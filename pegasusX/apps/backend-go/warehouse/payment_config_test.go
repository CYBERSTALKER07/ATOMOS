package warehouse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleOpsPaymentConfig_GET_UZOmitsForeignRails(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/payment-config?warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsPaymentConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body warehousePaymentConfigResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, listing := range body.Catalog {
		if listing.Code == "STRIPE" || listing.Code == "ADYEN" {
			t.Fatalf("UZ GET catalog listed %s", listing.Code)
		}
	}
	if !containsSelected(body.SelectedGateways, "CASH") || !containsSelected(body.SelectedGateways, "GLOBAL_PAY") {
		t.Fatalf("empty UZ config selected=%#v", body.SelectedGateways)
	}
	if containsSelected(body.SelectedGateways, "ADYEN") {
		t.Fatal("empty UZ config must not invent Adyen")
	}
	if body.CurrencyCode != "UZS" {
		t.Fatalf("currency_code=%q", body.CurrencyCode)
	}
	if body.MarketCode != "UZ" {
		t.Fatalf("market_code=%q", body.MarketCode)
	}
}

func TestHandleOpsPaymentConfig_POST_StripeForbiddenOnUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/payment-config?warehouse_id=wh-1", strings.NewReader(`{"selected_gateways":["STRIPE"]}`))
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsPaymentConfig(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != auth.ErrPackGatewayForbidden.Error() {
		t.Fatalf("error=%q", body["error"])
	}
}

func TestHandleOpsPaymentConfig_POST_AdyenForbiddenOnUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/payment-config?warehouse_id=wh-1", strings.NewReader(`{"selected_gateways":["ADYEN"]}`))
	req = withWarehouseClaims(req, auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	})
	rr := httptest.NewRecorder()
	svc.HandleOpsPaymentConfig(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func containsSelected(gateways []string, want string) bool {
	for _, gateway := range gateways {
		if gateway == want {
			return true
		}
	}
	return false
}
