package valuationroutes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
)

const testJWTSecret = "test-valuation-secret-0123456789ab"

func issueCookie(t *testing.T, c auth.Claims) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: testJWTSecret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func doReq(t *testing.T, r http.Handler, method, target string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, target, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestValuationRoutes_RBAC_TenantIsolation_And_ReceiptLifecycle(t *testing.T) {
	t.Parallel()
	svc := valuation.NewServiceWithRepository(valuation.NewMemoryRepository())
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testJWTSecret})

	whAdmin := issueCookie(t, auth.Claims{
		Subject:      "usr-wh-mgr",
		Role:         auth.RoleWarehouseAdmin,
		SupplierID:   "sup-v1",
		SupplierRole: auth.SupplierRoleWarehouseManager,
	})
	auditor := issueCookie(t, auth.Claims{
		Subject:      "usr-aud",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-v1",
		SupplierRole: auth.SupplierRoleAuditor,
	})
	retailer := issueCookie(t, auth.Claims{
		Subject: "usr-ret",
		Role:    auth.RoleRetailer,
	})

	// 1. Unauthenticated must 401.
	if rec := doReq(t, r, http.MethodGet, "/v1/inventory/valuation", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth GET valuation = %d, want 401", rec.Code)
	}

	// 2. Retailer must 403.
	if rec := doReq(t, r, http.MethodGet, "/v1/inventory/valuation", nil, retailer); rec.Code != http.StatusForbidden {
		t.Fatalf("retailer GET valuation = %d, want 403", rec.Code)
	}

	// 3. Warehouse admin cross-tenant IDOR must 403.
	if rec := doReq(t, r, http.MethodGet, "/v1/inventory/valuation?entity_id=sup-other", nil, whAdmin); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET valuation = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}

	// 4. Auditor cannot POST receipt (403), but CAN GET valuation & movements (200).
	receiptPayload := map[string]any{
		"warehouse_id":    "wh-a",
		"product_id":      "sku-a",
		"lot_id":          "lot-a",
		"receipt_ref_id":  "po-a-1",
		"quantity":        10,
		"unit_cost_minor": 2500,
		"vat_minor":       3000,
		"credit_account":  "AP:VENDOR",
		"vendor_id":       "vend-1",
		"idempotency_key": "rt-rec-1",
	}
	if rec := doReq(t, r, http.MethodPost, "/v1/inventory/cost-layers/receipt", receiptPayload, auditor); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST receipt = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}

	// 5. Warehouse admin POST receipt -> 201 Created, replay -> 200 OK.
	recCreate := doReq(t, r, http.MethodPost, "/v1/inventory/cost-layers/receipt", receiptPayload, whAdmin)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("POST receipt = %d, want 201 (%s)", recCreate.Code, recCreate.Body.String())
	}
	recReplay := doReq(t, r, http.MethodPost, "/v1/inventory/cost-layers/receipt", receiptPayload, whAdmin)
	if recReplay.Code != http.StatusOK {
		t.Fatalf("replay POST receipt = %d, want 200 (%s)", recReplay.Code, recReplay.Body.String())
	}

	// 6. Auditor GET valuation & GET movements -> 200 OK with expected integer totals.
	recVal := doReq(t, r, http.MethodGet, "/v1/inventory/valuation", nil, auditor)
	if recVal.Code != http.StatusOK {
		t.Fatalf("auditor GET valuation = %d, want 200 (%s)", recVal.Code, recVal.Body.String())
	}
	var valBody valuation.ValuationSummaryResponse
	if err := json.Unmarshal(recVal.Body.Bytes(), &valBody); err != nil {
		t.Fatalf("decode valuation summary: %v", err)
	}
	if valBody.TotalQtyRemaining != 10 || valBody.TotalValueMinor != 25000 {
		t.Fatalf("valBody = %+v, want qty=10 value=25000", valBody)
	}

	recMov := doReq(t, r, http.MethodGet, "/v1/inventory/movements?warehouse_id=wh-a", nil, auditor)
	if recMov.Code != http.StatusOK {
		t.Fatalf("auditor GET movements = %d, want 200 (%s)", recMov.Code, recMov.Body.String())
	}
	var movResp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(recMov.Body.Bytes(), &movResp); err != nil {
		t.Fatalf("decode movements: %v", err)
	}
	if movResp.Count != 1 {
		t.Fatalf("movements count = %d, want 1", movResp.Count)
	}
}
