package order

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

type stubPaymentRefunder struct {
	ref string
	err error
}

func (s stubPaymentRefunder) RefundCardPayment(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return s.ref, s.err
}

func insertRefundableOrder(t *testing.T, ctx context.Context, client *spanner.Client, orderID string, legs []map[string]any) {
	t.Helper()
	now := time.Now().UTC().Add(-2 * time.Minute)
	muts := []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"RetailerId":         "ret-refund",
			"SupplierId":         "sup-refund",
			"Status":             string(StatusCompleted),
			"OrderSource":        string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(10000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
	}
	for _, leg := range legs {
		leg["OrderId"] = orderID
		muts = append(muts, spanner.InsertMap("OrderPaymentLegs", leg))
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("insert refundable order: %v", err)
	}
}

func capturedLeg(legID, method string, amount int64, idemKey string) map[string]any {
	now := time.Now().UTC().Add(-2 * time.Minute)
	return map[string]any{
		"LegId":          legID,
		"Method":         method,
		"AmountMinor":    amount,
		"Status":         string(PaymentStatusCaptured),
		"IdempotencyKey": idemKey,
		"CreatedAt":      now,
		"CapturedAt":     now,
	}
}

func readRefundRow(t *testing.T, ctx context.Context, client *spanner.Client, refundID string) (status string, ref spanner.NullString) {
	t.Helper()
	row, err := client.Single().ReadRow(ctx, "Refunds", spanner.Key{refundID}, []string{"Status", "ProviderRef"})
	if err != nil {
		t.Fatalf("read refund: %v", err)
	}
	if err := row.Columns(&status, &ref); err != nil {
		t.Fatalf("parse refund: %v", err)
	}
	return status, ref
}

func TestRefund_CardPartialSuccess(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_ok_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCard), 10000, "card-capture-"+orderID),
	})

	s := newMoneyPathGateService(client, nil)
	s.SetPaymentRefunder(stubPaymentRefunder{ref: "gp-refund-1"})
	s.SetFiscalProvider(PegasusReceiptProvider{})

	res, err := s.InitiateRefund(ctx, RefundRequest{
		OrderID: orderID, AmountMinor: 4000, ReasonCode: "CUSTOMER_RETURN", ActorID: "admin-1",
		IdempotencyKey: "rf-test-" + orderID,
	})
	if err != nil {
		t.Fatalf("InitiateRefund: %v", err)
	}
	if res.Status != RefundStatusCaptured || res.Method != RefundMethodCard {
		t.Fatalf("refund status/method = %s/%s, want CAPTURED/CARD", res.Status, res.Method)
	}
	if res.ProviderRef != "gp-refund-1" {
		t.Fatalf("provider ref = %q", res.ProviderRef)
	}
	status, ref := readRefundRow(t, ctx, client, res.RefundID)
	if status != RefundStatusCaptured || !ref.Valid || ref.StringVal != "gp-refund-1" {
		t.Fatalf("refund row = %s/%+v", status, ref)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "REFUND_SUCCEEDED"); got != 1 {
		t.Fatalf("REFUND_SUCCEEDED events = %d, want 1", got)
	}
	if res.CreditNoteID == "" {
		t.Fatal("credit note must be recorded for a confirmed refund")
	}
}

func TestRefund_ExceedsCapturedRejected(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_over_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCard), 5000, "card-capture-"+orderID),
	})

	s := newMoneyPathGateService(client, nil)
	s.SetPaymentRefunder(stubPaymentRefunder{ref: "unused"})
	_, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 9000, ActorID: "admin-1"})
	if !errors.Is(err, ErrRefundExceedsCaptured) {
		t.Fatalf("want ErrRefundExceedsCaptured, got %v", err)
	}
}

func TestRefund_CreditPortionRejected(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_credit_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCredit), 10000, "credit-leave-"+orderID),
	})

	s := newMoneyPathGateService(client, nil)
	_, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 5000, ActorID: "admin-1"})
	if !errors.Is(err, ErrRefundCreditPortion) {
		t.Fatalf("want ErrRefundCreditPortion, got %v", err)
	}
}

