# AWS-SQS-LITE

*A distributed Queue Service in GO (AWS SQS inspired)*

## Overview

SQS-lite is a simplified version of AWS SQS built for learning distributed systems, concurrency, and containerized development.

The system provides:

- **At-least-once delivery** - Workers may see duplicates (must be idempotent)
- **Visibility timeouts & leases** - Messages "locked" while in flight
- **Automatic retries** - Failed messages automatically requeue
- **Dead Letter Queues (DLQs)** - Messages that fail too often are quarantined
- **REST API** - For producers & workers
- **Prometheus metrics** - Production-ready monitoring

This is not meant to replace SQS — it's a learning project to deeply understand the design trade-offs in message queuing systems.

---

## ✨ Features Implemented

### Core Functionality
- ✅ **Enqueue** - Add messages with optional delay
- ✅ **Receive/Claim** - Atomically lease messages using PostgreSQL `FOR UPDATE SKIP LOCKED`
- ✅ **Acknowledge** - Delete successfully processed messages
- ✅ **Background Sweeper** - Automatically requeue expired messages or route to DLQ
- ✅ **Dead Letter Queue** - Failed messages automatically route to DLQ after max retries

### Observability
- ✅ **Prometheus Metrics** - Track enqueued, received, acked, requeued, and DLQ'd messages
- ✅ **Sweeper Metrics** - Monitor sweeper duration and errors
- ✅ **Health Check** - `/healthz` endpoint

### Testing & Demo
- ✅ **Integration Tests** - Comprehensive test suite
- ✅ **Interactive CLI Demo** - Visual demonstration of all features
- ✅ **Makefile** - Easy development workflow

---

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.23+

### 1. Start Database
```bash
make db-up
```

### 2. Run Server (in one terminal)
```bash
make run
```

### 3. Run Interactive Demo (in another terminal)
```bash
make demo
```

You'll see a beautiful colored output demonstrating:
1. Basic message flow (enqueue → receive → ack)
2. Sweeper requeuing expired messages
3. DLQ routing after max retries
4. Live Prometheus metrics

---

## 📖 API Reference

### Health Check
```bash
GET /healthz
```

### Enqueue Message
```bash
POST /v1/queues/{queue}/messages
Content-Type: application/json

{
  "body": {"task": "process-order"},
  "delay": 5000,          # Optional: milliseconds
  "max_retries": 3,       # Optional: defaults to 5
  "dlq": "failed-queue",  # Optional: DLQ name
  "trace_id": "xyz123"    # Optional: for tracing
}

Response: {"id": 123}
```

### Receive Messages
```bash
POST /v1/queues/{queue}:receive
Content-Type: application/json

{
  "max": 10,              # Max messages to receive (1-32)
  "visibility_ms": 30000  # Visibility timeout in milliseconds
}

Response: [
  {
    "id": 123,
    "body": {"task": "process-order"},
    "receipt": "123",
    "lease_until": "2026-01-07T...",
    "delivery_count": 1,
    "max_retries": 3,
    "dlq": "failed-queue"
  }
]
```

### Acknowledge Message
```bash
POST /v1/messages/{id}:ack
Content-Type: application/json

{}

Response: {"ok": true}
```

### Prometheus Metrics
```bash
GET /metrics
```

---

## 📊 Metrics

The following Prometheus metrics are exposed at `/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `sqs_messages_enqueued_total{queue}` | Counter | Total messages enqueued per queue |
| `sqs_messages_received_total{queue}` | Counter | Total messages received per queue |
| `sqs_messages_acked_total` | Counter | Total messages acknowledged |
| `sqs_messages_requeued_total` | Counter | Total messages requeued by sweeper |
| `sqs_messages_dlq_total` | Counter | Total messages sent to DLQ |
| `sqs_sweeper_duration_seconds` | Histogram | Sweeper execution duration |
| `sqs_sweeper_errors_total` | Counter | Total sweeper errors |

---

## 🛠️ Development

### Available Commands

```bash
make help              # Show all available commands
make db-up             # Start PostgreSQL database
make db-down           # Stop PostgreSQL database
make db-reset          # Reset database (down + up)
make run               # Run the API server
make demo              # Run interactive demo
make test              # Run all tests
make test-integration  # Run integration tests only
make build             # Build the binary
make clean             # Clean up containers and volumes
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | (required) | PostgreSQL connection string |
| `PORT` | 8080 | HTTP server port |
| `SWEEPER_INTERVAL` | 60 | Sweeper run interval (seconds) |
| `VISIBILITY_TIMEOUT` | 30 | Default visibility timeout (seconds) |
| `RECEIVE_MAX` | 10 | Default max messages per receive |
| `LOG_LEVEL` | info | Log level |

