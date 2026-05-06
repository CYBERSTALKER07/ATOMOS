package idempotency

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend-go/cache"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"error":"fail"}`))
}

// ─── Guard: no header → passthrough ─────────────────────────────────────────

func TestGuard_NoHeader_Passthrough(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	called := false
	handler := Guard(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	r := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if !called {
		t.Error("handler should be called when no Idempotency-Key header")
	}
	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ─── Guard: nil Redis → passthrough ─────────────────────────────────────────

func TestGuard_NilRedis_Passthrough(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	cache.Client = nil

	called := false
	handler := Guard(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Idempotency-Key", "test-key")
	w := httptest.NewRecorder()
	handler(w, r)

	if !called {
		t.Error("handler should be called when Redis is nil (degraded)")
	}
}

// ─── Guard: first request → execute + cache ─────────────────────────────────

func TestGuard_FirstRequest_ExecutesAndCaches(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	callCount := 0
	handler := Guard(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"n":1}`))
	})

	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Idempotency-Key", "key-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if callCount != 1 {
		t.Errorf("handler called %d times, want 1", callCount)
	}
	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	// Verify cached in Redis
	val, err := cache.Client.Get(context.Background(), "idem:key-1").Result()
	if err != nil {
		t.Fatalf("expected cached value, got error: %v", err)
	}
	var cr cachedResponse
	if err := json.Unmarshal([]byte(val), &cr); err != nil {
		t.Fatalf("unmarshal cached: %v", err)
	}
	if cr.StatusCode != 200 {
		t.Errorf("cached status = %d, want 200", cr.StatusCode)
	}
	if cr.Body != `{"n":1}` {
		t.Errorf("cached body = %q, want {\"n\":1}", cr.Body)
	}
}

// ─── Guard: duplicate → replay from cache ───────────────────────────────────

func TestGuard_Duplicate_ReplaysCached(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	callCount := 0
	handler := Guard(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"n":1}`))
	})

	// First request
	r1 := httptest.NewRequest("POST", "/", nil)
	r1.Header.Set("Idempotency-Key", "dup-key")
	w1 := httptest.NewRecorder()
	handler(w1, r1)

	// Second request (duplicate)
	r2 := httptest.NewRequest("POST", "/", nil)
	r2.Header.Set("Idempotency-Key", "dup-key")
	w2 := httptest.NewRecorder()
	handler(w2, r2)

	if callCount != 1 {
		t.Errorf("handler called %d times, want 1 (duplicate should be replayed)", callCount)
	}
	if w2.Code != 200 {
		t.Errorf("replay status = %d, want 200", w2.Code)
	}
	if w2.Body.String() != `{"n":1}` {
		t.Errorf("replay body = %q, want {\"n\":1}", w2.Body.String())
	}
}

// ─── Guard: non-2xx → not cached ────────────────────────────────────────────

func TestGuard_Non2xx_NotCached(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	handler := Guard(errorHandler)

	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Idempotency-Key", "err-key")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 500 {
		t.Errorf("status = %d, want 500", w.Code)
	}

	// Should NOT be cached
	_, err := cache.Client.Get(context.Background(), "idem:err-key").Result()
	if err == nil {
		t.Error("non-2xx response should not be cached")
	}
}

// ─── Guard: lock contention → 409 ──────────────────────────────────────────

func TestGuard_LockContention_409(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	// Pre-acquire the lock
	cache.Client.SetNX(context.Background(), "idem:locked-key:lock", "1", 30*time.Second)

	handler := Guard(okHandler)
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Idempotency-Key", "locked-key")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

// ─── Guard: different keys → independent ────────────────────────────────────

func TestGuard_DifferentKeys_Independent(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	callCount := 0
	handler := Guard(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	r1 := httptest.NewRequest("POST", "/", nil)
	r1.Header.Set("Idempotency-Key", "key-a")
	handler(httptest.NewRecorder(), r1)

	r2 := httptest.NewRequest("POST", "/", nil)
	r2.Header.Set("Idempotency-Key", "key-b")
	handler(httptest.NewRecorder(), r2)

	if callCount != 2 {
		t.Errorf("handler called %d times, want 2 (different keys)", callCount)
	}
}

// ─── Purge ──────────────────────────────────────────────────────────────────

func TestPurge_RemovesCachedKey(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	// Seed a cached value
	cache.Client.Set(context.Background(), "idem:purge-me", "data", 24*time.Hour)

	if err := Purge(context.Background(), "purge-me"); err != nil {
		t.Fatalf("Purge: %v", err)
	}

	_, err := cache.Client.Get(context.Background(), "idem:purge-me").Result()
	if err == nil {
		t.Error("expected key to be deleted after Purge")
	}
}

func TestPurge_NilClient_NoError(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	cache.Client = nil

	if err := Purge(context.Background(), "anything"); err != nil {
		t.Errorf("Purge with nil client should return nil, got %v", err)
	}
}

func TestPurge_NonexistentKey_NoError(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	if err := Purge(context.Background(), "does-not-exist"); err != nil {
		t.Errorf("Purge nonexistent key should not error, got %v", err)
	}
}

// ─── Route adoption regression: factory dispatch / supply-request transitions
// and warehouse dispatch-lock acquire/release / supply-request transitions are
// wrapped via Guard in factoryroutes/warehouseroutes. This test pins the
// contract: a state-changing handler must replay the original 2xx response on
// a duplicate call carrying the same Idempotency-Key, even if the underlying
// handler would otherwise mutate state again.
func TestGuard_DispatchAndLockRoutes_Replay(t *testing.T) {
	orig := cache.Client
	defer func() { cache.Client = orig }()
	setupMiniredis(t)

	mutationCount := 0
	dispatchHandler := Guard(func(w http.ResponseWriter, r *http.Request) {
		mutationCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"manifest_id":"m-001"}`))
	})

	for _, key := range []string{
		"factory-dispatch-key",
		"warehouse-dispatch-lock-key",
		"warehouse-supply-request-transition-key",
	} {
		first := httptest.NewRequest("POST", "/", nil)
		first.Header.Set("Idempotency-Key", key)
		w1 := httptest.NewRecorder()
		dispatchHandler(w1, first)
		if w1.Code != http.StatusOK {
			t.Fatalf("%s first call status=%d, want 200", key, w1.Code)
		}

		retry := httptest.NewRequest("POST", "/", nil)
		retry.Header.Set("Idempotency-Key", key)
		w2 := httptest.NewRecorder()
		dispatchHandler(w2, retry)
		if w2.Code != http.StatusOK {
			t.Fatalf("%s retry status=%d, want 200 replay", key, w2.Code)
		}
		if w1.Body.String() != w2.Body.String() {
			t.Fatalf("%s retry body drift: %q vs %q", key, w1.Body.String(), w2.Body.String())
		}
	}

	if mutationCount != 3 {
		t.Errorf("mutation handler called %d times, want 3 (one per key, retries replay)", mutationCount)
	}
}

// ─── Constants ──────────────────────────────────────────────────────────────

func TestConstants(t *testing.T) {
	if headerKey != "Idempotency-Key" {
		t.Errorf("headerKey = %q, want Idempotency-Key", headerKey)
	}
	if keyPrefix != "idem:" {
		t.Errorf("keyPrefix = %q, want idem:", keyPrefix)
	}
	if idempTTL != 24*time.Hour {
		t.Errorf("idempTTL = %v, want 24h", idempTTL)
	}
}
