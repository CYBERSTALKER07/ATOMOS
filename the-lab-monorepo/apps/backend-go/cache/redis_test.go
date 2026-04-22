package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr
}

// ─── Init ───────────────────────────────────────────────────────────────────

func TestInit_EmptyAddr_NilClient(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	Client = nil
	Init("")
	if Client != nil {
		t.Error("Init with empty addr should leave Client nil")
	}
}

func TestInit_ValidAddr_SetsClient(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	Client = nil
	Init(mr.Addr())
	if Client == nil {
		t.Error("Init with valid addr should set Client")
	}
}

func TestInit_BadAddr_NilClient(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	Client = nil
	Init("127.0.0.1:1") // unreachable port
	if Client != nil {
		t.Error("Init with unreachable addr should leave Client nil after failed ping")
	}
}

// ─── MarkResolved ───────────────────────────────────────────────────────────

func TestMarkResolved_Success(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	setupMiniredis(t)

	ctx := context.Background()
	if err := MarkResolved(ctx, 42); err != nil {
		t.Fatalf("MarkResolved: %v", err)
	}
	if !IsResolved(ctx, 42) {
		t.Error("offset 42 should be resolved after MarkResolved")
	}
}

func TestMarkResolved_Idempotent(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	setupMiniredis(t)

	ctx := context.Background()
	MarkResolved(ctx, 99)
	MarkResolved(ctx, 99) // second call should not error
	if !IsResolved(ctx, 99) {
		t.Error("offset 99 should still be resolved")
	}
}

func TestMarkResolved_NilClient_NoError(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	Client = nil

	if err := MarkResolved(context.Background(), 1); err != nil {
		t.Errorf("MarkResolved with nil client should return nil, got %v", err)
	}
}

// ─── IsResolved ─────────────────────────────────────────────────────────────

func TestIsResolved_NotPresent(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	setupMiniredis(t)

	if IsResolved(context.Background(), 999) {
		t.Error("offset 999 should not be resolved")
	}
}

func TestIsResolved_NilClient_ReturnsFalse(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	Client = nil

	if IsResolved(context.Background(), 1) {
		t.Error("IsResolved with nil client should return false")
	}
}

// ─── ResolvedOffsets ────────────────────────────────────────────────────────

func TestResolvedOffsets_MultipleEntries(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	setupMiniredis(t)

	ctx := context.Background()
	MarkResolved(ctx, 10)
	MarkResolved(ctx, 20)
	MarkResolved(ctx, 30)

	offsets := ResolvedOffsets(ctx)
	if len(offsets) != 3 {
		t.Fatalf("expected 3 offsets, got %d", len(offsets))
	}
	for _, key := range []string{"10", "20", "30"} {
		if !offsets[key] {
			t.Errorf("expected offset %s in resolved set", key)
		}
	}
}

func TestResolvedOffsets_Empty(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	setupMiniredis(t)

	offsets := ResolvedOffsets(context.Background())
	if len(offsets) != 0 {
		t.Errorf("expected 0 offsets, got %d", len(offsets))
	}
}

func TestResolvedOffsets_NilClient_EmptyMap(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	Client = nil

	offsets := ResolvedOffsets(context.Background())
	if offsets == nil {
		t.Error("ResolvedOffsets with nil client should return non-nil empty map")
	}
	if len(offsets) != 0 {
		t.Errorf("expected 0 offsets, got %d", len(offsets))
	}
}

// ─── PurgeResolutionLedger ──────────────────────────────────────────────────

func TestPurgeResolutionLedger_ClearsAll(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	setupMiniredis(t)

	ctx := context.Background()
	MarkResolved(ctx, 1)
	MarkResolved(ctx, 2)

	if err := PurgeResolutionLedger(ctx); err != nil {
		t.Fatalf("PurgeResolutionLedger: %v", err)
	}

	offsets := ResolvedOffsets(ctx)
	if len(offsets) != 0 {
		t.Errorf("expected 0 offsets after purge, got %d", len(offsets))
	}
}

func TestPurgeResolutionLedger_NilClient_NoError(t *testing.T) {
	orig := Client
	defer func() { Client = orig }()
	Client = nil

	if err := PurgeResolutionLedger(context.Background()); err != nil {
		t.Errorf("PurgeResolutionLedger with nil client should return nil, got %v", err)
	}
}

// ─── DLQResolvedKey constant ────────────────────────────────────────────────

func TestDLQResolvedKey_Value(t *testing.T) {
	if DLQResolvedKey != "dlq:resolved_offsets" {
		t.Errorf("DLQResolvedKey = %q, want dlq:resolved_offsets", DLQResolvedKey)
	}
}
