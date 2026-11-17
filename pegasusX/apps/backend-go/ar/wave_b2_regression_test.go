package ar

import (
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func TestRecordPayment_OverpaymentSplitsAdvanceBalance(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	client := openARSpannerTestClient(t)
	defer client.Close()

	ctx := t.Context()
	sid := "sup_ar_overpay_" + uuid.NewString()[:10]
	rid := "ret_ar_overpay_" + uuid.NewString()[:10]
	cleanupARGLTestState(t, client, sid)

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	svc := NewService(NewSpannerRepository(client))
	svc.SetNow(func() time.Time { return now })

	inv, err := svc.OpenFromCreditLeave(ctx, OpenFromCreditLeaveRequest{
		SupplierID: sid, RetailerID: rid, OrderID: "ord_overpay_" + uuid.NewString()[:8],
		AmountMinor: 50000, Currency: "UZS", CreditLeaveAt: now.Add(-24 * time.Hour), TermsDays: 14,
	})
	if err != nil {
		t.Fatalf("OpenFromCreditLeave: %v", err)
	}

	// Pay 80,000 against 50,000 invoice -> 50,000 pays down invoice, 30,000 moves to ArAdvanceBalances.
	paidInv, err := svc.RecordPayment(ctx, inv.InvoiceID, 80000, "overpay-key-"+inv.InvoiceID, "UZS")
	if err != nil {
		t.Fatalf("RecordPayment overpayment: %v", err)
	}
	if paidInv.Status != StatusPaid || paidInv.BalanceMinor != 0 {
		t.Fatalf("invoice status=%s balance=%d, want PAID / 0", paidInv.Status, paidInv.BalanceMinor)
	}

	row, err := client.Single().ReadRow(ctx, "ArAdvanceBalances", spanner.Key{rid, sid}, []string{"BalanceMinor", "Currency"})
	if err != nil {
		t.Fatalf("read ArAdvanceBalances: %v", err)
	}
	var advBal int64
	var ccy string
	if err := row.Columns(&advBal, &ccy); err != nil {
		t.Fatalf("parse ArAdvanceBalances: %v", err)
	}
	if advBal != 30000 || ccy != "UZS" {
		t.Fatalf("advance balance=%d currency=%s, want 30000 UZS", advBal, ccy)
	}
}
