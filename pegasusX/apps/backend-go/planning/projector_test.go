package planning

import (
	"testing"
)

func TestProjectSnapshot_stockouts(t *testing.T) {
	snap := NetworkSnapshot{
		SupplierID: "S1",
		Inventory:  map[string]int64{"SKU1": 10},
		OpenDemand: map[string]int64{"SKU1": 20, "SKU2": 5},
		WarehouseCount: 2,
		OpenOrderCount: 3,
	}
	out := ProjectSnapshot(snap, ScenarioInput{DemandDeltaPct: 0})
	if len(out.StockoutSKUs) == 0 {
		t.Fatal("expected stockout SKUs")
	}
	if out.Mode != "twin_snapshot" {
		t.Fatalf("mode: %s", out.Mode)
	}
}
