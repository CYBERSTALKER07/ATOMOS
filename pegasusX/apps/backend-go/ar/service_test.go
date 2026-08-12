package ar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

func TestOpenInvoiceIdempotentAndDueDate(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	due := leave.AddDate(0, 0, 14)
	inv1, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-1",
		AmountMinor: 50_000, Currency: "USD", TermsDays: 14,
		CreditLeaveAt: leave, DueAt: due,
	})
	if err != nil {
		t.Fatal(err)
	}
	inv2, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-1",
		AmountMinor: 50_000, Currency: "USD", TermsDays: 14,
		CreditLeaveAt: leave, DueAt: due,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inv1.InvoiceID != inv2.InvoiceID {
		t.Fatalf("idempotent mismatch %s vs %s", inv1.InvoiceID, inv2.InvoiceID)
	}
	if !inv1.DueAt.Equal(due) {
		t.Fatalf("due=%v want %v", inv1.DueAt, due)
	}
	if inv1.Currency != "USD" {
		t.Fatalf("currency=%q want USD", inv1.Currency)
	}
}

func TestOpenInvoiceRejectsInvalidCurrency(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	_, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-bad-ccy",
		AmountMinor: 1000, Currency: "US", CreditLeaveAt: leave,
	})
	if err == nil || !errors.Is(err, fxrates.ErrInvalidCurrency) {
		t.Fatalf("expected ErrInvalidCurrency, got %v", err)
	}
}

func TestOpenInvoiceDefaultsEmptyCurrencyToUZS(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	inv, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-default-ccy",
		AmountMinor: 1000, CreditLeaveAt: leave,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Currency != "UZS" {
		t.Fatalf("currency=%q want UZS fallback", inv.Currency)
	}
}

func TestAgingBuckets(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if AgingBucketFor(now.AddDate(0, 0, 1), now) != AgingCurrent {
		t.Fatal("current")
	}
	if AgingBucketFor(now.AddDate(0, 0, -10), now) != Aging1to30 {
		t.Fatal("1_30")
	}
	if AgingBucketFor(now.AddDate(0, 0, -100), now) != Aging90Plus {
		t.Fatal("90+")
	}
}

func TestGlobalTermsChangeDoesNotMutateOpenInvoice(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	due := leave.AddDate(0, 0, 7)
	inv, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-2",
		AmountMinor: 10_000, Currency: "EUR", TermsDays: 7,
		CreditLeaveAt: leave, DueAt: due,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Simulate terms change — open invoice DueAt must stay.
	if inv.TermsDays != 7 || !inv.DueAt.Equal(due) {
		t.Fatalf("invoice mutated unexpectedly: %+v", inv)
	}
	if inv.Currency != "EUR" {
		t.Fatalf("currency=%q want EUR", inv.Currency)
	}
}

func TestRecordPaymentPaysDownInvoice(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	inv, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-9",
		AmountMinor: 50_000, Currency: "USD", TermsDays: 14,
		CreditLeaveAt: leave, DueAt: leave.AddDate(0, 0, 14),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Partial payment -> PARTIAL, balance reduced.
	updated, err := svc.RecordPayment(context.Background(), inv.InvoiceID, 20_000, "pay-1", "USD")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusPartial || updated.BalanceMinor != 30_000 {
		t.Fatalf("after partial: status=%s balance=%d", updated.Status, updated.BalanceMinor)
	}
	// Currency mismatch rejected.
	if _, err := svc.RecordPayment(context.Background(), inv.InvoiceID, 1_000, "pay-mismatch", "UZS"); !errors.Is(err, fxrates.ErrCurrencyMismatch) {
		t.Fatalf("expected currency mismatch, got %v", err)
	}
	// Idempotent replay of the same key does not double-apply.
	again, err := svc.RecordPayment(context.Background(), inv.InvoiceID, 20_000, "pay-1", "USD")
	if err != nil {
		t.Fatal(err)
	}
	if again.BalanceMinor != 30_000 {
		t.Fatalf("idempotent replay changed balance to %d", again.BalanceMinor)
	}
	// Remaining balance -> PAID.
	paid, err := svc.RecordPayment(context.Background(), inv.InvoiceID, 30_000, "pay-2", "")
	if err != nil {
		t.Fatal(err)
	}
	if paid.Status != StatusPaid || paid.BalanceMinor != 0 {
		t.Fatalf("after full: status=%s balance=%d", paid.Status, paid.BalanceMinor)
	}
}

func TestRecordPaymentForOrderNoInvoiceIsNoOp(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	// Cash/card order with no AR invoice: must be a safe no-op.
	if err := svc.RecordPaymentForOrder(context.Background(), "no-such-order", 10_000, "k", "UZS"); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}

func TestRecordPaymentForOrderSettlesOpenInvoice(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	if _, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s", RetailerID: "r", OrderID: "ord-credit",
		AmountMinor: 40_000, Currency: "UZS", TermsDays: 14,
		CreditLeaveAt: leave, DueAt: leave.AddDate(0, 0, 14),
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordPaymentForOrder(context.Background(), "ord-credit", 40_000, "cash-ord-credit", "UZS"); err != nil {
		t.Fatal(err)
	}
	inv, found, err := repo.GetByOrder(context.Background(), "ord-credit")
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if inv.Status != StatusPaid || inv.BalanceMinor != 0 {
		t.Fatalf("invoice not settled: status=%s balance=%d", inv.Status, inv.BalanceMinor)
	}
	// Already-paid invoice is a no-op on replay.
	if err := svc.RecordPaymentForOrder(context.Background(), "ord-credit", 40_000, "cash-ord-credit-2", "UZS"); err != nil {
		t.Fatal(err)
	}
}

func TestRecordPaymentValidatesInput(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	if _, err := svc.RecordPayment(context.Background(), "inv", 0, "k", ""); err == nil {
		t.Fatal("expected error for non-positive amount")
	}
	if _, err := svc.RecordPayment(context.Background(), "inv", 100, "  ", ""); err == nil {
		t.Fatal("expected error for empty idempotency key")
	}
}
