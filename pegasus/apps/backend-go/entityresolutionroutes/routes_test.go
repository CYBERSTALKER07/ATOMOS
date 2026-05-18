package entityresolutionroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_ResolveRejectsInvalidRole(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("driver-user", "DRIVER", "", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader(`{"entity_type":"ANY","query":"ord"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRegisterRoutes_ExplainRejectsInvalidRole(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("driver-user", "DRIVER", "", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/explain", strings.NewReader(`{"entity_type":"ORDER","entity_id":"ord-1"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRegisterRoutes_ResolveRouteMounted(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/entity-resolution/resolve", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRegisterRoutes_ExplainRouteMounted(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/entity-resolution/explain", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRegisterRoutes_ResolveRejectsMalformedJSON(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterRoutes_ExplainRejectsMalformedJSON(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/explain", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterRoutes_ResolveWithRegionIDRequiresSpannerClient(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve?region_id=reg-1", strings.NewReader(`{"entity_type":"ANY","query":"ord"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRegisterRoutes_ExplainWithRegionIDRequiresSpannerClient_AdminRole(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "ADMIN", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/explain?region_id=reg-1", strings.NewReader(`{"entity_type":"ORDER","entity_id":"ord-1"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRegisterRoutes_ResolveMissingTokenReturnsUnauthorized(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader(`{"entity_type":"ANY","query":"ord"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRegisterRoutes_ExplainMissingTokenReturnsUnauthorized(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/explain", strings.NewReader(`{"entity_type":"ORDER","entity_id":"ord-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRegisterRoutes_ResolveOptionsBypassesAuthAndReturnsMethodNotAllowed(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/supplier/entity-resolution/resolve", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRegisterRoutes_ExplainOptionsBypassesAuthAndReturnsMethodNotAllowed(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: passthroughMiddleware,
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/supplier/entity-resolution/explain", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}
