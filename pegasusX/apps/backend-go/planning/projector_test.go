package planning

import (
	"testing"
)

func TestProjectSnapshot_stockouts(t *testing.T) {
	snap := NetworkSnapshot{
		SupplierID:     "S1",
		Inventory:      map[string]int64{"SKU1": 10},
		OpenDemand:     map[string]int64{"SKU1": 20, "SKU2": 5},
		UnitValueMinor: map[string]int64{"SKU1": 2500},
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
	// SKU1 shortfall 10 * 2500; SKU2 shortfall 5 * fallback 10000
	want := int64(10*2500 + 5*10000)
	if out.RevenueAtRiskMinor != want {
		t.Fatalf("RaR want %d got %d", want, out.RevenueAtRiskMinor)
	}
	if out.UnitValueSource != "mixed" {
		t.Fatalf("unit_value_source want mixed got %s", out.UnitValueSource)
	}
}

func TestProjectSnapshot_pricedOnly(t *testing.T) {
	snap := NetworkSnapshot{
		SupplierID:     "S1",
		Inventory:      map[string]int64{"A": 0},
		OpenDemand:     map[string]int64{"A": 3},
		UnitValueMinor: map[string]int64{"A": 1500},
	}
	out := ProjectSnapshot(snap, ScenarioInput{})
	if out.RevenueAtRiskMinor != 4500 {
		t.Fatalf("want 4500 got %d", out.RevenueAtRiskMinor)
	}
	if out.UnitValueSource != "products" {
		t.Fatalf("want products got %s", out.UnitValueSource)
	}
}

func TestProjectSnapshot_fallbackOnly(t *testing.T) {
	t.Setenv("SCENARIO_UNIT_VALUE_MINOR", "2000")
	snap := NetworkSnapshot{
		SupplierID: "S1",
		Inventory:  map[string]int64{"A": 0},
		OpenDemand: map[string]int64{"A": 4},
	}
	out := ProjectSnapshot(snap, ScenarioInput{})
	if out.RevenueAtRiskMinor != 8000 {
		t.Fatalf("want 8000 got %d", out.RevenueAtRiskMinor)
	}
	if out.UnitValueSource != "fallback" {
		t.Fatalf("want fallback got %s", out.UnitValueSource)
	}
}

func TestHeuristicRevenueAtRisk(t *testing.T) {
	rar, src := heuristicRevenueAtRisk(
		map[string]int64{"A": 100},
		[]string{"A", "B"},
		map[string]int64{"A": 5},
	)
	if rar != 5*100+1*10000 {
		t.Fatalf("rar=%d", rar)
	}
	if src != "mixed" {
		t.Fatalf("src=%s", src)
	}
}
