namespace MicroserviceDemoOrderService.Model
{
    public sealed class OrderItem
    {
        public int Id { get; set; } // Primary key for the OrderItem table
        public int ProductId { get; set; }

        // Navigation property back to the Order
        public int OrderId { get; set; }
        public Order Order { get; set; } = null!;
    }
}
