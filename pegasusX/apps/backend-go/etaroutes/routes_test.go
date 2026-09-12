package etaroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/eta"
)

func etaRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("eta-test-secret"))
	RegisterRoutes(r, Deps{})
	return r
}

func issueETAToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject: "eta-user", Role: role, SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "eta-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestETARoutesUnauthenticated(t *testing.T) {
	r := etaRouter()
	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodGet, "/v1/etas/route/r1"},
		{http.MethodGet, "/v1/etas/stop/s1"},
		{http.MethodPost, "/v1/etas/recalculate"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status=%d body=%s want 401", tc.method, tc.path, rr.Code, rr.Body.String())
		}
	}
}

func TestETARecalculateRejectsRetailer(t *testing.T) {
	r := etaRouter()
	req := httptest.NewRequest(http.MethodPost, "/v1/etas/recalculate", strings.NewReader(`{"route_id":"r1"}`))
	req.Header.Set("Authorization", "Bearer "+issueETAToken(t, auth.RoleRetailer))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", rr.Code, rr.Body.String())
	}
}

func TestETAGetAllowsWarehouse(t *testing.T) {
	r := etaRouter()
	req := httptest.NewRequest(http.MethodGet, "/v1/etas/route/r1", nil)
	req.Header.Set("Authorization", "Bearer "+issueETAToken(t, auth.RoleWarehouse))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Fatalf("warehouse GET must pass role gate, status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestETAGetUnknownRouteNotFound(t *testing.T) {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("eta-test-secret"))
	RegisterRoutes(r, Deps{Service: eta.NewService(nil)})
	req := httptest.NewRequest(http.MethodGet, "/v1/etas/route/r1", nil)
	req.Header.Set("Authorization", "Bearer "+issueETAToken(t, auth.RoleAdmin))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s want 404", rr.Code, rr.Body.String())
	}
}

func TestETARecalculateRequiresRouteID(t *testing.T) {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("eta-test-secret"))
	RegisterRoutes(r, Deps{Service: eta.NewService(nil)})
	req := httptest.NewRequest(http.MethodPost, "/v1/etas/recalculate", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+issueETAToken(t, auth.RoleWarehouseAdmin))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s want 400", rr.Code, rr.Body.String())
	}
}

func TestETARecalculateAllowsWarehouseAdmin(t *testing.T) {
	r := etaRouter()
	req := httptest.NewRequest(http.MethodPost, "/v1/etas/recalculate", strings.NewReader(`{"route_id":"r1"}`))
	req.Header.Set("Authorization", "Bearer "+issueETAToken(t, auth.RoleWarehouseAdmin))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Fatalf("warehouse admin POST must pass role gate, status=%d body=%s", rr.Code, rr.Body.String())
	}
}
