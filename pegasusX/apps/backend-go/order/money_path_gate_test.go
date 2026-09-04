package order

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

// Phase-0 money-path gate: Spanner-emulator proofs that the ledger never
// asserts money that did not move.
//
//	(a) capture failure never writes CAPTURED
//	(b) duplicate idempotency key never double-records (DB-enforced)
//	(d) shop-closed credit debt is always recorded (balance + CREDIT leg)
//
// (c) lives in payment/money_path_gate_test.go (empty GP creds = hard error).

type stubPaymentCapturer struct {
	ref string
	err error
}

func (s stubPaymentCapturer) CaptureCardPayment(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return s.ref, s.err
}

func newMoneyPathGateService(client *spanner.Client, capturer PaymentCapturer) *Service {
	// Lag the service clock: the emulator container clock can trail the host,
	// and Spanner rejects client timestamps in the future.
	now := time.Now().UTC().Add(-2 * time.Minute)
	creditSvc := credit.NewService(credit.NewSpannerRepository(client))
	creditSvc.SetNow(func() time.Time { return now })
	seq := 0
	return &Service{
		spannerClient:   client,
		paymentCapturer: capturer,
		credit:          creditSvc,
		now:             func() time.Time { return now },
		newID: func() string {
			seq++
			return fmt.Sprintf("gate-id-%d-%d", now.UnixNano(), seq)
		},
		log: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}

func insertFiscalizingCardOrder(t *testing.T, ctx context.Context, client *spanner.Client, orderID string, amountMinor int64) {
	t.Helper()
	now := time.Now().UTC().Add(-2 * time.Minute)
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":      orderID,
			"RetailerId":   "ret-gate",
			"SupplierId":   "sup-gate",
			"Status":             string(StatusFiscalizing),
			"OrderSource":        string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson":      []byte("[]"),
			"FiscalStatus":       string(FiscalStatusPending),
			"TotalMinor":   amountMinor,
			"Currency":     "UZS",
			"Version":      int64(1),
			"CreatedAt":    now,
			"UpdatedAt":    now,
		}),
		spanner.InsertMap("OrderPaymentLegs", map[string]any{
			"OrderId":        orderID,
			"LegId":          "leg-card-1",
			"Method":         string(MethodCard),
			"AmountMinor":    amountMinor,
			"Status":         string(PaymentStatusPending),
			"IdempotencyKey": "card-capture-" + orderID,
			"CreatedAt":      now,
		}),
	})
	if err != nil {
		t.Fatalf("insert fiscalizing card order: %v", err)
	}
}

func gateOrderRecord(orderID string, amountMinor int64) Order {
	return Order{
		OrderID:    orderID,
		SupplierID: "sup-gate",
		RetailerID: "ret-gate",
		Currency:   "UZS",
		TotalMinor: amountMinor,
		Status:     StatusFiscalizing,
		FiscalStatus: FiscalStatusPending,
		PendingFiscalReceipts: []FiscalReceiptRow{{
			OrderID:     orderID,
			AttemptID:   "att-" + orderID,
			SupplierID:  "sup-gate",
			RetailerID:  "ret-gate",
			Status:      FiscalAttemptPending,
			AmountMinor: amountMinor,
			Currency:    "UZS",
		}},
	}
}

func readLegStatus(t *testing.T, ctx context.Context, client *spanner.Client, orderID, legID string) (string, spanner.NullString) {
	t.Helper()
	row, err := client.Single().ReadRow(ctx, "OrderPaymentLegs", spanner.Key{orderID, legID}, []string{"Status", "ProviderRef"})
	if err != nil {
		t.Fatalf("read leg: %v", err)
	}
	var status string
	var ref spanner.NullString
	if err := row.Columns(&status, &ref); err != nil {
		t.Fatalf("parse leg: %v", err)
	}
	return status, ref
}

func countOutboxEventsContaining(t *testing.T, ctx context.Context, client *spanner.Client, orderID, needle string) int {
	t.Helper()
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Payload FROM OutboxEvents WHERE AggregateId = @oid`,
		Params: map[string]any{"oid": orderID},
	})
	defer iter.Stop()
	count := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query outbox: %v", err)
		}
		var payload []byte
		if err := row.Columns(&payload); err != nil {
			t.Fatalf("parse outbox payload: %v", err)
		}
		if bytes.Contains(payload, []byte(needle)) {
			count++
		}
	}
	return count
}

func TestMoneyPathGate_CaptureFailureNeverWritesCaptured(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_gate_fail_%d", time.Now().UnixNano())
	insertFiscalizingCardOrder(t, ctx, client, orderID, 10000)

	s := newMoneyPathGateService(client, stubPaymentCapturer{err: errors.New("pspsim: capture declined")})
	err := s.settleOutstandingCardPayment(ctx, gateOrderRecord(orderID, 10000))
	if err == nil {
		t.Fatal("settleOutstandingCardPayment must fail when the provider capture fails")
	}

	status, ref := readLegStatus(t, ctx, client, orderID, "leg-card-1")
	if status == string(PaymentStatusCaptured) {
		t.Fatal("capture failure must never write CAPTURED")
	}
	if status != string(PaymentStatusFailed) {
		t.Fatalf("leg status = %s, want FAILED", status)
	}
	if ref.Valid {
		t.Fatalf("failed capture must not record a provider ref, got %q", ref.StringVal)
	}

	ex := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OrderSettlementExceptions WHERE OrderId = @oid AND Type = 'CARD_CAPTURE_FAILED' AND Status = 'OPEN'`,
		Params: map[string]any{"oid": orderID},
	})
	defer ex.Stop()
	row, err := ex.Next()
	if err != nil {
		t.Fatalf("read exceptions: %v", err)
	}
	var n int64
	if err := row.Columns(&n); err != nil {
		t.Fatalf("parse exceptions: %v", err)
	}
	if n != 1 {
		t.Fatalf("CARD_CAPTURE_FAILED exceptions = %d, want 1", n)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "PAYMENT_CLEARED"); got != 0 {
		t.Fatalf("PAYMENT_CLEARED emitted %d times for a failed capture, want 0", got)
	}
}

