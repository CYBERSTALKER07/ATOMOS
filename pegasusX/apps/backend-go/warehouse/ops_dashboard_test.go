package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleDashboard_DoesNotInventHistoryOrTodayMoney(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/dashboard?warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"sparkline_active_orders", "sparkline_revenue", "sparkline_completed", "completed_today", "today_revenue"} {
		if _, has := body[key]; has {
			t.Fatalf("must not emit invented %s", key)
		}
	}
	if body["history_available"] != false {
		t.Fatalf("history_available=%v", body["history_available"])
	}
	if body["completed_today_available"] != false {
		t.Fatalf("completed_today_available=%v", body["completed_today_available"])
	}
	if body["today_revenue_available"] != false {
		t.Fatalf("today_revenue_available=%v", body["today_revenue_available"])
	}
}

func TestHandleDashboard_ETag304(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/dashboard?warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", rr.Code, rr.Body.String())
	}
	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/dashboard?warehouse_id=wh-1", nil)
	req2 = withWarehouseClaims(req2, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	svc.HandleDashboard(rr2, req2)
	if rr2.Code != http.StatusNotModified {
		t.Fatalf("second status=%d want 304 body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestHandleDashboard_FullOrderAndTruckDuty(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID: "sup-1",
		OpsOrders: func(context.Context, string, int) ([]OrderRow, error) {
			return []OrderRow{
				{OrderID: "o1", Status: "PENDING"},
				{OrderID: "o2", Status: "FISCAL_FAILED"},
				{OrderID: "o3", Status: "EN_ROUTE"},
			}, nil
		},
		OpsDrivers: func(context.Context, string) ([]PortalDriver, error) {
			return []PortalDriver{
				{DriverID: "d1", IsActive: true, OnShift: true, TruckStatus: "AVAILABLE", VehicleID: "v1", VehicleIsActive: true},
				{DriverID: "d2", IsActive: true, OnShift: false, UnavailableReason: "RETURNING_TO_WAREHOUSE"},
				{DriverID: "d3", IsActive: true, OnShift: false},
				{DriverID: "d4", IsActive: true, OnShift: true},
				{DriverID: "d5", IsActive: true, OnShift: true, VehicleID: "v5", VehicleIsActive: false, VehicleUnavailableReason: "MAINTENANCE"},
			}, nil
		},
		OpsVehicles: func(context.Context, string) ([]PortalVehicle, error) {
			return []PortalVehicle{
				{VehicleID: "v5", IsActive: false, UnavailableReason: "MAINTENANCE"},
			}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/dashboard?warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	orders, _ := body["orders_by_status"].(map[string]any)
	if len(orders) != 17 {
		t.Fatalf("orders_by_status keys=%d want 17", len(orders))
	}
	if orders["PENDING"] != float64(1) || orders["FISCAL_FAILED"] != float64(1) || orders["IN_TRANSIT"] != float64(1) {
		t.Fatalf("orders_by_status=%v", orders)
	}
	if orders["COMPLETED"] != float64(0) {
		t.Fatalf("zero COMPLETED must stay visible, got %v", orders["COMPLETED"])
	}
	duty, _ := body["truck_duty"].(map[string]any)
	for _, key := range []string{"OFF_SHIFT", "RETURNING_TO_WAREHOUSE", "UNASSIGNED", "VEHICLE_INACTIVE", "AVAILABLE"} {
		if _, ok := duty[key]; !ok {
			t.Fatalf("truck_duty missing %s: %v", key, duty)
		}
	}
	if duty["AVAILABLE"] != float64(1) || duty["RETURNING_TO_WAREHOUSE"] != float64(1) || duty["OFF_SHIFT"] != float64(1) {
		t.Fatalf("truck_duty=%v", duty)
	}
	if duty["UNASSIGNED"] != float64(1) || duty["VEHICLE_INACTIVE"] != float64(1) {
		t.Fatalf("truck_duty unassigned/inactive=%v", duty)
	}
	if body["demand_source"] != "empty" {
		t.Fatalf("demand_source=%v want empty", body["demand_source"])
	}
	holds, _ := body["hold_reasons"].([]any)
	if len(holds) == 0 {
		t.Fatalf("hold_reasons empty: %v", body["hold_reasons"])
	}
	foundMaintenance := false
	for _, row := range holds {
		item, _ := row.(map[string]any)
		if item["code"] == "MAINTENANCE" {
			foundMaintenance = true
		}
	}
	if !foundMaintenance {
		t.Fatalf("hold_reasons missing MAINTENANCE: %v", holds)
	}
	fleet, _ := body["fleet_status"].([]any)
	if len(fleet) != 8 {
		t.Fatalf("fleet_status len=%d want 8", len(fleet))
	}
}

func TestClassifyWarehouseTruckDuty(t *testing.T) {
	if got := classifyWarehouseTruckDuty(PortalDriver{IsActive: false}); got != "INACTIVE" {
		t.Fatalf("inactive=%s", got)
	}
	if got := classifyWarehouseTruckDuty(PortalDriver{IsActive: true, OnShift: false, UnavailableReason: "RETURNING_TO_WAREHOUSE"}); got != "RETURNING_TO_WAREHOUSE" {
		t.Fatalf("returning=%s", got)
	}
	if got := classifyWarehouseTruckDuty(PortalDriver{IsActive: true, OnShift: true}); got != "UNASSIGNED" {
		t.Fatalf("unassigned=%s", got)
	}
}
