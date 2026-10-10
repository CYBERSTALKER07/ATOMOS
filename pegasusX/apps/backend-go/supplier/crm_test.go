package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestHandleCRMRetailers_NoSpanner503(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/crm/retailers", nil)
	rr := httptest.NewRecorder()
	svc.HandleCRMRetailers(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, has := body["retailers"]; has {
		t.Fatal("must not invent retailers[] without Spanner")
	}
	if body["error"] != "crm_unavailable" {
		t.Fatalf("error=%v", body["error"])
	}
}

func TestHandleCRMRetailers_EmptyListShape(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"retailers": []crmRetailer{}})
	var body struct {
		Retailers []crmRetailer `json:"retailers"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Retailers == nil {
		t.Fatal("retailers must be [] not null")
	}
	if len(body.Retailers) != 0 {
		t.Fatalf("len=%d", len(body.Retailers))
	}
}

func TestHandleCRMRetailerDetail_MissingID(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/crm/retailers/", nil)
	rctx := chi.NewRouteContext()
	req = req.WithContext(req.Context())
	_ = rctx
	rr := httptest.NewRecorder()
	svc.HandleCRMRetailerDetail(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing retailerId must 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCRMRetailerDetail_NoSpanner503(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/crm/retailers/ret-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("retailerId", "ret-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	svc.HandleCRMRetailerDetail(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("no Spanner must 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCRMRetailerJSONOmitsEmptyEmail(t *testing.T) {
	raw, err := json.Marshal(crmRetailer{RetailerID: "r1", RetailerName: "Shop", Lifetime: 10, OrderCount: 1, Status: "ACTIVE"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["email"]; has {
		t.Fatal("empty email must omit (omitempty); never invent an address")
	}
	for _, key := range []string{"retailer_id", "retailer_name", "lifetime", "order_count", "status"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing %s", key)
		}
	}
}

func TestCRMRetailerJSONIncludesEmailWhenSet(t *testing.T) {
	raw, err := json.Marshal(crmRetailer{RetailerID: "r1", RetailerName: "Shop", Email: "shop@example.uz", Lifetime: 10, OrderCount: 1, Status: "ACTIVE"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["email"] != "shop@example.uz" {
		t.Fatalf("email=%v", m["email"])
	}
}

func TestCRMStatusActiveWindow(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	if crmStatus(now.Add(-29*24*time.Hour), now) != "ACTIVE" {
		t.Fatal("29d should be ACTIVE")
	}
	if crmStatus(now.Add(-30*24*time.Hour), now) != "ACTIVE" {
		t.Fatal("exactly 30d should be ACTIVE")
	}
	if crmStatus(now.Add(-31*24*time.Hour), now) != "INACTIVE" {
		t.Fatal("31d should be INACTIVE")
	}
	if crmStatus(time.Time{}, now) != "INACTIVE" {
		t.Fatal("zero last order")
	}
}

func TestCountCRMLineItems(t *testing.T) {
	if countCRMLineItems([]byte(`[{"sku":"a"},{"product_id":"b"}]`)) != 2 {
		t.Fatal("sku + product_id lines")
	}
	if countCRMLineItems(nil) != 0 {
		t.Fatal("empty")
	}
}
