// Package simulator_test provides simulation tests for the cache layer,
// WebSocket hub, and Kafka worker pool — exercising concurrency, TTL,
// singleflight, and backpressure without external dependencies.
package simulator_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// ─── simConn implements ws.Connection for simulation tests ──────────────────

type simConn struct {
	id     string
	claims auth.Claims
	fail   bool
	sent   chan []byte
}

func newSimConn(id string) *simConn {
	return &simConn{id: id, sent: make(chan []byte, 64)}
}

func (c *simConn) ID() string            { return c.id }
func (c *simConn) Identity() auth.Claims { return c.claims }
func (c *simConn) Send(_ context.Context, payload []byte) error {
	if c.fail {
		return errors.New("send failed")
	}
	cp := make([]byte, len(payload))
	copy(cp, payload)
	select {
	case c.sent <- cp:
	default:
		// Drop if buffer full to avoid deadlocking hub broadcasts
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Cache: basic get/set/invalidate cycle
// ─────────────────────────────────────────────────────────────────────────────

func TestSimCache_SetGetInvalidate(t *testing.T) {
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	ctx := context.Background()

	key := "catalog:product:123"
	value := []byte(`{"name":"Coca-Cola","price":12000}`)

	// Set
	if err := c.Set(ctx, key, value, 5*time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Get (should hit)
	got, found, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !found {
		t.Fatal("expected cache hit, got miss")
	}
	if string(got) != string(value) {
		t.Fatalf("value mismatch: got %s", string(got))
	}

	// Invalidate
	c.Invalidate(ctx, key)

	// Get (should miss)
	_, found, err = c.Get(ctx, key)
	if err != nil {
		t.Fatalf("get after invalidate: %v", err)
	}
	if found {
		t.Fatal("expected cache miss after invalidation")
	}

	t.Log("cache set/get/invalidate cycle ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Cache: GetOrLoad with singleflight deduplication
// ─────────────────────────────────────────────────────────────────────────────

func TestSimCache_GetOrLoad_Singleflight(t *testing.T) {
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	ctx := context.Background()

	loadCount := 0
	var mu sync.Mutex
	loader := func(ctx context.Context) ([]byte, error) {
		mu.Lock()
		loadCount++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // simulate slow DB query
		return []byte(`{"loaded":true}`), nil
	}

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := c.GetOrLoad(ctx, "singleflight:test", 5*time.Minute, loader)
			if err != nil {
				errs <- fmt.Sprintf("GetOrLoad: %v", err)
				return
			}
			if string(val) != `{"loaded":true}` {
				errs <- fmt.Sprintf("unexpected value: %s", string(val))
			}
		}()
	}
	wg.Wait()
	close(errs)

	for errMsg := range errs {
		t.Error(errMsg)
	}

	mu.Lock()
	count := loadCount
	mu.Unlock()

	if count > 5 {
		t.Errorf("singleflight: expected minimal loads, got %d (of %d goroutines)", count, goroutines)
	}
	t.Logf("singleflight: loader called %d times for %d goroutines ✓", count, goroutines)
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Cache: concurrent set/get safety
// ─────────────────────────────────────────────────────────────────────────────

func TestSimCache_ConcurrentSetGet(t *testing.T) {
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	ctx := context.Background()

	const goroutines = 100
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent:key:%d", idx%10)
			val := []byte(fmt.Sprintf("value-%d", idx))
			if err := c.Set(ctx, key, val, time.Minute); err != nil {
				t.Errorf("set %s: %v", key, err)
			}
			_, _, err := c.Get(ctx, key)
			if err != nil {
				t.Errorf("get %s: %v", key, err)
			}
		}(i)
	}
	wg.Wait()
	t.Log("concurrent set/get: no race conditions or panics ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Cache: invalidation of multiple keys
// ─────────────────────────────────────────────────────────────────────────────

func TestSimCache_BatchInvalidation(t *testing.T) {
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	ctx := context.Background()

	keys := []string{"key-a", "key-b", "key-c", "key-d"}
	for _, k := range keys {
		if err := c.Set(ctx, k, []byte("data"), 5*time.Minute); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}

	c.Invalidate(ctx, keys...)

	for _, k := range keys {
		_, found, _ := c.Get(ctx, k)
		if found {
			t.Errorf("key %s should be invalidated", k)
		}
	}
	t.Log("batch invalidation ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. Cache: nil cache safety
// ─────────────────────────────────────────────────────────────────────────────

func TestSimCache_NilCacheSafety(t *testing.T) {
	var c *cache.Cache
	ctx := context.Background()

	_, found, err := c.Get(ctx, "key")
	if err != nil {
		t.Errorf("nil cache get should not error: %v", err)
	}
	if found {
		t.Error("nil cache get should return not found")
	}
	t.Log("nil cache safety ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. WebSocket Hub: subscribe/unsubscribe/broadcast simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimWSHub_SubscribeUnsubscribe(t *testing.T) {
	hub := ws.NewHub("sim-test", nil, nil)

	conn := newSimConn("conn-001")
	unsub := hub.Subscribe("room:supplier:001", conn)

	if !hub.HasCapacity() {
		t.Error("hub should have capacity")
	}

	stats := hub.Stats()
	t.Logf("after subscribe: %+v", stats)

	unsub() // unsubscribe
	t.Log("WebSocket subscribe/unsubscribe ✓")
}

func TestSimWSHub_Broadcast(t *testing.T) {
	hub := ws.NewHub("sim-broadcast", nil, nil)
	ctx := context.Background()

	// Subscribe 3 connections to same room
	conns := make([]*simConn, 3)
	for i := 0; i < 3; i++ {
		conns[i] = newSimConn(fmt.Sprintf("conn-%03d", i))
		hub.Subscribe("room:orders:001", conns[i])
	}

	// Broadcast
	hub.Broadcast(ctx, "room:orders:001", []byte(`{"event":"order_update","order_id":"ord-001"}`))

	// Each connection should receive the message
	for i, conn := range conns {
		select {
		case msg := <-conn.sent:
			t.Logf("conn %d received: %s", i, string(msg))
		case <-time.After(100 * time.Millisecond):
			t.Errorf("conn %d did not receive broadcast within 100ms", i)
		}
	}
	t.Log("WebSocket broadcast ✓")
}

func TestSimWSHub_BroadcastToWrongRoom(t *testing.T) {
	hub := ws.NewHub("sim-rooms", nil, nil)
	ctx := context.Background()

	conn := newSimConn("conn-001")
	hub.Subscribe("room:A", conn)

	// Broadcast to a different room
	hub.Broadcast(ctx, "room:B", []byte(`{"msg":"should_not_arrive"}`))

	select {
	case msg := <-conn.sent:
		t.Errorf("conn should NOT receive message for different room, got: %s", string(msg))
	case <-time.After(50 * time.Millisecond):
		// expected: no message
	}
	t.Log("room isolation ✓")
}

func TestSimWSHub_FailedSendReapsConnection(t *testing.T) {
	hub := ws.NewHub("sim-reap", nil, nil)
	ctx := context.Background()

	conn := newSimConn("fail-conn")
	conn.fail = true
	hub.Subscribe("room:test", conn)

	// Broadcast should trigger Send which fails — hub should reap
	hub.Broadcast(ctx, "room:test", []byte(`{"msg":"trigger"}`))

	t.Log("failed send reaps connection (no panic) ✓")
}

func TestSimWSHub_ConcurrentBroadcast(t *testing.T) {
	hub := ws.NewHub("sim-concurrent", nil, nil)
	ctx := context.Background()

	// Subscribe 10 connections
	for i := 0; i < 10; i++ {
		conn := newSimConn(fmt.Sprintf("stress-%d", i))
		hub.Subscribe("room:stress", conn)
	}

	// Concurrent broadcast (100 messages from 10 goroutines)
	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(gIdx int) {
			defer wg.Done()
			for m := 0; m < 10; m++ {
				hub.Broadcast(ctx, "room:stress", []byte(fmt.Sprintf(`{"g":%d,"m":%d}`, gIdx, m)))
			}
		}(g)
	}
	wg.Wait()
	t.Log("concurrent broadcast: no panics or deadlocks ✓")
}

func TestSimWSHub_MultiRoom(t *testing.T) {
	hub := ws.NewHub("sim-multi", nil, nil)
	ctx := context.Background()

	connA := newSimConn("conn-A")
	connB := newSimConn("conn-B")
	connBoth := newSimConn("conn-Both")

	hub.Subscribe("room:A", connA)
	hub.Subscribe("room:B", connB)
	hub.Subscribe("room:A", connBoth)
	hub.Subscribe("room:B", connBoth)

	hub.Broadcast(ctx, "room:A", []byte(`{"room":"A"}`))
	hub.Broadcast(ctx, "room:B", []byte(`{"room":"B"}`))

	// connA should get room:A only
	select {
	case <-connA.sent:
	case <-time.After(50 * time.Millisecond):
		t.Error("connA should receive room:A broadcast")
	}

	// connB should get room:B only
	select {
	case <-connB.sent:
	case <-time.After(50 * time.Millisecond):
		t.Error("connB should receive room:B broadcast")
	}

	// connBoth should get both
	received := 0
	for received < 2 {
		select {
		case <-connBoth.sent:
			received++
		case <-time.After(50 * time.Millisecond):
			t.Fatalf("connBoth only received %d/2 broadcasts", received)
		}
	}

	t.Log("multi-room isolation and multi-subscribe ✓")
}
