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
