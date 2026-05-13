package proximity

import (
	"context"
	"errors"
	"testing"
)

func TestResolveWarehouseByH3Distance_PrefersLowerRing(t *testing.T) {
	candidates := []warehouseResolverCandidate{
		{
			match:      &WarehouseMatch{WarehouseId: "wh-b"},
			h3Distance: 2,
		},
		{
			match:      &WarehouseMatch{WarehouseId: "wh-a"},
			h3Distance: 1,
		},
	}

	got := resolveWarehouseByH3Distance(context.Background(), "supplier-1", "872830828ffffff", candidates, nil)
	if got == nil {
		t.Fatal("expected warehouse match")
	}
	if got.WarehouseId != "wh-a" {
		t.Fatalf("warehouse id = %q, want %q", got.WarehouseId, "wh-a")
	}
}

func TestResolveWarehouseByH3Distance_RoundRobinTie(t *testing.T) {
	calls := 0
	next := func(_ context.Context, _ string) (int64, error) {
		calls++
		return 2, nil
	}

	candidates := []warehouseResolverCandidate{
		{
			match:      &WarehouseMatch{WarehouseId: "wh-b"},
			h3Distance: 0,
		},
		{
			match:      &WarehouseMatch{WarehouseId: "wh-a"},
			h3Distance: 0,
		},
	}

	got := resolveWarehouseByH3Distance(context.Background(), "supplier-1", "872830828ffffff", candidates, next)
	if got == nil {
		t.Fatal("expected warehouse match")
	}
	if got.WarehouseId != "wh-b" {
		t.Fatalf("warehouse id = %q, want %q", got.WarehouseId, "wh-b")
	}
	if calls != 1 {
		t.Fatalf("round-robin provider calls = %d, want 1", calls)
	}
}

func TestResolveWarehouseByH3Distance_TieFallbackOnSequenceError(t *testing.T) {
	next := func(_ context.Context, _ string) (int64, error) {
		return 0, errors.New("redis unavailable")
	}

	candidates := []warehouseResolverCandidate{
		{
			match:      &WarehouseMatch{WarehouseId: "wh-z"},
			h3Distance: 0,
		},
		{
			match:      &WarehouseMatch{WarehouseId: "wh-a"},
			h3Distance: 0,
		},
	}

	got := resolveWarehouseByH3Distance(context.Background(), "supplier-1", "872830828ffffff", candidates, next)
	if got == nil {
		t.Fatal("expected warehouse match")
	}
	if got.WarehouseId != "wh-a" {
		t.Fatalf("warehouse id = %q, want %q", got.WarehouseId, "wh-a")
	}
}
