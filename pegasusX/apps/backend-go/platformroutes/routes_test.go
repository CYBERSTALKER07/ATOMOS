package platformroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

func platformRouter(h *platform.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("platform-test-secret"))
	RegisterRoutes(r, Deps{Handler: h, JWTSecret: "platform-test-secret"})
	return r
}

func issuePlatformToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "platform-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "platform-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestPlatformRoutes_NilHandler(t *testing.T) {
	r := platformRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/platform/client-policy", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestPlatformRoutes_Unauthenticated(t *testing.T) {
	r := platformRouter(&platform.Handler{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/v1/platform/client-policy"},
		{http.MethodGet, "/v1/media/upload-ticket"},
		{http.MethodGet, "/v1/auth/session"},
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

func TestPlatformRoutes_RoleGuard(t *testing.T) {
	r := platformRouter(&platform.Handler{})

	retailerToken := issuePlatformToken(t, auth.RoleRetailer)

	// Retailer cannot PUT client-policy (Admin required)
	req := httptest.NewRequest(http.MethodPut, "/v1/platform/client-policy", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer updating client-policy returned %d, want 403 Forbidden", rr.Code)
	}
}
