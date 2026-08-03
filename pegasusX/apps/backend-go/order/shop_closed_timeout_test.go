package order

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

func TestDecideShopClosedTimeout_CreditLeaveWhenActiveAndAvailable(t *testing.T) {
	order := &Order{TotalMinor: 1000}
	profile := &credit.Profile{
		Status:               credit.StatusActive,
		CreditLimitMinor:     10000,
		CurrentBalanceMinor:  0,
		AvailableCreditMinor: 10000,
	}
	cfg := TimeoutConfig{MaxAutoCreditMinor: 50000}
	if d := DecideShopClosedTimeout(order, profile, cfg); d != DecisionCreditLeave {
		t.Fatalf("want CREDIT_LEAVE, got %s", d)
	}
}

func TestDecideShopClosedTimeout_FrozenBlocksCredit(t *testing.T) {
	order := &Order{TotalMinor: 1000}
	profile := &credit.Profile{
		Status:               credit.StatusFrozen,
		CreditLimitMinor:     10000,
		AvailableCreditMinor: 10000,
	}
	cfg := TimeoutConfig{MaxAutoCreditMinor: 50000}
	if d := DecideShopClosedTimeout(order, profile, cfg); d != DecisionReturnToWarehouse {
		t.Fatalf("want RETURN_TO_WAREHOUSE, got %s", d)
	}
}

func TestDecideShopClosedTimeout_ForceBypassWhenEnabled(t *testing.T) {
	order := &Order{TotalMinor: 1000}
	profile := &credit.Profile{Status: credit.StatusFrozen}
	cfg := TimeoutConfig{AllowForceBypass: true}
	if d := DecideShopClosedTimeout(order, profile, cfg); d != DecisionForceBypass {
		t.Fatalf("want FORCE_BYPASS, got %s", d)
	}
}

func TestDecideShopClosedTimeout_NoProfileReturns(t *testing.T) {
	order := &Order{TotalMinor: 1000}
	cfg := TimeoutConfig{}
	if d := DecideShopClosedTimeout(order, nil, cfg); d != DecisionReturnToWarehouse {
		t.Fatalf("want RETURN_TO_WAREHOUSE, got %s", d)
	}
}

func TestDecideShopClosedTimeout_InsufficientCredit(t *testing.T) {
	order := &Order{TotalMinor: 5000}
	profile := &credit.Profile{
		Status:              credit.StatusActive,
		CreditLimitMinor:    1000,
		CurrentBalanceMinor: 0,
	}
	cfg := TimeoutConfig{MaxAutoCreditMinor: 50000}
	if d := DecideShopClosedTimeout(order, profile, cfg); d != DecisionReturnToWarehouse {
		t.Fatalf("want RETURN_TO_WAREHOUSE, got %s", d)
	}
}

func TestCanLeaveOnCredit_IgnoresRiskTier(t *testing.T) {
	order := &Order{TotalMinor: 1000}
	profile := &credit.Profile{
		Status:           credit.StatusActive,
		CreditLimitMinor: 10000,
		RiskTier:         credit.RiskTierBlock, // must not block after scoring removal
	}
	if err := CanLeaveOnCredit(order, profile, TimeoutConfig{}); err != nil {
		t.Fatalf("RiskTier must not gate credit leave: %v", err)
	}
}
