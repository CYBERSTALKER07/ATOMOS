package taxroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
)

func taxRouter(svc *tax.Service) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc)
	return r
}

func TestTaxRoutes_NilService(t *testing.T) {
	r := taxRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/tax-regimes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestTaxRoutes_MountWithService(t *testing.T) {
	svc := &tax.Service{}
	r := taxRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/tax-regimes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Since Spanner is nil, it will return internal error or execute handler, but definitely not 404
	if rr.Code == http.StatusNotFound {
		t.Fatalf("expected route to be mounted, got 404")
	}
}
