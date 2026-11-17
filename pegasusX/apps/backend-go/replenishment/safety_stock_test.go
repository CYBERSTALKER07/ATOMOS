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
		{-1.0, 2.054},
		{0.0, 2.054},
		{0.40, 0.0},
		{0.50, 0.0},
		{0.55, 0.126},
		{0.60, 0.253},
		{0.65, 0.385},
		{0.70, 0.524},
		{0.75, 0.674},
		{0.80, 0.842},
		{0.85, 1.036},
		{0.90, 1.282},
		{0.95, 1.645},
		{0.98, 2.054},
		{0.99, 2.326},
		{0.995, 2.576},
	}
	for _, tc := range cases {
		got := NormalZ(tc.sl)
		if tc.want == 0.0 {
			if got != 0.0 {
				t.Fatalf("NormalZ(%v)=%v want 0.0", tc.sl, got)
			}
		} else {
			// Acklam continuous approximation matches 3-decimal common levels within 0.001
			if math.Abs(got-tc.want) > 0.001 {
				t.Fatalf("NormalZ(%v)=%v want ~%v (diff %v)", tc.sl, got, tc.want, math.Abs(got-tc.want))
			}
		}
	}
}

func TestNormalZMonotonicity(t *testing.T) {
	prevZ := -1.0
	for sl := 0.01; sl <= 0.999; sl += 0.01 {
		z := NormalZ(sl)
		if z < prevZ {
			t.Fatalf("NormalZ(%v)=%v is not monotonic (prev was %v)", sl, z, prevZ)
		}
		if sl < 0.85 && z > NormalZ(0.85) {
			t.Fatalf("NormalZ(%v)=%v exceeds NormalZ(0.85)=%v", sl, z, NormalZ(0.85))
		}
		prevZ = z
	}
}

func TestAcklamContinuousQuantiles(t *testing.T) {
	// Verify intermediate continuous quantiles like 0.979 produce ~2.0335
	z979 := NormalZ(0.979)
	if math.Abs(z979-2.03352) > 1e-4 {
		t.Fatalf("NormalZ(0.979)=%v want ~2.03352", z979)
	}

	// High quantile 0.999 -> ~3.0902
	z999 := NormalZ(0.999)
	if math.Abs(z999-3.09023) > 1e-4 {
		t.Fatalf("NormalZ(0.999)=%v want ~3.09023", z999)
	}

	// Median boundary
	if got := NormalZ(0.50); got != 0.0 {
		t.Fatalf("NormalZ(0.50)=%v want 0.0", got)
	}

	// Safe clamp for extreme service levels > 0.9999
	zExtreme := NormalZ(0.99999)
	zClamped := NormalZ(0.9999)
	if zExtreme != zClamped {
		t.Fatalf("NormalZ(0.99999)=%v want clamped to NormalZ(0.9999)=%v", zExtreme, zClamped)
	}
}

func TestSafetyStockUnitsFormula(t *testing.T) {
	// SS = NormalZ(0.98) * sqrt(2*4 + 10^2*1)
	in := SafetyStockInputs{
		DBar: 10, SigmaD: 2, L: 2, SigmaL: 1, ServiceLevel: 0.98,
	}
	wantZ := NormalZ(0.98)
	want := wantZ * math.Sqrt(2*4+100*1)
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
	wantZ := NormalZ(0.98)
	if math.Abs(res.ZAlpha-wantZ) > 1e-9 {
		t.Fatalf("z=%v want %v", res.ZAlpha, wantZ)
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
