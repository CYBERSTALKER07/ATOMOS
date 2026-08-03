package segment

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

func TestNormalizeRetailerSegment(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{SegmentA, SegmentA},
		{"STRATEGIC", SegmentA},
		{SegmentB, SegmentB},
		{"STANDARD", SegmentB},
		{SegmentC, SegmentC},
		{"OPPORTUNISTIC", SegmentC},
		{"", SegmentC},
		{"unknown", SegmentC},
	}
	for _, tc := range tests {
		if got := NormalizeRetailerSegment(tc.in); got != tc.out {
			t.Errorf("NormalizeRetailerSegment(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

func TestNormalizeVelocityClass(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{VelocityA, VelocityA},
		{VelocityB, VelocityB},
		{VelocityC, VelocityC},
		{"", VelocityB},
		{"unknown", VelocityB},
	}
	for _, tc := range tests {
		if got := NormalizeVelocityClass(tc.in); got != tc.out {
			t.Errorf("NormalizeVelocityClass(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

func TestComputePriorityScore(t *testing.T) {
	policy := ServicePolicy{
		PriorityWeight:  100,
		CreditRiskBoost: 20,
	}
	if got := ComputePriorityScore(policy, credit.RiskTierLow, false); got != 120 {
		t.Fatalf("low tier boost: got %d want 120", got)
	}
	if got := ComputePriorityScore(policy, credit.RiskTierMedium, false); got != 110 {
		t.Fatalf("medium tier boost: got %d want 110", got)
	}
	if got := ComputePriorityScore(policy, credit.RiskTierHigh, false); got != 100 {
		t.Fatalf("high tier boost: got %d want 100", got)
	}
	if got := ComputePriorityScore(policy, credit.RiskTierLow, true); got != 130 {
		t.Fatalf("strategic flag: got %d want 130", got)
	}
	zeroBoost := ServicePolicy{PriorityWeight: 50, CreditRiskBoost: 0}
	if got := ComputePriorityScore(zeroBoost, credit.RiskTierLow, false); got != 50 {
		t.Fatalf("zero boost: got %d want 50", got)
	}
}

func TestDefaultPolicyWeights(t *testing.T) {
	a := DefaultPolicy("S1", SegmentA, VelocityA)
	if a.PriorityWeight != 105 {
		t.Fatalf("segment A + velocity A weight: got %d want 105", a.PriorityWeight)
	}
	c := DefaultPolicy("S1", SegmentC, VelocityC)
	if c.PriorityWeight != 30 {
		t.Fatalf("segment C weight: got %d want 30", c.PriorityWeight)
	}
}
