package suppliercatalogroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"
	"backend-go/supplier"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_PricingRulesUseIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Pricing:     &supplier.PricingService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "pricing-rules"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/pricing/rules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "pricing-rules" {
		t.Fatalf("idempotency guard header = %q, want pricing-rules", got)
	}
}

func TestRegisterRoutes_PricingRuleActionUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Pricing:     &supplier.PricingService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "pricing-rule-action"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/pricing/rules/tier-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "pricing-rule-action" {
		t.Fatalf("idempotency guard header = %q, want pricing-rule-action", got)
	}
}

func TestRegisterRoutes_RetailerOverridesUseIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		RetailerPricing: &supplier.RetailerPricingService{},
		Log:             passthroughMiddleware,
		Idempotency:     markerMiddleware("X-Idempotency-Guard", "retailer-overrides"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/pricing/retailer-overrides", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "retailer-overrides" {
		t.Fatalf("idempotency guard header = %q, want retailer-overrides", got)
	}
}

func TestRegisterRoutes_RetailerOverrideActionUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		RetailerPricing: &supplier.RetailerPricingService{},
		Log:             passthroughMiddleware,
		Idempotency:     markerMiddleware("X-Idempotency-Guard", "retailer-override-action"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/pricing/retailer-overrides/override-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "retailer-override-action" {
		t.Fatalf("idempotency guard header = %q, want retailer-override-action", got)
	}
}

func TestRegisterRoutes_ProductCreateUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "product-create"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/products", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "product-create" {
		t.Fatalf("idempotency guard header = %q, want product-create", got)
	}
}

func TestRegisterRoutes_ProductDetailUpdateUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "product-detail"),
	})

	req := httptest.NewRequest(http.MethodPut, "/v1/supplier/products/sku-1", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "product-detail" {
		t.Fatalf("idempotency guard header = %q, want product-detail", got)
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
