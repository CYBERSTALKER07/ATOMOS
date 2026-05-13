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

func TestRegisterRoutes_InventoryAdjustUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "inventory-adjust"),
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/inventory", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "inventory-adjust" {
		t.Fatalf("idempotency guard header = %q, want inventory-adjust", got)
	}
}

func TestRegisterRoutes_InventoryImportCreateUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "inventory-import-create"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "inventory-import-create" {
		t.Fatalf("idempotency guard header = %q, want inventory-import-create", got)
	}
}

func TestRegisterRoutes_InventoryImportActionUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "inventory-import-apply"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import/session-1/apply", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "inventory-import-apply" {
		t.Fatalf("idempotency guard header = %q, want inventory-import-apply", got)
	}
}

func TestRegisterRoutes_InventoryImportsSandboxCreateUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "inventory-imports-sandbox-create"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "inventory-imports-sandbox-create" {
		t.Fatalf("idempotency guard header = %q, want inventory-imports-sandbox-create", got)
	}
}

func TestRegisterRoutes_InventoryImportsSandboxApproveUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "inventory-imports-sandbox-approve"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/session-1/approve", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "inventory-imports-sandbox-approve" {
		t.Fatalf("idempotency guard header = %q, want inventory-imports-sandbox-approve", got)
	}
}

func TestRegisterRoutes_InventoryImportsSandboxUploadedUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "inventory-imports-sandbox-uploaded"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/session-1/uploaded", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "inventory-imports-sandbox-uploaded" {
		t.Fatalf("idempotency guard header = %q, want inventory-imports-sandbox-uploaded", got)
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
