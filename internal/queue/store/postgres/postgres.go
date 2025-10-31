package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aridsondez/AWS-SQS-LITE/internal/queue"
	"github.com/aridsondez/AWS-SQS-LITE/internal/queue/store"
)

// Ensure *PostgresStore implements store.Store at compile time.
var _ store.Store = (*PostgresStore)(nil)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// helper: convert a Go duration to a Postgres interval literal like "12.500000s".
func toInterval(d time.Duration) string {
	// We’ll use seconds with fractional precision.
	return fmt.Sprintf("%fs", d.Seconds())
}

// SQL templates
const (
	sqlEnqueue = `
INSERT INTO messages (queue, body, not_before, max_retries, dlq, trace_id)
VALUES ($1, $2, now() + $3::interval, $4, $5, $6)
RETURNING id;`

	// Single CTE TX pattern: pick -> update -> return rows
	sqlClaim = `
WITH picked AS (
  SELECT id
  FROM messages
  WHERE queue = $1
    AND lease_until IS NULL
    AND not_before <= now()
  ORDER BY id
  FOR UPDATE SKIP LOCKED
  LIMIT $2
),
updated AS (
  UPDATE messages m
  SET lease_until   = now() + $3::interval,
      delivery_count = m.delivery_count + 1
  FROM picked
  WHERE m.id = picked.id
  RETURNING m.*
)
SELECT * FROM updated;`

	sqlAck = `DELETE FROM messages WHERE id = $1;`
)

// Enqueue inserts a message with optional delay.
func (p *PostgresStore) Enqueue(ctx context.Context, m queue.Message, delay time.Duration) (int64, error) {
	// TODO: set sensible defaults if m.MaxRetries == 0, etc.
	if m.MaxRetries == 0{
		m.MaxRetries = 5
	}
	
	interval := toInterval(delay)

	var id int64
	err := p.pool.QueryRow(ctx, sqlEnqueue,
		m.Queue,
		m.Body,
		interval,     // $3 interval
		m.MaxRetries, // $4
		m.DLQ,        // $5
		m.TraceID,    // $6
	).Scan(&id)
	return id, err
}

// Claim leases up to opts.Limit messages for opts.Visibility.
func (p *PostgresStore) Claim(ctx context.Context, opts queue.ClaimOptions) ([]queue.Message, error) {
	interval := toInterval(opts.Visibility)

	rows, err := p.pool.Query(ctx, sqlClaim, opts.Queue, opts.Limit, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []queue.Message
	for rows.Next() {
		var m queue.Message
		// NOTE: Column order must match RETURNING m.* (table order).
		err = rows.Scan(
			&m.ID,
			&m.Queue,
			&m.Body,
			&m.EnqueuedAt,
			&m.NotBefore,
			&m.LeaseUntil,
			&m.DeliveryCount,
			&m.MaxRetries,
			&m.DLQ,
			&m.TraceID,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Ack deletes the message by its ID.
func (p *PostgresStore) Ack(ctx context.Context, id int64) (bool, error) {
	ct, err := p.pool.Exec(ctx, sqlAck, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
