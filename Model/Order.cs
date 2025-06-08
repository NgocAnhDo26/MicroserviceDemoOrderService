using System.ComponentModel.DataAnnotations.Schema;
using OrderService.Model;

namespace OrderService.Models;

public class Order
{
    public int Id { get; set; }
    public int UserId { get; set; }
    public decimal TotalAmount { get; set; }
    public DateTime OrderDate { get; set; }
    public virtual ICollection<OrderItem> OrderItems { get; set; } = new List<OrderItem>();

    // --- These properties are for display/DTO purposes only ---
    [NotMapped]
    public string? UserName { get; set; }

    [NotMapped]
    public List<string> ProductNames { get; set; } = new List<string>();

    [NotMapped]
    public List<int> ProductIds => OrderItems.Select(oi => oi.ProductId).ToList();
}

public class OrderCreateRequest
{
    public int UserId { get; set; }
    public List<int> ProductIds { get; set; } = new List<int>();
}