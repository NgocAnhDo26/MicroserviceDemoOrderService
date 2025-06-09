using MicroserviceDemoOrderService.Data;
using MicroserviceDemoOrderService.DTOs;
using MicroserviceDemoOrderService.Model;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace MicroserviceDemoOrderService.Controllers;

[Route("api/[controller]")]
[ApiController]
public class OrdersController(
    OrderDbContext context,
    IHttpClientFactory httpClientFactory,
    IConfiguration configuration)
    : ControllerBase
{
    // Use the context

    // Inject the DbContext

    // --- GET Methods ---
    // When retrieving orders, we should include the related items.
    [HttpGet]
    public async Task<ActionResult<IEnumerable<Order>>> GetOrders()
    {
        // Use Include() to load the related OrderItems
        return await context.Orders.Include(o => o.OrderItems).ToListAsync();
    }

    [HttpGet("{id:int}")]
    public async Task<ActionResult<Order>> GetOrder(int id)
    {
        var order = await context.Orders
            .Include(o => o.OrderItems) // Eager to load the items
            .FirstOrDefaultAsync(o => o.Id == id);

        if (order == null)
        {
            return NotFound();
        }

        return order;
    }


    // --- POST Method (The main change) ---
    [HttpPost]
    public async Task<ActionResult<Order>> CreateOrder([FromBody] OrderCreateRequest orderRequest)
    {
        if (orderRequest.ProductIds.Count == 0)
        {
            return BadRequest("Order request is invalid or has no products.");
        }

        var client = httpClientFactory.CreateClient();
        var userServiceUrl = configuration["ServiceUrls:UserService"];
        var productServiceUrl = configuration["ServiceUrls:ProductService"];

        // 1. Validate User (same as before)
        var userResponse = await client.GetAsync($"{userServiceUrl}/api/users/{orderRequest.UserId}");
        if (!userResponse.IsSuccessStatusCode)
        {
            return BadRequest($"User with ID {orderRequest.UserId} not found or UserService error.");
        }

        var user = await userResponse.Content.ReadFromJsonAsync<UserDto>();
        if (user == null) return BadRequest("Failed to deserialize user data.");


        // 2. Get Product Details and Calculate Total (same as before)
        decimal totalAmount = 0;
        var productNames = new List<string>();
        foreach (var productId in orderRequest.ProductIds)
        {
            var productResponse = await client.GetAsync($"{productServiceUrl}/api/products/{productId}");
            if (!productResponse.IsSuccessStatusCode)
            {
                return BadRequest($"Product with ID {productId} not found or ProductService error.");
            }

            var product = await productResponse.Content.ReadFromJsonAsync<ProductDto>();
            if (product == null) return BadRequest($"Failed to deserialize product data for ID {productId}.");

            productNames.Add(product.Name);
            totalAmount += product.Price;
        }

        // 3. Create the Order and OrderItem entities
        var newOrder = new Order
        {
            UserId = orderRequest.UserId,
            TotalAmount = totalAmount,
            OrderDate = DateTime.UtcNow,
            // Assign display-only properties
            UserName = user.Name,
            ProductNames = productNames
        };

        // Create an OrderItem for each product ID and add it to the order
        foreach (var orderItem in orderRequest.ProductIds.Select(productId => new OrderItem
                 {
                     ProductId = productId,
                     Order = newOrder // EF Core will automatically link this
                 }))
        {
            newOrder.OrderItems.Add(orderItem);
        }

        // 4. Save to the database
        context.Orders.Add(newOrder);
        await context.SaveChangesAsync(); // This saves the Order and all its related OrderItems in one transaction

        return CreatedAtAction(nameof(GetOrder), new { id = newOrder.Id }, newOrder);
    }
}