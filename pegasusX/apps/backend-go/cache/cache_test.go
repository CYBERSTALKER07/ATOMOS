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

func TestMGetAndBatchInvalidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	backend := NewInMemoryBackend()
	cacheClient := New(backend, nil)

	if err := cacheClient.Set(ctx, "{sup:1}:profile", []byte("p1"), time.Minute); err != nil {
		t.Fatalf("set k1: %v", err)
	}
	if err := cacheClient.Set(ctx, "{sup:1}:orders", []byte("o1"), time.Minute); err != nil {
		t.Fatalf("set k2: %v", err)
	}

	got, err := cacheClient.MGet(ctx, "{sup:1}:profile", "{sup:1}:orders", "{sup:1}:missing")
	if err != nil {
		t.Fatalf("MGet error: %v", err)
	}
	if string(got["{sup:1}:profile"]) != "p1" || string(got["{sup:1}:orders"]) != "o1" {
		t.Fatalf("unexpected MGet result: %v", got)
	}
	if _, exists := got["{sup:1}:missing"]; exists {
		t.Fatalf("expected missing key to be absent from MGet map")
	}

	msgs, cancel, err := backend.Subscribe(ctx, InvalidationChannel)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer cancel()

	cacheClient.Invalidate(ctx, "{sup:1}:profile", "{sup:1}:orders")

	received := make(map[string]bool)
	for i := 0; i < 2; i++ {
		select {
		case m := <-msgs:
			received[string(m)] = true
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for batch invalidation message %d", i)
		}
	}
	if !received["{sup:1}:profile"] || !received["{sup:1}:orders"] {
		t.Fatalf("expected both invalidated keys to be published, got %v", received)
	}
}

func TestKeyPrefixNormalization_ClusterHashTags(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"{sup:123}:profile":          "sup:profile",
		"{dash:supplier:123}:view":   "dash:view",
		"{idem:abc-123}":             "idem",
		"{driver:drv-9}:last_loc":    "driver:last_loc",
		"semcache:pricing:deadbeef":  "semcache",
		"rl:ip:127.0.0.1":            "rl",
		"":                           "unknown",
	}
	for input, want := range cases {
		if got := keyPrefix(input); got != want {
			t.Errorf("keyPrefix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestExtractClusterSlotTag(t *testing.T) {
	t.Parallel()

	if got := extractClusterSlotTag("{sup:123}:profile"); got != "sup:123" {
		t.Fatalf("expected slot tag sup:123, got %q", got)
	}
	if got := extractClusterSlotTag("plain:key"); got != "plain:key" {
		t.Fatalf("expected full key as slot identity for plain key, got %q", got)
	}
	if got := extractClusterSlotTag("{}:empty"); got != "{}:empty" {
		t.Fatalf("expected full key as slot identity for empty braces per Redis Cluster spec, got %q", got)
	}
}

func TestRecordServerInfoMetrics(t *testing.T) {
	t.Parallel()

	sampleInfo := "# Memory\r\nused_memory:1048576\r\nmaxmemory:4194304\r\nmem_fragmentation_ratio:1.12\r\n# Stats\r\ninstantaneous_ops_per_sec:840\r\nkeyspace_hits:900\r\nkeyspace_misses:100\r\nrejected_connections:0\r\n# Clients\r\nconnected_clients:24\r\nblocked_clients:1\r\n"
	RecordServerInfoMetrics(sampleInfo)
	RecordSlowlogLength(3)
}

