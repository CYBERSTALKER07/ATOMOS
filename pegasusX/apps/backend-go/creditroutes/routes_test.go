package creditroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

func creditRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("credit-test-secret"))
	RegisterRoutes(r, d)
	return r
}

func issueCreditToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "credit-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "credit-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCreditRoutes_AllNil(t *testing.T) {
	r := creditRouter(Deps{})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/credit-profile", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestCreditRoutes_Unauthenticated(t *testing.T) {
	r := creditRouter(Deps{
		Service:       &credit.Service{},
		PolicyService: &credit.PolicyService{},
	})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/retailer/credit-profile"},
		{http.MethodGet, "/v1/supplier/credit-profiles"},
		{http.MethodGet, "/v1/supplier/credit-scores"},
		{http.MethodGet, "/v1/supplier/credit-program"},
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

func TestCreditRoutes_RoleGuard(t *testing.T) {
	r := creditRouter(Deps{
		Service:       &credit.Service{},
		PolicyService: &credit.PolicyService{},
	})

	retailerToken := issueCreditToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/credit-scores", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}
