package supplieroperationsroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_ReturnResolveUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "returns-resolve"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/returns/resolve", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "returns-resolve" {
		t.Fatalf("idempotency guard header = %q, want returns-resolve", got)
	}
}

func TestRegisterRoutes_FleetDriverCreateUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "fleet-driver-create"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/fleet/drivers", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "fleet-driver-create" {
		t.Fatalf("idempotency guard header = %q, want fleet-driver-create", got)
	}
}

func TestRegisterRoutes_FleetDriverAssignUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "fleet-driver-assign"),
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/fleet/drivers/", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "fleet-driver-assign" {
		t.Fatalf("idempotency guard header = %q, want fleet-driver-assign", got)
	}
}

func TestRegisterRoutes_FleetVehicleCreateUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "fleet-vehicle-create"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/fleet/vehicles", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "fleet-vehicle-create" {
		t.Fatalf("idempotency guard header = %q, want fleet-vehicle-create", got)
	}
}

func TestRegisterRoutes_ReconcileReturnsUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "reconcile-returns"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/inventory/reconcile-returns", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "reconcile-returns" {
		t.Fatalf("idempotency guard header = %q, want reconcile-returns", got)
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
