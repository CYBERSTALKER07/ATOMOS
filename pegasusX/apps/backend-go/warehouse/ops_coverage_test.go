package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

func TestHandleOpsCoverage_GET_UZPackCountry(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/coverage", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsCoverage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body warehouseOpsCoverageResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.WarehouseID != "wh-1" {
		t.Fatalf("warehouse_id=%q", body.WarehouseID)
	}
	if body.CountryCode != "UZ" {
		t.Fatalf("country_code=%q", body.CountryCode)
	}
	if body.Mode != proximity.CoverageModeCountryClosest {
		t.Fatalf("mode=%q", body.Mode)
	}
	if body.Pins == nil || body.Cities == nil {
		t.Fatal("pins/cities must be arrays, not null")
	}
}

func TestHandleOpsCoverage_PUT_Rejected(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/ops/coverage", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsCoverage(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rr.Code)
	}
}

func TestHandleOpsCoverage_PlannedFailsClosed(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/coverage", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "CA",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsCoverage(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleOpsSupplyFactory_GET_Engine(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	svc.resolveSupplyContext = func(_ context.Context, warehouseID string) (warehouseSupplyContext, error) {
		if warehouseID != "wh-1" {
			t.Fatalf("warehouseID=%q", warehouseID)
		}
		return warehouseSupplyContext{FactoryID: "fac-engine", TransferMode: supplier.TransferModeTruck}, nil
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/supply-factory", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsSupplyFactory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body warehouseOpsSupplyFactoryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.FactoryID != "fac-engine" || body.Source != "engine" || body.CountryCode != "UZ" {
		t.Fatalf("body=%+v", body)
	}
}

func TestHandleOpsSupplyFactory_Unassigned(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	svc.resolveSupplyContext = func(context.Context, string) (warehouseSupplyContext, error) {
		return warehouseSupplyContext{}, proximity.ErrFactoryUnassigned
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/supply-factory", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsSupplyFactory(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != proximity.ErrFactoryUnassigned.Error() {
		t.Fatalf("error=%q", body["error"])
	}
}

func TestHandleOpsSupplyFactory_PUT_Rejected(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/ops/supply-factory", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsSupplyFactory(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rr.Code)
	}
}
