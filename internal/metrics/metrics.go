package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Messages enqueued counter
	MessagesEnqueued = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqs_messages_enqueued_total",
			Help: "Total number of messages enqueued",
		},
		[]string{"queue"},
	)

	// Messages received counter
	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqs_messages_received_total",
			Help: "Total number of messages received",
		},
		[]string{"queue"},
	)

	// Messages acknowledged counter
	MessagesAcked = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_messages_acked_total",
			Help: "Total number of messages acknowledged",
		},
	)

	// Messages requeued by sweeper
	MessagesRequeued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_messages_requeued_total",
			Help: "Total number of messages requeued by sweeper",
		},
	)

	// Messages sent to DLQ
	MessagesDLQd = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_messages_dlq_total",
			Help: "Total number of messages sent to DLQ",
		},
	)

	// Sweeper run duration
	SweeperDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sqs_sweeper_duration_seconds",
			Help:    "Time taken for sweeper to process messages",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Sweeper errors counter
	SweeperErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sqs_sweeper_errors_total",
			Help: "Total number of sweeper errors",
		},
	)
)
