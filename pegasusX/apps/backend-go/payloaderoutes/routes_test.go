package payloaderoutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
)

func payloaderRouter(svc *payload.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("payloader-test-secret"))
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "payloader-test-secret"})
	return r
}

func issuePayloadToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "payload-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "payloader-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestPayloadRoutes_NilService(t *testing.T) {
	r := payloaderRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/payloader/trucks", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestPayloadRoutes_Unauthenticated(t *testing.T) {
	r := payloaderRouter(&payload.Service{})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/payloader/trucks"},
		{http.MethodGet, "/v1/payloader/manifests"},
		{http.MethodPost, "/v1/payloader/manifests/seal-all"},
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

func TestPayloadRoutes_RoleGuard(t *testing.T) {
	r := payloaderRouter(&payload.Service{})

	driverToken := issuePayloadToken(t, auth.RoleDriver)

	// Driver role should not have access to payloader routes
	req := httptest.NewRequest(http.MethodGet, "/v1/payloader/trucks", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver accessing payloader routes returned %d, want 403 Forbidden", rr.Code)
	}
}
