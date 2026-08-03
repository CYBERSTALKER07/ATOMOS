package credit

import (
	"testing"
)

func TestCalculateRiskScore(t *testing.T) {
	cases := []struct {
		name        string
		inputs      RiskInputs
		wantScore   int64
		wantTier    RiskTier
		wantLimit   int64
	}{
		{
			name: "Excellent",
			inputs: RiskInputs{
				PaymentRate:    0.95,
				ClaimRate:      0.05,
				VelocityScore:  0.8,
				UtilisationBps: 2000,
				AccountAgeDays: 90,
			},
			wantScore: 100,
			wantTier:  RiskTierLow,
			wantLimit: 1000000,
		},
		{
			name: "High claims",
			inputs: RiskInputs{
				PaymentRate:    0.95,
				ClaimRate:      0.15,
				VelocityScore:  0.8,
				UtilisationBps: 2000,
				AccountAgeDays: 90,
			},
			wantScore: 70,
			wantTier:  RiskTierMedium,
			wantLimit: 500000,
		},
		{
			name: "Low payments",
			inputs: RiskInputs{
				PaymentRate:    0.70,
				ClaimRate:      0.05,
				VelocityScore:  0.8,
				UtilisationBps: 2000,
				AccountAgeDays: 90,
			},
			wantScore: 80,
			wantTier:  RiskTierLow,
			wantLimit: 1000000,
		},
		{
			name: "High claims and low payments",
			inputs: RiskInputs{
				PaymentRate:    0.70,
				ClaimRate:      0.15,
				VelocityScore:  0.8,
				UtilisationBps: 2000,
				AccountAgeDays: 90,
			},
			wantScore: 50,
			wantTier:  RiskTierHigh,
			wantLimit: 0,
		},
		{
			name: "High utilisation and young account",
			inputs: RiskInputs{
				PaymentRate:    0.95,
				ClaimRate:      0.02,
				VelocityScore:  0.1,
				UtilisationBps: 9000,
				AccountAgeDays: 10,
				ShopClosedRate: 0.1,
			},
			wantScore: 50,
			wantTier:  RiskTierHigh,
			wantLimit: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score, tier, limit := calculateRiskScore(tc.inputs)
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
