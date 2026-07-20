package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type flakyBackend struct {
	fail bool
	data map[string][]byte
}

func (f *flakyBackend) Get(_ context.Context, key string) ([]byte, bool, error) {
	if f.fail {
		return nil, false, errors.New("redis down")
	}
	v, ok := f.data[key]
	return v, ok, nil
}

func (f *flakyBackend) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if f.fail {
		return errors.New("redis down")
	}
	if f.data == nil {
		f.data = map[string][]byte{}
	}
	f.data[key] = value
	return nil
}

func (f *flakyBackend) Delete(_ context.Context, _ ...string) error {
	if f.fail {
		return errors.New("redis down")
	}
	return nil
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
