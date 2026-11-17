package reconroutes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
)

const testJWTSecret = "test-recon-secret-0123456789abcdef"

func issueTestCookie(t *testing.T, c auth.Claims) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(c, auth.IssueOptions{Secret: testJWTSecret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func doJSONRequest(t *testing.T, r http.Handler, method, target string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("marshal body: %v", err)
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

func TestReconRoutes_Lifecycle_And_FailClosedTenantIsolation(t *testing.T) {
	t.Parallel()
	repo := recon.NewMemoryRepository()
	repo.SeedSnapshot(recon.Snapshot{
		EntityID:             "sup-a",
		PeriodID:             "2026-11",
		Currency:             "UZS",
		OpenARBalanceMinor:   200_000,
		GLARRetailerNetMinor: 150_000,
	})
	svc := recon.NewServiceWithRepository(repo)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: testJWTSecret})

	adminA := issueTestCookie(t, auth.Claims{
		Subject:    "usr-admin-a",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-a",
		MarketCode: "UZ",
	})
	driverA := issueTestCookie(t, auth.Claims{
		Subject:    "usr-drv-a",
		Role:       auth.RoleDriver,
		SupplierID: "sup-a",
	})
	platformAdmin := issueTestCookie(t, auth.Claims{
		Subject: "usr-platform",
		Role:    auth.RolePlatformAdmin,
	})

	// 1. Unauthenticated fails closed (401).
	if res := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs", nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("unauth GET runs = %d, want 401", res.Code)
	}

	// 2. Non-admin role fails closed (403).
	if res := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs", nil, driverA); res.Code != http.StatusForbidden {
		t.Fatalf("driver GET runs = %d, want 403", res.Code)
	}

	// 3. Cross-tenant query/body attempt by supplier admin fails closed (403).
	if res := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs?entity=sup-b", nil, adminA); res.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET runs = %d, want 403", res.Code)
	}
	if res := doJSONRequest(t, r, http.MethodPost, "/v1/finance/recon/run", map[string]any{"entity_id": "sup-b"}, adminA); res.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant POST run = %d, want 403", res.Code)
	}

	// 4. Platform admin without ?entity= fails closed (400 entity_required).
	if res := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs", nil, platformAdmin); res.Code != http.StatusBadRequest {
		t.Fatalf("platform GET without entity = %d, want 400", res.Code)
	}

	// 4b. Malformed period_id fails closed with 400 invalid_period on both POST run and GET runs.
	if res := doJSONRequest(t, r, http.MethodPost, "/v1/finance/recon/run", map[string]any{"period_id": "2026-13"}, adminA); res.Code != http.StatusBadRequest {
		t.Fatalf("invalid POST period_id = %d, want 400", res.Code)
	}
	if res := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs?period_id=not-a-period", nil, adminA); res.Code != http.StatusBadRequest {
		t.Fatalf("invalid GET period_id = %d, want 400", res.Code)
	}

	// 5. Authorized POST /v1/finance/recon/run triggers sweep and returns 1 break.
	postRes := doJSONRequest(t, r, http.MethodPost, "/v1/finance/recon/run", map[string]any{"period_id": "2026-11"}, adminA)
	if postRes.Code != http.StatusOK {
		t.Fatalf("POST run = %d body=%s, want 200", postRes.Code, postRes.Body.String())
	}
	var runResp struct {
		Run    recon.ReconciliationRun     `json:"run"`
		Breaks []recon.ReconciliationBreak `json:"breaks"`
	}
	if err := json.Unmarshal(postRes.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode runResp: %v", err)
	}
	if runResp.Run.BreakCount != 1 || len(runResp.Breaks) != 1 || runResp.Run.Status != recon.RunStatusBreaksFound {
		t.Fatalf("unexpected runResp: %+v", runResp)
	}

	// 6. GET /v1/finance/recon/runs & GET /v1/finance/recon/breaks as platform admin with ?entity=sup-a.
	runsRes := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs?entity=sup-a", nil, platformAdmin)
	if runsRes.Code != http.StatusOK {
		t.Fatalf("platform GET runs = %d body=%s", runsRes.Code, runsRes.Body.String())
	}
	breaksRes := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/breaks?status=OPEN", nil, adminA)
	if breaksRes.Code != http.StatusOK {
		t.Fatalf("GET breaks = %d body=%s", breaksRes.Code, breaksRes.Body.String())
	}
	var breaksPayload struct {
		Count  int                         `json:"count"`
		Breaks []recon.ReconciliationBreak `json:"breaks"`
	}
	if err := json.Unmarshal(breaksRes.Body.Bytes(), &breaksPayload); err != nil || breaksPayload.Count != 1 {
		t.Fatalf("decode breaks: err=%v payload=%+v", err, breaksPayload)
	}

	// 7. PATCH /v1/finance/recon/breaks/{id} RESOLVE transitions status cleanly.
	bid := breaksPayload.Breaks[0].BreakID
	patchRes := doJSONRequest(t, r, http.MethodPatch, "/v1/finance/recon/breaks/"+bid, map[string]any{
		"action": "RESOLVE",
		"note":   "matched via manual journal",
	}, adminA)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("PATCH break = %d body=%s, want 200", patchRes.Code, patchRes.Body.String())
	}
	var patchPayload struct {
		Break recon.ReconciliationBreak `json:"break"`
	}
	if err := json.Unmarshal(patchRes.Body.Bytes(), &patchPayload); err != nil || patchPayload.Break.Status != recon.BreakStatusResolved {
		t.Fatalf("patchPayload = (%+v, %v), want RESOLVED", patchPayload, err)
	}

	// 8. AUDITOR can read runs/breaks (200) but cannot mutate /run or /breaks/{id} (403); AGENT cannot mutate (403).
	audCookie := issueTestCookie(t, auth.Claims{Subject: "aud-recon", Role: auth.RoleAdmin, SupplierID: "sup-a", SupplierRole: auth.SupplierRoleAuditor})
	agentCookie := issueTestCookie(t, auth.Claims{Subject: "agent-recon", Role: auth.RoleAdmin, SupplierID: "sup-a", ActorType: auth.ActorTypeAgent, SupplierRole: auth.SupplierRoleAgent})
	if res := doJSONRequest(t, r, http.MethodGet, "/v1/finance/recon/runs", nil, audCookie); res.Code != http.StatusOK {
		t.Fatalf("auditor GET runs = %d, want 200", res.Code)
	}
	if res := doJSONRequest(t, r, http.MethodPost, "/v1/finance/recon/run", map[string]any{"period_id": "2026-11"}, audCookie); res.Code != http.StatusForbidden {
		t.Fatalf("auditor POST run = %d, want 403", res.Code)
	}
	if res := doJSONRequest(t, r, http.MethodPatch, "/v1/finance/recon/breaks/"+bid, map[string]any{"action": "RESOLVE"}, audCookie); res.Code != http.StatusForbidden {
		t.Fatalf("auditor PATCH break = %d, want 403", res.Code)
	}
	if res := doJSONRequest(t, r, http.MethodPost, "/v1/finance/recon/run", map[string]any{"period_id": "2026-11"}, agentCookie); res.Code != http.StatusForbidden {
		t.Fatalf("agent POST run = %d, want 403", res.Code)
	}
}
