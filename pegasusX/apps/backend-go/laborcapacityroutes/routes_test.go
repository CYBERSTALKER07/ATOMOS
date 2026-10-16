package laborcapacityroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacity"
)

func laborCapacityRouter(svc *laborcapacity.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("labor-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueLaborToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "labor-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "labor-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestLaborCapacityRoutes_NilService(t *testing.T) {
	r := laborCapacityRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/labor-capacity/zone-capacity", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestLaborCapacityRoutes_Unauthenticated(t *testing.T) {
	svc := &laborcapacity.Service{}
	r := laborCapacityRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/labor-capacity/driver-score/drv-1"},
		{http.MethodGet, "/v1/labor-capacity/zone-capacity"},
		{http.MethodPost, "/v1/labor-capacity/driver-availability"},
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

func TestLaborCapacityRoutes_RoleGuard(t *testing.T) {
	svc := &laborcapacity.Service{}
	r := laborCapacityRouter(svc)

	retailerToken := issueLaborToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/labor-capacity/zone-capacity", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}
