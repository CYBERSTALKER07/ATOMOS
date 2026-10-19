package supplierroutes

import (
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
