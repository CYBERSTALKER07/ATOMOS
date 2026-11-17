package controltowerroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
)

func controlTowerRouter(h *controltower.Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("controltower-test-secret"))
	RegisterRoutes(r, Deps{Handlers: h})
	return r
}

func issueControlTowerToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "ct-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "controltower-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestControlTowerRoutes_NilHandlers(t *testing.T) {
	r := controlTowerRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/exceptions/scored", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestControlTowerRoutes_Unauthenticated(t *testing.T) {
	r := controlTowerRouter(&controltower.Handlers{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/control-tower/exceptions/scored"},
		{http.MethodGet, "/v1/control-tower/playbooks"},
		{http.MethodPost, "/v1/control-tower/playbooks"},
		{http.MethodPost, "/v1/control-tower/evaluate"},
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

func TestControlTowerRoutes_RoleGuard(t *testing.T) {
	r := controlTowerRouter(&controltower.Handlers{})

	driverToken := issueControlTowerToken(t, auth.RoleDriver)

	req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/playbooks", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver accessing control tower returned %d, want 403 Forbidden", rr.Code)
	}
}

func TestControlTowerRoutes_ObservabilityEndpoints(t *testing.T) {
	r := controlTowerRouter(&controltower.Handlers{})

	endpoints := []string{
		"/v1/control-tower/cells",
		"/v1/control-tower/metrics/spanner",
		"/v1/control-tower/metrics/kafka",
		"/v1/control-tower/metrics/redis",
		"/v1/control-tower/waterfall",
		"/v1/control-tower/costs/attributed",
		"/v1/control-tower/scorecards",
	}

	adminToken := issueControlTowerToken(t, auth.RoleAdmin)
	platformAdminToken := issueControlTowerToken(t, auth.RolePlatformAdmin)
	driverToken := issueControlTowerToken(t, auth.RoleDriver)

	for _, ep := range endpoints {
		// 1. RoleAdmin should succeed (200 OK)
		reqAdmin := httptest.NewRequest(http.MethodGet, ep, nil)
		reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
		recAdmin := httptest.NewRecorder()
		r.ServeHTTP(recAdmin, reqAdmin)
		if recAdmin.Code != http.StatusOK {
			t.Fatalf("endpoint %s with RoleAdmin: expected 200, got %d (body: %s)", ep, recAdmin.Code, recAdmin.Body.String())
		}

		// 2. RolePlatformAdmin should also succeed (200 OK)
		reqPlat := httptest.NewRequest(http.MethodGet, ep, nil)
		reqPlat.Header.Set("Authorization", "Bearer "+platformAdminToken)
		recPlat := httptest.NewRecorder()
		r.ServeHTTP(recPlat, reqPlat)
		if recPlat.Code != http.StatusOK {
			t.Fatalf("endpoint %s with RolePlatformAdmin: expected 200, got %d (body: %s)", ep, recPlat.Code, recPlat.Body.String())
		}

		// 3. RoleDriver should be forbidden (403 Forbidden)
		reqDriver := httptest.NewRequest(http.MethodGet, ep, nil)
		reqDriver.Header.Set("Authorization", "Bearer "+driverToken)
		recDriver := httptest.NewRecorder()
		r.ServeHTTP(recDriver, reqDriver)
		if recDriver.Code != http.StatusForbidden {
			t.Fatalf("endpoint %s with RoleDriver: expected 403, got %d", ep, recDriver.Code)
		}

		// 4. Unauthenticated should return 401 Unauthorized
		reqNoAuth := httptest.NewRequest(http.MethodGet, ep, nil)
		recNoAuth := httptest.NewRecorder()
		r.ServeHTTP(recNoAuth, reqNoAuth)
		if recNoAuth.Code != http.StatusUnauthorized {
			t.Fatalf("endpoint %s without auth: expected 401, got %d", ep, recNoAuth.Code)
		}
	}
}
