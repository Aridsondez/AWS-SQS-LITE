package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aridsondez/AWS-SQS-LITE/pkg/worker"
)

// Example: Order processing worker
func main() {
	// Create worker
	w := worker.New(worker.Config{
		BaseURL:    "http://localhost:8080",
		PollDelay:  1 * time.Second,
		BatchSize:  10,
		Visibility: 30 * time.Second,
	})

	// Register handlers for different queues
	w.Handle("orders", processOrder)
	w.Handle("emails", sendEmail)
	w.Handle("notifications", sendNotification)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("🚀 Worker started - press Ctrl+C to stop")

	// Run worker (blocks until Ctrl+C)
	if err := w.Run(ctx); err != nil {
		log.Fatalf("Worker error: %v", err)
	}

	log.Println("👋 Worker stopped gracefully")
}

// processOrder handles order processing
func processOrder(ctx context.Context, msg *worker.Message) error {
	var order struct {
		OrderID  string  `json:"order_id"`
		Customer string  `json:"customer"`
		Amount   float64 `json:"amount"`
	}

	if err := json.Unmarshal(msg.Body, &order); err != nil {
		return fmt.Errorf("invalid order data: %w", err)
	}

	log.Printf("📦 Processing order %s for %s ($%.2f)",
		order.OrderID, order.Customer, order.Amount)

	// Simulate processing time
	time.Sleep(2 * time.Second)

	// Simulate business logic
	if err := chargeCustomer(order.OrderID, order.Amount); err != nil {
		return fmt.Errorf("charge failed: %w", err)
	}

	if err := updateInventory(order.OrderID); err != nil {
		return fmt.Errorf("inventory update failed: %w", err)
	}

	log.Printf("✅ Order %s completed successfully", order.OrderID)
	return nil
}

// sendEmail handles email sending
func sendEmail(ctx context.Context, msg *worker.Message) error {
	var email struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := json.Unmarshal(msg.Body, &email); err != nil {
		return fmt.Errorf("invalid email data: %w", err)
	}

	log.Printf("📧 Sending email to %s: %s", email.To, email.Subject)

	// Simulate email sending
	time.Sleep(1 * time.Second)

	log.Printf("✅ Email sent to %s", email.To)
	return nil
}

// sendNotification handles push notifications
func sendNotification(ctx context.Context, msg *worker.Message) error {
	var notification struct {
		UserID  string `json:"user_id"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(msg.Body, &notification); err != nil {
		return fmt.Errorf("invalid notification data: %w", err)
	}

	log.Printf("🔔 Sending notification to user %s: %s",
		notification.UserID, notification.Message)

	// Simulate notification sending
	time.Sleep(500 * time.Millisecond)

	log.Printf("✅ Notification sent to user %s", notification.UserID)
	return nil
}

// Mock functions (in real app, these would call actual services)
func chargeCustomer(orderID string, amount float64) error {
	log.Printf("  💳 Charging customer $%.2f for order %s", amount, orderID)
	return nil
}

func updateInventory(orderID string) error {
	log.Printf("  📊 Updating inventory for order %s", orderID)
	return nil
}
