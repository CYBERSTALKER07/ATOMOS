package entityresolutionroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func entityResolutionRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("er-test-secret"))
	RegisterRoutes(r, d)
	return r
}

func TestEntityResolutionRoutes_NilSpanner(t *testing.T) {
	r := entityResolutionRouter(Deps{Spanner: nil})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes when Spanner is nil, got %d", rr.Code)
	}
}
