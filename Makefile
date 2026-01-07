.PHONY: help db-up db-down db-reset run test test-integration clean build migrate-test demo run-worker run-producer

# Default target
help:
	@echo "AWS SQS Lite - Available Commands"
	@echo "=================================="
	@echo "make db-up           - Start PostgreSQL database"
	@echo "make db-down         - Stop PostgreSQL database"
	@echo "make db-reset        - Reset database (down + up)"
	@echo "make migrate-test    - Create test database and run migrations"
	@echo "make run             - Run the API server"
	@echo "make demo            - Run interactive demo (requires server running)"
	@echo "make run-worker      - Run example worker (requires server running)"
	@echo "make run-producer    - Run example producer (requires server running)"
	@echo "make test            - Run all tests"
	@echo "make test-integration - Run integration tests only"
	@echo "make build           - Build the binary"
	@echo "make clean           - Clean up containers and volumes"

# Start database
db-up:
	@echo "Starting PostgreSQL..."
	docker-compose up -d
	@echo "Waiting for database to be ready..."
	@sleep 3
	@docker exec sqs-lite-postgres pg_isready -U postgres || (echo "Database not ready" && exit 1)
	@echo "Database is ready!"

# Stop database
db-down:
	@echo "Stopping PostgreSQL..."
	docker-compose down

# Reset database (down + up)
db-reset: db-down db-up

# Create test database and run migrations
migrate-test:
	@echo "Creating test database..."
	@docker exec sqs-lite-postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'aws_sqs_lite_test'" | grep -q 1 || \
		docker exec sqs-lite-postgres psql -U postgres -c "CREATE DATABASE aws_sqs_lite_test;"
	@echo "Running migrations on test database..."
	@docker exec sqs-lite-postgres psql -U postgres -d aws_sqs_lite_test -f /docker-entrypoint-initdb.d/0001_init.sql
	@echo "Test database ready!"

# Run the server
run:
	@echo "Starting AWS SQS Lite server..."
	DATABASE_URL="postgres://postgres:password@localhost:5432/aws_sqs_lite?sslmode=disable" \
	PORT=8080 \
	SWEEPER_INTERVAL=5 \
	go run cmd/api/main.go

# Run interactive demo
demo:
	@echo "Running interactive demo..."
	@echo "Make sure the server is running with 'make run' in another terminal"
	@sleep 1
	go run cmd/demo/main.go

# Run example worker
run-worker:
	@echo "Starting example worker..."
	@echo "Make sure the server is running with 'make run' in another terminal"
	@sleep 1
	go run examples/worker/main.go

# Run example producer
run-producer:
	@echo "Running example producer..."
	@echo "Make sure the server is running with 'make run' in another terminal"
	@sleep 1
	go run examples/producer/main.go

# Run all tests
test: migrate-test
	@echo "Running all tests..."
	go test -v ./...

# Run integration tests only
test-integration: migrate-test
	@echo "Running integration tests..."
	go test -v ./tests

# Build binary
build:
	@echo "Building AWS SQS Lite..."
	go build -o bin/sqs-lite cmd/api/main.go
	@echo "Binary created at bin/sqs-lite"

# Clean everything
clean:
	@echo "Cleaning up..."
	docker-compose down -v
	rm -rf bin/
	@echo "Cleanup complete!"
