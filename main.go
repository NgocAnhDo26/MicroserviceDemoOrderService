package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type OrderCreateRequest struct {
	UserID     int   `json:"userId"`
	ProductIds []int `json:"productIds"`
}

type Order struct {
	ID          int         `json:"id"`
	UserID      int         `json:"userId"`
	TotalAmount float64     `json:"totalAmount"`
	OrderDate   time.Time   `json:"orderDate"`
	OrderItems  []OrderItem `json:"orderItems,omitempty"`
}

type OrderItem struct {
	ID        int `json:"id"`
	OrderID   int `json:"orderId"`
	ProductID int `json:"productId"`
}

type Product struct {
	ID    string  `json:"_id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type App struct {
	DB     *pgxpool.Pool
	Client *http.Client
}

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Printf("ERROR: Returning status %d with message: %s", statusCode, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(map[string]string{"error": message})
	if err != nil {
		return
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not load .env file")
	}

	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	app := &App{
		DB:     dbpool,
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	r := mux.NewRouter()

	r.HandleFunc("/api/orders", app.createOrderHandler).Methods("POST")
	r.HandleFunc("/api/orders", app.getAllOrdersHandler).Methods("GET")
	r.HandleFunc("/api/orders/user/{id}", app.getOrdersByUserIDHandler).Methods("GET")

	// 2. All CORS configuration has been removed from this service.
	// The API Gateway is now responsible for handling CORS.

	log.Println("OrderService starting on port 8080...")
	// 3. Start the server with just the router, no CORS wrapper.
	log.Fatal(http.ListenAndServe(":8080", r))
}

// --- Handlers (getAllOrdersHandler, getOrdersByUserIDHandler, createOrderHandler) ---
// Note: The logic inside the handler functions does not need to change.
// They are omitted here for brevity but are the same as the previous version.

func (a *App) getAllOrdersHandler(w http.ResponseWriter, _ *http.Request) {
	log.Println("--- DEBUG[GetAll]: Starting getAllOrdersHandler ---")
	w.Header().Set("Content-Type", "application/json")

	var orders []*Order
	orderQuery := `SELECT id, userid, totalamount, orderdate FROM orders ORDER BY orderdate DESC`

	rows, err := a.DB.Query(context.Background(), orderQuery)
	if err != nil {
		writeJSONError(w, "Failed to query all orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orderMap := make(map[int]*Order)

	for rows.Next() {
		o := &Order{}
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalAmount, &o.OrderDate); err != nil {
			writeJSONError(w, "Failed to scan order row", http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
		orderMap[o.ID] = o
	}
	if err := rows.Err(); err != nil {
		writeJSONError(w, "Error after iterating order rows", http.StatusInternalServerError)
		return
	}
	log.Printf("--- DEBUG[GetAll]: Found %d orders.", len(orders))

	if len(orders) == 0 {
		_ = json.NewEncoder(w).Encode([]*Order{})
		return
	}

	var orderIDs []int
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
	}

	log.Printf("--- DEBUG[GetAll]: Fetching items for all %d order IDs ---", len(orderIDs))
	itemsQuery := `SELECT id, orderid, productid FROM orderitems WHERE orderid = ANY($1)`
	itemRows, err := a.DB.Query(context.Background(), itemsQuery, orderIDs)
	if err != nil {
		writeJSONError(w, "Failed to query order items", http.StatusInternalServerError)
		return
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item OrderItem
		if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID); err != nil {
			writeJSONError(w, "Failed to scan order item row", http.StatusInternalServerError)
			return
		}

		if order, found := orderMap[item.OrderID]; found {
			order.OrderItems = append(order.OrderItems, item)
		}
	}

	err = json.NewEncoder(w).Encode(orders)
	if err != nil {
		return
	}
	log.Println("--- DEBUG[GetAll]: Finished getAllOrdersHandler and sent response. ---")
}

func (a *App) getOrdersByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userIDStr, ok := vars["id"]
	if !ok {
		writeJSONError(w, "User ID not provided", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeJSONError(w, "Invalid User ID format", http.StatusBadRequest)
		return
	}

	var orders []*Order
	orderQuery := `SELECT id, userid, totalamount, orderdate FROM orders WHERE userid = $1 ORDER BY orderdate DESC`

	rows, err := a.DB.Query(context.Background(), orderQuery, userID)
	if err != nil {
		log.Printf("Error querying orders for user %d: %v. Returning empty list.", userID, err)
		_ = json.NewEncoder(w).Encode([]*Order{})
		return
	}
	defer rows.Close()

	orderMap := make(map[int]*Order)

	for rows.Next() {
		o := &Order{}
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalAmount, &o.OrderDate); err != nil {
			log.Printf("Error scanning order row for user %d: %v", userID, err)
			writeJSONError(w, "Failed to process user orders", http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
		orderMap[o.ID] = o
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error after iterating user order rows: %v. Returning empty list.", err)
		_ = json.NewEncoder(w).Encode([]*Order{})
		return
	}

	if len(orders) == 0 {
		_ = json.NewEncoder(w).Encode(orders)
		return
	}

	var orderIDs []int
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
	}

	itemsQuery := `SELECT id, orderid, productid FROM orderitems WHERE orderid = ANY($1)`
	itemRows, err := a.DB.Query(context.Background(), itemsQuery, orderIDs)
	if err != nil {
		log.Printf("Error querying order items for orders %v: %v. Returning orders without item details.", orderIDs, err)
	} else {
		defer itemRows.Close()
		for itemRows.Next() {
			var item OrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID); err != nil {
				log.Printf("Error scanning order item row: %v", err)
				continue
			}
			if order, found := orderMap[item.OrderID]; found {
				order.OrderItems = append(order.OrderItems, item)
			}
		}
	}

	err = json.NewEncoder(w).Encode(orders)
	if err != nil {
		return
	}
}

func (a *App) createOrderHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("--- DEBUG: Starting createOrderHandler ---")
	w.Header().Set("Content-Type", "application/json")

	var req OrderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("--- DEBUG: Decoded request: UserID=%d, ProductIds=%v", req.UserID, req.ProductIds)

	// 1. Validate User
	log.Println("--- DEBUG: Step 1: Validating user ---")
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:7001" // Default for local dev
	}
	userResp, err := a.Client.Get(fmt.Sprintf("%s/api/users/%d", userServiceURL, req.UserID))
	if err != nil {
		writeJSONError(w, "Failed to contact user service", http.StatusServiceUnavailable)
		return
	}
	if userResp.StatusCode != http.StatusOK {
		writeJSONError(w, fmt.Sprintf("User with ID %d not found", req.UserID), userResp.StatusCode)
		return
	}
	err = userResp.Body.Close()
	if err != nil {
		return
	}
	log.Println("--- DEBUG: User validation successful ---")

	// 2. Get Product Details and Calculate Total
	log.Println("--- DEBUG: Step 2: Fetching product details ---")
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "http://localhost:8082" // Default for local dev
	}
	var totalAmount float64
	for _, pid := range req.ProductIds {
		log.Printf("--- DEBUG: Fetching product with ID: %d ---", pid)
		resp, err := a.Client.Get(fmt.Sprintf("%s/api/products/%d", productServiceURL, pid))
		if err != nil {
			writeJSONError(w, "Failed to contact product service", http.StatusServiceUnavailable)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			errorMsg := fmt.Sprintf("Failed to fetch product with ID %d; downstream service returned status %d", pid, resp.StatusCode)
			writeJSONError(w, errorMsg, resp.StatusCode)
			return
		}
		var p Product
		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			log.Printf("Error decoding product JSON: %v", err)
			writeJSONError(w, "Failed to decode product data", http.StatusInternalServerError)
			return
		}
		log.Printf("--- DEBUG: Fetched product %d with price %.2f ---", pid, p.Price)
		totalAmount += p.Price
	}
	log.Printf("--- DEBUG: Total amount calculated: %.2f ---", totalAmount)

	// 3. Save to Database within a Transaction
	log.Println("--- DEBUG: Step 3: Starting database transaction ---")
	ctx := context.Background()
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		writeJSONError(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			log.Printf("Warning: Failed to rollback transaction: %v", err)
		} else {
			log.Println("--- DEBUG: Transaction rolled back successfully ---")
		}
	}(tx, ctx)

	log.Println("--- DEBUG: Inserting into 'orders' table ---")
	var orderID int
	orderSQL := `INSERT INTO orders (userid, totalamount, orderdate) VALUES ($1, $2, $3) RETURNING id`
	err = tx.QueryRow(ctx, orderSQL, req.UserID, totalAmount, time.Now()).Scan(&orderID)
	if err != nil {
		writeJSONError(w, "Failed to create order", http.StatusInternalServerError)
		return
	}
	log.Printf("--- DEBUG: Inserted into 'orders' with new orderID: %d ---", orderID)

	log.Println("--- DEBUG: Preparing statement for 'orderitems' table ---")
	_, err = tx.Prepare(ctx, "order_item_insert", `INSERT INTO orderitems (orderid, productid) VALUES ($1, $2)`)
	if err != nil {
		writeJSONError(w, "Failed to prepare statement for order items", http.StatusInternalServerError)
		return
	}

	for _, pid := range req.ProductIds {
		log.Printf("--- DEBUG: Inserting into 'orderitems': orderID=%d, productID=%d ---", orderID, pid)
		_, err := tx.Exec(ctx, "order_item_insert", orderID, pid)
		if err != nil {
			writeJSONError(w, "Failed to create order item", http.StatusInternalServerError)
			return
		}
	}
	log.Println("--- DEBUG: All order items inserted ---")

	log.Println("--- DEBUG: Committing transaction ---")
	if err := tx.Commit(ctx); err != nil {
		writeJSONError(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}
	log.Println("--- DEBUG: Transaction committed successfully ---")

	log.Printf("--- DEBUG: Fetching newly created order %d to return to client ---", orderID)
	var newOrder Order
	err = a.DB.QueryRow(context.Background(), `SELECT id, userid, totalamount, orderdate FROM orders WHERE id = $1`, orderID).Scan(&newOrder.ID, &newOrder.UserID, &newOrder.TotalAmount, &newOrder.OrderDate)
	if err != nil {
		writeJSONError(w, "Failed to fetch created order details", http.StatusInternalServerError)
		return
	}

	itemRows, err := a.DB.Query(context.Background(), `SELECT id, orderid, productid FROM orderitems WHERE orderid = $1`, orderID)
	if err != nil {
		log.Printf("Warning: could not fetch items for newly created order %d: %v", orderID, err)
	} else {
		defer itemRows.Close()
		for itemRows.Next() {
			var item OrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID); err != nil {
				log.Printf("Warning: could not scan item for newly created order %d: %v", orderID, err)
				continue
			}
			newOrder.OrderItems = append(newOrder.OrderItems, item)
		}
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(newOrder)
	if err != nil {
		return
	}
	log.Println("--- DEBUG: createOrderHandler finished successfully, returned full order object ---")
}
