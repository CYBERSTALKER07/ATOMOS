package fleetroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend-go/auth"
	"backend-go/cache"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func TestRegisterRoutes_FleetDispatchUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newFleetTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:dispatch-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/fleet/dispatch", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "dispatch-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterRoutes_FleetReassignUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newFleetTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:reassign-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/fleet/reassign", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "reassign-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterRoutes_WildcardRoutesMounted(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	router := newFleetTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "drivers status wildcard", method: http.MethodPut, path: "/v1/fleet/drivers/driver-1/status"},
		{name: "trucks dispatcher wildcard", method: http.MethodPost, path: "/v1/fleet/trucks/truck-1/seal"},
		{name: "route complete wildcard", method: http.MethodPost, path: "/v1/fleet/route/route-1/complete"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code == http.StatusNotFound {
				t.Fatalf("status = %d, want non-404 for %s", response.Code, tc.path)
			}
		})
	}
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}

func newFleetTestRouter(t *testing.T) *chi.Mux {
	t.Helper()

	origMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = origMux })

	origClient := cache.Client
	mr := setupMiniredis(t)
	t.Cleanup(func() {
		cache.Client = origClient
		mr.Close()
	})

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: passthroughMiddleware,
	})
	return router
}

func setupMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr
}
