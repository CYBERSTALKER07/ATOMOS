package analytics

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"backend-go/telemetry"
)

// ─── Lightweight Process Metrics ────────────────────────────────────────────
// Exposes runtime, connection, and request counters at GET /v1/metrics.
// Prometheus metrics are exposed separately at GET /metrics.

var (
	// Request counters (atomically incremented by middleware)
	TotalRequests  atomic.Int64
	ActiveRequests atomic.Int64
	TotalErrors    atomic.Int64

	// WS counters (incremented by hubs)
	WSConnections atomic.Int64

	startTime = time.Now()
)

// IncrementRequest should be called at the start of each HTTP request.
func IncrementRequest() { TotalRequests.Add(1); ActiveRequests.Add(1) }

// DecrementRequest should be called when the HTTP request finishes.
func DecrementRequest() { ActiveRequests.Add(-1) }

// IncrementError tracks 5xx responses.
func IncrementError() { TotalErrors.Add(1) }

// routeRegistrar captures the minimal route registration surface shared by
// chi routers and net/http ServeMux.
type routeRegistrar interface {
	Handle(pattern string, h http.Handler)
}

// RegisterMetricsRoutes mounts legacy JSON metrics and Prometheus metrics.
func RegisterMetricsRoutes(r routeRegistrar, log func(http.HandlerFunc) http.HandlerFunc) {
	r.Handle("/metrics", telemetry.Handler())
	r.Handle("/v1/metrics", log(HandleMetrics))
}

// HandleMetrics returns process-level metrics as JSON.
// Designed for load balancers, dashboards, and lightweight monitoring.
func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	resp := map[string]interface{}{
		"uptime_seconds":  int64(time.Since(startTime).Seconds()),
		"go_version":      runtime.Version(),
		"goroutines":      runtime.NumGoroutine(),
		"num_cpu":         runtime.NumCPU(),
		"total_requests":  TotalRequests.Load(),
		"active_requests": ActiveRequests.Load(),
		"total_errors":    TotalErrors.Load(),
		"ws_connections":  WSConnections.Load(),
		"memory": map[string]interface{}{
			"alloc_mb":          mem.Alloc / 1024 / 1024,
			"total_alloc_mb":    mem.TotalAlloc / 1024 / 1024,
			"sys_mb":            mem.Sys / 1024 / 1024,
			"heap_objects":      mem.HeapObjects,
			"gc_cycles":         mem.NumGC,
			"gc_pause_total_ms": mem.PauseTotalNs / 1_000_000,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
