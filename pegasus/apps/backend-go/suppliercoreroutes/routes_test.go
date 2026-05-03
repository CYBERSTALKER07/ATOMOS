package suppliercoreroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"
	"backend-go/supplier"
	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_OrderVettingUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Vetting:     &supplier.OrderVettingService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "supplier-vet"),
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/supplier/orders/vet", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "supplier-vet" {
		t.Fatalf("idempotency guard header = %q, want supplier-vet", got)
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
