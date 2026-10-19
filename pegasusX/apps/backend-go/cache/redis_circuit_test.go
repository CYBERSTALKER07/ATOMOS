package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

type flakyBackend struct {
	mu   sync.Mutex
	fail bool
	data map[string][]byte
}

func (f *flakyBackend) setFail(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail = v
}

func (f *flakyBackend) Get(_ context.Context, key string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return nil, false, errors.New("redis down")
	}
	v, ok := f.data[key]
	return v, ok, nil
}

func (f *flakyBackend) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("redis down")
	}
	if f.data == nil {
		f.data = map[string][]byte{}
	}
	f.data[key] = value
	return nil
}

func (f *flakyBackend) Delete(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("redis down")
	}
	for _, k := range keys {
		delete(f.data, k)
	}
	return nil
}

func (f *flakyBackend) IncrBy(_ context.Context, key string, amount int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return 0, errors.New("redis down")
	}
	return amount, nil
}

func (f *flakyBackend) DecrBy(_ context.Context, _ string, amount int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return 0, errors.New("redis down")
	}
	return -amount, nil
}

func (f *flakyBackend) Publish(context.Context, string, []byte) error { return nil }
func (f *flakyBackend) Subscribe(context.Context, string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	return ch, func() {}, nil
}
func (f *flakyBackend) Close() error { return nil }

func TestCircuitBreakerBackend_FailClosedDoesNotUseMemory(t *testing.T) {
	primary := &flakyBackend{fail: true}
	fallback := NewInMemoryBackend()
	_ = fallback.Set(context.Background(), "k", []byte("stale"), time.Minute)

	cb := NewCircuitBreakerBackendWithMode(primary, fallback, true)
	// Trip the breaker by recording enough failures through Do.
	for i := 0; i < 6; i++ {
		_, _, _ = cb.Get(context.Background(), "k")
	}
	val, ok, err := cb.Get(context.Background(), "k")
	if err == nil {
		t.Fatalf("expected error in fail-closed mode, got val=%q ok=%v", val, ok)
	}
	if ok {
		t.Fatal("fail-closed must not return fallback hit")
	}
}

func TestCircuitBreakerBackend_FallbackWhenAllowed(t *testing.T) {
	primary := &flakyBackend{fail: true}
	fallback := NewInMemoryBackend()
	_ = fallback.Set(context.Background(), "k", []byte("mem"), time.Minute)

	cb := NewCircuitBreakerBackendWithMode(primary, fallback, false)
	for i := 0; i < 6; i++ {
		_, _, _ = cb.Get(context.Background(), "k")
	}
	val, ok, err := cb.Get(context.Background(), "k")
	if err != nil {
		t.Fatalf("expected fallback success: %v", err)
	}
	if !ok || string(val) != "mem" {
		t.Fatalf("fallback miss: ok=%v val=%q", ok, val)
	}
}

func TestCircuitBreakerBackend_DegradedSetDelete_FallbackAllowed(t *testing.T) {
	primary := &flakyBackend{fail: true}
	fallback := NewInMemoryBackend()

	cb := NewCircuitBreakerBackendWithMode(primary, fallback, false)
	// Trip breaker
	for i := 0; i < 6; i++ {
		_, _, _ = cb.Get(context.Background(), "probe")
	}
	if cb.Breaker().State() != circuit.StateOpen {
		t.Fatalf("expected breaker to be open, got %s", cb.Breaker().State())
	}

	// Set during open state should route to fallback
	err := cb.Set(context.Background(), "order-123", []byte("payload-xyz"), time.Minute)
	if err != nil {
		t.Fatalf("expected Set to succeed in fallback mode: %v", err)
	}

	// Get should retrieve from fallback
	val, ok, err := cb.Get(context.Background(), "order-123")
	if err != nil || !ok || string(val) != "payload-xyz" {
		t.Fatalf("expected fallback to return payload-xyz, got val=%q ok=%v err=%v", val, ok, err)
	}

	// IncrBy should route to fallback
	count, err := cb.IncrBy(context.Background(), "counter", 5)
	if err != nil || count != 5 {
		t.Fatalf("expected fallback IncrBy 5, got count=%d err=%v", count, err)
	}

	// Delete during open state should route to fallback
	err = cb.Delete(context.Background(), "order-123")
	if err != nil {
		t.Fatalf("expected Delete to succeed in fallback mode: %v", err)
	}
	_, ok, _ = cb.Get(context.Background(), "order-123")
	if ok {
		t.Fatal("expected order-123 to be deleted from fallback")
	}
}

