package factory

import (
	"testing"
	"time"
)

func TestDecideLockAcquire_EmptyLockWins(t *testing.T) {
	got := DecideLockAcquire(time.Now().UTC(), "wh-1", 10, lockSnapshot{})
	if !got.Acquired {
		t.Fatal("expected acquire on empty lock")
	}
}

func TestDecideLockAcquire_HigherVelocityPreempts(t *testing.T) {
	now := time.Now().UTC()
	got := DecideLockAcquire(now, "wh-fast", 20, lockSnapshot{
		Present:    true,
		AcquiredBy: "wh-slow",
		Priority:   5,
		ExpiresAt:  now.Add(5 * time.Minute),
	})
	if !got.Acquired {
		t.Fatal("expected preemption")
	}
}

func TestDecideLockAcquire_LowerVelocityDenied(t *testing.T) {
	now := time.Now().UTC()
	got := DecideLockAcquire(now, "wh-slow", 1, lockSnapshot{
		Present:    true,
		AcquiredBy: "wh-fast",
		Priority:   9,
		ExpiresAt:  now.Add(5 * time.Minute),
	})
	if got.Acquired {
		t.Fatal("expected deny")
	}
	if got.HeldBy != "wh-fast" {
		t.Fatalf("held_by=%q", got.HeldBy)
	}
}

func TestDecideLockAcquire_ExpiredReacquired(t *testing.T) {
	now := time.Now().UTC()
	got := DecideLockAcquire(now, "wh-1", 1, lockSnapshot{
		Present:    true,
		AcquiredBy: "wh-old",
		Priority:   99,
		ExpiresAt:  now.Add(-time.Minute),
	})
	if !got.Acquired {
		t.Fatal("expected reacquire after TTL")
	}
}
