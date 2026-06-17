package idempotency

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestGuard_AcquiresThenReplays(t *testing.T) {
	t.Parallel()
	store := NewInMemoryStore()
	ctx := context.Background()

	_, hit, err := Guard(ctx, store, "key-1", "hash-a")
	if err != nil || hit {
		t.Fatalf("first guard hit=%v err=%v", hit, err)
	}

	if err := store.Save(ctx, "key-1", Record{
		BodyHash:   "hash-a",
		StatusCode: 200,
		Response:   []byte(`{"ok":true}`),
	}, time.Hour); err != nil {
		t.Fatalf("save: %v", err)
	}

	rec, hit, err := Guard(ctx, store, "key-1", "hash-a")
	if err != nil || !hit || rec.StatusCode != 200 {
		t.Fatalf("replay hit=%v status=%d err=%v", hit, rec.StatusCode, err)
	}
}

func TestGuard_ConflictOnBodyMismatch(t *testing.T) {
	t.Parallel()
	store := NewInMemoryStore()
	ctx := context.Background()

	_, _, err := Guard(ctx, store, "key-2", "hash-a")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	_ = store.Save(ctx, "key-2", Record{BodyHash: "hash-a", StatusCode: 200}, time.Hour)

	_, hit, err := Guard(ctx, store, "key-2", "hash-b")
	if !errors.Is(err, ErrConflict) || !hit {
		t.Fatalf("want conflict hit=true got hit=%v err=%v", hit, err)
	}
}

func TestGuard_InProgressWhileHeld(t *testing.T) {
	t.Parallel()
	store := NewInMemoryStore()
	ctx := context.Background()

	_, _, err := Guard(ctx, store, "key-3", "hash-a")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	_, hit, err := Guard(ctx, store, "key-3", "hash-a")
	if !errors.Is(err, ErrInProgress) || hit {
		t.Fatalf("want in progress hit=false got hit=%v err=%v", hit, err)
	}
}

func TestGuard_ConcurrentAcquireOneWins(t *testing.T) {
	t.Parallel()
	store := NewInMemoryStore()
	ctx := context.Background()
	const key = "concurrent-key"
	const hash = "hash-concurrent"

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, 2)

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := Guard(ctx, store, key, hash)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var inProgress int
	var acquired int
	for err := range results {
		switch {
		case err == nil:
			acquired++
		case errors.Is(err, ErrInProgress):
			inProgress++
		default:
			t.Fatalf("unexpected guard error: %v", err)
		}
	}
	if acquired != 1 || inProgress != 1 {
		t.Fatalf("want one acquire and one in-progress, got acquired=%d inProgress=%d", acquired, inProgress)
	}

	if err := store.Save(ctx, key, Record{
		BodyHash:   hash,
		StatusCode: 200,
		Response:   []byte(`{"ok":true}`),
	}, time.Hour); err != nil {
		t.Fatalf("save: %v", err)
	}

	rec, hit, err := Guard(ctx, store, key, hash)
	if err != nil || !hit || rec.StatusCode != 200 {
		t.Fatalf("replay hit=%v status=%d err=%v", hit, rec.StatusCode, err)
	}

	_, hit, err = Guard(ctx, store, key, "different-hash")
	if !errors.Is(err, ErrConflict) || !hit {
		t.Fatalf("want conflict on body mismatch got hit=%v err=%v", hit, err)
	}
}
