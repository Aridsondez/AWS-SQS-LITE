package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL = "http://localhost:8080"
	colorReset = "\033[0m"
	colorRed = "\033[31m"
	colorGreen = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan = "\033[36m"
	colorBold = "\033[1m"
)

type Message struct {
	Body       map[string]interface{} `json:"body"`
	MaxRetries int                    `json:"max_retries,omitempty"`
	DLQ        string                 `json:"dlq,omitempty"`
}

type EnqueueResponse struct {
	ID int64 `json:"id"`
}

type ReceiveRequest struct {
	Max          int `json:"max"`
	VisibilityMS int `json:"visibility_ms"`
}

type ReceivedMessage struct {
	ID            int64                  `json:"id"`
	Body          map[string]interface{} `json:"body"`
	Receipt       string                 `json:"receipt"`
	DeliveryCount int                    `json:"delivery_count"`
	MaxRetries    int                    `json:"max_retries"`
}

func main() {
	printHeader()

	// Check server is running
	if !checkServer() {
		fmt.Printf("%s✗ Server not running. Please run 'make run' first.%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Server is running%s\n\n", colorGreen, colorReset)

	// Run demo scenarios
	fmt.Printf("%s=== AWS SQS Lite Demo ===%s\n\n", colorBold+colorCyan, colorReset)

	scenario1_BasicFlow()
	time.Sleep(2 * time.Second)

	scenario2_SweeperRequeue()
	time.Sleep(2 * time.Second)

	scenario3_DLQRouting()
	time.Sleep(2 * time.Second)

	displayMetrics()

	printFooter()
}

func printHeader() {
	fmt.Print(colorCyan + colorBold)
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         AWS SQS LITE - INTERACTIVE DEMO                   ║")
	fmt.Println("║         Message Queue with DLQ & Auto-Retry               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Print(colorReset)
	fmt.Println()
}

func printFooter() {
	fmt.Println()
	fmt.Print(colorCyan)
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Demo Complete!                         ║")
	fmt.Println("║  View live metrics at: http://localhost:8080/metrics      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Print(colorReset)
}

func checkServer() bool {
	resp, err := http.Get(baseURL + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func scenario1_BasicFlow() {
	printScenario("Scenario 1: Basic Message Flow (Enqueue → Receive → Ack)")

	// 1. Enqueue
	fmt.Printf("%s→ Enqueuing message to 'orders' queue...%s\n", colorYellow, colorReset)
	msg := Message{
		Body: map[string]interface{}{
			"order_id": "ORD-12345",
			"customer": "John Doe",
			"total":    99.99,
		},
		MaxRetries: 3,
	}

	msgID := enqueueMessage("orders", msg)
	fmt.Printf("%s  ✓ Message enqueued with ID: %d%s\n", colorGreen, msgID, colorReset)
	time.Sleep(1 * time.Second)

	// 2. Receive
	fmt.Printf("%s→ Receiving message from 'orders' queue...%s\n", colorYellow, colorReset)
	messages := receiveMessages("orders", 1, 30000)
	if len(messages) > 0 {
		fmt.Printf("%s  ✓ Received message ID: %d, Delivery Count: %d%s\n",
			colorGreen, int64(messages[0].ID), messages[0].DeliveryCount, colorReset)
		fmt.Printf("    Body: %v\n", messages[0].Body)
	}
	time.Sleep(1 * time.Second)

	// 3. Ack
	fmt.Printf("%s→ Acknowledging message...%s\n", colorYellow, colorReset)
	ackMessage(int64(messages[0].ID))
	fmt.Printf("%s  ✓ Message acknowledged and deleted%s\n", colorGreen, colorReset)

	// 4. Verify empty
	fmt.Printf("%s→ Verifying queue is empty...%s\n", colorYellow, colorReset)
	messages = receiveMessages("orders", 1, 30000)
	if len(messages) == 0 {
		fmt.Printf("%s  ✓ Queue is empty (message successfully processed)%s\n", colorGreen, colorReset)
	}

	fmt.Println()
}

func scenario2_SweeperRequeue() {
	printScenario("Scenario 2: Sweeper Requeues Expired Messages")

	// 1. Enqueue
	fmt.Printf("%s→ Enqueuing task message...%s\n", colorYellow, colorReset)
	msg := Message{
		Body: map[string]interface{}{
			"task": "process-payment",
			"amount": 50.00,
		},
		MaxRetries: 5,
	}

	msgID := enqueueMessage("tasks", msg)
	fmt.Printf("%s  ✓ Message enqueued with ID: %d%s\n", colorGreen, msgID, colorReset)
	time.Sleep(1 * time.Second)

	// 2. Receive with short visibility
	fmt.Printf("%s→ Receiving message with 2-second visibility timeout...%s\n", colorYellow, colorReset)
	messages := receiveMessages("tasks", 1, 2000)
	if len(messages) > 0 {
		fmt.Printf("%s  ✓ Message leased (delivery_count=%d)%s\n",
			colorGreen, messages[0].DeliveryCount, colorReset)
	}
	time.Sleep(1 * time.Second)

	// 3. Don't ack - simulate worker crash
	fmt.Printf("%s→ Simulating worker crash (not acknowledging)...%s\n", colorMagenta, colorReset)
	fmt.Printf("%s  ⏳ Waiting for visibility timeout to expire...%s\n", colorBlue, colorReset)
	time.Sleep(3 * time.Second)

	// 4. Sweeper should have requeued
	fmt.Printf("%s→ Attempting to receive again (should be requeued by sweeper)...%s\n", colorYellow, colorReset)
	messages = receiveMessages("tasks", 1, 30000)
	if len(messages) > 0 {
		fmt.Printf("%s  ✓ Message requeued! Delivery count: %d%s\n",
			colorGreen, messages[0].DeliveryCount, colorReset)
		ackMessage(int64(messages[0].ID))
		fmt.Printf("%s  ✓ Cleaned up message%s\n", colorGreen, colorReset)
	}

	fmt.Println()
}

func scenario3_DLQRouting() {
	printScenario("Scenario 3: Dead Letter Queue Routing After Max Retries")

	dlqName := "failed-tasks"

	// 1. Enqueue with low max_retries
	fmt.Printf("%s→ Enqueuing message with max_retries=2 and DLQ='%s'...%s\n",
		colorYellow, dlqName, colorReset)
	msg := Message{
		Body: map[string]interface{}{
			"task": "risky-operation",
			"will_fail": true,
		},
		MaxRetries: 2,
		DLQ:        dlqName,
	}

	msgID := enqueueMessage("main-tasks", msg)
	fmt.Printf("%s  ✓ Message enqueued with ID: %d%s\n", colorGreen, msgID, colorReset)
	time.Sleep(1 * time.Second)

	// 2. Simulate 2 failed attempts
	for attempt := 1; attempt <= 2; attempt++ {
		fmt.Printf("%s→ Attempt %d: Receiving and failing...%s\n", colorYellow, attempt, colorReset)
		messages := receiveMessages("main-tasks", 1, 2000)
		if len(messages) > 0 {
			fmt.Printf("%s  ✓ Received (delivery_count=%d)%s\n",
				colorGreen, messages[0].DeliveryCount, colorReset)
			fmt.Printf("%s  ✗ Simulating failure (not acking)...%s\n", colorRed, colorReset)
		}

		fmt.Printf("%s  ⏳ Waiting for sweeper to requeue...%s\n", colorBlue, colorReset)
		time.Sleep(4 * time.Second)
	}

	// 3. After max retries, should go to DLQ
	fmt.Printf("%s→ Waiting for sweeper to route to DLQ...%s\n", colorBlue, colorReset)
	time.Sleep(4 * time.Second)

	// 4. Check main queue (should be empty)
	fmt.Printf("%s→ Checking main queue...%s\n", colorYellow, colorReset)
	messages := receiveMessages("main-tasks", 1, 30000)
	if len(messages) == 0 {
		fmt.Printf("%s  ✓ Main queue is empty%s\n", colorGreen, colorReset)
	}

	// 5. Check DLQ (should have message)
	fmt.Printf("%s→ Checking DLQ '%s'...%s\n", colorYellow, dlqName, colorReset)
	dlqMessages := receiveMessages(dlqName, 1, 30000)
	if len(dlqMessages) > 0 {
		fmt.Printf("%s  ✓ Message found in DLQ!%s\n", colorGreen, colorReset)
		fmt.Printf("    ID: %d, Body: %v\n", int64(dlqMessages[0].ID), dlqMessages[0].Body)

		// Clean up
		ackMessage(int64(dlqMessages[0].ID))
		fmt.Printf("%s  ✓ Cleaned up DLQ message%s\n", colorGreen, colorReset)
	}

	fmt.Println()
}

func displayMetrics() {
	printScenario("Live Prometheus Metrics")

	resp, err := http.Get(baseURL + "/metrics")
	if err != nil {
		fmt.Printf("%s✗ Failed to fetch metrics%s\n", colorRed, colorReset)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")

	metrics := []string{
		"sqs_messages_enqueued_total",
		"sqs_messages_received_total",
		"sqs_messages_acked_total",
		"sqs_messages_requeued_total",
		"sqs_messages_dlq_total",
		"sqs_sweeper_errors_total",
		"sqs_sweeper_duration_seconds_count",
	}

	for _, line := range lines {
		for _, metric := range metrics {
			if strings.HasPrefix(line, metric) && !strings.Contains(line, "#") {
				// Colorize the output
				parts := strings.Split(line, " ")
				if len(parts) == 2 {
					metricName := parts[0]
					value := parts[1]

					// Color metric name
					fmt.Printf("%s%-40s%s %s%s%s\n",
						colorCyan, metricName, colorReset,
						colorGreen+colorBold, value, colorReset)
				}
			}
		}
	}

	fmt.Printf("\n%sView full metrics: %shttp://localhost:8080/metrics%s\n",
		colorYellow, colorBlue+colorBold, colorReset)
}

func printScenario(title string) {
	fmt.Printf("%s%s┌─────────────────────────────────────────────────────────────┐%s\n",
		colorBold, colorMagenta, colorReset)
	fmt.Printf("%s%s│ %-59s │%s\n",
		colorBold, colorMagenta, title, colorReset)
	fmt.Printf("%s%s└─────────────────────────────────────────────────────────────┘%s\n",
		colorBold, colorMagenta, colorReset)
}

func enqueueMessage(queue string, msg Message) int64 {
	payload, _ := json.Marshal(msg)
	resp, err := http.Post(
		fmt.Sprintf("%s/v1/queues/%s/messages", baseURL, queue),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		fmt.Printf("%s✗ Enqueue failed: %v%s\n", colorRed, err, colorReset)
		return 0
	}
	defer resp.Body.Close()

	var result EnqueueResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ID
}

func receiveMessages(queue string, max int, visibilityMS int) []ReceivedMessage {
	req := ReceiveRequest{
		Max:          max,
		VisibilityMS: visibilityMS,
	}
	payload, _ := json.Marshal(req)

	resp, err := http.Post(
		fmt.Sprintf("%s/v1/queues/%s:receive", baseURL, queue),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		fmt.Printf("%s✗ Receive failed: %v%s\n", colorRed, err, colorReset)
		return nil
	}
	defer resp.Body.Close()

	var messages []ReceivedMessage
	json.NewDecoder(resp.Body).Decode(&messages)
	return messages
}

func ackMessage(id int64) {
	payload := []byte("{}")
	resp, err := http.Post(
		fmt.Sprintf("%s/v1/messages/%d:ack", baseURL, id),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		fmt.Printf("%s✗ Ack failed: %v%s\n", colorRed, err, colorReset)
		return
	}
	defer resp.Body.Close()
}
