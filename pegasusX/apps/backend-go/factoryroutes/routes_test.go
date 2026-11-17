package factoryroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
)

func factoryRouter(svc *factory.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("factory-test-secret"))
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "factory-test-secret"})
	return r
}

func issueFactoryToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:      "factory-user",
		Role:         role,
		SupplierID:   "sup-1",
		HomeNodeType: "FACTORY",
		HomeNodeID:   "fac-1",
	}, auth.IssueOptions{Secret: "factory-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestFactoryRoutes_NilService(t *testing.T) {
	r := factoryRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/factories", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestFactoryRoutes_Unauthenticated(t *testing.T) {
	svc := &factory.Service{}
	r := factoryRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/factory/manifests"},
		{http.MethodGet, "/v1/factories"},
		{http.MethodPost, "/v1/factory/dispatch"},
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

func TestFactoryRoutes_RoleGuard(t *testing.T) {
	svc := &factory.Service{}
	r := factoryRouter(svc)

	retailerToken := issueFactoryToken(t, auth.RoleRetailer)

	// Retailer cannot access factory manifests
	req := httptest.NewRequest(http.MethodGet, "/v1/factory/manifests", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer accessing factory manifests returned %d, want 403 Forbidden", rr.Code)
	}
}

func TestFactorySetup_UnauthorizedWithoutToken(t *testing.T) {
	svc := &factory.Service{}
	r := factoryRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/setup", strings.NewReader(`{"address":"Factory 1","lat":41.3,"lng":69.2}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated factory setup, got %d", rr.Code)
	}
}

func TestFactorySetup_ForbiddenForRetailerAndDriver(t *testing.T) {
	svc := &factory.Service{}
	r := factoryRouter(svc)

	for _, role := range []auth.Role{auth.RoleRetailer, auth.RoleDriver} {
		tok := issueFactoryToken(t, role)
		req := httptest.NewRequest(http.MethodPost, "/v1/factory/setup", strings.NewReader(`{"address":"Factory 1","lat":41.3,"lng":69.2}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for role %s attempting factory setup, got %d", role, rr.Code)
		}
	}
}

func TestFactorySetup_AllowedForAuthorizedRoles(t *testing.T) {
	svc := &factory.Service{}
	r := factoryRouter(svc)

	for _, role := range []auth.Role{auth.RoleFactory, auth.RoleFactoryAdmin, auth.RoleAdmin} {
		tok := issueFactoryToken(t, role)
		req := httptest.NewRequest(http.MethodPost, "/v1/factory/setup", strings.NewReader(`{"address":"Factory 1","lat":41.3,"lng":69.2}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// With an empty service (no Spanner), it should reach the handler and return 503 (Spanner not configured),
		// proving the role guard allowed the request through (not 401 Unauthorized or 403 Forbidden).
		if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
			t.Fatalf("role %s blocked by auth guard with status %d", role, rr.Code)
		}
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected handler 503 (Spanner not configured), got %d", rr.Code)
		}
	}
}
