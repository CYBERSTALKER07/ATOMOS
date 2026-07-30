package order

import "testing"

func TestDecideShopClosedTimeout_creditLeaveLowRisk(t *testing.T) {
	d := DecideShopClosedTimeout(ShopClosedTimeoutInput{
		RiskTier:             ShopClosedRiskLow,
		ProfileStatus:        "ACTIVE",
		AvailableCreditMinor: 500_000,
		OrderTotalMinor:      100_000,
		CreditAllowed:        true,
	})
	if d.Resolution != TimeoutCreditLeave {
		t.Fatalf("got %s want CREDIT_LEAVE (%s)", d.Resolution, d.Reason)
	}
}

func TestDecideShopClosedTimeout_highRiskReturns(t *testing.T) {
	d := DecideShopClosedTimeout(ShopClosedTimeoutInput{
		RiskTier:             ShopClosedRiskHigh,
		ProfileStatus:        "ACTIVE",
		AvailableCreditMinor: 1_000_000,
		OrderTotalMinor:      50_000,
		CreditAllowed:        true,
	})
	if d.Resolution != TimeoutReturnToWarehouse {
		t.Fatalf("got %s want RETURN_TO_WAREHOUSE", d.Resolution)
	}
}

func TestDecideShopClosedTimeout_frozenBlocksCredit(t *testing.T) {
	d := DecideShopClosedTimeout(ShopClosedTimeoutInput{
		RiskTier:             ShopClosedRiskLow,
		ProfileStatus:        "FROZEN",
		AvailableCreditMinor: 1_000_000,
		OrderTotalMinor:      10_000,
		CreditAllowed:        false,
	})
	if d.Resolution != TimeoutReturnToWarehouse {
		t.Fatalf("got %s want RETURN_TO_WAREHOUSE for frozen", d.Resolution)
	}
}

func TestDecideShopClosedTimeout_forceBypassWhenEnabled(t *testing.T) {
	d := DecideShopClosedTimeout(ShopClosedTimeoutInput{
		RiskTier:             ShopClosedRiskMedium,
		ProfileStatus:        "ACTIVE",
		AvailableCreditMinor: 0,
		OrderTotalMinor:      200_000,
		CreditAllowed:        false,
		ForceBypassEnabled:   true,
	})
	if d.Resolution != TimeoutForceBypass {
		t.Fatalf("got %s want FORCE_BYPASS", d.Resolution)
	}
}

func TestDecideShopClosedTimeout_noProfileLowValueReschedule(t *testing.T) {
	d := DecideShopClosedTimeout(ShopClosedTimeoutInput{
		RiskTier:               ShopClosedRiskMedium,
		ProfileStatus:          "",
		OrderTotalMinor:        5_000,
		LowValueThresholdMinor: 10_000,
	})
	if d.Resolution != TimeoutReschedule {
		t.Fatalf("got %s want RESCHEDULE", d.Resolution)
	}
}

func TestDecideShopClosedTimeout_insufficientCredit(t *testing.T) {
	d := DecideShopClosedTimeout(ShopClosedTimeoutInput{
		RiskTier:             ShopClosedRiskLow,
		ProfileStatus:        "ACTIVE",
		AvailableCreditMinor: 1_000,
		OrderTotalMinor:      50_000,
		CreditAllowed:        false,
	})
	if d.Resolution != TimeoutReturnToWarehouse {
		t.Fatalf("got %s want RETURN_TO_WAREHOUSE", d.Resolution)
	}
}

func TestNormalizeShopClosedReason(t *testing.T) {
	if got := NormalizeShopClosedReason(""); got != ShopClosedReasonClosed {
		t.Fatalf("empty → %s", got)
	}
	if got := NormalizeShopClosedReason("no_answer"); got != ShopClosedReasonNoAnswer {
		t.Fatalf("no_answer → %s", got)
	}
	if got := NormalizeShopClosedReason("weird"); got != ShopClosedReasonOther {
		t.Fatalf("weird → %s", got)
	}
}
