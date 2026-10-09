package optimizerclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

func TestClient_Solve_TraceContextPropagation(t *testing.T) {
	t.Parallel()

	expectedTraceID := "trace-client-test-999"
	expectedTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	var receivedTraceID string
	var receivedTraceParent string
	var receivedAuthHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != contract.SolvePath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		receivedTraceID = r.Header.Get("X-Trace-Id")
		receivedTraceParent = r.Header.Get("traceparent")
		receivedAuthHeader = r.Header.Get(contract.AuthHeader)

		resp := contract.SolveResponse{
			V:       contract.V,
			TraceID: receivedTraceID,
			Routes:  []contract.Route{},
			Orphans: []contract.Orphan{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cli := New(srv.URL, "test-api-key")

	ctx := outbox.WithTraceParent(context.Background(), expectedTraceParent)
	ctx = outbox.WithTraceID(ctx, expectedTraceID)

	in := SolveInput{
		TraceID:       expectedTraceID,
		DepotLat:      41.311081,
		DepotLng:      69.240562,
		DepartureTime: time.Now(),
		Orders: []dispatch.GeoOrder{
			{
				OrderID: "order-1",
				Lat:     41.312,
				Lng:     69.241,
				Volume:  0.5,
			},
		},
		Fleet: []dispatch.AvailableDriver{
			{
				DriverID:    "drv-1",
				VehicleID:   "veh-1",
				MaxVolumeVU: 10.0,
			},
		},
	}

	res, err := cli.Solve(ctx, in)
	if err != nil {
		t.Fatalf("unexpected solve error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil AssignmentResult")
	}

	if receivedAuthHeader != "test-api-key" {
		t.Fatalf("expected AuthHeader 'test-api-key', got %q", receivedAuthHeader)
	}
	if receivedTraceID != expectedTraceID {
		t.Fatalf("expected X-Trace-Id %q, got %q", expectedTraceID, receivedTraceID)
	}
	if receivedTraceParent != expectedTraceParent {
		t.Fatalf("expected traceparent %q, got %q", expectedTraceParent, receivedTraceParent)
	}
}

func TestClient_Solve_TraceContextPropagation_FromContextFallback(t *testing.T) {
	t.Parallel()

	expectedTraceID := "ctx-trace-fallback-111"

	var receivedTraceID string
	var receivedTraceParent string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceID = r.Header.Get("X-Trace-Id")
		receivedTraceParent = r.Header.Get("traceparent")

		resp := contract.SolveResponse{
			V:       contract.V,
			TraceID: receivedTraceID,
			Routes:  []contract.Route{},
			Orphans: []contract.Orphan{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cli := New(srv.URL, "test-api-key")

	ctx := outbox.WithTraceID(context.Background(), expectedTraceID)

	in := SolveInput{
		TraceID:       "", // Empty in input struct, must fall back to context
		DepotLat:      41.311081,
		DepotLng:      69.240562,
		DepartureTime: time.Now(),
		Orders: []dispatch.GeoOrder{
			{
				OrderID: "order-1",
				Lat:     41.312,
				Lng:     69.241,
				Volume:  0.5,
			},
		},
		Fleet: []dispatch.AvailableDriver{
			{
				DriverID:    "drv-1",
				VehicleID:   "veh-1",
				MaxVolumeVU: 10.0,
			},
		},
	}

	_, err := cli.Solve(ctx, in)
	if err != nil {
		t.Fatalf("unexpected solve error: %v", err)
	}

	if receivedTraceID != expectedTraceID {
		t.Fatalf("expected X-Trace-Id %q, got %q", expectedTraceID, receivedTraceID)
	}
	if !strings.HasPrefix(receivedTraceParent, "00-") {
		t.Fatalf("expected valid W3C traceparent starting with '00-', got %q", receivedTraceParent)
	}
}

func TestClient_Solve_CircuitBreaker_TripsAndRecovers(t *testing.T) {
	t.Parallel()

	var failMode atomic.Bool
	failMode.Store(true)
	var requestCount atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if failMode.Load() {
			w.WriteHeader(http.StatusGatewayTimeout)
			_ = json.NewEncoder(w).Encode(contract.ErrorResponse{
				Code:    contract.ErrCodeTimeout,
				Message: "solver timed out",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(contract.SolveResponse{
			V:      contract.V,
			Routes: []contract.Route{},
		})
	}))
	defer srv.Close()

	breaker := circuit.New("test-optimizer", circuit.Config{
		FailureThreshold: 3,
		FailureWindow:    5 * time.Second,
		OpenDuration:     50 * time.Millisecond,
	})

	cli := New(srv.URL, "key").WithBreaker(breaker)

	input := SolveInput{
		DepotLat: 41.311,
		DepotLng: 69.240,
		Orders: []dispatch.GeoOrder{
			{OrderID: "o1", Lat: 41.312, Lng: 69.241, Volume: 1.0},
		},
		Fleet: []dispatch.AvailableDriver{
			{DriverID: "d1", VehicleID: "v1", MaxVolumeVU: 10.0},
		},
	}

	ctx := context.Background()

	// 1. Initial 3 calls fail with 504 Gateway Timeout -> trips circuit breaker
	for i := 1; i <= 3; i++ {
		_, err := cli.Solve(ctx, input)
		if err == nil {
			t.Fatalf("call %d: expected error", i)
		}
	}

	if breaker.State() != circuit.StateOpen {
		t.Fatalf("expected breaker StateOpen, got %v", breaker.State())
	}

	// 2. 4th call: breaker is OPEN -> fails immediately in <1ms without calling server
	reqBefore := requestCount.Load()
	start := time.Now()
	_, err := cli.Solve(ctx, input)
	duration := time.Since(start)

	if !errors.Is(err, circuit.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable when open, got %v", err)
	}
	if duration > 10*time.Millisecond {
		t.Fatalf("fail-fast open circuit took too long: %v", duration)
	}
	if requestCount.Load() != reqBefore {
		t.Fatalf("open circuit must not hit upstream HTTP server")
	}

	// 3. Wait for OpenDuration (50ms) to elapse -> transitions to StateHalfOpen
	time.Sleep(60 * time.Millisecond)

	// 4. Upstream recovers: switch server to 200 OK
	failMode.Store(false)

	// 5. Probe request succeeds -> breaker resets to StateClosed
	res, err := cli.Solve(ctx, input)
	if err != nil {
		t.Fatalf("expected successful recovery solve, got: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil AssignmentResult on recovery")
	}
	if breaker.State() != circuit.StateClosed {
		t.Fatalf("expected breaker to reset to StateClosed, got %v", breaker.State())
	}
}
