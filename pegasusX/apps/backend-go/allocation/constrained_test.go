package allocation

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
)

func TestWarehouseCandidatesPicksHighestScore(t *testing.T) {
	active := map[string]bool{"W1": true, "W2": true}
	inventory := map[string][]stockInfo{
		"SKU1": {
			{WarehouseId: "W1", Available: 100},
			{WarehouseId: "W2", Available: 100},
		},
		"SKU2": {
			{WarehouseId: "W1", Available: 50},
			{WarehouseId: "W2", Available: 50},
		},
	}
	qty := map[string]int64{"SKU1": 10, "SKU2": 5}
	lineContexts := map[string]segment.LineAllocationContext{
		"SKU1": {PriorityScore: 80, RetailerSegment: segment.SegmentA, SkuClass: segment.VelocityA},
		"SKU2": {PriorityScore: 20, RetailerSegment: segment.SegmentA, SkuClass: segment.VelocityB},
	}

	candidates := warehouseCandidates(active, inventory, qty, lineContexts)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	for _, c := range candidates {
		if c.score != 100 {
			t.Fatalf("candidate %s score: got %d want 100", c.warehouseID, c.score)
		}
		if c.slack != (100-10)+(50-5) {
			t.Fatalf("candidate %s slack: got %d want %d", c.warehouseID, c.slack, (100-10)+(50-5))
		}
	}
}

func TestWarehouseCandidatesSkipsInsufficientStock(t *testing.T) {
	active := map[string]bool{"W1": true, "W2": true}
	inventory := map[string][]stockInfo{
		"SKU1": {
			{WarehouseId: "W1", Available: 5},
			{WarehouseId: "W2", Available: 100},
		},
	}
	qty := map[string]int64{"SKU1": 10}
	lineContexts := map[string]segment.LineAllocationContext{
		"SKU1": {PriorityScore: 50, Policy: segment.DefaultPolicy("S1", segment.SegmentB, segment.VelocityB), RiskTier: credit.RiskTierMedium},
	}

	candidates := warehouseCandidates(active, inventory, qty, lineContexts)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].warehouseID != "W2" {
		t.Fatalf("expected W2, got %s", candidates[0].warehouseID)
	}
}

func TestWarehouseCandidatesEmptyWhenNoWarehouseFulfillsAll(t *testing.T) {
	active := map[string]bool{"W1": true}
	inventory := map[string][]stockInfo{
		"SKU1": {{WarehouseId: "W1", Available: 100}},
		"SKU2": {{WarehouseId: "W1", Available: 1}},
	}
	qty := map[string]int64{"SKU1": 10, "SKU2": 5}
	lineContexts := map[string]segment.LineAllocationContext{
		"SKU1": {PriorityScore: 10},
		"SKU2": {PriorityScore: 10},
	}

	if candidates := warehouseCandidates(active, inventory, qty, lineContexts); len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %d", len(candidates))
	}
}