---

## 🏗️ Architecture

### Components

1. **API Server** - HTTP REST API for message operations
2. **PostgreSQL Store** - Durable message storage with ACID guarantees
3. **Background Sweeper** - Goroutine that processes expired leases
4. **Prometheus Exporter** - Metrics endpoint for monitoring

### Message Lifecycle

```
1. ENQUEUE
   ↓
   Message stored in PostgreSQL
   (not_before = now() + delay)
   ↓
2. RECEIVE
   ↓
   Worker claims message using FOR UPDATE SKIP LOCKED
   (lease_until = now() + visibility_timeout)
   (delivery_count++)
   ↓
3a. ACK (Success)          3b. Timeout (Failure)
    ↓                          ↓
    Message deleted            Sweeper detects expired lease
                              ↓
                          4a. Requeue        4b. DLQ
                          (if < max_retries) (if >= max_retries)
                              ↓                  ↓
                          Back to step 2      Moved to DLQ queue
```

### Database Schema

```sql
CREATE TABLE messages (
  id               BIGSERIAL PRIMARY KEY,
  queue            TEXT NOT NULL,
  body             JSONB NOT NULL,
  enqueued_at      TIMESTAMPTZ DEFAULT now(),
  not_before       TIMESTAMPTZ DEFAULT now(),  -- Delay support
  lease_until      TIMESTAMPTZ,                 -- NULL = available
  delivery_count   INT DEFAULT 0,
  max_retries      INT DEFAULT 5,
  dlq              TEXT,                        -- DLQ queue name
  trace_id         TEXT
);

-- Indexes for performance
CREATE INDEX idx_messages_available ON messages (queue, not_before, id)
  WHERE lease_until IS NULL;

CREATE INDEX idx_messages_inflight ON messages (queue, lease_until)
  WHERE lease_until IS NOT NULL;
```

---

## 🧪 Testing

### Run All Tests
```bash
make test
```

### Run Integration Tests
```bash
make test-integration
```

### Integration Test Coverage
- Basic message flow (enqueue → receive → ack)
- Sweeper requeues expired messages
- DLQ routing after max retries

---

## 📁 Project Structure

```
AWS-SQS-LITE/
├── cmd/
│   ├── api/              # API server entrypoint
│   └── demo/             # Interactive demo CLI
├── internal/
│   ├── api/              # HTTP handlers & routing
│   ├── config/           # Configuration management
│   ├── metrics/          # Prometheus metrics
│   └── queue/
│       ├── models.go     # Data structures
│       ├── services.go   # Business logic
│       ├── store/        # Storage interface
│       │   └── postgres/ # PostgreSQL implementation
│       └── sweeper/      # Background sweeper
├── migrations/           # Database migrations
├── tests/                # Integration tests
├── docker-compose.yml    # Docker services
├── Makefile             # Development commands
└── README.md
```

---

## 🎯 Learning Outcomes

By building this project, you'll gain hands-on experience with:

- **Go Concurrency** - Goroutines, channels, contexts for producer/consumer patterns
- **Database Transactions** - Using PostgreSQL `FOR UPDATE SKIP LOCKED` for safe concurrency
- **Distributed Systems** - Message queues, delivery guarantees, retry strategies
- **Observability** - Prometheus metrics, structured logging
- **API Design** - RESTful APIs for infrastructure tools
- **Testing** - Integration tests for distributed systems
- **DevOps** - Docker, Docker Compose, Makefiles

---

## 🔮 Future Enhancements

- [ ] **Change Visibility** - Extend lease duration for long-running tasks
- [ ] **Batch Operations** - Send/delete multiple messages at once
- [ ] **Long Polling** - Wait for messages instead of immediate empty response
- [ ] **Queue Stats** - GET /v1/queues/{queue}/stats endpoint
- [ ] **FIFO Queues** - Message ordering guarantees
- [ ] **Exponential Backoff** - Configurable backoff strategies
- [ ] **Structured Logging** - Replace basic log with zerolog
- [ ] **gRPC API** - High-performance alternative to REST
- [ ] **Worker SDK** - Client library for workers
- [ ] **Load Testing** - Performance benchmarks

---

## 📝 License

This project is for educational purposes. Feel free to use it for learning!

---

## 🙏 Acknowledgments

Inspired by AWS SQS and built to learn distributed systems engineering.

**Tech Stack:**
- Go - Fast, simple concurrency
- PostgreSQL - ACID compliance & transactional locks
- Prometheus - Production-grade metrics
- Docker - Containerization
