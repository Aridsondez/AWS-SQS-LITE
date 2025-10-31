package store

import (
	"context"
	"time"

	"github.com/aridsondez/AWS-SQS-LITE/internal/queue"
)

// Store is the DB-agnostic interface the rest of the app uses.
type Store interface {
	// Enqueue inserts a message (delay can be 0).
	Enqueue(ctx context.Context, m queue.Message, delay time.Duration) (int64, error)

	// Claim atomically leases up to Limit messages from a queue.
	Claim(ctx context.Context, opts queue.ClaimOptions) ([]queue.Message, error)

	// Ack deletes the message by ID; returns true if deleted.
	Ack(ctx context.Context, id int64) (bool, error)
}