func TestMoneyPathGate_CaptureSuccessConfirmsLegThenFiscalEvents(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_gate_ok_%d", time.Now().UnixNano())
	insertFiscalizingCardOrder(t, ctx, client, orderID, 10000)

	s := newMoneyPathGateService(client, stubPaymentCapturer{ref: "gp-ref-gate-1"})
	if err := s.settleOutstandingCardPayment(ctx, gateOrderRecord(orderID, 10000)); err != nil {
		t.Fatalf("settleOutstandingCardPayment: %v", err)
	}

	status, ref := readLegStatus(t, ctx, client, orderID, "leg-card-1")
	if status != string(PaymentStatusCaptured) {
		t.Fatalf("leg status = %s, want CAPTURED", status)
	}
	if !ref.Valid || ref.StringVal != "gp-ref-gate-1" {
		t.Fatalf("provider ref = %+v, want gp-ref-gate-1", ref)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "PAYMENT_CLEARED"); got != 1 {
		t.Fatalf("PAYMENT_CLEARED events = %d, want 1", got)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "FISCAL_RECEIPT_REQUESTED"); got != 1 {
		t.Fatalf("FISCAL_RECEIPT_REQUESTED events = %d, want 1", got)
	}
}

func TestMoneyPathGate_DuplicateIdempotencyKeyNeverDoubleRecords(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_gate_idem_%d", time.Now().UnixNano())
	now := time.Now().UTC().Add(-2 * time.Minute)
	key := "card-capture-" + orderID

	// OrderPaymentLegs is interleaved in Orders — the parent row must exist.
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"RetailerId":         "ret-gate",
			"SupplierId":         "sup-gate",
			"Status":             string(StatusFiscalizing),
			"OrderSource":        string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(5000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
	})
	if err != nil {
		t.Fatalf("insert parent order: %v", err)
	}

	s := newMoneyPathGateService(client, nil)
	first := PaymentLeg{
		OrderID: orderID, LegID: "leg-idem-1", Method: MethodCard,
		AmountMinor: 5000, Status: PaymentStatusCaptured,
		IdempotencyKey: key, CreatedAt: now, CapturedAt: spanner.NullTime{Time: now, Valid: true},
	}
	if err := s.recordPaymentLegStandalone(ctx, first); err != nil {
		t.Fatalf("first leg insert: %v", err)
	}
	dup := first
	dup.LegID = "leg-idem-2"
	if err := s.recordPaymentLegStandalone(ctx, dup); err == nil {
		t.Fatal("duplicate IdempotencyKey must be rejected by the unique index")
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT LegId FROM OrderPaymentLegs WHERE IdempotencyKey = @key`,
		Params: map[string]any{"key": key},
	})
	defer iter.Stop()
	legs := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query legs: %v", err)
		}
		legs++
	}
	if legs != 1 {
		t.Fatalf("legs with idempotency key = %d, want exactly 1", legs)
	}
}

func TestMoneyPathGate_ShopClosedCreditDebtAlwaysRecorded(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_gate_credit_%d", suffix)
	retailerID := fmt.Sprintf("ret-gate-credit-%d", suffix)
	supplierID := fmt.Sprintf("sup-gate-credit-%d", suffix)
	now := time.Now().UTC()
	past := now.Add(-10 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "Status", "OrderSource", "ConfirmationStatus", "LineItemsJson", "TotalMinor", "Currency", "Version", "ShopClosedGraceEndsAt", "CreatedAt", "UpdatedAt"},
			[]any{orderID, retailerID, supplierID, string(StatusShopClosedPending), string(OrderSourceManual), string(ConfirmationStatusConfirmed), []byte("[]"), int64(10000), "UZS", int64(1), past, past, past},
		),
		spanner.Insert("RetailerCreditProfiles",
			[]string{"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor", "AvailableCreditMinor", "Status", "RiskScore", "DelinquencyCount", "Version", "CreatedAt", "UpdatedAt"},
			[]any{retailerID, supplierID, int64(100000), int64(0), int64(100000), string(credit.StatusActive), int64(800), int64(0), int64(1), past, past},
		),
	})
	if err != nil {
		t.Fatalf("insert shop-closed fixture: %v", err)
	}

	t.Setenv("AR_INVOICES_ENABLED", "1")
	s := newMoneyPathGateService(client, nil)
	arRepo := ar.NewSpannerRepository(client)
	s.ar = ar.NewService(arRepo)
	if err := s.processShopClosedTimeouts(ctx); err != nil {
		t.Fatalf("processShopClosedTimeouts: %v", err)
	}

	row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status"})
	if err != nil {
		t.Fatalf("read order: %v", err)
	}
	var status string
	if err := row.Columns(&status); err != nil {
		t.Fatalf("parse order: %v", err)
	}
	if status != string(StatusDeliveredOnCredit) {
		t.Fatalf("order status = %s, want DELIVERED_ON_CREDIT", status)
	}

	prof, err := client.Single().ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{"CurrentBalanceMinor"})
	if err != nil {
		t.Fatalf("read credit profile: %v", err)
	}
	var balance int64
	if err := prof.Columns(&balance); err != nil {
		t.Fatalf("parse credit profile: %v", err)
	}
	if balance != 10000 {
		t.Fatalf("credit balance = %d, want 10000 (debt recorded)", balance)
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderPaymentLegs
		      WHERE OrderId = @oid AND Method = @m AND Status = @st`,
		Params: map[string]any{"oid": orderID, "m": string(MethodCredit), "st": string(PaymentStatusCaptured)},
	})
	defer iter.Stop()
	row2, err := iter.Next()
	if err == iterator.Done {
		t.Fatal("no CAPTURED CREDIT payment leg recorded for shop-closed credit leave")
	}
	if err != nil {
		t.Fatalf("query credit leg: %v", err)
	}
	var legAmount int64
	if err := row2.Columns(&legAmount); err != nil {
		t.Fatalf("parse credit leg: %v", err)
	}
	if legAmount != 10000 {
		t.Fatalf("credit leg amount = %d, want 10000", legAmount)
	}

	// Credit leave must open the AR open item (collectible revenue).
	inv, found, err := arRepo.GetByOrder(ctx, orderID)
	if err != nil {
		t.Fatalf("get AR invoice: %v", err)
	}
	if !found {
		t.Fatal("no AR invoice opened for shop-closed credit leave")
	}
	if inv.BalanceMinor != 10000 || inv.Status != ar.StatusOpen {
		t.Fatalf("AR invoice balance/status = %d/%s, want 10000/OPEN", inv.BalanceMinor, inv.Status)
	}
}

