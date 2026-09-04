package order

import (
	"testing"
	"time"
)

func TestValidateSplitTenderPlan(t *testing.T) {
	// Valid match
	plan := SplitTenderPlan{
		WalletMinor: 20_000,
		CreditMinor: 30_000,
		CardMinor:   40_000,
		CashMinor:   10_000,
	}
	if err := ValidateSplitTenderPlan(100_000, plan); err != nil {
		t.Fatalf("expected valid plan, got %v", err)
	}

	// Mismatch
	if err := ValidateSplitTenderPlan(90_000, plan); err == nil {
		t.Fatal("expected error on total mismatch")
	}

	// Negative leg
	badPlan := SplitTenderPlan{
		WalletMinor: -10_000,
		CreditMinor: 110_000,
	}
	if err := ValidateSplitTenderPlan(100_000, badPlan); err == nil {
		t.Fatal("expected error on negative leg")
	}
}

func TestBuildSplitTenderLegs(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	plan := SplitTenderPlan{
		WalletMinor:   25_000,
		CreditMinor:   25_000,
		CardMinor:     25_000,
		CashMinor:     25_000,
		ProviderToken: "tok_psp_123",
	}

	legs := BuildSplitTenderLegs("ord-split-1", plan, now)
	if len(legs) != 4 {
		t.Fatalf("expected 4 legs, got %d", len(legs))
	}

	for _, leg := range legs {
		if leg.OrderID != "ord-split-1" {
			t.Errorf("orderID mismatch %s", leg.OrderID)
		}
		switch leg.Method {
		case MethodWallet:
			if leg.AmountMinor != 25_000 || leg.Status != PaymentStatusCaptured {
				t.Errorf("wallet leg invalid %+v", leg)
			}
		case MethodCredit:
			if leg.AmountMinor != 25_000 || leg.Status != PaymentStatusAuthorized {
				t.Errorf("credit leg invalid %+v", leg)
			}
		case MethodCard:
			if leg.AmountMinor != 25_000 || leg.Status != PaymentStatusAuthorized || !leg.ProviderRef.Valid || leg.ProviderRef.StringVal != "tok_psp_123" {
				t.Errorf("card leg invalid %+v", leg)
			}
		case MethodCash:
			if leg.AmountMinor != 25_000 || leg.Status != PaymentStatusPending {
				t.Errorf("cash leg invalid %+v", leg)
			}
		}
	}
}
