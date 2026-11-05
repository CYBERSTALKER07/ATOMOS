package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
		w.Header().Set("Content-Type", "application/json")
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
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("replay should set Idempotency-Replayed: true, got %q", second.Header().Get("Idempotency-Replayed"))
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

func TestMiddlewareDoesNotCacheErrorResponses(t *testing.T) {
	store := NewInMemoryStore()
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"invalid_status_transition"}`))
	})

	mw := Middleware(store)(handler)
	body := []byte(`{"status":"CANCELLED"}`)

	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/v1/order/ord-1/status", bytes.NewReader(body))
		req.Header.Set("Idempotency-Key", "admin-order-status:ord-1:CANCELLED")
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		return rec
	}

	first := do()
	if first.Code != http.StatusConflict || calls.Load() != 1 {
		t.Fatalf("first status=%d calls=%d", first.Code, calls.Load())
	}

	second := do()
	if second.Code != http.StatusConflict || calls.Load() != 2 {
		t.Fatalf("retry should hit handler again: status=%d calls=%d", second.Code, calls.Load())
	}
}

func TestMiddlewareRejectsPayloadOver1MiB(t *testing.T) {
	store := NewInMemoryStore()
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(store)(handler)

	// Payload of 1 MiB + 10 bytes
	oversized := bytes.Repeat([]byte("x"), (1<<20)+10)
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewReader(oversized))
	req.Header.Set("Idempotency-Key", "oversized-key-1")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("handler should not have been called for oversized payload")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", rec.Header().Get("Content-Type"))
	}
	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "payload_too_large" {
		t.Fatalf("expected error payload_too_large, got %v (body: %s)", errResp, rec.Body.String())
	}

	// Verify no lock was acquired in store
	if _, found, _ := store.Load(context.Background(), ScopeKey("", "POST /v1/order/create", "oversized-key-1")); found {
		t.Fatal("oversized request must not create record or acquire lock in store")
	}
}

func TestMiddlewareDoesNotCacheResponseOver1MiB(t *testing.T) {
	store := NewInMemoryStore()
	bigPayload := bytes.Repeat([]byte("a"), (1<<20)+100)
	var calls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bigPayload)
	})

	mw := Middleware(store)(handler)
	key := "big-resp-key"

	req := httptest.NewRequest(http.MethodPost, "/v1/export/data", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected handler called once, got %d", calls.Load())
	}

	// Lock must be released (not held in progress) and response not saved
	scopedKey := ScopeKey("", "POST /v1/export/data", key)
	recRecord, found, err := store.Load(context.Background(), scopedKey)
	if err != nil {
		t.Fatalf("unexpected load err: %v", err)
	}
	if found {
		t.Fatalf("response > 1 MiB should NOT be cached in store, but found: %+v", recRecord)
	}

	// Immediate next request should execute handler again (lock was released)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/export/data", bytes.NewReader([]byte(`{}`)))
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if calls.Load() != 2 {
		t.Fatalf("expected handler called again after release, got %d", calls.Load())
	}
}

func TestMiddlewareConcurrentRequestReturns409RetryAfter1(t *testing.T) {
	store := NewInMemoryStore()
	handlerBlocked := make(chan struct{})
	unblockHandler := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerBlocked)
		<-unblockHandler
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	mw := Middleware(store)(handler)
	key := "concurrent-test-key"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodPost, "/v1/trade/order", bytes.NewReader([]byte(`{"val":1}`)))
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
	}()

	<-handlerBlocked

	// Concurrent second request with same key
	req2 := httptest.NewRequest(http.MethodPost, "/v1/trade/order", bytes.NewReader([]byte(`{"val":1}`)))
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict for in-progress request, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") != "1" {
		t.Fatalf("expected Retry-After: 1, got %q", rec2.Header().Get("Retry-After"))
	}
	var errResp map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &errResp); err != nil || errResp["error"] != "request_in_progress" {
		t.Fatalf("expected error request_in_progress, got %v (body: %s)", errResp, rec2.Body.String())
	}

	close(unblockHandler)
	wg.Wait()
}

func TestMiddlewarePayloadMismatchReturns422(t *testing.T) {
	store := NewInMemoryStore()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	mw := Middleware(store)(handler)
	key := "mismatch-key"

	// First request succeeds
	req1 := httptest.NewRequest(http.MethodPost, "/v1/resource/update", bytes.NewReader([]byte(`{"data":"A"}`)))
	req1.Header.Set("Idempotency-Key", key)
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	// Second request with SAME key but DIFFERENT payload -> 422 Unprocessable Entity
	req2 := httptest.NewRequest(http.MethodPost, "/v1/resource/update", bytes.NewReader([]byte(`{"data":"B"}`)))
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 Unprocessable Entity, got %d", rec2.Code)
	}
	if rec2.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", rec2.Header().Get("Content-Type"))
	}
	var errResp map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &errResp); err != nil || errResp["error"] != "idempotency_key_payload_mismatch" {
		t.Fatalf("expected error idempotency_key_payload_mismatch, got %v (body: %s)", errResp, rec2.Body.String())
	}
}

func TestMiddlewareReplaySetsReplayedHeaderAndPreservesContentType(t *testing.T) {
	store := NewInMemoryStore()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"res-123"}`))
	})

	mw := Middleware(store)(handler)
	key := "replay-ct-key"

	req1 := httptest.NewRequest(http.MethodPost, "/v1/entities", bytes.NewReader([]byte(`{"name":"foo"}`)))
	req1.Header.Set("Idempotency-Key", key)
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec1.Code)
	}

	// Replay
	req2 := httptest.NewRequest(http.MethodPost, "/v1/entities", bytes.NewReader([]byte(`{"name":"foo"}`)))
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201 on replay, got %d", rec2.Code)
	}
	if rec2.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected Idempotency-Replayed: true, got %q", rec2.Header().Get("Idempotency-Replayed"))
	}
	if rec2.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("expected Content-Type application/problem+json preserved, got %q", rec2.Header().Get("Content-Type"))
	}
	if rec2.Body.String() != `{"id":"res-123"}` {
		t.Fatalf("expected body preserved, got %q", rec2.Body.String())
	}
}

