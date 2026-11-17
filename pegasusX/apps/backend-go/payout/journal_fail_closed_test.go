package payout

import (
	"errors"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
)

func TestApplySettlementJournalInTxn_FailsClosedWhenUnconfigured(t *testing.T) {
	orig := SettlementJournal
	t.Cleanup(func() { SettlementJournal = orig })
	SettlementJournal = nil

	err := applySettlementJournalInTxn(nil, "batch-1", StatusPaid, "sup-1", "UZS", 100000, 0, 5000)
	if err == nil || !strings.Contains(err.Error(), "settlement journal hook required") {
		t.Fatalf("expected fail-closed error when SettlementJournal is nil, got %v", err)
	}
}

func TestApplySettlementJournalInTxn_ZeroGrossDoesNotRequireHook(t *testing.T) {
	orig := SettlementJournal
	t.Cleanup(func() { SettlementJournal = orig })
	SettlementJournal = nil

	if err := applySettlementJournalInTxn(nil, "batch-1", StatusPaid, "sup-1", "UZS", 1000, 1000, 0); err != nil {
		t.Fatalf("expected nil when net gross <= 0, got %v", err)
	}
	if err := applySettlementJournalInTxn(nil, "batch-2", StatusSubmitted, "sup-1", "UZS", 1000, 0, 0); err != nil {
		t.Fatalf("expected nil for non-paid status, got %v", err)
	}
}

func TestApplySettlementJournalInTxn_PropagatesHookErrorAndSuccess(t *testing.T) {
	orig := SettlementJournal
	t.Cleanup(func() { SettlementJournal = orig })

	called := false
	SettlementJournal = func(_ *spanner.ReadWriteTransaction, batchID, supplierID, currency string, grossMinor, feeMinor int64) error {
		called = true
		if batchID != "batch-3" || supplierID != "sup-1" || currency != "UZS" || grossMinor != 9000 || feeMinor != 500 {
			t.Fatalf("unexpected args: %s %s %s %d %d", batchID, supplierID, currency, grossMinor, feeMinor)
		}
		return nil
	}
	if err := applySettlementJournalInTxn(nil, "batch-3", StatusPaid, "sup-1", "UZS", 10000, 1000, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected SettlementJournal hook to be invoked")
	}

	sentinel := errors.New("journal write failed")
	SettlementJournal = func(_ *spanner.ReadWriteTransaction, _, _, _ string, _, _ int64) error {
		return sentinel
	}
	if err := applySettlementJournalInTxn(nil, "batch-3", StatusPaid, "sup-1", "UZS", 10000, 0, 500); !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
