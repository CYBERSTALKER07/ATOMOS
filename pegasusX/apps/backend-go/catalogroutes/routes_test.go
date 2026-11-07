package catalogroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
)

func catalogRouter(svc *catalog.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("catalog-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueCatalogToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "catalog-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "catalog-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCatalogRoutes_NilService(t *testing.T) {
	r := catalogRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/catalog/categories", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestCatalogRoutes_MutationUnauthenticated(t *testing.T) {
	svc := &catalog.Service{}
	r := catalogRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/catalog/categories"},
		{http.MethodPost, "/v1/catalog/products"},
		{http.MethodPut, "/v1/catalog/products/prod-1"},
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

func TestCatalogRoutes_MutationRoleGuard(t *testing.T) {
	svc := &catalog.Service{}
	r := catalogRouter(svc)

	retailerToken := issueCatalogToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodPost, "/v1/catalog/products", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}

func TestCatalogRoutes_HandlerPayloadValidation(t *testing.T) {
	svc := &catalog.Service{}
	r := catalogRouter(svc)
	adminToken := issueCatalogToken(t, auth.RoleAdmin)

	// Invalid JSON body on CreateProduct should fail with 400 Bad Request inside the handler
	req := httptest.NewRequest(http.MethodPost, "/v1/catalog/products", strings.NewReader(`{invalid-json`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/catalog/products with malformed JSON returned %d, want 400", rr.Code)
	}

	// Invalid JSON body on CreateCategory should fail with 400 Bad Request inside the handler
	req2 := httptest.NewRequest(http.MethodPost, "/v1/catalog/categories", strings.NewReader(`{invalid-json`))
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/catalog/categories with malformed JSON returned %d, want 400", rr2.Code)
	}
}