func TestRefund_IdempotencyReplayReturnsSameRefund(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_idem_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCard), 10000, "card-capture-"+orderID),
	})

	s := newMoneyPathGateService(client, nil)
	s.SetPaymentRefunder(stubPaymentRefunder{ref: "gp-refund-idem"})
	key := "rf-idem-" + orderID
	first, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 3000, ActorID: "a", IdempotencyKey: key})
	if err != nil {
		t.Fatalf("first refund: %v", err)
	}
	second, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 3000, ActorID: "a", IdempotencyKey: key})
	if err != nil {
		t.Fatalf("replay refund: %v", err)
	}
	if first.RefundID != second.RefundID {
		t.Fatalf("replay returned refund %s, want %s", second.RefundID, first.RefundID)
	}
}

func TestRefund_ProviderFailureNeverCaptured(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_fail_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCard), 10000, "card-capture-"+orderID),
	})

	s := newMoneyPathGateService(client, nil)
	s.SetPaymentRefunder(stubPaymentRefunder{err: errors.New("pspsim: refund declined")})
	res, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 10000, ActorID: "a"})
	if err == nil {
		t.Fatal("provider failure must surface")
	}
	if res.Status != RefundStatusFailed {
		t.Fatalf("refund status = %s, want FAILED", res.Status)
	}
	// The reversal leg must never assert captured money.
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Status FROM OrderPaymentLegs WHERE OrderId = @oid AND Method = @m`,
		Params: map[string]any{"oid": orderID, "m": string(MethodRefund)},
	})
	defer iter.Stop()
	row, qErr := iter.Next()
	if qErr != nil {
		t.Fatalf("refund leg missing: %v", qErr)
	}
	var st string
	if err := row.Columns(&st); err != nil {
		t.Fatalf("parse leg: %v", err)
	}
	if st != string(PaymentStatusFailed) {
		t.Fatalf("refund leg status = %s, want FAILED", st)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "REFUND_SUCCEEDED"); got != 0 {
		t.Fatalf("REFUND_SUCCEEDED emitted %d times for a failed refund, want 0", got)
	}
}

func TestRefund_CashLedgerOnly(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_cash_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCash), 7000, "cash-collect-"+orderID),
	})

	s := newMoneyPathGateService(client, nil)
	res, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 7000, ActorID: "a"})
	if err != nil {
		t.Fatalf("cash refund: %v", err)
	}
	if res.Method != RefundMethodCash || res.Status != RefundStatusCaptured {
		t.Fatalf("cash refund = %s/%s, want CASH/CAPTURED", res.Method, res.Status)
	}
	if res.ProviderRef != "cash-ledger-reversal" {
		t.Fatalf("cash provider ref = %q", res.ProviderRef)
	}
}

func TestRefund_FiscalCorrectiveChainLinked(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_rf_fiscal_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, orderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCard), 10000, "card-capture-"+orderID),
	})
	now := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("OrderFiscalReceipts", map[string]any{
			"OrderId":         orderID,
			"AttemptId":       "att-" + orderID,
			"SupplierId":      "sup-refund",
			"RetailerId":      "ret-refund",
			"Provider":        FiscalProviderPegasus,
			"Status":          FiscalAttemptSuccess,
			"FiscalReceiptId": "PX-RCPT-ORIG-1",
			"AmountMinor":     int64(10000),
			"Currency":        "UZS",
			"CreatedAt":       now,
			"UpdatedAt":       now,
		}),
	}); err != nil {
		t.Fatalf("insert fiscal receipt: %v", err)
	}

	s := newMoneyPathGateService(client, nil)
	s.SetPaymentRefunder(stubPaymentRefunder{ref: "gp-refund-f"})
	s.SetFiscalProvider(PegasusReceiptProvider{})
	res, err := s.InitiateRefund(ctx, RefundRequest{OrderID: orderID, AmountMinor: 2500, ActorID: "a"})
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if res.CorrectiveEhfID == "" {
		t.Fatal("corrective receipt must be issued when a provider supports it and an original exists")
	}
	// Credit note links original + corrective.
	row, err := client.Single().ReadRow(ctx, "CreditNotes", spanner.Key{res.CreditNoteID}, []string{"OriginalEhfId", "CorrectiveEhfId"})
	if err != nil {
		t.Fatalf("read credit note: %v", err)
	}
	var orig, corr spanner.NullString
	if err := row.Columns(&orig, &corr); err != nil {
		t.Fatalf("parse credit note: %v", err)
	}
	if !orig.Valid || orig.StringVal != "PX-RCPT-ORIG-1" {
		t.Fatalf("original ehf = %+v", orig)
	}
	if !corr.Valid || corr.StringVal != res.CorrectiveEhfID {
		t.Fatalf("corrective ehf = %+v, want %s", corr, res.CorrectiveEhfID)
	}
}
