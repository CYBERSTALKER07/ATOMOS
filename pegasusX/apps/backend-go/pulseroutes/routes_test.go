package pulseroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulse"
)

func pulseRouter(h *pulse.Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("pulse-test-secret"))
	RegisterRoutes(r, Deps{Handlers: h})
	return r
}

func issuePulseToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "pulse-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "pulse-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestPulseRoutes_NilHandlers(t *testing.T) {
	r := pulseRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/pulse", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestPulseRoutes_Unauthenticated(t *testing.T) {
	r := pulseRouter(&pulse.Handlers{})

	endpoints := []string{
		"/v1/retailer/pulse",
		"/v1/supplier/pulse",
		"/v1/warehouse/ops/pulse",
		"/v1/driver/pulse",
		"/v1/payloader/pulse",
		"/v1/factory/pulse",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s without auth returned %d, want 401", ep, rr.Code)
		}
	}
}

func TestPulseRoutes_RoleGuard(t *testing.T) {
	r := pulseRouter(&pulse.Handlers{})

	driverToken := issuePulseToken(t, auth.RoleDriver)

	// Driver should not be allowed to access supplier pulse
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/pulse", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver accessing supplier pulse returned %d, want 403 Forbidden", rr.Code)
	}
}
