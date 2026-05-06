package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ─── Atomic Counters ────────────────────────────────────────────────────────

func TestIncrementRequest(t *testing.T) {
	before := TotalRequests.Load()
	activeBefore := ActiveRequests.Load()
	IncrementRequest()
	if TotalRequests.Load() != before+1 {
		t.Error("TotalRequests did not increment")
	}
	if ActiveRequests.Load() != activeBefore+1 {
		t.Error("ActiveRequests did not increment")
	}
}

func TestDecrementRequest(t *testing.T) {
	IncrementRequest() // ensure at least 1 active
	activeBefore := ActiveRequests.Load()
	DecrementRequest()
	if ActiveRequests.Load() != activeBefore-1 {
		t.Error("ActiveRequests did not decrement")
	}
}

func TestIncrementError(t *testing.T) {
	before := TotalErrors.Load()
	IncrementError()
	if TotalErrors.Load() != before+1 {
		t.Error("TotalErrors did not increment")
	}
}

// ─── HandleMetrics ──────────────────────────────────────────────────────────

func TestHandleMetrics_StatusOK(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/metrics", nil)
	HandleMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestHandleMetrics_ContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/metrics", nil)
	HandleMetrics(rr, req)
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
}

func TestHandleMetrics_ValidJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/metrics", nil)
	HandleMetrics(rr, req)

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check required keys
	requiredKeys := []string{
		"uptime_seconds", "go_version", "goroutines", "num_cpu",
		"total_requests", "active_requests", "total_errors",
		"ws_connections", "memory",
	}
	for _, key := range requiredKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("missing key %q in metrics response", key)
		}
	}
}

func TestHandleMetrics_MemorySubKeys(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/metrics", nil)
	HandleMetrics(rr, req)

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)

	mem, ok := result["memory"].(map[string]interface{})
	if !ok {
		t.Fatal("memory is not an object")
	}

	memKeys := []string{"alloc_mb", "total_alloc_mb", "sys_mb", "heap_objects", "gc_cycles", "gc_pause_total_ms"}
	for _, key := range memKeys {
		if _, ok := mem[key]; !ok {
			t.Errorf("missing memory key %q", key)
		}
	}
}

func TestHandleMetrics_UptimePositive(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/metrics", nil)
	HandleMetrics(rr, req)

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)

	uptime, ok := result["uptime_seconds"].(float64)
	if !ok {
		t.Fatal("uptime_seconds not a number")
	}
	if uptime < 0 {
		t.Error("uptime cannot be negative")
	}
}

func TestRegisterMetricsRoutes_MountsJSONAndPrometheus(t *testing.T) {
	mux := http.NewServeMux()
	RegisterMetricsRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next })

	jsonRecorder := httptest.NewRecorder()
	jsonReq := httptest.NewRequest(http.MethodGet, "/v1/metrics", nil)
	mux.ServeHTTP(jsonRecorder, jsonReq)
	if jsonRecorder.Code != http.StatusOK {
		t.Errorf("/v1/metrics status = %d, want 200", jsonRecorder.Code)
	}
	if jsonRecorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("/v1/metrics content-type = %q, want application/json", jsonRecorder.Header().Get("Content-Type"))
	}

	promRecorder := httptest.NewRecorder()
	promReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mux.ServeHTTP(promRecorder, promReq)
	if promRecorder.Code != http.StatusOK {
		t.Errorf("/metrics status = %d, want 200", promRecorder.Code)
	}
	if promRecorder.Header().Get("Content-Type") == "application/json" {
		t.Error("/metrics returned JSON content type, want Prometheus text format")
	}
}

func TestRegisterMetricsRoutes_ChiRouterCompatibility(t *testing.T) {
	router := chi.NewRouter()
	RegisterMetricsRoutes(router, func(next http.HandlerFunc) http.HandlerFunc { return next })

	jsonRecorder := httptest.NewRecorder()
	jsonReq := httptest.NewRequest(http.MethodGet, "/v1/metrics", nil)
	router.ServeHTTP(jsonRecorder, jsonReq)
	if jsonRecorder.Code != http.StatusOK {
		t.Errorf("/v1/metrics status = %d, want 200", jsonRecorder.Code)
	}

	promRecorder := httptest.NewRecorder()
	promReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(promRecorder, promReq)
	if promRecorder.Code != http.StatusOK {
		t.Errorf("/metrics status = %d, want 200", promRecorder.Code)
	}
}
