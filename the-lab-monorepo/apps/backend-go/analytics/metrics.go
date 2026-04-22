package analytics

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// ─── Lightweight Process Metrics (no Prometheus dependency) ─────────────────
// Exposes runtime, connection, and request counters at GET /v1/metrics.
// When Prometheus client_golang is added later, these counters can be
// registered as prometheus.Gauge/Counter trivially.

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