func TestMiddlewareDifferentiatedLockRetention_4xxReleasesImmediately(t *testing.T) {
	store := NewInMemoryStore()
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad_input"}`))
	})

	mw := Middleware(store)(handler)
	key := "err-4xx-key"

	req := httptest.NewRequest(http.MethodPost, "/v1/validate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Lock should be released immediately (no in-progress error on second try)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/validate", bytes.NewReader([]byte(`{}`)))
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if calls.Load() != 2 {
		t.Fatalf("expected handler called again after 4xx release, calls=%d", calls.Load())
	}
}

func TestMiddlewareDifferentiatedLockRetention_504HoldsLockFor10s(t *testing.T) {
	store := NewInMemoryStore()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`{"error":"gateway_timeout"}`))
	})

	mw := Middleware(store)(handler)
	key := "timeout-504-key"

	req := httptest.NewRequest(http.MethodPost, "/v1/external/sync", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "10" {
		t.Fatalf("expected Retry-After: 10, got %q", rec.Header().Get("Retry-After"))
	}

	// The lock must be held for 10s: immediately subsequent request gets 409 request_in_progress
	req2 := httptest.NewRequest(http.MethodPost, "/v1/external/sync", bytes.NewReader([]byte(`{}`)))
	req2.Header.Set("Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict while 504 lock is held, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") != "1" {
		t.Fatalf("expected Retry-After: 1 on concurrent attempt, got %q", rec2.Header().Get("Retry-After"))
	}
}

func TestMiddlewareDetachedContextSurvivesClientCancel(t *testing.T) {
	store := NewInMemoryStore()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate client cancellation during processing
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"done":true}`))
	})

	mw := Middleware(store)(handler)
	key := "cancel-test-key"

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/v1/action", bytes.NewReader([]byte(`{}`))).WithContext(ctx)
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()

	// Cancel context during processing
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()

	mw.ServeHTTP(rec, req)

	// Since detached context is used, the record save/release operation must succeed without corruption.
	scopedKey := ScopeKey("", "POST /v1/action", key)
	recRecord, found, err := store.Load(context.Background(), scopedKey)
	if err != nil {
		t.Fatalf("unexpected load err: %v", err)
	}
	if !found {
		t.Fatal("expected record to be saved despite client context cancellation")
	}
	if recRecord.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 saved, got %d", recRecord.StatusCode)
	}
}

func TestMiddlewarePartnerKeysDoNotCollide(t *testing.T) {
	store := NewInMemoryStore()
	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mw := Middleware(store)(handler)

	// Partner A request
	reqA := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"order":"A"}`))
	reqA.Header.Set("Idempotency-Key", "po-common-1001")
	ctxA := auth.WithTenant(reqA.Context(), auth.TenantContext{
		KeyID:      "pxk_partner_a",
		TenantType: "retailer",
		TenantID:   "ret_a",
		Source:     "partner",
	})
	reqA = reqA.WithContext(ctxA)
	recA := httptest.NewRecorder()
	mw.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("partner A expected 200, got %d", recA.Code)
	}

	// Partner B request with same key and different body
	reqB := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"order":"B"}`))
	reqB.Header.Set("Idempotency-Key", "po-common-1001")
	ctxB := auth.WithTenant(reqB.Context(), auth.TenantContext{
		KeyID:      "pxk_partner_b",
		TenantType: "retailer",
		TenantID:   "ret_b",
		Source:     "partner",
	})
	reqB = reqB.WithContext(ctxB)
	recB := httptest.NewRecorder()
	mw.ServeHTTP(recB, reqB)

	// Since Partner A and B have different KeyIDs, Partner B must NOT trigger a payload conflict!
	if recB.Code != http.StatusOK {
		t.Fatalf("partner B should not collide with partner A, got %d", recB.Code)
	}
	if handlerCalled != 2 {
		t.Fatalf("expected 2 distinct handler executions, got %d", handlerCalled)
	}
}

func TestMiddlewarePathSanitization(t *testing.T) {
	store := NewInMemoryStore()
	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	mw := Middleware(store)(handler)

	// First request: clean path /v1/orders
	req1 := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"sku":"A"}`))
	req1.Header.Set("Idempotency-Key", "path-test-key")
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("req1 status = %d", rec1.Code)
	}
	if handlerCalled != 1 {
		t.Fatalf("handlerCalled = %d, want 1", handlerCalled)
	}

	// Second request: dirty path with trailing slash /v1/orders/
	req2 := httptest.NewRequest(http.MethodPost, "/v1/orders/", strings.NewReader(`{"sku":"A"}`))
	req2.Header.Set("Idempotency-Key", "path-test-key")
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("req2 status = %d", rec2.Code)
	}
	if rec2.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected replay for trailing slash path, got header=%q", rec2.Header().Get("Idempotency-Replayed"))
	}
	if handlerCalled != 1 {
		t.Fatalf("handlerCalled after dirty path = %d, want 1 (replay expected)", handlerCalled)
	}

	// Third request: dirty path with double slash /v1//orders
	req3 := httptest.NewRequest(http.MethodPost, "/v1//orders", strings.NewReader(`{"sku":"A"}`))
	req3.Header.Set("Idempotency-Key", "path-test-key")
	rec3 := httptest.NewRecorder()
	mw.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("req3 status = %d", rec3.Code)
	}
	if rec3.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected replay for double slash path, got header=%q", rec3.Header().Get("Idempotency-Replayed"))
	}
	if handlerCalled != 1 {
		t.Fatalf("handlerCalled after double slash path = %d, want 1", handlerCalled)
	}
}

