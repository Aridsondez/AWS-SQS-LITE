# AWS SQS Lite - SDK Guide

Complete guide for using the Worker SDK and Client library.

---

## Installation

```bash
go get github.com/aridsondez/AWS-SQS-LITE/pkg/worker
go get github.com/aridsondez/AWS-SQS-LITE/pkg/client
```

---

## 🚀 Quick Start

### Producer (Enqueuing Tasks)

```go
package main

import (
    "context"
    "github.com/aridsondez/AWS-SQS-LITE/pkg/client"
)

func main() {
    // Create client
    c := client.NewClient("http://localhost:8080")

    // Enqueue a task
    messageID, err := c.Enqueue(context.Background(), "orders", map[string]interface{}{
        "order_id": "12345",
        "amount": 99.99,
    }, nil)

    if err != nil {
        panic(err)
    }

    log.Printf("Enqueued message: %d", messageID)
}
```

### Worker (Processing Tasks)

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/aridsondez/AWS-SQS-LITE/pkg/worker"
)

func main() {
    // Create worker
    w := worker.New(worker.Config{
        BaseURL: "http://localhost:8080",
    })

    // Register handler
    w.Handle("orders", processOrder)

    // Run worker (blocks)
    w.Run(context.Background())
}

func processOrder(ctx context.Context, msg *worker.Message) error {
    var order map[string]interface{}
    json.Unmarshal(msg.Body, &order)

    // Your business logic here
    log.Printf("Processing order: %v", order)

    return nil  // Success - message will be acked
}
```

---

## Client API

### Creating a Client

```go
c := client.NewClient("http://localhost:8080")
```

### Enqueue Options

```go
opts := &client.EnqueueOptions{
    Delay:      5 * time.Second,  // Delay delivery
    MaxRetries: 3,                 // Max retry attempts
    DLQ:        "failed-orders",   // Dead letter queue
    TraceID:    "trace-abc123",    // Correlation ID
}

messageID, err := c.Enqueue(ctx, "orders", payload, opts)
```

### Examples

#### Simple Enqueue
```go
id, err := c.Enqueue(ctx, "emails", map[string]string{
    "to": "user@example.com",
    "subject": "Welcome!",
}, nil)
```

#### Delayed Task
```go
id, err := c.Enqueue(ctx, "reminders", reminder, &client.EnqueueOptions{
    Delay: 1 * time.Hour,  // Send in 1 hour
})
```

#### With DLQ
```go
id, err := c.Enqueue(ctx, "payments", payment, &client.EnqueueOptions{
    MaxRetries: 2,
    DLQ: "failed-payments",
})
```

---

## 🔨 Worker SDK

### Configuration

```go
w := worker.New(worker.Config{
    BaseURL:    "http://localhost:8080",  // Required
    PollDelay:  1 * time.Second,          // Poll interval (default: 1s)
    BatchSize:  10,                       // Messages per poll (default: 10)
    Visibility: 30 * time.Second,         // Visibility timeout (default: 30s)
})
```

### Handler Function

```go
type HandlerFunc func(ctx context.Context, msg *Message) error
```

**Return `nil`** → Success (message acked)
**Return `error`** → Failure (message requeued)
**Panic** → Recovered, message requeued

### Message Structure

```go
type Message struct {
    ID            int64           // Message ID
    Body          json.RawMessage // Message payload
    Receipt       string          // Receipt handle
    LeaseUntil    *time.Time      // Lease expiration
    DeliveryCount int             // Retry attempt count
    MaxRetries    int             // Max allowed retries
    Queue         string          // Queue name
}
```

### Multiple Queues

```go
w := worker.New(config)

w.Handle("orders", processOrder)
w.Handle("emails", sendEmail)
w.Handle("notifications", sendNotification)

w.Run(ctx)  // Polls all queues in parallel
```

### Graceful Shutdown

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

w.Run(ctx)  // Stops on Ctrl+C
```

---

## 💡 Common Patterns

### Pattern 1: Database Updates

```go
func processOrder(ctx context.Context, msg *worker.Message) error {
    var order Order
    json.Unmarshal(msg.Body, &order)

    // Update database status
    db.UpdateOrderStatus(order.ID, "processing")

    // Do work
    if err := chargeCustomer(order); err != nil {
        db.UpdateOrderStatus(order.ID, "failed")
        return err  // Will requeue
    }

    // Mark complete
    db.UpdateOrderStatus(order.ID, "completed")
    return nil  // Success
}
```

### Pattern 2: External API Calls

