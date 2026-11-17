package demandroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
)

func demandRouter(svc *demand.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("demand-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueDemandToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "demand-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "demand-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestDemandRoutes_NilService(t *testing.T) {
	r := demandRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/demand/signals", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestDemandRoutes_Unauthenticated(t *testing.T) {
	svc := &demand.Service{}
	r := demandRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/demand/signals"},
		{http.MethodPost, "/v1/demand/signals"},
		{http.MethodGet, "/v1/demand/adjustments"},
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

func TestDemandRoutes_RoleGuard(t *testing.T) {
	svc := &demand.Service{}
	r := demandRouter(svc)

	retailerToken := issueDemandToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/demand/signals", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}
