package twin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockRepo struct {
	Repository // embed to satisfy interface
}

func (m *mockRepo) ListActiveRouteTwins(ctx context.Context, zoneH3 string) ([]RouteTwinView, error) {
	if zoneH3 == "zone1" {
		return []RouteTwinView{{RouteTwin: RouteTwin{RouteID: "r1", CurrentH3: "zone1"}}}, nil
	}
	return []RouteTwinView{}, nil
}

func (m *mockRepo) GetRouteTwin(ctx context.Context, routeID string) (*RouteTwinView, error) {
	if routeID == "r1" {
		return &RouteTwinView{RouteTwin: RouteTwin{RouteID: "r1"}}, nil
	}
	return nil, nil
}

func (m *mockRepo) GetVehicleInventory(ctx context.Context, routeID string) ([]VehicleInventory, error) {
	if routeID == "r1" {
		return []VehicleInventory{{RouteID: "r1", Sku: "sku1", QtyOnVehicle: 10}}, nil
	}
	return []VehicleInventory{}, nil
}

func TestHTTPHandler_ListActiveRoutes(t *testing.T) {
	handler := NewHTTPHandler(&mockRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/twin/routes/active?zoneH3=zone1", nil)
	rr := httptest.NewRecorder()

	handler.ListActiveRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", rr.Code)
	}

	var routes []RouteTwinView
	if err := json.NewDecoder(rr.Body).Decode(&routes); err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].RouteID != "r1" {
		t.Errorf("expected route r1, got %v", routes)
	}
}

func TestHTTPHandler_GetRoute(t *testing.T) {
	handler := NewHTTPHandler(&mockRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/twin/routes/r1", nil)
	rr := httptest.NewRecorder()

	handler.GetRoute(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", rr.Code)
	}

	var route RouteTwinView
	if err := json.NewDecoder(rr.Body).Decode(&route); err != nil {
		t.Fatal(err)
	}
	if route.RouteID != "r1" {
		t.Errorf("expected route r1, got %v", route)
	}

	// Test not found
	req2 := httptest.NewRequest(http.MethodGet, "/v1/twin/routes/r2", nil)
	rr2 := httptest.NewRecorder()
	handler.GetRoute(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound; got %v", rr2.Code)
	}
}

func TestHTTPHandler_GetRouteInventory(t *testing.T) {
	handler := NewHTTPHandler(&mockRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/twin/routes/r1/inventory", nil)
	rr := httptest.NewRecorder()

	handler.GetRouteInventory(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", rr.Code)
	}

	var inv []VehicleInventory
	if err := json.NewDecoder(rr.Body).Decode(&inv); err != nil {
		t.Fatal(err)
	}
	if len(inv) != 1 || inv[0].Sku != "sku1" {
		t.Errorf("expected sku1, got %v", inv)
	}
}
