package returnsroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
)

func returnsRouter(svc *returns.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("returns-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	RegisterDriverRoutes(r, Deps{Service: svc})
	RegisterSupplierHistory(r, Deps{Service: svc})
	return r
}

func issueReturnsToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "returns-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "returns-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestReturnsRoutes_NilService(t *testing.T) {
	r := returnsRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/returns/inbound", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestReturnsRoutes_Unauthenticated(t *testing.T) {
	svc := &returns.Service{}
	r := returnsRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/returns/inbound"},
		{http.MethodPost, "/v1/returns/inbound/sessions"},
		{http.MethodGet, "/v1/driver/return-goods"},
		{http.MethodGet, "/v1/supplier/returns/history"},
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

func TestReturnsRoutes_RoleGuard(t *testing.T) {
	svc := &returns.Service{}
	r := returnsRouter(svc)

	retailerToken := issueReturnsToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/returns/inbound", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}
