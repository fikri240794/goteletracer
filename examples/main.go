// Package main demonstrates the usage of goteletracer library
// This example shows various scenarios from simple to complex usage patterns
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/fikri240794/goteletracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	fmt.Println("=== GoTeleTracer Examples ===")

	// Example 1: Simple usage with nil config (noop tracer)
	fmt.Println("1. Simple usage with nil config (noop tracer)")
	runSimpleNoopExample()

	// Example 2: Basic tracer with configuration but invalid endpoint
	fmt.Println("\n2. Basic tracer with configuration (will fallback to noop due to invalid endpoint)")
	runBasicExample()

	// Example 3: Advanced usage with TracerProvider (recommended approach)
	fmt.Println("\n3. Advanced usage with TracerProvider (recommended)")
	runAdvancedExample()

	// Example 4: Error handling and validation
	fmt.Println("\n4. Error handling and configuration validation")
	runErrorHandlingExample()

	// Example 5: Complex nested spans with attributes and events
	fmt.Println("\n5. Complex nested spans with attributes and events")
	runComplexTracingExample()

	// Example 6: Graceful shutdown example
	fmt.Println("\n6. Graceful shutdown example")
	runShutdownExample()

	fmt.Println("\n=== All examples completed ===")
}

// runSimpleNoopExample demonstrates the simplest usage with nil config
func runSimpleNoopExample() {
	// Create a tracer with nil config - returns noop tracer
	tracer := goteletracer.NewTracer(nil)

	// Use the tracer (this will be a noop operation)
	result := performCalculation(context.Background(), tracer, 10, 20)
	fmt.Printf("Calculation result: %d (traced with noop tracer)\n", result)
}

// runBasicExample demonstrates basic usage with configuration
func runBasicExample() {
	// Create configuration for OpenTelemetry tracer
	// Note: This uses a mock endpoint that won't be reachable
	config := &goteletracer.Config{
		ServiceName:         "example-service",
		ExporterGRPCAddress: "localhost:4317", // This will likely fail to connect
	}

	// Create tracer - will fallback to noop if connection fails
	tracer := goteletracer.NewTracer(config)

	// Use the tracer
	result := performCalculation(context.Background(), tracer, 15, 25)
	fmt.Printf("Calculation result: %d (traced with basic tracer)\n", result)
}

// runAdvancedExample demonstrates the recommended TracerProvider approach
func runAdvancedExample() {
	// Create configuration with custom shutdown timeout
	config := &goteletracer.Config{
		ServiceName:         "advanced-example-service",
		ExporterGRPCAddress: "localhost:4317",
		ShutdownTimeout:     10 * time.Second,
	}

	// Create TracerProvider (recommended approach for production)
	provider, err := goteletracer.NewTracerProvider(config)
	if err != nil {
		fmt.Printf("Failed to create tracer provider: %v\n", err)
		fmt.Println("This is expected if no OTLP collector is running on localhost:4317")
		return
	}

	// Get tracer from provider
	tracer := provider.Tracer()

	// Use the tracer
	result := performDatabaseOperation(context.Background(), tracer, "user_123")
	fmt.Printf("Database operation result: %s (traced with TracerProvider)\n", result)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
	} else {
		fmt.Println("TracerProvider shutdown successfully")
	}
}

// runErrorHandlingExample demonstrates error handling and validation
func runErrorHandlingExample() {
	fmt.Println("Testing various error conditions:")

	// Test invalid configurations
	invalidConfigs := []*goteletracer.Config{
		nil, // nil config
		{ServiceName: "", ExporterGRPCAddress: "localhost:4317"},      // empty service name
		{ServiceName: "test", ExporterGRPCAddress: ""},                // empty address
		{ServiceName: "test", ExporterGRPCAddress: "invalid-address"}, // invalid address format
	}

	for i, config := range invalidConfigs {
		provider, err := goteletracer.NewTracerProvider(config)
		if err != nil {
			fmt.Printf("  Config %d: Error (expected): %v\n", i+1, err)
		} else {
			fmt.Printf("  Config %d: Unexpected success\n", i+1)
			if provider != nil {
				provider.Shutdown(context.Background())
			}
		}
	}
}

// runComplexTracingExample demonstrates complex tracing scenarios
func runComplexTracingExample() {
	// Use noop tracer for this example to avoid connection issues
	tracer := goteletracer.NewTracer(nil)

	// Simulate a complex business operation
	ctx := context.Background()
	err := processUserOrder(ctx, tracer, "order_456", "user_789")

	if err != nil {
		fmt.Printf("Order processing failed: %v\n", err)
	} else {
		fmt.Println("Order processed successfully with complex tracing")
	}
}

// runShutdownExample demonstrates proper shutdown handling
func runShutdownExample() {
	config := &goteletracer.Config{
		ServiceName:         "shutdown-example",
		ExporterGRPCAddress: "localhost:4317",
		ShutdownTimeout:     5 * time.Second,
	}

	provider, err := goteletracer.NewTracerProvider(config)
	if err != nil {
		fmt.Printf("Provider creation failed (expected): %v\n", err)
		return
	}

	// Simulate some work
	tracer := provider.Tracer()
	_, span := tracer.Start(context.Background(), "shutdown-example-work")
	time.Sleep(100 * time.Millisecond) // Simulate work
	span.End()

	// Test multiple shutdowns (should be safe)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err1 := provider.Shutdown(shutdownCtx)
	err2 := provider.Shutdown(shutdownCtx) // Second shutdown should be safe

	fmt.Printf("First shutdown error: %v\n", err1)
	fmt.Printf("Second shutdown error: %v\n", err2)
	fmt.Println("Shutdown example completed")
}

