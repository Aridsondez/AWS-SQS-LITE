package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// HandlerFunc processes a message and returns an error if processing failed.
// Returning nil means success (message will be acked).
// Returning an error means failure (message will be requeued).
type HandlerFunc func(ctx context.Context, msg *Message) error

// Message represents a message received from the queue
type Message struct {
	ID            int64           `json:"id"`
	Body          json.RawMessage `json:"body"`
	Receipt       string          `json:"receipt"`
	LeaseUntil    *time.Time      `json:"lease_until,omitempty"`
	DeliveryCount int             `json:"delivery_count"`
	MaxRetries    int             `json:"max_retries"`
	Queue         string          `json:"-"` // Set by worker
}

// Worker manages message processing from queues
type Worker struct {
	baseURL   string
	client    *http.Client
	handlers  map[string]HandlerFunc
	pollDelay time.Duration
	batchSize int
	visibility time.Duration
}

// Config for creating a new worker
type Config struct {
	BaseURL    string        // SQS Lite server URL
	PollDelay  time.Duration // Time between polling attempts (default: 1s)
	BatchSize  int           // Max messages to fetch per poll (default: 10)
	Visibility time.Duration // Visibility timeout (default: 30s)
}

// New creates a new Worker with the given configuration
func New(cfg Config) *Worker {
	if cfg.PollDelay == 0 {
		cfg.PollDelay = 1 * time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10
	}
	if cfg.Visibility == 0 {
		cfg.Visibility = 30 * time.Second
	}

	return &Worker{
		baseURL:    cfg.BaseURL,
		client:     &http.Client{Timeout: 10 * time.Second},
		handlers:   make(map[string]HandlerFunc),
		pollDelay:  cfg.PollDelay,
		batchSize:  cfg.BatchSize,
		visibility: cfg.Visibility,
	}
}

// Handle registers a handler function for a specific queue
func (w *Worker) Handle(queue string, handler HandlerFunc) {
	w.handlers[queue] = handler
	log.Printf("Registered handler for queue: %s", queue)
}

// Run starts the worker and blocks until context is cancelled
func (w *Worker) Run(ctx context.Context) error {
	if len(w.handlers) == 0 {
		return fmt.Errorf("no handlers registered")
	}

	log.Printf("Worker starting with %d queue(s)", len(w.handlers))

	// Start a goroutine for each queue
	for queue, handler := range w.handlers {
		go w.pollQueue(ctx, queue, handler)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Worker shutting down...")
	return nil
}

// pollQueue continuously polls a queue and processes messages
func (w *Worker) pollQueue(ctx context.Context, queue string, handler HandlerFunc) {
	ticker := time.NewTicker(w.pollDelay)
	defer ticker.Stop()

	log.Printf("Started polling queue: %s", queue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopped polling queue: %s", queue)
			return

		case <-ticker.C:
			messages, err := w.receiveMessages(ctx, queue)
			if err != nil {
				log.Printf("Error receiving from %s: %v", queue, err)
				continue
			}

			if len(messages) == 0 {
				continue // No messages available
			}

			log.Printf("Received %d message(s) from %s", len(messages), queue)

			// Process each message
			for _, msg := range messages {
				msg.Queue = queue
				w.processMessage(ctx, msg, handler)
			}
		}
	}
}

// processMessage handles a single message with error recovery
func (w *Worker) processMessage(ctx context.Context, msg *Message, handler HandlerFunc) {
	// Create a timeout context for the handler
	handlerCtx, cancel := context.WithTimeout(ctx, w.visibility-5*time.Second)
	defer cancel()

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC processing message %d from %s: %v (will requeue)",
				msg.ID, msg.Queue, r)
			// Don't ack - let it requeue
		}
	}()

	// Call the handler
	err := handler(handlerCtx, msg)

	if err != nil {
		log.Printf("Error processing message %d from %s (attempt %d/%d): %v",
			msg.ID, msg.Queue, msg.DeliveryCount, msg.MaxRetries, err)
		// Don't ack - let sweeper requeue or route to DLQ
		return
	}

	// Success - acknowledge the message
	if err := w.ackMessage(ctx, msg.ID); err != nil {
		log.Printf("Error acking message %d: %v", msg.ID, err)
		return
	}

	log.Printf("✓ Successfully processed message %d from %s", msg.ID, msg.Queue)
}

// receiveMessages fetches messages from a queue
func (w *Worker) receiveMessages(ctx context.Context, queue string) ([]*Message, error) {
	reqBody := map[string]interface{}{
		"max":           w.batchSize,
		"visibility_ms": int(w.visibility.Milliseconds()),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/queues/%s:receive", w.baseURL, queue)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("receive failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var messages []*Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// ackMessage acknowledges a message
func (w *Worker) ackMessage(ctx context.Context, messageID int64) error {
	url := fmt.Sprintf("%s/v1/messages/%d:ack", w.baseURL, messageID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ack failed: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}
