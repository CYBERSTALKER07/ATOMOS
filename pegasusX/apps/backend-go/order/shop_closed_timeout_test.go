package order

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/stretchr/testify/assert"
)

func TestDecideShopClosedTimeout_CreditLeaveLowRisk(t *testing.T) {
	order := &Order{TotalMinor: 100_000}
	profile := &credit.Profile{
		Status:               "ACTIVE",
		AvailableCreditMinor: 500_000,
		RiskTier:             credit.RiskTierLow,
	}
	cfg := TimeoutConfig{
		MaxAutoCreditMinor:       200_000,
		MaxRiskTierForAutoCredit: 1, // Low is 1
	}

	d := DecideShopClosedTimeout(order, profile, cfg)
	assert.Equal(t, DecisionCreditLeave, d)
}

func TestDecideShopClosedTimeout_HighRiskReturns(t *testing.T) {
	order := &Order{TotalMinor: 50_000}
	profile := &credit.Profile{
		Status:               "ACTIVE",
		AvailableCreditMinor: 1_000_000,
		RiskTier:             credit.RiskTierHigh,
	}
	cfg := TimeoutConfig{
		MaxAutoCreditMinor:       200_000,
		MaxRiskTierForAutoCredit: 1,
	}

	d := DecideShopClosedTimeout(order, profile, cfg)
	assert.Equal(t, DecisionReturnToWarehouse, d)
}

func TestDecideShopClosedTimeout_FrozenBlocksCredit(t *testing.T) {
	order := &Order{TotalMinor: 10_000}
	profile := &credit.Profile{
		Status:               "FROZEN",
		AvailableCreditMinor: 1_000_000,
		RiskTier:             credit.RiskTierLow,
	}
	cfg := TimeoutConfig{
		MaxAutoCreditMinor:       200_000,
		MaxRiskTierForAutoCredit: 1,
	}

	d := DecideShopClosedTimeout(order, profile, cfg)
	assert.Equal(t, DecisionReturnToWarehouse, d, "frozen should return to warehouse")
}

func TestDecideShopClosedTimeout_ForceBypassWhenEnabled(t *testing.T) {
	order := &Order{TotalMinor: 200_000}
	profile := &credit.Profile{
		Status:               "ACTIVE",
		AvailableCreditMinor: 0,
		RiskTier:             credit.RiskTierMedium,
	}
	cfg := TimeoutConfig{
		MaxAutoCreditMinor:       100_000,
		MaxRiskTierForAutoCredit: 1,
		AllowForceBypass:         true,
	}

	d := DecideShopClosedTimeout(order, profile, cfg)
	assert.Equal(t, DecisionForceBypass, d)
}

func TestDecideShopClosedTimeout_NoProfileLowValueReturns(t *testing.T) {
	order := &Order{TotalMinor: 10_000}
	cfg := TimeoutConfig{
		MaxAutoCreditMinor:       100_000,
		MaxRiskTierForAutoCredit: 1,
	}

	d := DecideShopClosedTimeout(order, nil, cfg)
	assert.Equal(t, DecisionReturnToWarehouse, d, "no profile defaults to return to warehouse")
}

func TestDecideShopClosedTimeout_InsufficientCredit(t *testing.T) {
	order := &Order{TotalMinor: 200_000}
	profile := &credit.Profile{
		Status:               "ACTIVE",
		AvailableCreditMinor: 50_000, // less than order total
		RiskTier:             credit.RiskTierLow,
	}
	cfg := TimeoutConfig{
		MaxAutoCreditMinor:       500_000,
		MaxRiskTierForAutoCredit: 1,
	}

	d := DecideShopClosedTimeout(order, profile, cfg)
	assert.Equal(t, DecisionReturnToWarehouse, d)
}
