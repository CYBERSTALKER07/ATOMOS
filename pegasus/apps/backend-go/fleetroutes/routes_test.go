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
