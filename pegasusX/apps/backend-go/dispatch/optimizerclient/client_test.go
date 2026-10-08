package optimizerclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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
