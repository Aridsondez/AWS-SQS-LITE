package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client for enqueueing messages to SQS Lite
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a new SQS Lite client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// EnqueueOptions for customizing message enqueue
type EnqueueOptions struct {
	Delay      time.Duration 
	MaxRetries int           // Max retry attempts (default: 5)
	DLQ        string        // Dead letter queue name
	TraceID    string        // Optional trace ID for correlation
}

// Enqueue sends a message to a queue
func (c *Client) Enqueue(ctx context.Context, queue string, body interface{}, opts *EnqueueOptions) (int64, error) {
	if opts == nil {
		opts = &EnqueueOptions{}
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("marshal body: %w", err)
	}

	req := map[string]interface{}{
		"body": json.RawMessage(bodyJSON),
	}

	if opts.Delay > 0 {
		req["delay"] = int(opts.Delay.Milliseconds())
	}
	if opts.MaxRetries > 0 {
		req["max_retries"] = opts.MaxRetries
	}
	if opts.DLQ != "" {
		req["dlq"] = opts.DLQ
	}
	if opts.TraceID != "" {
		req["trace_id"] = opts.TraceID
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	url := fmt.Sprintf("%s/v1/queues/%s/messages", c.baseURL, queue)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("enqueue failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.ID, nil
}
