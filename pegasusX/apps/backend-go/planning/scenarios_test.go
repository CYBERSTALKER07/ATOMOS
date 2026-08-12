package planning

import "testing"

func TestScenarioCompareDeltas(t *testing.T) {
	left := ScenarioResult{
		SLARiskPct:         10,
		FleetVolume:        100,
		RevenueAtRiskMinor: 5000,
		StockoutSKUs:       []string{"a"},
		CapacityBreach:     false,
	}
	right := ScenarioResult{
		SLARiskPct:         25,
		FleetVolume:        80,
		RevenueAtRiskMinor: 9000,
		StockoutSKUs:       []string{"a", "b", "c"},
		CapacityBreach:     true,
	}
	got := ScenarioCompareResult{
		Left:  left,
		Right: right,
		Deltas: ScenarioCompareDeltas{
			SLARiskPctDelta:         right.SLARiskPct - left.SLARiskPct,
			FleetVolumeDelta:        right.FleetVolume - left.FleetVolume,
			RevenueAtRiskMinorDelta: right.RevenueAtRiskMinor - left.RevenueAtRiskMinor,
			StockoutCountDelta:      len(right.StockoutSKUs) - len(left.StockoutSKUs),
			CapacityBreachChanged:   right.CapacityBreach != left.CapacityBreach,
		},
	}
	if got.Deltas.SLARiskPctDelta != 15 {
		t.Fatalf("sla delta %v", got.Deltas.SLARiskPctDelta)
	}
	if got.Deltas.FleetVolumeDelta != -20 {
		t.Fatalf("fleet delta %v", got.Deltas.FleetVolumeDelta)
	}
	if got.Deltas.RevenueAtRiskMinorDelta != 4000 {
		t.Fatalf("rar delta %v", got.Deltas.RevenueAtRiskMinorDelta)
	}
	if got.Deltas.StockoutCountDelta != 2 {
		t.Fatalf("stockout delta %v", got.Deltas.StockoutCountDelta)
	}
	if !got.Deltas.CapacityBreachChanged {
		t.Fatal("capacity change expected")
	}
}
