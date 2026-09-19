package ar

import (
	"context"
	"testing"
	"time"
)

// G1.A: RecordPaymentForOrderInTxn — memory-repo path (nil txn → sequential apply).

func TestRecordPaymentForOrderInTxn_NoInvoiceNoOp(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	err := svc.RecordPaymentForOrderInTxn(context.Background(), nil, "ord-none", 10_000, "ar-cash-collect-ord-none", "UZS")
	if err != nil {
		t.Fatalf("no invoice should be no-op: %v", err)
	}
}

func TestRecordPaymentForOrderInTxn_PaysDown(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	svc.SetNow(func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) })

	_, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-credit",
		AmountMinor: 50_000, Currency: "UZS", CreditLeaveAt: svc.now(),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	err = svc.RecordPaymentForOrderInTxn(context.Background(), nil, "ord-credit", 50_000, "ar-cash-collect-ord-credit", "UZS")
	if err != nil {
		t.Fatalf("pay-down: %v", err)
	}
	inv, found, err := repo.GetByOrder(context.Background(), "ord-credit")
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if inv.BalanceMinor != 0 || inv.Status != StatusPaid {
		t.Fatalf("want PAID/0, got status=%s bal=%d", inv.Status, inv.BalanceMinor)
	}
}

func TestRecordPaymentForOrderInTxn_Idempotent(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)

	_, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-idem",
		AmountMinor: 40_000, Currency: "UZS", CreditLeaveAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	key := "ar-cash-collect-ord-idem"
	if err := svc.RecordPaymentForOrderInTxn(context.Background(), nil, "ord-idem", 40_000, key, "UZS"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := svc.RecordPaymentForOrderInTxn(context.Background(), nil, "ord-idem", 40_000, key, "UZS"); err != nil {
		t.Fatalf("second: %v", err)
	}
	inv, _, _ := repo.GetByOrder(context.Background(), "ord-idem")
	if inv.BalanceMinor != 0 {
		t.Fatalf("double apply corrupted balance: %d", inv.BalanceMinor)
	}
}

func TestRecordPaymentForOrderInTxn_RejectsBadInput(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	if err := svc.RecordPaymentForOrderInTxn(context.Background(), nil, "o", 0, "k", ""); err == nil {
		t.Fatal("expected amount error")
	}
	if err := svc.RecordPaymentForOrderInTxn(context.Background(), nil, "o", 100, "  ", ""); err == nil {
		t.Fatal("expected idempotency error")
	}
}
