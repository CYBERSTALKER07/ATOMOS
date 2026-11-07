package credit

import (
	"context"
	"testing"
)

// G1-A2: ClearBalanceInTxn fails closed via sequential path for memory/test repos.

func TestClearBalanceInTxn_MemoryFallback(t *testing.T) {
	repo := &testCreditRepo{}
	svc := newTestService(repo)

	if err := svc.ClearBalanceInTxn(context.Background(), nil, "ret-1", "sup-1", "ord-1", 25_000); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if repo.adjustDelta != -25_000 {
		t.Fatalf("delta = %d, want -25000", repo.adjustDelta)
	}
}

func TestClearBalanceInTxn_ZeroAmountNoOp(t *testing.T) {
	repo := &testCreditRepo{}
	svc := newTestService(repo)
	if err := svc.ClearBalanceInTxn(context.Background(), nil, "ret-1", "sup-1", "ord-1", 0); err != nil {
		t.Fatalf("zero amount: %v", err)
	}
	if repo.adjustDelta != 0 {
		t.Fatalf("expected no adjust, got %d", repo.adjustDelta)
	}
}

func TestClearBalanceInTxn_NilService(t *testing.T) {
	var svc *Service
	err := svc.ClearBalanceInTxn(context.Background(), nil, "r", "s", "o", 100)
	if err == nil {
		t.Fatal("expected error for nil service")
	}
}
