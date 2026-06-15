package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func bodyHashHex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func TestMiddlewareReplaysCompletedRequest(t *testing.T) {
	store := NewInMemoryStore()
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	mw := Middleware(store)(handler)
	body := []byte(`{"items":[]}`)

	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", bytes.NewReader(body))
		req.Header.Set("Idempotency-Key", "checkout:test")
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		return rec
	}

	first := do()
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d body=%s", first.Code, http.StatusCreated, first.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("handler calls = %d, want 1", calls.Load())
	}

	second := do()
	if second.Code != http.StatusCreated {
		t.Fatalf("replay status = %d, want %d body=%s", second.Code, http.StatusCreated, second.Body.String())
	}
	if second.Body.String() != `{"ok":true}` {
		t.Fatalf("replay body = %q", second.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("handler calls after replay = %d, want 1", calls.Load())
	}
}

func TestMiddlewareSkipsHandlerGuardWhenClaimed(t *testing.T) {
	store := NewInMemoryStore()
	body := []byte(`{"items":[]}`)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if _, hit, err := Guard(r.Context(), store, "checkout:claimed", bodyHashHex(payload)); err != nil || hit {
			t.Fatalf("handler guard should noop when middleware claimed: hit=%v err=%v", hit, err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"handled":true}`))
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewReader(body))
	req.Header.Set("Idempotency-Key", "checkout:claimed")
	rec := httptest.NewRecorder()
	Middleware(store)(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGuardSkipsWhenClaimed(t *testing.T) {
	store := NewInMemoryStore()
	ctx := WithClaimed(context.Background())
	if _, hit, err := Guard(ctx, store, "k", "hash"); err != nil || hit {
		t.Fatalf("Guard on claimed ctx: hit=%v err=%v", hit, err)
	}
}

func TestMiddlewareIgnoresGETWithoutKey(t *testing.T) {
	store := NewInMemoryStore()
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	rec := httptest.NewRecorder()
	Middleware(store)(handler).ServeHTTP(rec, req)

	if calls.Load() != 1 || rec.Code != http.StatusOK {
		t.Fatalf("GET should pass through: calls=%d code=%d", calls.Load(), rec.Code)
	}
}
