package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleOpsTreasuryInvoicesUnavailableWithoutSpanner(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/treasury?view=invoices&warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1"})
	rr := httptest.NewRecorder()
	svc.HandleOpsTreasury(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusServiceUnavailable, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, has := body["invoices"]; has {
		t.Fatal("must not invent invoices[] on query failure")
	}
	if body["error"] != "invoices_unavailable" {
		t.Fatalf("error=%v", body["error"])
	}
}

func TestHandleOpsAnalyticsFallbackMarksBreakdownsUnavailable(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/analytics?period=7d&warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1"})
	rr := httptest.NewRecorder()
	svc.HandleOpsAnalytics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, has := body["top_products"]; has {
		t.Fatal("fallback must not advertise empty top_products")
	}
	if _, has := body["daily_breakdown"]; has {
		t.Fatal("fallback must not advertise empty daily_breakdown")
	}
	if body["top_products_available"] != false {
		t.Fatalf("top_products_available=%v", body["top_products_available"])
	}
	if body["daily_breakdown_available"] != false {
		t.Fatalf("daily_breakdown_available=%v", body["daily_breakdown_available"])
	}
}

func TestHandleOpsFinancialsOmitsInventedGatewayBreakdown(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1", Currency: "UZS"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/financials?period=2026-05&warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1"})
	rr := httptest.NewRecorder()
	svc.HandleOpsFinancials(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, has := body["gateway_breakdown"]; has {
		t.Fatal("must not invent gateway_breakdown")
	}
	if body["gateway_breakdown_available"] != false {
		t.Fatalf("gateway_breakdown_available=%v", body["gateway_breakdown_available"])
	}
	if body["daily_revenue_available"] != false {
		t.Fatalf("daily_revenue_available=%v without Spanner", body["daily_revenue_available"])
	}
	if body["platform_fee_available"] != false {
		t.Fatalf("platform_fee_available=%v", body["platform_fee_available"])
	}
}

func TestHandleOpsFinancialsGatewayAvailableFromQuery(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1", Currency: "UZS"})
	svc.gatewayBreakdownQuery = func(ctx context.Context, warehouseID, period string) ([]map[string]any, bool) {
		return []map[string]any{{"gateway": "PAYME", "amount_minor": int64(1500)}}, true
	}
	svc.platformFeeQuery = func(ctx context.Context, warehouseID, period string) (int64, bool) {
		return 200, true
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/financials?period=2026-05&warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1"})
	rr := httptest.NewRecorder()
	svc.HandleOpsFinancials(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["gateway_breakdown_available"] != true {
		t.Fatalf("gateway_breakdown_available=%v body=%s", body["gateway_breakdown_available"], rr.Body.String())
	}
	if body["platform_fee_available"] != true {
		t.Fatalf("platform_fee_available=%v", body["platform_fee_available"])
	}
	if body["platform_fee"] != float64(200) {
		t.Fatalf("platform_fee=%v", body["platform_fee"])
	}
}

func TestDemandForecastNoScaffoldWithoutSeed(t *testing.T) {
	_ = os.Unsetenv("WAREHOUSE_PORTAL_SEED")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/demand/forecast?warehouse_id=wh-1&days=3", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1"})
	rr := httptest.NewRecorder()
	svc.HandleDemandForecast(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["source"] != "empty" {
		t.Fatalf("source=%v want empty", body["source"])
	}
	products, _ := body["products"].([]any)
	for _, raw := range products {
		row, _ := raw.(map[string]any)
		id, _ := row["product_id"].(string)
		if id == "prod-1" || id == "prod-2" {
			t.Fatalf("scaffold SKU leaked: %v", row)
		}
	}
	if len(products) != 0 {
		t.Fatalf("products=%v want empty without seed", products)
	}
}
