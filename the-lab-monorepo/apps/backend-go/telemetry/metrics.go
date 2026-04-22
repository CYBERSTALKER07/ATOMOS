// Package telemetry — Prometheus metric exports.
//
// Registers all V.O.I.D. operational gauges once at init time and exposes
// them for collection by the Prometheus sidecar via /metrics. The HTTP
// handler is mounted by main.go via RegisterMetricsHandler.
//
// Counter/gauge ownership:
//   - kafka_consumer_lag_seconds      — updated by kafka/workerpool
//   - outbox_relay_lag_seconds        — updated by outbox/relay
//   - redis_circuit_breaker_state     — updated by cache/circuitbreaker
//   - spanner_stale_read_age_seconds  — updated by bootstrap Spanner dialer
//   - ws_pubsub_failures_total        — updated by ws hubs
//   - grpc_optimizer_calls_total      — updated by dispatch/plan
//   - grpc_optimizer_latency_seconds  — updated by dispatch/plan
package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ─── Kafka consumer lag ───────────────────────────────────────────────────────

// KafkaConsumerLag measures the age of the last committed offset relative to
// the topic head. Exported as kafka_consumer_lag_seconds{topic,partition}.
// Alert threshold: > 10 s sustained for 1 min (per doctrine).
var KafkaConsumerLag = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "consumer_lag_seconds",
		Help:      "Offset lag of each consumer group partition in seconds.",
	},
	[]string{"topic", "partition"},
)

// ─── Outbox relay lag ─────────────────────────────────────────────────────────

// OutboxRelayLag measures the age of the oldest unpublished outbox row in
// seconds. Exported as void_outbox_relay_lag_seconds.
// Alert threshold: > 60 s (stuck-event watchdog per doctrine).
var OutboxRelayLag = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "void",
	Subsystem: "outbox",
	Name:      "relay_lag_seconds",
	Help:      "Age of the oldest unpublished OutboxEvents row in seconds.",
})

// ─── Redis circuit-breaker state ──────────────────────────────────────────────

// RedisCircuitBreakerState is 0 (CLOSED), 1 (HALF-OPEN), or 2 (OPEN).
// Exported as void_redis_circuit_breaker_state.
// Alert: sustained 2 (OPEN) > 5 min pages on-call.
var RedisCircuitBreakerState = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "void",
	Subsystem: "redis",
	Name:      "circuit_breaker_state",
	Help:      "Redis circuit breaker state: 0=CLOSED 1=HALF_OPEN 2=OPEN.",
})

// ─── Spanner stale read age ───────────────────────────────────────────────────

// SpannerStaleReadAge is the configured stale read bound in seconds. Surfaced
// so an alert can fire when the effective staleness unexpectedly drifts from
// the 15 s default (e.g. during Spanner leader elections).
var SpannerStaleReadAge = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "void",
	Subsystem: "spanner",
	Name:      "stale_read_age_seconds",
	Help:      "Configured stale read bound in seconds (target: 15 s).",
})

// ─── WebSocket Pub/Sub failures ──────────────────────────────────────────────

// WSPubSubFailures counts Redis Pub/Sub broadcast failures. Exported as
// void_ws_pubsub_failures_total{hub}. A pod broadcasting locally but failing
// cross-pod relay is exactly the scenario this detects.
var WSPubSubFailures = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "pubsub_failures_total",
		Help:      "Total Redis Pub/Sub broadcast failures per hub.",
	},
	[]string{"hub"},
)

// ─── gRPC optimizer calls ────────────────────────────────────────────────────

// GRPCOptimizerCalls counts calls to the gRPC optimizer service, labelled by
// result (ok | timeout | error | fallback).
var GRPCOptimizerCalls = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "grpc",
		Name:      "optimizer_calls_total",
		Help:      "Total gRPC optimizer Solve calls by result.",
	},
	[]string{"result"},
)

// GRPCOptimizerLatency is a histogram of gRPC optimizer call durations.
var GRPCOptimizerLatency = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: "void",
	Subsystem: "grpc",
	Name:      "optimizer_latency_seconds",
	Help:      "Latency of gRPC optimizer Solve calls.",
	Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1.0, 1.5, 2.5},
})

// ─── Active order counts ──────────────────────────────────────────────────────

// ActiveOrders is a real-time gauge of orders in each status bucket.
// Updated by order/service.go on state transitions.
var ActiveOrders = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "void",
		Subsystem: "orders",
		Name:      "active_total",
		Help:      "Current count of active orders per status.",
	},
	[]string{"status"},
)

// ─── HTTP handler ──────────────────────────────────────────────────────────────

// RegisterMetricsHandler mounts the Prometheus text handler at /metrics on
// the provided mux. Called from main() before ListenAndServe.
func RegisterMetricsHandler(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}

// Handler returns the Prometheus HTTP handler for use with chi.Mount.
func Handler() http.Handler {
	return promhttp.Handler()
}
