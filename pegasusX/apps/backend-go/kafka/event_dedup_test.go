package kafka

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInMemoryEventDedupDropsDuplicate(t *testing.T) {
	t.Parallel()
	store := NewInMemoryEventDedup(time.Minute)
	ctx := context.Background()
	ok1, err := store.ShouldProcess(ctx, "evt-1")
	if err != nil || !ok1 {
		t.Fatalf("first process: ok=%v err=%v", ok1, err)
	}
	ok2, err := store.ShouldProcess(ctx, "evt-1")
	if err != nil {
		t.Fatalf("second process err: %v", err)
	}
	if ok2 {
		t.Fatal("expected duplicate to be dropped")
	}
}

func TestDedupKeyForMessageStable(t *testing.T) {
	t.Parallel()
	k1 := DedupKeyForMessage("pegasusx-orders", 2, 99)
	k2 := DedupKeyForMessage("pegasusx-orders", 2, 99)
	if k1 != k2 {
		t.Fatalf("keys differ: %q vs %q", k1, k2)
	}
}

func TestDedupKeyForConsumerGroupIndependent(t *testing.T) {
	t.Parallel()
	store := NewInMemoryEventDedup(time.Minute)
	ctx := context.Background()
	msgKey := DedupKeyForMessage("pegasusx-main", 0, 42)
	orderKey := DedupKeyForConsumerGroup("void-order-mutator", "pegasusx-main", 0, 42)
	dispatchKey := DedupKeyForConsumerGroup("void-notification-dispatcher", "pegasusx-main", 0, 42)

	okOrder, err := store.ShouldProcess(ctx, orderKey)
	if err != nil || !okOrder {
		t.Fatalf("order consumer first: ok=%v err=%v", okOrder, err)
	}
	okDispatch, err := store.ShouldProcess(ctx, dispatchKey)
	if err != nil || !okDispatch {
		t.Fatalf("notification dispatcher should not be suppressed by order consumer: ok=%v err=%v", okDispatch, err)
	}
	if orderKey == dispatchKey || orderKey == msgKey {
		t.Fatal("expected distinct dedup keys per consumer group")
	}
}

func TestConcurrentInMemoryEventDedup(t *testing.T) {
	t.Parallel()
	store := NewInMemoryEventDedup(time.Minute)
	ctx := context.Background()
	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := store.ShouldProcess(ctx, "concurrent-key")
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 1 {
		t.Fatalf("allowed=%d want 1", allowed)
	}
}
