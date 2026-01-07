package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aridsondez/AWS-SQS-LITE/internal/api"
	"github.com/aridsondez/AWS-SQS-LITE/internal/queue/store/postgres"
	"github.com/aridsondez/AWS-SQS-LITE/internal/queue/sweeper"
)

const testDBURL = "postgres://postgres:password@localhost:5432/aws_sqs_lite_test?sslmode=disable"

func setupTestServer(t *testing.T) (*http.Server, *sweeper.Sweeper, *pgxpool.Pool) {
	ctx := context.Background()
	
	pool, err := pgxpool.New(ctx, testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}
	
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping test DB: %v", err)
	}
	
	// Clean up test data
	_, _ = pool.Exec(ctx, "DELETE FROM messages")
	
	store := postgres.New(pool)
	
	// Create sweeper with short interval for testing
	swp := sweeper.New(store, 2*time.Second)
	go swp.Start(ctx)
	
	srv := api.NewServer(":9999", store)
	go func() {
		_ = srv.ListenAndServe()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	return srv, swp, pool
}


func TestBasicFlow(t *testing.T) {
	srv, swp, pool := setupTestServer(t)
	defer srv.Shutdown(context.Background())
	defer swp.Stop()
	defer pool.Close()
	
	fmt.Println("\n=== Test 1: Basic Enqueue → Receive → Ack ===")
	
	enqueuePayload := map[string]interface{}{
		"body": map[string]string{"task": "process-order"},
		"max_retries": 3,
	}
	
	msgID := enqueueMessage(t, "test-queue", enqueuePayload)
	fmt.Printf("✓ Enqueued message ID: %d\n", msgID)
	
	// Receive the message
	messages := receiveMessages(t, "test-queue", 1, 30000)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	fmt.Printf("✓ Received message ID: %d, Delivery Count: %d\n", 
		int64(messages[0]["id"].(float64)), 
		int(messages[0]["delivery_count"].(float64)))
	
	ackMessage(t, int64(messages[0]["id"].(float64)))
	fmt.Println("✓ Acknowledged message")
	
	messages = receiveMessages(t, "test-queue", 1, 30000)
	if len(messages) != 0 {
		t.Fatalf("Expected 0 messages after ack, got %d", len(messages))
	}
	fmt.Println("✓ Queue is empty after ack")
}

func TestSweeperRequeue(t *testing.T) {
	srv, swp, pool := setupTestServer(t)
	defer srv.Shutdown(context.Background())
	defer swp.Stop()
	defer pool.Close()
	
	fmt.Println("\n=== Test 2: Sweeper Requeues Expired Messages ===")
	
	// Enqueue a message
	enqueuePayload := map[string]interface{}{
		"body": map[string]string{"task": "test-requeue"},
		"max_retries": 5,
	}
	
	msgID := enqueueMessage(t, "requeue-test", enqueuePayload)
	fmt.Printf("✓ Enqueued message ID: %d\n", msgID)
	
	// Receive with very short visibility (1 second)
	messages := receiveMessages(t, "requeue-test", 1, 1000)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	fmt.Printf("✓ Received message with 1s visibility timeout\n")
	
	fmt.Println("Waiting 3 seconds for sweeper to requeue...")
	time.Sleep(3 * time.Second)
	
	messages = receiveMessages(t, "requeue-test", 1, 30000)
	if len(messages) != 1 {
		t.Fatalf("Expected message to be requeued, got %d messages", len(messages))
	}
	
	deliveryCount := int(messages[0]["delivery_count"].(float64))
	if deliveryCount != 2 {
		t.Fatalf("Expected delivery_count=2, got %d", deliveryCount)
	}
	fmt.Printf("✓ Message requeued! Delivery count: %d\n", deliveryCount)
}

func TestDLQRouting(t *testing.T) {
	srv, swp, pool := setupTestServer(t)
	defer srv.Shutdown(context.Background())
	defer swp.Stop()
	defer pool.Close()
	
	fmt.Println("\n=== Test 3: DLQ Routing After Max Retries ===")
	
	dlqName := "failed-queue"
	
	// Enqueue a message with low max_retries and DLQ
	enqueuePayload := map[string]interface{}{
		"body": map[string]string{"task": "will-fail"},
		"max_retries": 2,
		"dlq": dlqName,
	}
	
	msgID := enqueueMessage(t, "main-queue", enqueuePayload)
	fmt.Printf("✓ Enqueued message ID: %d (max_retries=2, dlq=%s)\n", msgID, dlqName)
	
	// Receive and don't ack - 2 times
	for i := 1; i <= 2; i++ {
		messages := receiveMessages(t, "main-queue", 1, 1000)
		if len(messages) != 1 {
			t.Fatalf("Attempt %d: Expected 1 message, got %d", i, len(messages))
		}
		fmt.Printf("✓ Received attempt %d, delivery_count=%d\n", 
			i, int(messages[0]["delivery_count"].(float64)))
		
		// Wait for sweeper to requeue
		time.Sleep(3 * time.Second)
	}
	

	fmt.Println("⏳ Waiting for sweeper to route to DLQ...")
	time.Sleep(3 * time.Second)
	
	messages := receiveMessages(t, "main-queue", 1, 30000)
	if len(messages) != 0 {
		t.Fatalf("Expected main queue to be empty, got %d messages", len(messages))
	}
	fmt.Println("✓ Main queue is empty")
	
	// DLQ should have the message
	dlqMessages := receiveMessages(t, dlqName, 1, 30000)
	if len(dlqMessages) != 1 {
		t.Fatalf("Expected 1 message in DLQ, got %d", len(dlqMessages))
	}
	fmt.Printf("✓ Message routed to DLQ! Body: %v\n", dlqMessages[0]["body"])
}


func enqueueMessage(t *testing.T, queue string, payload map[string]interface{}) int64 {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:9999/v1/queues/%s/messages", queue),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Enqueue returned %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return int64(result["id"].(float64))
}

func receiveMessages(t *testing.T, queue string, max int, visibilityMS int) []map[string]interface{} {
	payload := map[string]interface{}{
		"max": max,
		"visibility_ms": visibilityMS,
	}
	body, _ := json.Marshal(payload)
	
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:9999/v1/queues/%s:receive", queue),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	defer resp.Body.Close()
	
	var messages []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&messages)
	return messages
}

func ackMessage(t *testing.T, id int64) {
	payload := map[string]interface{}{}
	body, _ := json.Marshal(payload)
	
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:9999/v1/messages/%d:ack", id),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Ack failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Ack returned %d", resp.StatusCode)
	}
}
