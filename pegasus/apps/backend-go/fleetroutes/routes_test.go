package fleetroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_FleetDispatchUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: passthroughMiddleware,
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/fleet/dispatch", strings.NewReader("{"))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "dispatch-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestRegisterRoutes_FleetReassignUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: passthroughMiddleware,
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/fleet/reassign", strings.NewReader("{"))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "reassign-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}
