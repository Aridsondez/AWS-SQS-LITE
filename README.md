# AWS-SQS-LITE

*A distributed Queue Service in GO (AWS SQS inspired)*

## Overview 
SQS-lite is a simplified version of AWS SQS built for learning distributed systems, concurancy, and containerized development

The system provides:

At-least-once delivery (workers may see duplicates → must be idempotent).

Visibility timeouts & leases (messages “locked” while in flight).

Retries with exponential backoff + jitter.

Dead Letter Queues (DLQs) (messages that fail too often are quarantined).

API/CLI for producers & workers.

Basic monitoring/metrics with Prometheus.

This is not meant to replace SQS — it’s a learning project to deeply understand the design trade-offs in message queuing systems.

## Why This Project?

Building SQS-Lite gives hands-on practice with concepts that are common in distributed infrastructure:

Go concurrency: goroutines, channels, contexts for producer/consumer patterns.

Database-backed queues: using Postgres with FOR UPDATE SKIP LOCKED for safe multi-consumer message claims.

Delivery guarantees: understanding at-least-once vs exactly-once semantics.

Backoff strategies: exponential backoff with jitter to avoid thundering herds.

Observability: Prometheus counters/histograms, structured logs.

Containerization: Docker/Docker Compose to run API, workers, and DB together.

Recruiters love this kind of project because it shows you can design, implement, and explain infra-level software — the same skills used at companies like AWS, HashiCorp, and Cloudflare.

## Tech Stack

Language: Go (fast, simple concurrency, industry standard for infra).

Storage: PostgreSQL (durability + transactional locks).

API: REST (JSON). gRPC may be added later.

Observability: Prometheus for metrics, structured logging with zerolog.

Containers: Docker + docker-compose (API, DB, Prometheus).

CLI: Cobra in Go, wrapping API calls for enqueue/receive/ack.

## Architecture

Core components:

API Server: Enqueue, receive, ack, change-visibility, stats.

Queue Manager: Implements leases, retries, DLQs.

Storage Layer: Postgres transactions for concurrency control.

Sweeper: Background task to detect expired leases and requeue or DLQ messages.

Worker SDK & CLI: Client libraries for producers/consumers.

Metrics: Prometheus endpoint for observability.

Message lifecycle:

Enqueue → message stored in Postgres with not_before timestamp.

Receive → worker claims messages via FOR UPDATE SKIP LOCKED, sets lease_until.

Ack → worker signals completion → message deleted.

Visibility timeout expires → message becomes available again.

Retries exceed limit → message moved to Dead Letter Queue.

## Projected File Structure
sqs-lite/
├── cmd/
│   ├── api/        # API server entrypoint
│   ├── worker/     # Example worker process
│   └── cli/        # CLI using Cobra
├── internal/
│   ├── api/        # HTTP handlers & routes
│   ├── config/     # Env config loader
│   ├── metrics/    # Prometheus metrics registration
│   └── queue/      # Core queue logic
│       ├── store/  # Postgres-backed store
│       ├── models.go
│       └── service.go
├── migrations/     # SQL migrations for Postgres schema
├── deploy/
│   └── prometheus/ # Prometheus config
├── scripts/        # Dev helpers, notes
├── Dockerfile
├── docker-compose.yml
├── README.md
└── go.mod

## Development Plan (6 Weeks)

Week 1 → Repo setup, schema design, in-memory queue prototype.

Week 2 → Postgres integration, basic enqueue/receive/ack.

Week 3 → Change-visibility, sweeper, retries.

Week 4 → Dead Letter Queues + Prometheus metrics.

Week 5 → CLI tooling, worker SDK, docs.

Week 6 → Load testing, polish, LinkedIn demo post.

## Learning Outcomes

By the end of this project you’ll be comfortable with:

Writing concurrent Go code with goroutines/channels.

Designing APIs for infra tools.

Using Postgres for concurrency control (FOR UPDATE SKIP LOCKED).

Implementing exponential backoff & retry policies.

Deploying multi-service systems with Docker Compose.

Explaining distributed system trade-offs to recruiters.