func TestMiddlewarePrincipalNamespacing(t *testing.T) {
	store := NewInMemoryStore()
	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"paid"}`))
	})
	mw := Middleware(store)(handler)

	body1 := `{"amount":100}`
	body2 := `{"amount":200}`
	body3 := `{"amount":300}`

	// 1. User claims: subject "target_123" -> scoped "user:target_123"
	reqUser := httptest.NewRequest(http.MethodPost, "/v1/payments", strings.NewReader(body1))
	reqUser.Header.Set("Idempotency-Key", "same-raw-key")
	ctxUser := auth.WithClaims(reqUser.Context(), auth.Claims{Subject: "target_123"})
	reqUser = reqUser.WithContext(ctxUser)
	recUser := httptest.NewRecorder()
	mw.ServeHTTP(recUser, reqUser)
	if recUser.Code != http.StatusOK {
		t.Fatalf("reqUser status = %d", recUser.Code)
	}

	// 2. Partner claims: tenant "target_123", key "pxk_key_1" -> scoped "partner:target_123:pxk_key_1"
	reqPartner := httptest.NewRequest(http.MethodPost, "/v1/payments", strings.NewReader(body2))
	reqPartner.Header.Set("Idempotency-Key", "same-raw-key")
	ctxPartner := auth.WithTenant(reqPartner.Context(), auth.TenantContext{
		TenantID: "target_123",
		KeyID:    "pxk_key_1",
		Source:   "partner",
	})
	reqPartner = reqPartner.WithContext(ctxPartner)
	recPartner := httptest.NewRecorder()
	mw.ServeHTTP(recPartner, reqPartner)
	if recPartner.Code != http.StatusOK {
		t.Fatalf("reqPartner status = %d (should not conflict with user:target_123)", recPartner.Code)
	}

	// 3. Tenant context: supplier "target_123" -> scoped "tenant:target_123"
	reqTenant := httptest.NewRequest(http.MethodPost, "/v1/payments", strings.NewReader(body3))
	reqTenant.Header.Set("Idempotency-Key", "same-raw-key")
	ctxTenant := auth.WithTenant(reqTenant.Context(), auth.TenantContext{
		SupplierID: "target_123",
		Source:     "internal",
	})
	reqTenant = reqTenant.WithContext(ctxTenant)
	recTenant := httptest.NewRecorder()
	mw.ServeHTTP(recTenant, reqTenant)
	if recTenant.Code != http.StatusOK {
		t.Fatalf("reqTenant status = %d (should not conflict with user:target_123 or partner:target_123:pxk_key_1)", recTenant.Code)
	}

	// All 3 callers had the SAME raw key and route, but DIFFERENT principal namespaces and bodies,
	// so all 3 must execute successfully without 422 payload mismatch.
	if handlerCalled != 3 {
		t.Fatalf("expected 3 distinct handler executions, got %d", handlerCalled)
	}
}
