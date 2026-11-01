package bootstrap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

// TestMiddlewarePipelineOrdering_MaxBytesBeforeIdempotency verifies CG-3:
// MaxBytesMiddleware MUST be placed and executed BEFORE IdempotencyMiddleware
// in the request ingress pipeline.
func TestMiddlewarePipelineOrdering_MaxBytesBeforeIdempotency(t *testing.T) {
	store := idempotency.NewInMemoryStore()
	idemMW := IdempotencyMiddleware(store)
	maxBytesMW := MaxBytesMiddleware(500) // 500 byte limit for test

	handlerReached := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReached = true
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	// Pipeline: MaxBytesMiddleware -> IdempotencyMiddleware -> Handler
	pipeline := maxBytesMW(idemMW(handler))

	// Request exceeding MaxBytes limit (1000 bytes) with an Idempotency-Key
	oversizedBody := bytes.Repeat([]byte("a"), 1000)
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(oversizedBody))
	req.Header.Set("Idempotency-Key", "test-pipe-order-key")
	rec := httptest.NewRecorder()

	pipeline.ServeHTTP(rec, req)

	// Since MaxBytesMiddleware precedes IdempotencyMiddleware:
	// When IdempotencyMiddleware tries to read r.Body, MaxBytesReader returns an error.
	if handlerReached {
		t.Fatal("handler should not have been reached for request exceeding MaxBytes limit")
	}

	// Verify idempotency store was not populated
	scopedKey := idempotency.ScopeKey("", "POST /v1/orders", "test-pipe-order-key")
	if _, found, _ := store.Load(context.Background(), scopedKey); found {
		t.Fatal("store should not have recorded or cached when MaxBytes rejects")
	}
}
