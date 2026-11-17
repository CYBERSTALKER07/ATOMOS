package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestReliabilityActorKey_TenantShared(t *testing.T) {
	t.Setenv("TENANT_RATE_LIMIT_SHARED", "true")
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/profile", nil)
	ctx := auth.WithClaims(req.Context(), auth.Claims{Subject: "user-a", Role: auth.RoleAdmin, SupplierID: "sup-1"})
	ctx = auth.WithTenant(ctx, auth.TenantContext{SupplierID: "sup-1", Source: "jwt"})
	req = req.WithContext(ctx)
	got := reliabilityActorKey(req)
	if got != "tenant:sup-1" {
		t.Fatalf("got=%q want tenant:sup-1", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/supplier/profile", nil)
	ctx2 := auth.WithClaims(req2.Context(), auth.Claims{Subject: "user-b", Role: auth.RoleAdmin, SupplierID: "sup-1"})
	ctx2 = auth.WithTenant(ctx2, auth.TenantContext{SupplierID: "sup-1", Source: "jwt"})
	req2 = req2.WithContext(ctx2)
	if reliabilityActorKey(req2) != got {
		t.Fatalf("shared tenant key mismatch")
	}
}

func TestReliabilityMiddleware_RateLimitExceeded(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	cfg := DefaultReliabilityConfig()
	cfg.RateLimitWindow = time.Minute
	cfg.RateLimitDefaultMax = 1
	cfg.PriorityMaxInFlight = 8
	cfg.CircuitFailureThreshold = 100

	middleware := newReliabilityMiddlewareWithClock(cfg, func() time.Time { return current })
	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	first := httptest.NewRecorder()
	h.ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusOK)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, req)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatalf("Retry-After header not set")
	}
}

func TestReliabilityMiddleware_HealthExemptFromRateLimit(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	cfg := DefaultReliabilityConfig()
	cfg.RateLimitWindow = time.Minute
	cfg.RateLimitDefaultMax = 1
	cfg.PriorityMaxInFlight = 8
	cfg.CircuitFailureThreshold = 100

	middleware := newReliabilityMiddlewareWithClock(cfg, func() time.Time { return current })
	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	for i := 0; i < 5; i++ {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("health request %d status = %d, want %d", i+1, res.Code, http.StatusOK)
		}
	}
}

func TestReliabilityMiddleware_ShedsOperationalWhenCapacityIsFull(t *testing.T) {
	cfg := DefaultReliabilityConfig()
	cfg.PriorityMaxInFlight = 1
	cfg.RateLimitDefaultMax = 100
	cfg.CircuitFailureThreshold = 100

	middleware := newReliabilityMiddlewareWithClock(cfg, time.Now)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	go func() {
		defer close(done)
		firstReq := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
		firstReq.RemoteAddr = "127.0.0.1:2001"
		firstRes := httptest.NewRecorder()
		h.ServeHTTP(firstRes, firstReq)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not enter handler")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	secondReq.RemoteAddr = "127.0.0.1:2002"
	secondRes := httptest.NewRecorder()
	h.ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusServiceUnavailable {
		t.Fatalf("second request status = %d, want %d", secondRes.Code, http.StatusServiceUnavailable)
	}

	close(release)
	<-done
}

func TestReliabilityMiddleware_CriticalPathWaitsForPermit(t *testing.T) {
	cfg := DefaultReliabilityConfig()
	cfg.PriorityMaxInFlight = 1
	cfg.CriticalAcquireTimeout = 250 * time.Millisecond
	cfg.RateLimitDefaultMax = 100
	cfg.RateLimitPaymentMax = 100
	cfg.CircuitFailureThreshold = 100

	middleware := newReliabilityMiddlewareWithClock(cfg, time.Now)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	go func() {
		defer close(done)
		firstReq := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
		firstReq.RemoteAddr = "127.0.0.1:3001"
		firstRes := httptest.NewRecorder()
		h.ServeHTTP(firstRes, firstReq)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not enter handler")
	}

	go func() {
		time.Sleep(25 * time.Millisecond)
		close(release)
	}()

	criticalReq := httptest.NewRequest(http.MethodPost, "/v1/payment/chargeback", nil)
	criticalReq.RemoteAddr = "127.0.0.1:3002"
	criticalRes := httptest.NewRecorder()
	h.ServeHTTP(criticalRes, criticalReq)
	if criticalRes.Code != http.StatusOK {
		t.Fatalf("critical request status = %d, want %d", criticalRes.Code, http.StatusOK)
	}

	<-done
}

func TestReliabilityMiddleware_CircuitBreakerRejectsAfterFailures(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	cfg := DefaultReliabilityConfig()
	cfg.CircuitFailureThreshold = 2
	cfg.CircuitOpenDuration = 30 * time.Second
	cfg.RateLimitDefaultMax = 100

	middleware := newReliabilityMiddlewareWithClock(cfg, func() time.Time { return current })
	var callCount int32
	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req.RemoteAddr = "127.0.0.1:4001"

	first := httptest.NewRecorder()
	h.ServeHTTP(first, req)
	if first.Code != http.StatusInternalServerError {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusInternalServerError)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, req)
	if second.Code != http.StatusInternalServerError {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusInternalServerError)
	}

	third := httptest.NewRecorder()
	h.ServeHTTP(third, req)
	if third.Code != http.StatusServiceUnavailable {
		t.Fatalf("third request status = %d, want %d", third.Code, http.StatusServiceUnavailable)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("handler call count = %d, want %d", callCount, 2)
	}
}

