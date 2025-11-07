package api

import(
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aridsondez/AWS-SQS-LITE/internal/queue"
	"github.com/aridsondez/AWS-SQS-LITE/internal/queue/store"
)

type Server struct {
	store store.Store
	addr  string
	timeout time.Duration
}

func NewServer(addr string, s store.Store) *http.Server {
	srv := &Server{
		store: s,
		addr:  addr,
		timeout: 5 * time.Second,
	}
	r:= chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(srv.timeout))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_,_ = w.Write([]byte("ok"))
	})

	//r.Handle("/metrics", promhttp.Handler())

	r.Route("/v1", func(r chi.Router) {
		// enqueue: POST /v1/queues/{queue}/messages
		r.Post("/queues/{queue}/messages", srv.handleEnqueue)

		// receive: POST /v1/queues/{queue}:receive
		r.Post("/queues/{queue}:receive", srv.handleReceive)

		// ack: POST /v1/messages/{id}:ack
		r.Post("/messages/{id}:ack", srv.handleAck)
	})

	return &http.Server{
		Addr:    srv.addr,
		Handler: r,
	}
}

type enqueueRequest struct {
	Body  json.RawMessage `json:"body"`
	DelayMS int64          `json:"delay,omitempty"` // miliseconds
	MaxRetries int        `json:"max_retries,omitempty"`
	DLQ       *string     `json:"dlq,omitempty"`
	TraceID   *string     `json:"trace_id,omitempty"`
}

type enqueueResponse struct {
	ID int64 `json:"id"`
}

type receiveRequest struct {
	Max          int   `json:"max"`             // e.g., 1..32
	VisibilityMS int64 `json:"visibility_ms"`   // e.g., 30000
}

type receivedMessage struct {
	ID            int64           `json:"id"`
	Body          json.RawMessage `json:"body"`
	Receipt       string          `json:"receipt"` // for now, just the ID as string
	LeaseUntil    *time.Time      `json:"lease_until,omitempty"`
	DeliveryCount int             `json:"delivery_count"`
	MaxRetries    int             `json:"max_retries"`
	DLQ           *string         `json:"dlq,omitempty"`
	TraceID       *string         `json:"trace_id,omitempty"`
}

type ackRequest struct {
	Receipt string `json:"receipt,omitempty"` // placeholder for future opaque receipts
}

type ackResponse struct {
	OK bool `json:"ok"`
}

// ---------- Handlers ----------


func (s *Server) handleEnqueue(w http.ResponseWriter, r *http.Request) {
	qname := chi.URLParam(r, "queue")
	if qname == "" {
		httpError(w, http.StatusBadRequest, "missing queue path param")
		return
	}
	var req enqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json: %v", err)
		return
	}
	if len(req.Body) == 0 || string(req.Body) == "null" {
		httpError(w, http.StatusBadRequest, "`body` is required")
		return
	}
	if req.MaxRetries <= 0 {
		req.MaxRetries = 5
	}
	delay := time.Duration(req.DelayMS) * time.Millisecond

	msg := queue.Message{
		Queue:      qname,
		Body:       []byte(req.Body),
		MaxRetries: req.MaxRetries,
		DLQ:        req.DLQ,
		TraceID:    req.TraceID,
	}

	ctx := r.Context()
	id, err := s.store.Enqueue(ctx, msg, delay)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "enqueue failed: %v", err)
		return
	}
	writeJSON(w, http.StatusCreated, &enqueueResponse{ID: id})
}

func (s *Server) handleReceive(w http.ResponseWriter, r *http.Request) {
	qname := chi.URLParam(r, "queue")
	if qname == "" {
		httpError(w, http.StatusBadRequest, "missing queue path param")
		return
	}
	var req receiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json: %v", err)
		return
	}
	if req.Max <= 0 || req.Max > 32 {
		req.Max = 1
	}
	vis := time.Duration(req.VisibilityMS) * time.Millisecond
	if vis <= 0 {
		vis = 30 * time.Second
	}

	ctx := r.Context()
	out, err := s.store.Claim(ctx, queue.ClaimOptions{
		Queue:      qname,
		Limit:      req.Max,
		Visibility: vis,
	})
	if err != nil {
		httpError(w, http.StatusInternalServerError, "claim failed: %v", err)
		return
	}

	resp := make([]receivedMessage, 0, len(out))
	for _, m := range out {
		resp = append(resp, receivedMessage{
			ID:            m.ID,
			Body:          json.RawMessage(m.Body),
			Receipt:       strconv.FormatInt(m.ID, 10), 
			LeaseUntil:    m.LeaseUntil,
			DeliveryCount: m.DeliveryCount,
			MaxRetries:    m.MaxRetries,
			DLQ:           m.DLQ,
			TraceID:       m.TraceID,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAck(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		httpError(w, http.StatusBadRequest, "missing message id")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid id: %v", err)
		return
	}
	// Optionally verify receipt body, currently ignored:
	var req ackRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	ok, err := s.store.Ack(r.Context(), id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "ack failed: %v", err)
		return
	}
	if !ok {
		// id wasn’t found (already acked/expired & reclaimed); 404 is reasonable
		httpError(w, http.StatusNotFound, "message not found")
		return
	}
	writeJSON(w, http.StatusOK, &ackResponse{OK: true})
}

// ---------- helpers ----------

func httpError(w http.ResponseWriter, code int, format string, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	msg := fmt.Sprintf(format, args...)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": msg,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// If you want to run background jobs (like a sweeper) tied to request context:
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}