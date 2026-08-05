package replenishment

import (
	"math"
	"testing"
)

func TestRetailerSafetyStockLegacyVsV2(t *testing.T) {
	demand := 10.0
	legacy := demand * 0.15
	if legacy != 1.5 {
		t.Fatalf("legacy=%v", legacy)
	}

	ss := retailerSafetyStock(demand, 0, 2, 1, 0.98, true, true)
	if ss <= 0 {
		t.Fatalf("v2 SS should be >0, got %v", ss)
	}
	// Assumed σ_d = max(2.5, 1)=2.5 → SS = 2.054*sqrt(2*6.25 + 100*1) > legacy 1.5
	want := SafetyStockUnits(SafetyStockInputs{
		DBar: demand, SigmaD: math.Max(demand*0.25, 1), L: 2, SigmaL: 1, ServiceLevel: 0.98,
	})
	if math.Abs(ss-want) > 1e-9 {
		t.Fatalf("SS=%v want %v", ss, want)
	}
	if ss <= legacy {
		t.Fatalf("expected v2 SS %v > legacy %v when σ assumed", ss, legacy)
	}
}

func TestRetailerSafetyStockFeedsSuggestedQty(t *testing.T) {
	demand := 10.0
	lead := 3.0
	ss := retailerSafetyStock(demand, 4, lead, 1, 0.98, false, true)
	qty := ComputeSuggestedQty(demand, lead, ss, 5, 0)
	// target = 10*3 + ss - 5
	want := int64(math.Ceil(demand*lead + ss - 5))
	if qty != want {
		t.Fatalf("suggested=%d want %d (ss=%v)", qty, want, ss)
	}
}

func TestRetailerSafetyStockZeroDemandAssumedFloor(t *testing.T) {
	ss := retailerSafetyStock(0, 0, 2, 1, 0.98, true, true)
	// d̄=0, σ=1 → SS = 2.054*sqrt(2*1 + 0) > 0
	if ss <= 0 {
		t.Fatalf("SS=%v", ss)
	}
}
