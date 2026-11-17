package cashreconroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
)

func cashReconRouter(h *cashrecon.Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("cashrecon-test-secret"))
	RegisterRoutes(r, Deps{Handlers: h})
	return r
}

func issueCashReconToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "cashrecon-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "cashrecon-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCashReconRoutes_NilHandlers(t *testing.T) {
	r := cashReconRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/cash-reconciliations", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestCashReconRoutes_Unauthenticated(t *testing.T) {
	r := cashReconRouter(&cashrecon.Handlers{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/driver/cash-reconciliations"},
		{http.MethodPost, "/v1/driver/cash-reconciliations"},
		{http.MethodGet, "/v1/supplier/cash-reconciliations"},
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

func TestCashReconRoutes_RoleGuard(t *testing.T) {
	r := cashReconRouter(&cashrecon.Handlers{})

	driverToken := issueCashReconToken(t, auth.RoleDriver)

	// Driver should not be allowed to access supplier cash reconciliations
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/cash-reconciliations", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver accessing supplier cash-reconciliations returned %d, want 403 Forbidden", rr.Code)
	}
}
