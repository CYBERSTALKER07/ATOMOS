package retailerroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

func retailerRouter(svc *retailer.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("retailer-test-secret"))
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "retailer-test-secret"})
	return r
}

func issueRetailerToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "retailer-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "retailer-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestRetailerRoutes_NilService(t *testing.T) {
	r := retailerRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/me", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestRetailerRoutes_Unauthenticated(t *testing.T) {
	r := retailerRouter(&retailer.Service{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/retailer/me"},
		{http.MethodGet, "/v1/retailer/capabilities"},
		{http.MethodPost, "/v1/retailer/capabilities/pack-1/enable"},
		{http.MethodGet, "/v1/retailer/cards"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestRetailerRoutes_RoleGuard(t *testing.T) {
	r := retailerRouter(&retailer.Service{})

	driverToken := issueRetailerToken(t, auth.RoleDriver)

	// Driver cannot access retailer routes
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/me", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver accessing retailer routes returned %d, want 403 Forbidden", rr.Code)
	}
}

func TestReviewKyc_Unauthenticated(t *testing.T) {
	r := retailerRouter(&retailer.Service{})

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-123/review", strings.NewReader(`{"status":"APPROVED"}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated review KYC, got %d", rr.Code)
	}
}

func TestReviewKyc_ForbiddenForRetailerAndDriver(t *testing.T) {
	r := retailerRouter(&retailer.Service{})

	for _, role := range []auth.Role{auth.RoleRetailer, auth.RoleDriver} {
		tok := issueRetailerToken(t, role)
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-123/review", strings.NewReader(`{"status":"APPROVED"}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for role %s attempting KYC review, got %d", role, rr.Code)
		}
	}
}

func TestReviewKyc_AllowedForAdminRoles(t *testing.T) {
	r := retailerRouter(&retailer.Service{})

	for _, role := range []auth.Role{auth.RoleAdmin, auth.RolePlatformAdmin} {
		tok := issueRetailerToken(t, role)
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-123/review", strings.NewReader(`{"status":"APPROVED"}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// Must pass the auth & role guard (not 401 or 403).
		if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
			t.Fatalf("admin role %s blocked by auth guard: status=%d", role, rr.Code)
		}
		// Empty service returns 500 repo_not_supported from handler
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected handler 500 (repo_not_supported), got %d", rr.Code)
		}
	}
}