func TestCircuitBreakerBackend_DegradedSetDelete_FailClosed(t *testing.T) {
	primary := &flakyBackend{fail: true}
	fallback := NewInMemoryBackend()

	cb := NewCircuitBreakerBackendWithMode(primary, fallback, true) // failClosed = true
	// Trip breaker
	for i := 0; i < 6; i++ {
		_, _, _ = cb.Get(context.Background(), "probe")
	}
	if cb.Breaker().State() != circuit.StateOpen {
		t.Fatalf("expected breaker to be open, got %s", cb.Breaker().State())
	}

	// Set during open state in failClosed mode must reject
	err := cb.Set(context.Background(), "key", []byte("val"), time.Minute)
	if err != circuit.ErrUpstreamUnavailable {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}

	// Fallback backend should NOT contain the key
	_, ok, _ := fallback.Get(context.Background(), "key")
	if ok {
		t.Fatal("fallback in-memory store should not have received key when failClosed=true")
	}

	// Delete must reject
	err = cb.Delete(context.Background(), "key")
	if err != circuit.ErrUpstreamUnavailable {
		t.Fatalf("expected ErrUpstreamUnavailable for Delete, got %v", err)
	}

	// IncrBy must reject
	_, err = cb.IncrBy(context.Background(), "key", 1)
	if err != circuit.ErrUpstreamUnavailable {
		t.Fatalf("expected ErrUpstreamUnavailable for IncrBy, got %v", err)
	}
}

func TestCircuitBreakerBackend_AutoRecovery_HalfOpenToClosed(t *testing.T) {
	primary := &flakyBackend{
		fail: true,
		data: map[string][]byte{"healthy-key": []byte("healthy-val")},
	}
	fallback := NewInMemoryBackend()

	// Short durations for instant deterministic testing
	b := circuit.New("test-redis-recovery", circuit.Config{
		FailureThreshold: 2,
		FailureWindow:    50 * time.Millisecond,
		OpenDuration:     30 * time.Millisecond,
	})
	cb := NewCircuitBreakerBackendWithBreaker(primary, fallback, b, true)

	// Cause 2 failures to trip to StateOpen
	_, _, err1 := cb.Get(context.Background(), "k")
	_, _, err2 := cb.Get(context.Background(), "k")
	if err1 == nil || err2 == nil {
		t.Fatal("expected errors during initial primary outage")
	}
	if cb.Breaker().State() != circuit.StateOpen {
		t.Fatalf("expected StateOpen, got %s", cb.Breaker().State())
	}

	// Immediate next call should fast-fail with circuit.ErrUpstreamUnavailable
	_, _, fastFailErr := cb.Get(context.Background(), "k")
	if fastFailErr != circuit.ErrUpstreamUnavailable {
		t.Fatalf("expected fast-fail with ErrUpstreamUnavailable, got %v", fastFailErr)
	}

	// Primary recovers
	primary.setFail(false)

	// Wait past OpenDuration so state transitions to Half-Open
	time.Sleep(35 * time.Millisecond)

	// Half-open trial request should succeed on primary
	val, ok, err := cb.Get(context.Background(), "healthy-key")
	if err != nil || !ok || string(val) != "healthy-val" {
		t.Fatalf("expected successful probe in half-open state, got val=%q ok=%v err=%v", val, ok, err)
	}

	// Breaker should now be closed
	if cb.Breaker().State() != circuit.StateClosed {
		t.Fatalf("expected StateClosed after successful recovery probe, got %s", cb.Breaker().State())
	}

	// Subsequent calls succeed cleanly
	val, ok, err = cb.Get(context.Background(), "healthy-key")
	if err != nil || !ok || string(val) != "healthy-val" {
		t.Fatalf("expected clean hit in closed state, got val=%q ok=%v err=%v", val, ok, err)
	}
}

func TestCircuitBreakerBackend_TimeoutDegradation(t *testing.T) {
	primary := &flakyBackend{fail: true}
	fallback := NewInMemoryBackend()
	b := circuit.New("test-redis-timeout", circuit.Config{
		FailureThreshold: 3,
		FailureWindow:    time.Second,
		OpenDuration:     time.Minute,
	})
	cb := NewCircuitBreakerBackendWithBreaker(primary, fallback, b, false)

	// Simulate context timeout / cancel on operations
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Millisecond))
		_, _, _ = cb.Get(ctx, "timeout-probe")
		cancel()
	}

	// Breaker should have registered the failures and tripped
	if cb.Breaker().State() != circuit.StateOpen {
		t.Fatalf("expected StateOpen after timeouts, got %s", cb.Breaker().State())
	}
}