```go
func sendEmail(ctx context.Context, msg *worker.Message) error {
    var email EmailTask
    json.Unmarshal(msg.Body, &email)

    // Call external service
    err := sendgridClient.Send(email)
    if err != nil {
        return fmt.Errorf("sendgrid error: %w", err)
    }

    return nil
}
```

### Pattern 3: Webhook Callbacks

```go
func processWithCallback(ctx context.Context, msg *worker.Message) error {
    var task struct {
        Data        map[string]interface{} `json:"data"`
        CallbackURL string                 `json:"callback_url"`
    }
    json.Unmarshal(msg.Body, &task)

    // Do work
    result := doWork(task.Data)

    // Notify via webhook
    http.Post(task.CallbackURL, "application/json", ...)

    return nil
}
```

### Pattern 4: Retry with Exponential Backoff

```go
func unreliableTask(ctx context.Context, msg *worker.Message) error {
    // Check delivery count for backoff
    if msg.DeliveryCount > 1 {
        backoff := time.Duration(msg.DeliveryCount) * 2 * time.Second
        log.Printf("Retry %d, backing off %v", msg.DeliveryCount, backoff)
        time.Sleep(backoff)
    }

    // Try the task
    return externalAPI.Call()
}
```

---

## Error Handling

### Success
```go
return nil  // Message acknowledged and deleted
```

### Retriable Error
```go
return fmt.Errorf("temporary failure")  // Message requeued
```

### Panic Recovery
```go
panic("something went wrong")  // Recovered, message requeued
```

### Dead Letter Queue
After `max_retries` failures, message automatically routes to DLQ.

---

## 📊 Monitoring

### Logging

The worker automatically logs:
- Messages received
- Processing success/failure
- Panics
- Ack errors

### Custom Logging

```go
func myHandler(ctx context.Context, msg *worker.Message) error {
    log.Printf("Processing message %d (attempt %d/%d)",
        msg.ID, msg.DeliveryCount, msg.MaxRetries)

    // Your logic

    return nil
}
```

### Metrics

Worker processing is tracked in Prometheus:
- `sqs_messages_received_total{queue}`
- `sqs_messages_acked_total`
- View at: `http://localhost:8080/metrics`

---

## 🎯 Best Practices

### 1. **Idempotency**
Messages may be delivered more than once. Make handlers idempotent:

```go
func processPayment(ctx context.Context, msg *worker.Message) error {
    var payment Payment
    json.Unmarshal(msg.Body, &payment)

    // Check if already processed
    if db.PaymentExists(payment.ID) {
        log.Printf("Payment %s already processed, skipping", payment.ID)
        return nil  // Ack without reprocessing
    }

    // Process payment
    return stripe.Charge(payment)
}
```

### 2. **Timeouts**
Use context timeouts for external calls:

```go
func callExternalAPI(ctx context.Context, msg *worker.Message) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return apiClient.CallWithContext(ctx, msg.Body)
}
```

### 3. **Structured Data**
Use structs for type safety:

```go
type OrderTask struct {
    OrderID  string  `json:"order_id"`
    Amount   float64 `json:"amount"`
    Customer string  `json:"customer"`
}

func processOrder(ctx context.Context, msg *worker.Message) error {
    var order OrderTask
    if err := json.Unmarshal(msg.Body, &order); err != nil {
        return fmt.Errorf("invalid order: %w", err)
    }

    // Type-safe access
    log.Printf("Processing order %s for %s", order.OrderID, order.Customer)
    return nil
}
```

### 4. **DLQ Configuration**
Always set DLQ for critical tasks:

```go
c.Enqueue(ctx, "payments", payment, &client.EnqueueOptions{
    MaxRetries: 3,
    DLQ: "failed-payments",  // Failed messages go here
})
```

### 5. **Graceful Shutdown**
Always use context cancellation:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

if err := w.Run(ctx); err != nil {
    log.Fatal(err)
}
```

---

## 🧪 Testing

### Mock Handler

```go
func TestHandler(t *testing.T) {
    msg := &worker.Message{
        ID:   1,
        Body: json.RawMessage(`{"test": "data"}`),
    }

    err := myHandler(context.Background(), msg)
    if err != nil {
        t.Errorf("Handler failed: %v", err)
    }
}
```

---

## 🔗 Examples

See `examples/` directory for complete examples:
- `examples/worker/` - Full worker implementation
- `examples/producer/` - Producer examples

Run them:
```bash
make run-worker    # Start worker
make run-producer  # Enqueue tasks
```

---

## 📚 Additional Resources

- [Main README](../README.md) - Project overview
- [API Reference](../README.md#api-reference) - HTTP API docs
- [Architecture](../README.md#architecture) - System design
