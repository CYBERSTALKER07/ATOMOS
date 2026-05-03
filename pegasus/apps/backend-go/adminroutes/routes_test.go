package adminroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/order"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func TestRegisterRoutes_ShopClosedResolveUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newAdminTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:shop-closed-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/shop-closed/resolve", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "shop-closed-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterRoutes_ApproveCancelUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newAdminTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:approve-cancel-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/approve-cancel", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "approve-cancel-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterRoutes_ResolveCreditUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newAdminTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:resolve-credit-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/resolve-credit", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "resolve-credit-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterRoutes_ApproveEarlyCompleteUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newAdminTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:approve-early-complete-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/route/approve-early-complete", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "approve-early-complete-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterRoutes_NegotiationResolveUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := newAdminTestRouter(t)
	cache.Client.SetNX(context.Background(), "idem:negotiation-resolve-test:lock", "1", 30*time.Second)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/negotiate/resolve", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", "negotiation-resolve-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func newAdminTestRouter(t *testing.T) *chi.Mux {
	t.Helper()

	origMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = origMux })

	origClient := cache.Client
	mr := setupAdminMiniredis(t)
	t.Cleanup(func() {
		cache.Client = origClient
		mr.Close()
	})

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Order: &order.OrderService{},
		Log:   passthroughMiddleware,
	})
	return router
}

func setupAdminMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}
