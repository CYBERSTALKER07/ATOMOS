package supplierplanningroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"
	"backend-go/factory"
	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_SupplyLanesUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		SupplyLanes: &factory.SupplyLanesService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "supply-lanes"),
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/supplier/supply-lanes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "supply-lanes" {
		t.Fatalf("idempotency guard header = %q, want supply-lanes", got)
	}
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}

func markerMiddleware(name, value string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(name, value)
			next(w, r)
		}
	}
}
