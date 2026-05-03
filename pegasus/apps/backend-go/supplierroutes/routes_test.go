package supplierroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"
	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_BillingSetupUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "billing-setup"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/billing/setup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "billing-setup" {
		t.Fatalf("idempotency guard header = %q, want billing-setup", got)
	}
}

func TestRegisterRoutes_PaymentConfigUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "payment-config"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/payment-config", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "payment-config" {
		t.Fatalf("idempotency guard header = %q, want payment-config", got)
	}
}

func TestRegisterRoutes_OrgInviteUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "org-invite"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/org/members/invite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "org-invite" {
		t.Fatalf("idempotency guard header = %q, want org-invite", got)
	}
}

func supplierTestToken(t *testing.T) string {
	t.Helper()

	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}
	return token
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
