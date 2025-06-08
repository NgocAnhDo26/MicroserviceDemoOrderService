using OrderService.Models;

namespace OrderService.Model
{
    public class OrderItem
    {
        public int Id { get; set; } // Primary key for the OrderItem table
        public int ProductId { get; set; }

        // Navigation property back to the Order
        public int OrderId { get; set; }
        public virtual Order Order { get; set; } = null!;
    }
}
