package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

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