// performCalculation demonstrates simple span creation with arithmetic operation
func performCalculation(ctx context.Context, tracer trace.Tracer, a, b int) int {
	ctx, span := tracer.Start(ctx, "perform_calculation")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.Int("input.a", a),
		attribute.Int("input.b", b),
	)

	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)

	result := a + b
	span.SetAttributes(attribute.Int("result", result))

	return result
}

// performDatabaseOperation demonstrates span with events and status
func performDatabaseOperation(ctx context.Context, tracer trace.Tracer, userID string) string {
	ctx, span := tracer.Start(ctx, "database_operation")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
		attribute.String("user.id", userID),
	)

	// Add span event
	span.AddEvent("query_started", trace.WithAttributes(
		attribute.String("query", "SELECT * FROM users WHERE id = ?"),
	))

	// Simulate database query time
	time.Sleep(50 * time.Millisecond)

	// Simulate random success/failure
	if rand.Float32() < 0.1 { // 10% chance of failure
		span.RecordError(fmt.Errorf("database connection timeout"))
		span.SetStatus(codes.Error, "Database operation failed")
		return ""
	}

	span.AddEvent("query_completed")
	span.SetStatus(codes.Ok, "Database operation successful")

	return fmt.Sprintf("User data for %s", userID)
}

// processUserOrder demonstrates nested spans with complex business logic
func processUserOrder(ctx context.Context, tracer trace.Tracer, orderID, userID string) error {
	ctx, span := tracer.Start(ctx, "process_user_order")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.String("user.id", userID),
	)

	// Step 1: Validate user
	if err := validateUser(ctx, tracer, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "User validation failed")
		return err
	}

	// Step 2: Process payment
	if err := processPayment(ctx, tracer, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Payment processing failed")
		return err
	}

	// Step 3: Update inventory
	if err := updateInventory(ctx, tracer, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Inventory update failed")
		return err
	}

	// Step 4: Send notification
	if err := sendNotification(ctx, tracer, userID, orderID); err != nil {
		// Non-critical error - log but don't fail the order
		span.AddEvent("notification_failed", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))
	}

	span.SetStatus(codes.Ok, "Order processed successfully")
	return nil
}

// validateUser demonstrates a child span for user validation
func validateUser(ctx context.Context, tracer trace.Tracer, userID string) error {
	ctx, span := tracer.Start(ctx, "validate_user")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", userID))

	// Simulate validation logic
	time.Sleep(20 * time.Millisecond)

	if userID == "" {
		err := fmt.Errorf("user ID cannot be empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid user ID")
		return err
	}

	span.SetStatus(codes.Ok, "User validation successful")
	return nil
}

// processPayment demonstrates payment processing with spans
func processPayment(ctx context.Context, tracer trace.Tracer, orderID string) error {
	ctx, span := tracer.Start(ctx, "process_payment")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.String("payment.method", "credit_card"),
	)

	// Simulate payment processing
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional payment failure
	if rand.Float32() < 0.05 { // 5% chance of failure
		err := fmt.Errorf("payment declined")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Payment processing failed")
		return err
	}

	span.SetAttributes(attribute.String("payment.status", "approved"))
	span.SetStatus(codes.Ok, "Payment processed successfully")
	return nil
}

// updateInventory demonstrates inventory management with spans
func updateInventory(ctx context.Context, tracer trace.Tracer, orderID string) error {
	ctx, span := tracer.Start(ctx, "update_inventory")
	defer span.End()

	span.SetAttributes(attribute.String("order.id", orderID))

	// Simulate inventory update
	time.Sleep(30 * time.Millisecond)

	span.AddEvent("inventory_lock_acquired")

	// Simulate inventory check and update
	time.Sleep(20 * time.Millisecond)

	span.AddEvent("inventory_updated", trace.WithAttributes(
		attribute.Int("items.updated", 3),
	))

	span.SetStatus(codes.Ok, "Inventory updated successfully")
	return nil
}

// sendNotification demonstrates notification sending with spans
func sendNotification(ctx context.Context, tracer trace.Tracer, userID, orderID string) error {
	ctx, span := tracer.Start(ctx, "send_notification")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", userID),
		attribute.String("order.id", orderID),
		attribute.String("notification.type", "email"),
	)

	// Simulate notification sending
	time.Sleep(40 * time.Millisecond)

	// Simulate occasional notification failure
	if rand.Float32() < 0.15 { // 15% chance of failure
		err := fmt.Errorf("email service unavailable")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Notification sending failed")
		return err
	}

	span.SetStatus(codes.Ok, "Notification sent successfully")
	return nil
}

func init() {
	// Initialize random seed for demo purposes
	rand.Seed(time.Now().UnixNano())

	// Set up basic logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
