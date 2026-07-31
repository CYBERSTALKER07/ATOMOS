package segment

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

func TestBootstrapSegment(t *testing.T) {
	cases := []struct {
		name string
		in   RetailerBootstrapInput
		want string
	}{
		{
			name: "high risk to C",
			in: RetailerBootstrapInput{RiskTier: credit.RiskTierHigh, OrderCount: 100},
			want: SegmentC,
		},
		{
			name: "claims spike to C",
			in:   RetailerBootstrapInput{ClaimCount: 3, OrderCount: 2, RiskTier: credit.RiskTierLow},
			want: SegmentC,
		},
		{
			name: "top volume low risk to A",
			in: RetailerBootstrapInput{
				InTopVolume: true, CreditScore: 75, RiskTier: credit.RiskTierLow, OrderCount: 5,
			},
			want: SegmentA,
		},
		{
			name: "volume threshold to A",
			in:   RetailerBootstrapInput{RiskTier: credit.RiskTierLow, OrderCount: 25},
			want: SegmentA,
		},
		{
			name: "medium risk to B",
			in:   RetailerBootstrapInput{RiskTier: credit.RiskTierMedium, OrderCount: 10},
			want: SegmentB,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := bootstrapSegment(tc.in); got != tc.want {
				t.Fatalf("bootstrapSegment() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBootstrapSkuClass(t *testing.T) {
	fast := bootstrapSkuClass(SkuBootstrapInput{
		Sku:          "FAST",
		OrderQty:     500,
		VelocityRank: 0.9,
		PriceMinor:   1000,
		MedianPrice:  1000,
	})
	if fast.VelocityClass != VelocityA {
		t.Fatalf("fast velocity: got %s want A", fast.VelocityClass)
	}

	slow := bootstrapSkuClass(SkuBootstrapInput{
		Sku:          "SLOW",
		OrderQty:     1,
		VelocityRank: 0.1,
		PriceMinor:   1000,
		MedianPrice:  1000,
	})
	if slow.VelocityClass != VelocityC {
		t.Fatalf("slow velocity: got %s want C", slow.VelocityClass)
	}

	strategic := bootstrapSkuClass(SkuBootstrapInput{
		Sku:           "PREM",
		OrderQty:      10,
		VelocityRank:  0.5,
		HandlingClass: "PREMIUM",
		MedianPrice:   1000,
		PriceMinor:    500,
	})
	if !strategic.StrategicFlag {
		t.Fatal("expected strategic flag for PREMIUM handling")
	}
}

func TestTopVolumePercentile(t *testing.T) {
	stats := []RetailerOrderStats{
		{RetailerID: "R1", OrderCount: 100},
		{RetailerID: "R2", OrderCount: 80},
		{RetailerID: "R3", OrderCount: 60},
		{RetailerID: "R4", OrderCount: 40},
		{RetailerID: "R5", OrderCount: 20},
	}
	top := topVolumePercentile(stats, 0.20)
	if !top["R1"] {
		t.Fatal("expected R1 in top volume")
	}
	if len(top) != 1 {
		t.Fatalf("expected 1 top retailer, got %d", len(top))
	}
}

func TestIsManualSegment(t *testing.T) {
	if !isManualSegment(manualReasonPrefix + "override") {
		t.Fatal("expected manual prefix to be detected")
	}
	if isManualSegment(bootstrapReason) {
		t.Fatal("bootstrap reason should not be manual")
	}
}
