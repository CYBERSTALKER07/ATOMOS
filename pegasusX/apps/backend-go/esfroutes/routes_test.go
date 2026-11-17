package esfroutes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

const testESFJWTSecret = "test-esf-routes-secret-0123456789"

func issueESFCookie(t *testing.T, c auth.Claims) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: testESFJWTSecret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func doESFReq(t *testing.T, r http.Handler, method, target string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
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

func TestESFRoutes_AllEndpoints_RBAC_IDOR_And_Lifecycle(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	repo := esf.NewMemoryRepository()
	repo.SeedOrderFiscalSource(esf.OrderFiscalSource{
		OrderID:         "ord-r1",
		SupplierID:      "sup-r1",
		RetailerID:      "ret-r1",
		SupplierTIN:     "200100100",
		CounterpartyTIN: "300200200",
		Currency:        "UZS",
		TotalMinor:      112_000,
		TotalNetMinor:   100_000,
		TotalVatMinor:   12_000,
		TotalGrossMinor: 112_000,
		DominantVatBps:  1200,
		CreatedAt:       now,
	})
	repo.SeedGLVATBalance("sup-r1", "2026-11", "UZS", 12_000, 6_000)
	repo.SeedProductCompliance(esf.MemoryProductCompliance{
		ProductID: "sku-miss", SupplierID: "sup-r1", Name: "Flour 5kg", IsActive: true,
	})

	soliqSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"ehf_id": "ehf-rt-1"},
		})
	}))
	defer soliqSrv.Close()

	signer, err := fiscal.NewDevHMACSigner([]byte("test-esf-routes-hmac-secret-32byte"))
	if err != nil {
		t.Fatalf("NewDevHMACSigner: %v", err)
	}
	sc := soliq.NewClient(soliq.SoliqConfig{BaseURL: soliqSrv.URL, Operator: "direct", TIN: "200100100", Timeout: 5 * time.Second})
	svc := esf.NewService(esf.ServiceConfig{
		Repo: repo, SoliqClient: sc, Signer: signer, Now: func() time.Time { return now },
	})

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testESFJWTSecret})

	admin := issueESFCookie(t, auth.Claims{Subject: "usr-adm", Role: auth.RoleAdmin, SupplierID: "sup-r1", SupplierRole: auth.SupplierRoleOwner})
	auditor := issueESFCookie(t, auth.Claims{Subject: "usr-aud", Role: auth.RoleAdmin, SupplierID: "sup-r1", SupplierRole: auth.SupplierRoleAuditor})
	warehouse := issueESFCookie(t, auth.Claims{Subject: "usr-wh", Role: auth.RoleWarehouse, SupplierID: "sup-r1"})

	// 1. Unauthenticated 401, Warehouse role 403, cross-tenant IDOR 403.
	if rec := doESFReq(t, r, http.MethodGet, "/v1/fiscal/esf", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth GET /v1/fiscal/esf = %d, want 401", rec.Code)
	}
	if rec := doESFReq(t, r, http.MethodGet, "/v1/fiscal/esf", nil, warehouse); rec.Code != http.StatusForbidden {
		t.Fatalf("warehouse GET /v1/fiscal/esf = %d, want 403", rec.Code)
	}
	if rec := doESFReq(t, r, http.MethodGet, "/v1/fiscal/esf?entity_id=sup-other", nil, admin); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET /v1/fiscal/esf = %d, want 403", rec.Code)
	}

	// 2. Auditor blocked from POST /v1/fiscal/esf (403), Admin creates -> 201.
	if rec := doESFReq(t, r, http.MethodPost, "/v1/fiscal/esf", map[string]any{"order_id": "ord-r1"}, auditor); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST /v1/fiscal/esf = %d, want 403", rec.Code)
	}
	recCreate := doESFReq(t, r, http.MethodPost, "/v1/fiscal/esf", map[string]any{"order_id": "ord-r1", "operator": "direct"}, admin)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("admin POST /v1/fiscal/esf = %d (%s), want 201", recCreate.Code, recCreate.Body.String())
	}
	var outDoc esf.EsfDocument
	if err := json.Unmarshal(recCreate.Body.Bytes(), &outDoc); err != nil || outDoc.EsfID == "" {
		t.Fatalf("decode outbound ESF: (%+v, %v)", outDoc, err)
	}

	// 3. POST /v1/fiscal/esf/inbound -> 201.
	recIn := doESFReq(t, r, http.MethodPost, "/v1/fiscal/esf/inbound", map[string]any{
		"po_id": "po-rt-1", "counterparty_tin": "400999888", "total_minor": 56000, "vat_minor": 6000, "vat_rate_bps": 1200, "period_id": "2026-11",
	}, admin)
	if recIn.Code != http.StatusCreated {
		t.Fatalf("POST /v1/fiscal/esf/inbound = %d (%s), want 201", recIn.Code, recIn.Body.String())
	}

	// 4. Auditor CAN read GET /v1/fiscal/esf, /v1/fiscal/esf/{id}, /v1/fiscal/vat-summary, /v1/fiscal/missing-ikpu -> 200.
	if rec := doESFReq(t, r, http.MethodGet, "/v1/fiscal/esf", nil, auditor); rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/fiscal/esf = %d, want 200", rec.Code)
	}
	if rec := doESFReq(t, r, http.MethodGet, "/v1/fiscal/esf/"+outDoc.EsfID, nil, auditor); rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/fiscal/esf/{id} = %d, want 200", rec.Code)
	}
	recVAT := doESFReq(t, r, http.MethodGet, "/v1/fiscal/vat-summary?period_id=2026-11", nil, auditor)
	if recVAT.Code != http.StatusOK {
		t.Fatalf("GET /v1/fiscal/vat-summary = %d (%s), want 200", recVAT.Code, recVAT.Body.String())
	}
	var vat esf.VATSummaryReport
	if err := json.Unmarshal(recVAT.Body.Bytes(), &vat); err != nil || !vat.Reconciled || vat.DiffMinor != 0 {
		t.Fatalf("unexpected VATSummary response: %+v err=%v", vat, err)
	}

	recIKPU := doESFReq(t, r, http.MethodGet, "/v1/fiscal/missing-ikpu", nil, auditor)
	if recIKPU.Code != http.StatusOK {
		t.Fatalf("GET /v1/fiscal/missing-ikpu = %d, want 200", recIKPU.Code)
	}
	var ikpu esf.MissingIKPUResponse
	if err := json.Unmarshal(recIKPU.Body.Bytes(), &ikpu); err != nil || ikpu.Count != 1 {
		t.Fatalf("unexpected missing-ikpu: %+v err=%v", ikpu, err)
	}

	// 5. Resend and REJECT transition -> 200, second transition -> 409 Conflict.
	if rec := doESFReq(t, r, http.MethodPost, "/v1/fiscal/esf/"+outDoc.EsfID+"/resend", map[string]any{}, admin); rec.Code != http.StatusOK {
		t.Fatalf("POST /v1/fiscal/esf/{id}/resend = %d (%s), want 200", rec.Code, rec.Body.String())
	}
	if rec := doESFReq(t, r, http.MethodPost, "/v1/fiscal/esf/"+outDoc.EsfID+"/status", map[string]any{"action": "REJECT", "reason": "disputed"}, admin); rec.Code != http.StatusOK {
		t.Fatalf("POST /v1/fiscal/esf/{id}/status = %d (%s), want 200", rec.Code, rec.Body.String())
	}
	if rec := doESFReq(t, r, http.MethodPost, "/v1/fiscal/esf/"+outDoc.EsfID+"/status", map[string]any{"action": "ACCEPT"}, admin); rec.Code != http.StatusConflict {
		t.Fatalf("terminal transition status = %d, want 409 Conflict", rec.Code)
	}
}
