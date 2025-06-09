using MicroserviceDemoOrderService.Data;
using Microsoft.EntityFrameworkCore;
using System.Text.Json.Serialization; // Add this using directive

namespace MicroserviceDemoOrderService;

public static class Program
{
    public static void Main(string[] args)
    {
        const string myAllowSpecificOrigins = "_myAllowSpecificOrigins";

        var builder = WebApplication.CreateBuilder(args);

        // Add services to the container.
        var connectionString = builder.Configuration.GetConnectionString("DefaultConnection");
        builder.Services.AddDbContext<OrderDbContext>(options =>
            options.UseSqlServer(connectionString));

        builder.Services.AddCors(options =>
        {
            options.AddPolicy(name: myAllowSpecificOrigins,
                policy =>
                {
                    policy.WithOrigins("https://simple-microservice-demo.vercel.app",
                                       "http://localhost:3000")
                          .AllowAnyHeader()
                          .AllowAnyMethod();
                });
        });

        // Configure JSON options to handle cycles
        builder.Services.AddControllers().AddJsonOptions(options =>
        {
            options.JsonSerializerOptions.ReferenceHandler = ReferenceHandler.Preserve;
        });
        
        // Learn more about configuring Swagger/OpenAPI at https://aka.ms/aspnetcore/swashbuckle
        builder.Services.AddEndpointsApiExplorer();
        // builder.Services.AddSwaggerGen();
        builder.Services.AddHttpClient();

        var app = builder.Build();

        // Configure the HTTP request pipeline.
        if (app.Environment.IsDevelopment())
        {
            // app.UseSwagger();
            // app.UseSwaggerUI();
        }

        app.UseHttpsRedirection();

        app.UseCors(myAllowSpecificOrigins);

        app.UseAuthorization();


        app.MapControllers();

        app.Run();
    }
}