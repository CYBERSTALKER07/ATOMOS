package replenishment

import (
	"math"
	"testing"
)

func TestNormalZCommonLevels(t *testing.T) {
	cases := []struct {
		sl   float64
		want float64
	}{
		{0.90, 1.282},
		{0.95, 1.645},
		{0.98, 2.054},
		{0.99, 2.326},
	}
	for _, tc := range cases {
		if got := NormalZ(tc.sl); got != tc.want {
			t.Fatalf("NormalZ(%v)=%v want %v", tc.sl, got, tc.want)
		}
	}
}

func TestSafetyStockUnitsFormula(t *testing.T) {
	// SS = 2.054 * sqrt(2*4 + 10^2*1) = 2.054 * sqrt(8+100) = 2.054*sqrt(108)
	in := SafetyStockInputs{
		DBar: 10, SigmaD: 2, L: 2, SigmaL: 1, ServiceLevel: 0.98,
	}
	want := 2.054 * math.Sqrt(2*4+100*1)
	got := SafetyStockUnits(in)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("SS=%v want %v", got, want)
	}
}

func TestComputeReorderPointIncludesSafety(t *testing.T) {
	in := SafetyStockInputs{
		DBar: 10, SigmaD: 2, L: 2, SigmaL: 1, ServiceLevel: 0.98,
	}
	res := ComputeReorderPoint(in)
	base := in.DBar * in.L
	if res.ReorderPoint <= base {
		t.Fatalf("ROP %v should exceed d̄·L %v when σ>0", res.ReorderPoint, base)
	}
	if math.Abs(res.ReorderPoint-(base+res.SafetyStock)) > 1e-9 {
		t.Fatalf("ROP != d̄·L + SS: %v vs %v", res.ReorderPoint, base+res.SafetyStock)
	}
	if res.ZAlpha != 2.054 {
		t.Fatalf("z=%v", res.ZAlpha)
	}
}

func TestLegacyReorderPointMatches115(t *testing.T) {
	got := LegacyReorderPoint(10, 2)
	want := 10*2 + 10*2*0.15
	if got != want {
		t.Fatalf("legacy=%v want %v", got, want)
	}
}

func TestComputeSuggestedQtySubtractsInTransit(t *testing.T) {
	stock := skuStock{
		CurrentStock:   5,
		InTransitQty:   10,
		UnfulfilledQty: 0,
	}
	// ROP 30 → need 30 - (5+10) = 15
	got := computeSuggestedQty(stock, 30)
	if got != 15 {
		t.Fatalf("suggested=%d want 15", got)
	}
}
