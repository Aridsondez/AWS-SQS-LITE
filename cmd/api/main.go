package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aridsondez/AWS-SQS-LITE/internal/api"
	"github.com/aridsondez/AWS-SQS-LITE/internal/config"
	pgstore "github.com/aridsondez/AWS-SQS-LITE/internal/queue/store/postgres"
	"github.com/aridsondez/AWS-SQS-LITE/internal/queue/sweeper"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, cfg.DBConnectionTimeout)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(connectCtx); err != nil {
		log.Fatalf("pgx ping: %v", err)
	}

	store := pgstore.New(pool)

	swp := sweeper.New(store, cfg.SweeperInterval)
	go swp.Start(ctx)

	addr := fmt.Sprintf(":%d", cfg.Port)
	httpSrv := api.NewServer(addr, store)

	log.Printf("HTTP server listening on %s", addr)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	_ = httpSrv.Shutdown(context.Background())
}
