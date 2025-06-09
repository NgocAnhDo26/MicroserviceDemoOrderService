using System.ComponentModel.DataAnnotations.Schema;

namespace MicroserviceDemoOrderService.Model;

public sealed class Order
{
    public int Id { get; set; }
    public int UserId { get; set; }
    public decimal TotalAmount { get; set; }
    public DateTime OrderDate { get; set; }
    public ICollection<OrderItem> OrderItems { get; set; } = new List<OrderItem>();

    // --- These properties are for display/DTO purposes only ---
    [NotMapped]
    public string? UserName { get; set; }

    [NotMapped]
    public List<string> ProductNames { get; set; } = [];

    [NotMapped]
    public List<int> ProductIds => OrderItems.Select(oi => oi.ProductId).ToList();
}

public class OrderCreateRequest
{
    public int UserId { get; set; }
    public List<int> ProductIds { get; set; } = [];
}