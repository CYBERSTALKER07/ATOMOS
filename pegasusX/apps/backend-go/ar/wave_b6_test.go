package ar

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestOpenFromCreditLeave_DisabledFailsClosed(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "false")
	svc := NewService(NewMemoryRepository())
	_, err := svc.OpenFromCreditLeave(context.Background(), OpenFromCreditLeaveRequest{
		SupplierID: "s1", RetailerID: "r1", OrderID: "o1", AmountMinor: 1000, Currency: "UZS",
		CreditLeaveAt: time.Now().UTC(),
	})
	if err != ErrInvoicesDisabled {
		t.Fatalf("err=%v want ErrInvoicesDisabled", err)
	}
}

func TestOpenFromCreditLeave_IdempotentPerOrder(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	req := OpenFromCreditLeaveRequest{
		SupplierID: "s1", RetailerID: "r1", OrderID: "ord-b6", AmountMinor: 5000, Currency: "UZS",
		CreditLeaveAt: time.Now().UTC(), TermsDays: 14,
	}
	inv1, err := svc.OpenFromCreditLeave(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	inv2, err := svc.OpenFromCreditLeave(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if inv1.InvoiceID != inv2.InvoiceID {
		t.Fatalf("not idempotent: %s vs %s", inv1.InvoiceID, inv2.InvoiceID)
	}
}

func TestEventARInvoiceAgingUpdatedConstant(t *testing.T) {
	if events.EventARInvoiceAgingUpdated != "AR_INVOICE_AGING_UPDATED" {
		t.Fatalf("got %q", events.EventARInvoiceAgingUpdated)
	}
}
