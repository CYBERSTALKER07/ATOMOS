package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func decodeDashboard(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode dashboard: %v body=%s", err, rr.Body.String())
	}
	return body
}

func intField(body map[string]any, key string) int {
	switch v := body[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return -1
	}
}

func countMap(t *testing.T, body map[string]any, key string) map[string]int {
	t.Helper()
	raw, ok := body[key].(map[string]any)
	if !ok {
		t.Fatalf("%s missing or not an object: %T", key, body[key])
	}
	out := make(map[string]int, len(raw))
	for k, v := range raw {
		n, _ := v.(float64)
		out[k] = int(n)
	}
	return out
}

func TestHandleDashboard_EmptyWhenSeedOff(t *testing.T) {
	t.Setenv("FACTORY_PORTAL_SEED", "")
	t.Setenv("USE_DEMO_SEED", "")
	svc := NewService(ServiceConfig{
		SupplierID: "supplier-empty",
		Now:        func() time.Time { return time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC) },
	})

	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, httptest.NewRequest(http.MethodGet, "/v1/factory/dashboard", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := decodeDashboard(t, rr)
	if body["source"] != "empty" {
		t.Fatalf("source=%v want empty", body["source"])
	}
	if body["plane"] != "factory_trucks" {
		t.Fatalf("plane=%v want factory_trucks", body["plane"])
	}
	if intField(body, "pending_transfers") != 0 || intField(body, "loading_transfers") != 0 {
		t.Fatalf("empty source invented transfer KPIs: %+v", body)
	}
	if intField(body, "staff_on_shift") != 0 {
		t.Fatalf("empty source invented staff_on_shift=%v", body["staff_on_shift"])
	}
	transfers := countMap(t, body, "transfers_by_state")
	if len(transfers) != len(factoryTransferStates) {
		t.Fatalf("transfers_by_state keys=%d want %d", len(transfers), len(factoryTransferStates))
	}
	for _, key := range factoryTransferStates {
		if transfers[key] != 0 {
			t.Fatalf("empty %s=%d", key, transfers[key])
		}
	}
	if _, ok := body["truck_duty"]; ok {
		t.Fatal("factory dashboard must not emit last-mile truck_duty")
	}
	if _, ok := body["orders_by_status"]; ok {
		t.Fatal("factory dashboard must not emit last-mile orders_by_status")
	}
}

func TestHandleDashboard_MemoryWhenSeedOn(t *testing.T) {
	t.Setenv("FACTORY_PORTAL_SEED", "true")
	svc := NewService(ServiceConfig{
		SupplierID: "supplier-memory",
		Now:        func() time.Time { return time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC) },
	})

	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, httptest.NewRequest(http.MethodGet, "/v1/factory/dashboard", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := decodeDashboard(t, rr)
	if body["source"] != "memory" {
		t.Fatalf("source=%v want memory", body["source"])
	}
	transfers := countMap(t, body, "transfers_by_state")
	if transfers["ASSIGNED"] < 1 || transfers["APPROVED"] < 1 || transfers["LOADING"] < 1 {
		t.Fatalf("seed transfers missing: %+v", transfers)
	}
	manifests := countMap(t, body, "manifests_by_state")
	if manifests["DRAFT"] < 1 {
		t.Fatalf("seed manifests missing DRAFT: %+v", manifests)
	}
	if intField(body, "staff_on_shift") != 2 {
		t.Fatalf("staff_on_shift=%d want 2 (no invented fallback)", intField(body, "staff_on_shift"))
	}
}

func TestHandleDashboard_SpannerBackedCounts(t *testing.T) {
	t.Setenv("FACTORY_PORTAL_SEED", "")
	now := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		SupplierID:    "supplier-spanner",
		FactoryNodeID: "factory-spanner",
		Now:           func() time.Time { return now },
		DashboardQuery: func(context.Context, string, string) (factoryDashboardSnapshot, error) {
			return factoryDashboardSnapshot{
				Transfers: []TransferRow{
					{TransferID: "tr-1", State: "CREATED"},
					{TransferID: "tr-2", State: "DISPATCHED"},
					{TransferID: "tr-3", State: "CANCELLED"},
				},
				Manifests: []ManifestRow{
					{ManifestID: "mf-1", State: manifestStateLoading},
					{ManifestID: "mf-2", State: manifestStateSealed},
				},
				Vehicles: []FleetVehicle{
					{VehicleID: "v1", State: "READY"},
					{VehicleID: "v2", State: "MAINTENANCE"},
				},
				Drivers: []FleetDriver{
					{DriverID: "d1", OnShift: true},
					{DriverID: "d2", OnShift: false},
				},
				Exceptions: []ManifestException{{
					ExceptionID: "ex-1",
					ManifestID:  "mf-1",
					TransferID:  "tr-1",
					Reason:      "OVERFLOW",
				}},
				Requests: []SupplyRequest{{
					RequestID: "sr-1",
					Status:    "SUBMITTED",
					CreatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339Nano),
				}},
				QC:          map[string]string{"sr-1": "FAIL"},
				QCAvailable: true,
			}, nil
		},
	})

	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, httptest.NewRequest(http.MethodGet, "/v1/factory/dashboard", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := decodeDashboard(t, rr)
	if body["source"] != "spanner" {
		t.Fatalf("source=%v want spanner", body["source"])
	}
	if body["plane"] != "factory_trucks" {
		t.Fatalf("plane=%v", body["plane"])
	}
	transfers := countMap(t, body, "transfers_by_state")
	if transfers["CREATED"] != 1 || transfers["DISPATCHED"] != 1 || transfers["CANCELLED"] != 1 {
		t.Fatalf("spanner transfers=%+v", transfers)
	}
	if transfers["PENDING"] != 0 {
		t.Fatalf("zero PENDING must stay visible, got %d", transfers["PENDING"])
	}
	manifests := countMap(t, body, "manifests_by_state")
	if manifests["LOADING"] != 1 || manifests["SEALED"] != 1 || manifests["DRAFT"] != 0 {
		t.Fatalf("spanner manifests=%+v", manifests)
	}
	vehicles := countMap(t, body, "vehicles_by_state")
	if vehicles["READY"] != 1 || vehicles["UNAVAILABLE"] != 1 {
		t.Fatalf("spanner vehicles=%+v", vehicles)
	}
	duty := countMap(t, body, "driver_duty")
	if duty["ON_SHIFT"] != 1 || duty["OFF_SHIFT"] != 1 {
		t.Fatalf("spanner driver_duty=%+v", duty)
	}
	if intField(body, "staff_on_shift") != 1 {
		t.Fatalf("staff_on_shift=%d want 1", intField(body, "staff_on_shift"))
	}
	if intField(body, "critical_insights") != 1 {
		t.Fatalf("critical_insights=%d want 1", intField(body, "critical_insights"))
	}
	qc := countMap(t, body, "qc_by_result")
	if qc["FAIL"] != 1 || body["qc_available"] != true {
		t.Fatalf("qc=%+v available=%v", qc, body["qc_available"])
	}
	if intField(body, "active_manifests") != 1 {
		t.Fatalf("active_manifests=%d want 1 (LOADING only)", intField(body, "active_manifests"))
	}
}
