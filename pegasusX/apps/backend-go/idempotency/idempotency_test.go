package idempotency

import (
	"context"
	"errors"
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
