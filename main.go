package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Models & DTOs ---

type OrderCreateRequest struct {
	UserID     int   `json:"userId"`
	ProductIds []int `json:"productIds"`
}

type Order struct {
	ID          int         `json:"id"`
	UserID      int         `json:"userId"`
	TotalAmount float64     `json:"totalAmount"`
	OrderDate   time.Time   `json:"orderDate"`
	OrderItems  []OrderItem `json:"orderItems"`
}

type OrderItem struct {
	ID        int `json:"id"`
	OrderID   int `json:"orderId"`
	ProductID int `json:"productId"`
}

// DTOs for external service responses
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Product struct {
	ID    string  `json:"_id"` // Note: Mongo uses _id
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// App struct to hold dependencies
type App struct {
	DB     *pgxpool.Pool
	Client *http.Client
}

func main() {

	err := godotenv.Load()
	if err != nil {
	
		log.Println("Warning: Could not load .env file")
	}
	// Database Connection
	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	app := &App{
		DB:     dbpool,
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	// Router
	r := mux.NewRouter()
	r.HandleFunc("/api/orders", app.createOrderHandler).Methods("POST")

	log.Println("OrderService starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func (a *App) createOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req OrderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Validate User (example call)
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	_, err := http.Get(fmt.Sprintf("%s/api/users/%d", userServiceURL, req.UserID))
	if err != nil {
		http.Error(w, "Failed to contact user service", http.StatusInternalServerError)
		return
	}

	// 2. Get Product Details and Calculate Total
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	var totalAmount float64
	for _, pid := range req.ProductIds {
		resp, err := http.Get(fmt.Sprintf("%s/api/products/%s", productServiceURL, pid)) // NOTE: This assumes Mongo IDs can be passed as int, which might not be true. In a real app, they'd be strings.
		if err != nil {
			http.Error(w, "Failed to contact product service", http.StatusInternalServerError)
			return
		}
		var p Product
		json.NewDecoder(resp.Body).Decode(&p)
		totalAmount += p.Price
		resp.Body.Close()
	}

	// 3. Save to Database within a Transaction
	ctx := context.Background()
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx) // Rollback if anything fails

	// Insert into Orders table
	var orderID int
	orderSQL := `INSERT INTO orders (userid, totalamount, orderdate) VALUES ($1, $2, $3) RETURNING id`
	err = tx.QueryRow(ctx, orderSQL, req.UserID, totalAmount, time.Now()).Scan(&orderID)
	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	// Insert into OrderItems table
	for _, pid := range req.ProductIds {
		itemSQL := `INSERT INTO orderitems (orderid, productid) VALUES ($1, $2)`
		_, err := tx.Exec(ctx, itemSQL, orderID, pid)
		if err != nil {
			http.Error(w, "Failed to create order item", http.StatusInternalServerError)
			return
		}
	}

	// If all good, commit the transaction
	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// For simplicity, we return a success message. A real app would fetch and return the created order.
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Order created successfully", "orderId": fmt.Sprintf("%d", orderID)})
}