// Rule: credit leave-behind is rejected when AR invoicing is off — the debt
// would be uncollectible. The worker must fail closed and leave the order
// SHOP_CLOSED_PENDING for the next tick instead of booking invisible debt.
func TestMoneyPathGate_ShopClosedCreditLeaveRejectedWhenAROff(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	t.Setenv("AR_INVOICES_ENABLED", "")

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_gate_aroff_%d", suffix)
	retailerID := fmt.Sprintf("ret-gate-aroff-%d", suffix)
	supplierID := fmt.Sprintf("sup-gate-aroff-%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)
	past := now.Add(-10 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "Status", "OrderSource", "ConfirmationStatus", "LineItemsJson", "TotalMinor", "Currency", "Version", "ShopClosedGraceEndsAt", "CreatedAt", "UpdatedAt"},
			[]any{orderID, retailerID, supplierID, string(StatusShopClosedPending), string(OrderSourceManual), string(ConfirmationStatusConfirmed), []byte("[]"), int64(10000), "UZS", int64(1), past, past, past},
		),
		spanner.Insert("RetailerCreditProfiles",
			[]string{"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor", "AvailableCreditMinor", "Status", "RiskScore", "DelinquencyCount", "Version", "CreatedAt", "UpdatedAt"},
			[]any{retailerID, supplierID, int64(100000), int64(0), int64(100000), string(credit.StatusActive), int64(800), int64(0), int64(1), past, past},
		),
	})
	if err != nil {
		t.Fatalf("insert fixture: %v", err)
	}

	s := newMoneyPathGateService(client, nil)
	s.ar = ar.NewService(ar.NewSpannerRepository(client))
	if err := s.resolveOneShopClosedTimeout(ctx, orderID); err == nil {
		t.Fatal("credit leave with AR off must fail closed")
	}

	row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status"})
	if err != nil {
		t.Fatalf("read order: %v", err)
	}
	var status string
	if err := row.Columns(&status); err != nil {
		t.Fatalf("parse order: %v", err)
	}
	if status != string(StatusShopClosedPending) {
		t.Fatalf("order status = %s, want SHOP_CLOSED_PENDING (no silent credit leave)", status)
	}
	prof, err := client.Single().ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{"CurrentBalanceMinor"})
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	var balance int64
	if err := prof.Columns(&balance); err != nil {
		t.Fatalf("parse profile: %v", err)
	}
	if balance != 0 {
		t.Fatalf("credit balance = %d, want 0 (no debt booked while AR off)", balance)
	}
}
