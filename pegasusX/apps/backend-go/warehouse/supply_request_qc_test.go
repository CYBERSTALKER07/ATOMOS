package warehouse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSupplyRequestQC_NoSpanner503(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/supply-requests/req-1/qc", nil)
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "qc_unavailable" {
		t.Fatalf("error=%v", body["error"])
	}
}

func TestHandleSupplyRequestQC_PostNoSpanner503(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/supply-requests/req-1/qc", strings.NewReader(`{"result":"PASS"}`))
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", rr.Code)
	}
}

func TestPortalRetailerJSONShapeFrozen(t *testing.T) {
	raw, err := json.Marshal(portalRetailer{
		RetailerID: "r1", BusinessName: "Shop", TotalOrders: 2, TotalRevenue: 100, LastOrderDate: "2026-08-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"retailer_id", "business_name", "total_orders", "total_revenue", "last_order_date"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("warehouse CRM field %s missing — do not hijack this shape", key)
		}
	}
	if _, has := m["lifetime"]; has {
		t.Fatal("warehouse CRM must not grow supplier CRM lifetime field")
	}
	if _, has := m["retailer_name"]; has {
		t.Fatal("warehouse CRM must keep business_name, not retailer_name")
	}
}
