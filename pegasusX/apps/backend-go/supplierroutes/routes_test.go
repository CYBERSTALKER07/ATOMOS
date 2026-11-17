package supplierroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

func supplierRouter(svc *supplier.Service) http.Handler {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "supplier-test-secret"})
	return r
}

func issueSupplierCookie(t *testing.T, role auth.Role) *http.Cookie {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "supplier-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "supplier-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: tok}
}

func TestSupplierRoutes_NilService(t *testing.T) {
	r := supplierRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/profile", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestSupplierRoutes_Unauthenticated(t *testing.T) {
	r := supplierRouter(&supplier.Service{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/supplier/profile"},
		{http.MethodGet, "/v1/supplier/topology"},
		{http.MethodGet, "/v1/supplier/payment-catalog"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without cookie returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestSupplierRoutes_RoleGuard(t *testing.T) {
	r := supplierRouter(&supplier.Service{})

	cookie := issueSupplierCookie(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/profile", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer cookie accessing supplier profile returned %d, want 403 Forbidden", rr.Code)
	}
}

func TestSupplierRoutes_W4EndpointsMountedAndGuarded(t *testing.T) {
	r := supplierRouter(&supplier.Service{})
	retailerCookie := issueSupplierCookie(t, auth.RoleRetailer)
	adminCookie := issueSupplierCookie(t, auth.RoleAdmin)

	guarded := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/supplier/retailer-links", ""},
		{http.MethodPost, "/v1/supplier/retailer-links", `{"retailer_id":"ret-1"}`},
		{http.MethodGet, "/v1/supplier/retailer-links/dedup", ""},
		{http.MethodPost, "/v1/supplier/retailer-links/ret-1/resolve", `{"action":"mark_clean"}`},
		{http.MethodPost, "/v1/supplier/planning/seasonal-holidays/seed", `{}`},
	}

	for _, ep := range guarded {
		unauthReq := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		unauthRR := httptest.NewRecorder()
		r.ServeHTTP(unauthRR, unauthReq)
		if unauthRR.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s unauth status=%d, want 401", ep.method, ep.path, unauthRR.Code)
		}

		forbiddenReq := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
		forbiddenReq.AddCookie(retailerCookie)
		forbiddenRR := httptest.NewRecorder()
		r.ServeHTTP(forbiddenRR, forbiddenReq)
		if forbiddenRR.Code != http.StatusForbidden {
			t.Fatalf("%s %s retailer role status=%d, want 403", ep.method, ep.path, forbiddenRR.Code)
		}
	}

	// Admin cookie reaches the retailer-links handler cleanly (not 404/401/403).
	adminReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/retailer-links", strings.NewReader(`{"retailer_id":"ret-1","phone":"90 123 45 67","tin":"301-234-567"}`))
	adminReq.AddCookie(adminCookie)
	adminRR := httptest.NewRecorder()
	r.ServeHTTP(adminRR, adminReq)
	if adminRR.Code != http.StatusOK {
		t.Fatalf("POST /v1/supplier/retailer-links admin status=%d body=%s", adminRR.Code, adminRR.Body.String())
	}

	// Seed holidays route reaches handler cleanly (503 when Spanner nil, proving route dispatch).
	seedReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/planning/seasonal-holidays/seed", strings.NewReader(`{}`))
	seedReq.AddCookie(adminCookie)
	seedRR := httptest.NewRecorder()
	r.ServeHTTP(seedRR, seedReq)
	if seedRR.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /v1/supplier/planning/seasonal-holidays/seed status=%d body=%s", seedRR.Code, seedRR.Body.String())
	}
}

func TestSupplierRoutes_EventsQueryTokenAuth(t *testing.T) {
	srv := httptest.NewServer(supplierRouter(&supplier.Service{}))
	defer srv.Close()

	// Unauthenticated -> 401
	resp, err := http.Get(srv.URL + "/v1/supplier/events")
	if err != nil {
		t.Fatalf("GET /v1/supplier/events: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d, want 401", resp.StatusCode)
	}

	// Raw session JWT in ?token= query string must be rejected (short-lived WS tickets only) -> 401
	rawAdminTok := issueSupplierCookie(t, auth.RoleAdmin).Value
	resp, err = http.Get(srv.URL + "/v1/supplier/events?token=" + rawAdminTok)
	if err != nil {
		t.Fatalf("GET /v1/supplier/events raw session query token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("raw session query token status=%d, want 401", resp.StatusCode)
	}

	// Retailer role WS ticket -> 403 even if SupplierID claim is populated
	retailerTicket, _, err := auth.IssueWSTicket(auth.Claims{
		Subject:    "retailer-user",
		Role:       auth.RoleRetailer,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "supplier-test-secret", TTL: 5 * time.Minute})
	if err != nil {
		t.Fatalf("issue retailer ticket: %v", err)
	}
	resp, err = http.Get(srv.URL + "/v1/supplier/events?token=" + retailerTicket)
	if err != nil {
		t.Fatalf("GET /v1/supplier/events retailer: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("retailer query token status=%d, want 403", resp.StatusCode)
	}

	// Valid supplier admin WS ticket -> 200 SSE stream
	adminTicket, _, err := auth.IssueWSTicket(auth.Claims{
		Subject:    "supplier-user",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "supplier-test-secret", TTL: 5 * time.Minute})
	if err != nil {
		t.Fatalf("issue admin ticket: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/supplier/events?token="+adminTicket, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/supplier/events admin: %v", err)
	}
	streamResp.Body.Close()
	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("admin query token status=%d, want 200", streamResp.StatusCode)
	}
	if ct := streamResp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type=%q, want text/event-stream", ct)
	}
}
