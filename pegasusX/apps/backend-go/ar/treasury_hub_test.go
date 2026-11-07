package ar

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestSupplierAgingSummaryAndDelinquency(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)

	var idSeq int64
	svc.newID = func() string { return fmt.Sprintf("ari_test_%d", atomic.AddInt64(&idSeq, 1)) }

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	// Current invoice
	if _, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1",
		AmountMinor: 100_000, Currency: "UZS", TermsDays: 14,
		CreditLeaveAt: now.AddDate(0, 0, -5), DueAt: now.AddDate(0, 0, 9),
	}); err != nil {
		t.Fatalf("open ord-1: %v", err)
	}

	// Overdue 10 days (1_30)
	if _, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "sup-1", RetailerID: "ret-2", OrderID: "ord-2",
		AmountMinor: 200_000, Currency: "UZS", TermsDays: 14,
		CreditLeaveAt: now.AddDate(0, 0, -24), DueAt: now.AddDate(0, 0, -10),
	}); err != nil {
		t.Fatalf("open ord-2: %v", err)
	}

	// Overdue 40 days (31_60) -> Delinquent lock
	if _, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "sup-1", RetailerID: "ret-3", OrderID: "ord-3",
		AmountMinor: 300_000, Currency: "UZS", TermsDays: 14,
		CreditLeaveAt: now.AddDate(0, 0, -54), DueAt: now.AddDate(0, 0, -40),
	}); err != nil {
		t.Fatalf("open ord-3: %v", err)
	}

	summary, err := svc.GetSupplierAgingSummary(context.Background(), "sup-1")
	if err != nil {
		t.Fatal(err)
	}

	if summary.TotalInvoicesCount != 3 {
		t.Fatalf("invoices_count=%d want 3", summary.TotalInvoicesCount)
	}
	if summary.TotalOpenMinor != 600_000 {
		t.Fatalf("total_open=%d want 600000", summary.TotalOpenMinor)
	}
	if summary.BucketCurrentMinor != 100_000 {
		t.Fatalf("current=%d want 100000", summary.BucketCurrentMinor)
	}
	if summary.Bucket1To30Minor != 200_000 {
		t.Fatalf("1_30=%d want 200000", summary.Bucket1To30Minor)
	}
	if summary.Bucket31To60Minor != 300_000 {
		t.Fatalf("31_60=%d want 300000", summary.Bucket31To60Minor)
	}
	if summary.TotalOverdueMinor != 500_000 {
		t.Fatalf("overdue=%d want 500000", summary.TotalOverdueMinor)
	}
	if summary.DelinquentRetailersCount != 1 {
		t.Fatalf("delinquent_retailers=%d want 1", summary.DelinquentRetailersCount)
	}

	// Check delinquency lock on ret-3
	lockStatus, err := svc.CheckRetailerDelinquencyLock(context.Background(), "ret-3", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if !lockStatus.IsLocked {
		t.Fatal("expected ret-3 to be locked due to 40 days overdue invoice")
	}

	// Check ret-1 is NOT locked
	lock1, err := svc.CheckRetailerDelinquencyLock(context.Background(), "ret-1", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if lock1.IsLocked {
		t.Fatal("expected ret-1 to NOT be locked")
	}
}

func TestRetailerPayInvoice(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	inv, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-pay",
		AmountMinor: 100_000, Currency: "UZS", TermsDays: 14,
		CreditLeaveAt: now, DueAt: now.AddDate(0, 0, 14),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Partial pay 40_000
	updated, err := svc.RetailerPayInvoice(context.Background(), "ret-1", inv.InvoiceID, RetailerPayInvoiceRequest{
		AmountMinor:   40_000,
		PaymentMethod: "WALLET",
	}, "idem-1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.BalanceMinor != 60_000 {
		t.Fatalf("balance=%d want 60000", updated.BalanceMinor)
	}
	if updated.Status != StatusPartial {
		t.Fatalf("status=%s want PARTIAL", updated.Status)
	}

	// Pay remainder 60_000
	paid, err := svc.RetailerPayInvoice(context.Background(), "ret-1", inv.InvoiceID, RetailerPayInvoiceRequest{
		AmountMinor:   60_000,
		PaymentMethod: "WALLET",
	}, "idem-2")
	if err != nil {
		t.Fatal(err)
	}
	if paid.BalanceMinor != 0 {
		t.Fatalf("balance=%d want 0", paid.BalanceMinor)
	}
	if paid.Status != StatusPaid {
		t.Fatalf("status=%s want PAID", paid.Status)
	}
}
