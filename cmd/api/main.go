package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aridsondez/AWS-SQS-LITE/internal/config"
	"github.com/aridsondez/AWS-SQS-LITE/internal/queue"
	pgstore "github.com/aridsondez/AWS-SQS-LITE/internal/queue/store/postgres" // PostgresStore impl
)

func main() {
	// Root context with cancel on Ctrl+C

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1) Load env/config
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	// 2) Connect to Postgres with a timeout
	connectCtx, cancel := context.WithTimeout(ctx, cfg.DBConnectionTimeout)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, cfg.DatabaseURL)
	if err != nil {
		panic(fmt.Errorf("pgxpool.New: %w", err))
	}
	defer pool.Close()

	// Verify the connection
	if err := pool.Ping(connectCtx); err != nil {
		panic(fmt.Errorf("pgx ping: %w", err))
	}

	// 3) Wire the store
	store := pgstore.New(pool)

	// --------- TEMP SMOKE TEST (remove once HTTP is wired) ----------
	// Enqueue -> Claim -> Ack, just to prove the store works.
	msg := queue.Message{
		Queue:      "dev",
		Body:       []byte(`{"hello":"world"}`),
		MaxRetries: 3,
		// DLQ:      ptr to "dev-dlq" if you want
	}
	id, err := store.Enqueue(ctx, msg, 0)
	if err != nil {
		panic(fmt.Errorf("enqueue: %w", err))
	}
	fmt.Println("enqueued id:", id)

	claimed, err := store.Claim(ctx, queue.ClaimOptions{
		Queue:      "dev",
		Limit:      1,
		Visibility: 10 * time.Second,
	})
	if err != nil {
		panic(fmt.Errorf("claim: %w", err))
	}
	fmt.Printf("claimed %d messages\n", len(claimed))
	if len(claimed) > 0 {
		ok, err := store.Ack(ctx, claimed[0].ID)
		if err != nil {
			panic(fmt.Errorf("ack: %w", err))
		}
		fmt.Println("ack ok:", ok)
	}

	// --------- TODO: HTTP server (next step) ----------
	// Next we’ll replace the smoke test with:
	// - chi router
	// - POST /v1/queues/{queue}/messages        -> Enqueue
	// - POST /v1/queues/{queue}:receive         -> Claim
	// - POST /v1/messages/{id}:ack              -> Ack
	// - GET  /healthz, GET /metrics
	fmt.Printf("DB OK. Ready to add HTTP on :%d\n", cfg.Port)
	<-ctx.Done()
}
