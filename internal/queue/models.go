package queue

import "time"

// Message is the durable queue row mapped to Go.
type Message struct {
	ID            int64
	Queue         string
	Body          []byte
	EnqueuedAt    time.Time
	NotBefore     time.Time
	LeaseUntil    *time.Time
	DeliveryCount int
	MaxRetries    int
	DLQ           *string
	TraceID       *string
}

// ClaimOptions controls how we receive messages.
type ClaimOptions struct {
	Queue      string
	Limit      int
	Visibility time.Duration
}
