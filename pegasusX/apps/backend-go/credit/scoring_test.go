package credit

import (
	"testing"
	"time"
)

func TestComputeRiskScore_healthy(t *testing.T) {
	p := Profile{
		RetailerID: "r1", SupplierID: "s1",
		CreditLimitMinor: 1_000_000, CurrentBalanceMinor: 100_000,
		Status: StatusActive, DelinquencyCount: 0,
	}
	sc := ComputeRiskScore(p, ScoreMetrics{MaxDaysPastDue: 0, PaymentsLast90d: 3, ExpectedPayments90: 3}, time.Now().UTC())
	if sc.Score < 70 {
		t.Fatalf("healthy score too low: %d factors=%s", sc.Score, sc.FactorsJSON)
	}
	if sc.RiskTier != RiskTierLow && sc.RiskTier != RiskTierMedium {
		t.Fatalf("tier=%s", sc.RiskTier)
	}
}

func TestComputeRiskScore_blacklisted(t *testing.T) {
	p := Profile{RetailerID: "r1", Status: StatusBlacklisted, CreditLimitMinor: 1_000_000}
	sc := ComputeRiskScore(p, ScoreMetrics{}, time.Now().UTC())
	if sc.Score != 0 || sc.RiskTier != RiskTierBlock {
		t.Fatalf("score=%d tier=%s", sc.Score, sc.RiskTier)
	}
}

func TestComputeRiskScore_highUtilDelinq(t *testing.T) {
	p := Profile{
		RetailerID: "r1", CreditLimitMinor: 100_000, CurrentBalanceMinor: 95_000,
		DelinquencyCount: 3, Status: StatusActive,
	}
	sc := ComputeRiskScore(p, ScoreMetrics{MaxDaysPastDue: 25}, time.Now().UTC())
	if sc.Score >= 50 {
		t.Fatalf("expected stressed score, got %d", sc.Score)
	}
}
