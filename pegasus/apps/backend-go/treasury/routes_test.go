package treasury

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_BatchSettleUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "batch-settle"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/treasury/batch-settle", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "batch-settle" {
		t.Fatalf("idempotency guard header = %q, want batch-settle", got)
	}
}

func TestRegisterRoutes_InvoiceStatusOverrideUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "invoice-status"),
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/treasury/invoice/status", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "invoice-status" {
		t.Fatalf("idempotency guard header = %q, want invoice-status", got)
	}
}

func TestRegisterRoutes_InternalPayoutPolicyOverrideUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "internal-payout-policy"),
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/internal/treasury/supplier-payout-policy", strings.NewReader("{"))
	req.Header.Set("X-Internal-Key", "test-internal-key")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "internal-payout-policy" {
		t.Fatalf("idempotency guard header = %q, want internal-payout-policy", got)
	}
}

func TestRegisterRoutes_InternalPayoutPolicyOverrideRejectsSupplierRole(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "internal-payout-policy"),
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/internal/treasury/supplier-payout-policy", strings.NewReader(`{"supplier_id":"sup-1","payout_mode":"HQ_SUPPLIER","reason":"x"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "internal-payout-policy" {
		t.Fatalf("idempotency guard header = %q, want internal-payout-policy", got)
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
