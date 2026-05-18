package cache

import (
	"context"
	"testing"
	"time"
)

func waitUntil(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %s", timeout)
}

func TestInvalidateDeletesAndPublishes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	backend := NewInMemoryBackend()
	cacheClient := New(backend, nil)

	msgs, cancel, err := backend.Subscribe(ctx, InvalidationChannel)
	if err != nil {
		t.Fatalf("subscribe invalidation channel: %v", err)
	}
	defer cancel()

	if err := cacheClient.Set(ctx, "k1", []byte("v1"), time.Minute); err != nil {
		t.Fatalf("set cache key: %v", err)
	}

	cacheClient.Invalidate(ctx, "k1")

	if _, found, err := cacheClient.Get(ctx, "k1"); err != nil {
		t.Fatalf("get cache key after invalidate: %v", err)
	} else if found {
		t.Fatalf("expected k1 to be deleted after invalidate")
	}

	select {
	case msg := <-msgs:
		if string(msg) != "k1" {
			t.Fatalf("unexpected invalidation payload: %q", string(msg))
		}
	case <-time.After(time.Second):
		t.Fatalf("expected invalidation publish message")
	}
}

func TestStartInvalidationSubscriberDeletesPublishedKeys(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backend := NewInMemoryBackend()
	cacheClient := New(backend, nil)

	if err := cacheClient.Set(ctx, "k2", []byte("v2"), time.Minute); err != nil {
		t.Fatalf("set cache key: %v", err)
	}

	go cacheClient.StartInvalidationSubscriber(ctx)

	waitUntil(t, time.Second, func() bool {
		_ = backend.Publish(ctx, InvalidationChannel, []byte("k2"))
		_, found, err := cacheClient.Get(ctx, "k2")
		if err != nil {
			return false
		}
		return !found
	})
}
