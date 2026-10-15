package credit

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestCheckOrder_RiskTierBlocked(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 10000,
			Status:              StatusActive,
			RiskTier:            RiskTierBlock,
		},
	}
	svc := newTestService(repo)

	res, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 15000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatal("expected order to be blocked for RiskTierBlock")
	}
	if res.Reason != "risk_tier_blocked" {
		t.Fatalf("expected reason risk_tier_blocked, got %s", res.Reason)
	}
	if res.Shortfall != 15000 {
		t.Fatalf("expected shortfall 15000, got %d", res.Shortfall)
	}
}

func TestCheckOrder_DelinquentAccountHold(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 10000,
			Status:              StatusActive,
			DelinquencyCount:    2, // default tolerance is 2
		},
	}
	svc := newTestService(repo)

	res, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 15000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatal("expected order to be blocked for delinquent account hold")
	}
	if res.Reason != "delinquent_account_hold" {
		t.Fatalf("expected reason delinquent_account_hold, got %s", res.Reason)
	}
}

func TestCheckOrder_RiskScoreCritical(t *testing.T) {
	t.Setenv("CREDIT_SCORE_AUTO_HOLD_ENABLED", "true")
	t.Setenv("CREDIT_SCORE_AUTO_HOLD_MAX", "25")

	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 10000,
			Status:              StatusActive,
			RiskScore:           20,
		},
	}
	svc := newTestService(repo)

	res, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 15000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatal("expected order to be blocked for critical risk score")
	}
	if res.Reason != "risk_score_critical" {
		t.Fatalf("expected reason risk_score_critical, got %s", res.Reason)
	}
}

func TestBumpDelinquency_AutoFreezesAtTolerance(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 10000,
			Status:              StatusActive,
			DelinquencyCount:    1,
		},
	}
	svc := newTestService(repo)

	// Bumping delinquency from 1 -> 2 reaches default tolerance (2) and must freeze profile
	err := svc.BumpDelinquency(context.Background(), "ret-1", "sup-1")
	if err != nil {
		t.Fatalf("BumpDelinquency failed: %v", err)
	}

	if repo.stored == nil {
		t.Fatal("expected stored profile to be updated")
	}
	if repo.stored.DelinquencyCount != 2 {
		t.Fatalf("expected delinquency count 2, got %d", repo.stored.DelinquencyCount)
	}
	if repo.stored.Status != StatusFrozen {
		t.Fatalf("expected status to be auto-frozen, got %s", repo.stored.Status)
	}
}

func TestFreezeAndUnfreezeProfile(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 10000,
			Status:              StatusActive,
		},
	}
	svc := newTestService(repo)

	// Freeze
	if err := svc.FreezeProfile(context.Background(), "ret-1", "sup-1", "dunning_escalation"); err != nil {
		t.Fatalf("FreezeProfile failed: %v", err)
	}
	if repo.stored.Status != StatusFrozen {
		t.Fatalf("expected StatusFrozen, got %s", repo.stored.Status)
	}

	// Update local state to frozen for unfreeze test
	repo.profile = *repo.stored

	// Unfreeze
	if err := svc.UnfreezeProfile(context.Background(), "ret-1", "sup-1", "repaid_in_full"); err != nil {
		t.Fatalf("UnfreezeProfile failed: %v", err)
	}
	if repo.stored.Status != StatusActive {
		t.Fatalf("expected StatusActive, got %s", repo.stored.Status)
	}
}

func TestOverdueCreditSweeper_FreezesDelinquent(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-delinq",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 50000,
			DelinquencyCount:    2,
			Status:              StatusActive,
		},
	}
	svc := newTestService(repo)

	frozen, err := svc.RunOverdueCreditSweeperOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOverdueCreditSweeperOnce failed: %v", err)
	}
	if frozen != 1 {
		t.Fatalf("expected 1 frozen profile, got %d", frozen)
	}
	if repo.stored == nil || repo.stored.Status != StatusFrozen {
		t.Fatalf("expected stored profile to be frozen, got %+v", repo.stored)
	}
}

func TestOverdueCreditSweeper_ConfigOverrides(t *testing.T) {
	os.Setenv("CREDIT_OVERDUE_SWEEPER_INTERVAL_SECONDS", "30")
	os.Setenv("CREDIT_OVERDUE_BATCH_SIZE", "50")
	defer func() {
		os.Unsetenv("CREDIT_OVERDUE_SWEEPER_INTERVAL_SECONDS")
		os.Unsetenv("CREDIT_OVERDUE_BATCH_SIZE")
	}()

	interval := overdueSweeperInterval()
	if interval != 30*time.Second {
		t.Fatalf("expected 30s interval, got %v", interval)
	}

	batch := overdueBatchSize()
	if batch != 50 {
		t.Fatalf("expected batch size 50, got %d", batch)
	}
}
