package globalproductsroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproducts"
)

func globalProductsRouter(svc *globalproducts.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("gp-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueGPToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "gp-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "gp-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestGlobalProductsRoutes_NilService(t *testing.T) {
	r := globalProductsRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/global-products/gp-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestGlobalProductsRoutes_Unauthenticated(t *testing.T) {
	svc := &globalproducts.Service{}
	r := globalProductsRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/supplier/products/p1/link-global"},
		{http.MethodGet, "/v1/admin/product-match-queue"},
		{http.MethodPost, "/v1/admin/product-match-queue/m1/resolve"},
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

func TestGlobalProductsRoutes_RoleGuard(t *testing.T) {
	svc := &globalproducts.Service{}
	r := globalProductsRouter(svc)

	retailerToken := issueGPToken(t, auth.RoleRetailer)

	// Retailer cannot link products or access match queue
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/products/p1/link-global", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer linking product returned %d, want 403 Forbidden", rr.Code)
	}
}