func TestReliabilityMiddleware_WebhookBypassesCircuitReject(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	cfg := DefaultReliabilityConfig()
	cfg.CircuitFailureThreshold = 1
	cfg.CircuitOpenDuration = 30 * time.Second
	cfg.RateLimitWebhookMax = 100

	middleware := newReliabilityMiddlewareWithClock(cfg, func() time.Time { return current })
	var callCount int32
	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", nil)
	req.RemoteAddr = "127.0.0.1:5001"

	for i := 0; i < 3; i++ {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusInternalServerError {
			t.Fatalf("webhook request %d status = %d, want %d", i+1, res.Code, http.StatusInternalServerError)
		}
	}
	if atomic.LoadInt32(&callCount) != 3 {
		t.Fatalf("handler call count = %d, want %d", callCount, 3)
	}
}

func TestReliabilityMiddleware_DualSemaphorePriorityHeadroom(t *testing.T) {
	cfg := DefaultReliabilityConfig()
	// Total: 20 permits. Reserved for foundry critical: 20% (min 8) -> 8 permits.
	// Non-critical limit: 20 - 8 = 12 permits.
	cfg.PriorityMaxInFlight = 20
	cfg.CriticalAcquireTimeout = 250 * time.Millisecond
	cfg.RateLimitDefaultMax = 1000

	middleware := newReliabilityMiddlewareWithClock(cfg, time.Now)
	defer middleware.Close()

	holdNonCritical := make(chan struct{})
	var enteredCount atomic.Int32
	var wg sync.WaitGroup

	h := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if classifyReliabilityPath(r.URL.Path) == reliabilityClassOperational {
			enteredCount.Add(1)
			<-holdNonCritical
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Launch 12 non-critical requests to saturate nonCriticalSem
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
			req.RemoteAddr = "127.0.0.1:6000"
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
		}(i)
	}

	// Wait until all 12 non-critical requests are in-flight
	for i := 0; i < 50; i++ {
		if enteredCount.Load() == 12 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if enteredCount.Load() != 12 {
		t.Fatalf("expected 12 non-critical in flight, got %d", enteredCount.Load())
	}

	// 13th non-critical request must be load shed (503 Service Unavailable)
	extraReq := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	extraReq.RemoteAddr = "127.0.0.1:6001"
	extraRec := httptest.NewRecorder()
	h.ServeHTTP(extraRec, extraReq)
	if extraRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for extra non-critical request, got %d", extraRec.Code)
	}

	// All 5 foundry critical paths must succeed because 8 headroom permits are reserved exclusively for them
	foundryPaths := []string{
		"/v1/foundry/cashier/sessions",
		"/v1/foundry/kinetic/readings",
		"/v1/foundry/fleet/dvir/check",
		"/v1/foundry/fleet/mid-route-rescue/status",
		"/v1/foundry/guards/evaluate",
	}

	for _, p := range foundryPaths {
		fcReq := httptest.NewRequest(http.MethodPost, p, nil)
		fcReq.RemoteAddr = "127.0.0.1:7000"
		fcRec := httptest.NewRecorder()
		h.ServeHTTP(fcRec, fcReq)
		if fcRec.Code != http.StatusOK {
			t.Fatalf("path %s should succeed using reserved headroom, got status %d", p, fcRec.Code)
		}
	}

	close(holdNonCritical)
	wg.Wait()
}

func TestReliabilityMiddleware_Close(t *testing.T) {
	cfg := DefaultReliabilityConfig()
	middleware := newReliabilityMiddlewareWithClock(cfg, time.Now)
	// Calling Close multiple times must be safe and idempotent
	middleware.Close()
	middleware.Close()
}
