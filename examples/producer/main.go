package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aridsondez/AWS-SQS-LITE/pkg/client"
)

// Example: Producer that enqueues tasks
func main() {
	// Create client
	c := client.NewClient("http://localhost:8080")
	ctx := context.Background()

	log.Println("🚀 Producer demo - enqueuing tasks...")
	fmt.Println()

	// Example 1: Simple order
	orderID, err := c.Enqueue(ctx, "orders", map[string]interface{}{
		"order_id":  "ORD-001",
		"customer":  "Alice Johnson",
		"amount":    149.99,
		"items":     []string{"Widget", "Gadget"},
	}, nil)
	if err != nil {
		log.Fatalf("Failed to enqueue order: %v", err)
	}
	log.Printf("✓ Enqueued order (message ID: %d)", orderID)

	// Example 2: Email with custom options
	emailID, err := c.Enqueue(ctx, "emails", map[string]interface{}{
		"to":      "customer@example.com",
		"subject": "Order Confirmation",
		"body":    "Your order has been placed successfully!",
	}, &client.EnqueueOptions{
		MaxRetries: 3,
		TraceID:    "trace-12345",
	})
	if err != nil {
		log.Fatalf("Failed to enqueue email: %v", err)
	}
	log.Printf("✓ Enqueued email (message ID: %d)", emailID)

	// Example 3: Delayed notification
	notifID, err := c.Enqueue(ctx, "notifications", map[string]interface{}{
		"user_id": "user-789",
		"message": "Your order will arrive tomorrow!",
	}, &client.EnqueueOptions{
		Delay: 5 * time.Second, // Deliver after 5 seconds
	})
	if err != nil {
		log.Fatalf("Failed to enqueue notification: %v", err)
	}
	log.Printf("✓ Enqueued delayed notification (message ID: %d, delay: 5s)", notifID)

	// Example 4: Task with DLQ
	taskID, err := c.Enqueue(ctx, "orders", map[string]interface{}{
		"order_id": "ORD-002",
		"customer": "Bob Smith",
		"amount":   299.99,
	}, &client.EnqueueOptions{
		MaxRetries: 2,
		DLQ:        "failed-orders",
	})
	if err != nil {
		log.Fatalf("Failed to enqueue task: %v", err)
	}
	log.Printf("✓ Enqueued order with DLQ (message ID: %d)", taskID)

	fmt.Println()
	log.Println("✅ All tasks enqueued successfully!")
	log.Println("📊 Check worker logs to see processing")
	log.Println("📈 View metrics at: http://localhost:8080/metrics")
}
