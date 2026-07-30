package credit

import (
	"testing"
)

func TestCalculateRiskScore(t *testing.T) {
	cases := []struct {
		name        string
		paymentRate float64
		claimRate   float64
		wantScore   int64
		wantTier    RiskTier
		wantLimit   int64
	}{
		{
			name:        "Excellent",
			paymentRate: 0.95,
			claimRate:   0.05,
			wantScore:   100,
			wantTier:    RiskTierLow,
			wantLimit:   1000000,
		},
		{
			name:        "High claims",
			paymentRate: 0.95,
			claimRate:   0.15,
			wantScore:   70,
			wantTier:    RiskTierMedium,
			wantLimit:   500000,
		},
		{
			name:        "Low payments",
			paymentRate: 0.70,
			claimRate:   0.05,
			wantScore:   80,
			wantTier:    RiskTierLow,
			wantLimit:   1000000,
		},
		{
			name:        "High claims and low payments",
			paymentRate: 0.70,
			claimRate:   0.15,
			wantScore:   50,
			wantTier:    RiskTierHigh,
			wantLimit:   0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score, tier, limit := calculateRiskScore(tc.paymentRate, tc.claimRate)
			if score != tc.wantScore {
				t.Errorf("expected score %d, got %d", tc.wantScore, score)
			}
			if tier != tc.wantTier {
				t.Errorf("expected tier %s, got %s", tc.wantTier, tier)
			}
			if limit != tc.wantLimit {
				t.Errorf("expected limit %d, got %d", tc.wantLimit, limit)
			}
		})
	}
}
