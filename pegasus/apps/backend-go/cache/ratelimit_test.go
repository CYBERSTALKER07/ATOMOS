package cache

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── DefaultRateLimit / AuthRateLimit ───────────────────────────────────────

func TestDefaultRateLimit_Values(t *testing.T) {
	cfg := DefaultRateLimit()
	if cfg.MaxHits != 120 {
		t.Errorf("MaxHits = %d, want 120", cfg.MaxHits)
	}
	if cfg.Window.Minutes() != 1 {
		t.Errorf("Window = %v, want 1m", cfg.Window)
	}
	if cfg.KeyFunc == nil {
		t.Error("KeyFunc is nil")
	}
}

func TestAuthRateLimit_Values(t *testing.T) {
	cfg := AuthRateLimit()
	if cfg.MaxHits != 10 {
		t.Errorf("MaxHits = %d, want 10", cfg.MaxHits)
	}
	if cfg.Window.Minutes() != 1 {
		t.Errorf("Window = %v, want 1m", cfg.Window)
	}
}

// ─── ipKey ──────────────────────────────────────────────────────────────────

func TestIpKey_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")
	got := ipKey(r)
	if got != "ip:203.0.113.50" {
		t.Errorf("got %q, want ip:203.0.113.50", got)
	}
}

func TestIpKey_SingleXFF(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	got := ipKey(r)
	if got != "ip:10.0.0.1" {
		t.Errorf("got %q, want ip:10.0.0.1", got)
	}
}

func TestIpKey_RemoteAddrFallback(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100:12345"
	got := ipKey(r)
	if got != "ip:192.168.1.100" {
		t.Errorf("got %q, want ip:192.168.1.100", got)
	}
}

func TestIpKey_RemoteAddrNoPort(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100"
	got := ipKey(r)
	if got != "ip:192.168.1.100" {
		t.Errorf("got %q, want ip:192.168.1.100", got)
	}
}

// ─── RateLimitMiddleware (no Redis → fail-open) ─────────────────────────────

func TestRateLimitMiddleware_NilRedis_PassThrough(t *testing.T) {
	// Client is nil → should pass through
	origClient := Client
	Client = nil
	defer func() { Client = origClient }()

	called := false
	inner := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}
	handler := RateLimitMiddleware(DefaultRateLimit())(inner)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if !called {
		t.Error("handler should be called when Redis is nil (fail-open)")
	}
	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ─── LimitBodyMiddleware ────────────────────────────────────────────────────

func TestLimitBodyMiddleware_UnderLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unexpected read error: %v", err)
		}
		w.Write(body)
	})
	handler := LimitBodyMiddleware(1024)(inner)

	r := httptest.NewRequest("POST", "/", strings.NewReader("hello world"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Body.String() != "hello world" {
		t.Errorf("body = %q, want %q", w.Body.String(), "hello world")
	}
}

func TestLimitBodyMiddleware_OverLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := LimitBodyMiddleware(10)(inner)

	bigBody := strings.Repeat("X", 100)
	r := httptest.NewRequest("POST", "/", strings.NewReader(bigBody))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", w.Code)
	}
}

func TestLimitBodyMiddleware_GetRequest_NoLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := LimitBodyMiddleware(10)(inner)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestLimitBodyMiddleware_ExactLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.Write(body)
	})
	handler := LimitBodyMiddleware(5)(inner)

	r := httptest.NewRequest("POST", "/", strings.NewReader("12345"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Body.String() != "12345" {
		t.Errorf("body = %q, want %q", w.Body.String(), "12345")
	}
}
