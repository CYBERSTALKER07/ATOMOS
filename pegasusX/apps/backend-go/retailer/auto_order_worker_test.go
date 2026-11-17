package retailer

import (
	"context"
	"testing"
	"time"
)

func TestRunAutoOrderForRetailer_ModeFallbackFromSettings(t *testing.T) {
	ctx := context.Background()

	t.Run("empty mode inherits shadow from settings", func(t *testing.T) {
		n := 0
		svc := NewService(ServiceConfig{
			Now: time.Now,
			NewID: func() string {
				n++
				return "run-" + string(rune('A'+n%26))
			},
		})
		_ = svc.saveAutoOrderDurable(ctx, "ret-shadow", "user-1", AutoOrderSettings{
			GlobalEnabled: true,
			ExecutionMode: AutoOrderModeShadow,
		})
		svc.SeedAutoOrderCandidates("ret-shadow", []AutoOrderCandidate{
			{SKU: "sku-1", ProductID: "sku-1", SupplierID: "sup-1", Qty: 2},
		})

		run := svc.RunAutoOrderForRetailer(ctx, "ret-shadow", "")
		if run.Mode != AutoOrderModeShadow {
			t.Fatalf("expected mode %q, got %q", AutoOrderModeShadow, run.Mode)
		}
		if run.DraftLines != 1 {
			t.Fatalf("expected 1 draft proposal, got %d (status=%s, msg=%s)", run.DraftLines, run.Status, run.Message)
		}
	})

	t.Run("empty mode inherits off from settings", func(t *testing.T) {
		n := 0
		svc := NewService(ServiceConfig{
			Now: time.Now,
			NewID: func() string {
				n++
				return "run-" + string(rune('A'+n%26))
			},
		})
		_ = svc.saveAutoOrderDurable(ctx, "ret-off", "user-1", AutoOrderSettings{
			GlobalEnabled: true,
			ExecutionMode: AutoOrderModeOff,
		})

		run := svc.RunAutoOrderForRetailer(ctx, "ret-off", "")
		if run.Mode != AutoOrderModeOff {
			t.Fatalf("expected mode %q, got %q", AutoOrderModeOff, run.Mode)
		}
		if run.Status != "SKIPPED_ALL" || run.Message != "execution_mode_off" {
			t.Fatalf("expected SKIPPED_ALL with execution_mode_off, got status=%q msg=%q", run.Status, run.Message)
		}
	})

	t.Run("explicit off mode overrides settings", func(t *testing.T) {
		n := 0
		svc := NewService(ServiceConfig{
			Now: time.Now,
			NewID: func() string {
				n++
				return "run-" + string(rune('A'+n%26))
			},
		})
		_ = svc.saveAutoOrderDurable(ctx, "ret-draft", "user-1", AutoOrderSettings{
			GlobalEnabled: true,
			ExecutionMode: AutoOrderModeDraft,
		})

		run := svc.RunAutoOrderForRetailer(ctx, "ret-draft", AutoOrderModeOff)
		if run.Mode != AutoOrderModeOff {
			t.Fatalf("expected mode %q, got %q", AutoOrderModeOff, run.Mode)
		}
		if run.Status != "SKIPPED_ALL" || run.Message != "execution_mode_off" {
			t.Fatalf("expected SKIPPED_ALL with execution_mode_off, got status=%q msg=%q", run.Status, run.Message)
		}
	})

	t.Run("empty mode defaults to draft when settings execution mode is empty", func(t *testing.T) {
		n := 0
		svc := NewService(ServiceConfig{
			Now: time.Now,
			NewID: func() string {
				n++
				return "run-" + string(rune('A'+n%26))
			},
		})
		_ = svc.saveAutoOrderDurable(ctx, "ret-empty", "user-1", AutoOrderSettings{
			GlobalEnabled: true,
			ExecutionMode: "",
		})
		svc.SeedAutoOrderCandidates("ret-empty", []AutoOrderCandidate{
			{SKU: "sku-draft", ProductID: "sku-draft", SupplierID: "sup-1", Qty: 5},
		})

		run := svc.RunAutoOrderForRetailer(ctx, "ret-empty", "")
		if run.Mode != AutoOrderModeDraft {
			t.Fatalf("expected mode %q, got %q", AutoOrderModeDraft, run.Mode)
		}
		if run.DraftLines != 1 {
			t.Fatalf("expected 1 draft line, got %d (status=%s msg=%s)", run.DraftLines, run.Status, run.Message)
		}
	})
}

func TestRunAutoOrderForRetailer_BucketDeduplication(t *testing.T) {
	ctx := context.Background()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "run-" + string(rune('A'+n%26))
		},
	})
	_ = svc.saveAutoOrderDurable(ctx, "ret-dedup", "user-1", AutoOrderSettings{
		GlobalEnabled: true,
		ExecutionMode: AutoOrderModeDraft,
	})
	svc.SeedAutoOrderCandidates("ret-dedup", []AutoOrderCandidate{
		{SKU: "sku-once", ProductID: "sku-once", SupplierID: "sup-1", Qty: 4},
	})

	firstRun := svc.RunAutoOrderForRetailer(ctx, "ret-dedup", AutoOrderModeDraft)
	if firstRun.DraftLines != 1 {
		t.Fatalf("expected first run to produce 1 draft line, got %d", firstRun.DraftLines)
	}

	secondRun := svc.RunAutoOrderForRetailer(ctx, "ret-dedup", AutoOrderModeDraft)
	if secondRun.DraftLines != 0 {
		t.Fatalf("expected second run to produce 0 draft lines due to bucket deduplication, got %d", secondRun.DraftLines)
	}
	skipped := false
	for _, sk := range secondRun.Skipped {
		if sk.SKU == "sku-once" && sk.Reason == "already_processed_bucket" {
			skipped = true
			break
		}
	}
	if !skipped {
		t.Fatalf("expected sku-once skipped with already_processed_bucket, got skipped: %+v", secondRun.Skipped)
	}
}

func TestSupplierIDForRetailerSKU_FavoritesFallback(t *testing.T) {
	ctx := context.Background()
	svc := NewService(ServiceConfig{})
	svc.mu.Lock()
	svc.favoriteSuppliers = map[string]map[string]bool{
		"ret-fav": {"sup-fav-1": true},
	}
	svc.mu.Unlock()

	sid := svc.supplierIDForRetailerSKU(ctx, "ret-fav", "sku-x")
	if sid != "sup-fav-1" {
		t.Fatalf("expected sup-fav-1 from favorites, got %q", sid)
	}
}

func TestMergeSalesHistory(t *testing.T) {
	sellThrough := map[string]int64{
		"2026-05-01": 10,
		"2026-05-03": 5,
	}
	orders := map[string]int64{
		"2026-05-02": 8,
		"2026-05-03": 12, // sellThrough takes precedence when > 0
	}
	res := mergeSalesHistory(sellThrough, orders)
	// Expected days: 2026-05-01 (10), 2026-05-02 (8), 2026-05-03 (5)
	if len(res) != 3 {
		t.Fatalf("expected 3 days, got %d", len(res))
	}
	if res[0] != 10 || res[1] != 8 || res[2] != 5 {
		t.Fatalf("unexpected merged history: %v", res)
	}

	emptyRes := mergeSalesHistory(nil, nil)
	if len(emptyRes) != 0 {
		t.Fatalf("expected empty slice for empty inputs, got %v", emptyRes)
	}
}
