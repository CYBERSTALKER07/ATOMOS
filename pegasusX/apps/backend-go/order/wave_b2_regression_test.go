package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

func TestCancelService_FlushesOutboxEventsToSpanner(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_cancel_flush_%d", suffix)
	supID := fmt.Sprintf("sup_cancel_flush_%d", suffix)
	retID := fmt.Sprintf("ret_cancel_flush_%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)

	repo := NewSpannerRepository(client)
	cur := &Order{
		OrderID:            orderID,
		SupplierID:         supID,
		RetailerID:         retID,
		Status:             StatusPending,
		Source:             OrderSourceManual,
		ConfirmationStatus: ConfirmationStatusConfirmed,
		TotalMinor:         12000,
		Currency:           "UZS",
		Version:            1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            cur.OrderID,
			"SupplierId":         cur.SupplierID,
			"RetailerId":         cur.RetailerID,
			"Status":             string(cur.Status),
			"OrderSource":        string(cur.Source),
			"ConfirmationStatus": string(cur.ConfirmationStatus),
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         cur.TotalMinor,
			"Currency":           cur.Currency,
			"Version":            cur.Version,
			"CreatedAt":          cur.CreatedAt,
			"UpdatedAt":          cur.UpdatedAt,
		}),
	}); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{spanner.Delete("Orders", spanner.Key{orderID})})
	})

	svc := NewService(ServiceConfig{
		Repo:          repo,
		SpannerClient: client,
		SupplierID:    supID,
		Currency:      "UZS",
		Log:           slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})
	resp, err := svc.cancelOrderWithReason(ctx, cur, retID, "RETAILER", "buyer_changed_mind", events.EventOrderStatusChanged)
	if err != nil {
		t.Fatalf("cancelOrderWithReason: %v", err)
	}
	if resp.Status != StatusCancelled {
		t.Fatalf("status=%s want CANCELLED", resp.Status)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "buyer_changed_mind"); got == 0 {
		t.Fatalf("expected flushed OutboxEvents row for order %s cancel reason, got 0", orderID)
	}
	if err := svc.enqueueCreditReleaseCompensation(ctx, cur, "credit_release_comp_test", errors.New("simulated release err")); err != nil {
		t.Fatalf("enqueueCreditReleaseCompensation: %v", err)
	}
	if got := countOutboxEventsContaining(t, ctx, client, orderID, "credit_release_comp_test"); got == 0 {
		t.Fatalf("expected flushed CreditReleaseCompensation OutboxEvents row for order %s, got 0", orderID)
	}
}

func TestApplyNegotiatedLineItems_PreservesDeliveryFee(t *testing.T) {
	t.Parallel()
	existing := []LineItem{
		{SKU: "SKU-1", Quantity: 2, UnitPrice: 1000},
		{SKU: "SKU-2", Quantity: 1, UnitPrice: 500},
	}
	proposed := []ProposedNegotiationItem{
		{SKUID: "SKU-1", OriginalQty: 2, ProposedQty: 1},
	}
	updated, total, err := applyNegotiatedLineItems(existing, proposed, 750)
	if err != nil {
		t.Fatalf("applyNegotiatedLineItems: %v", err)
	}
	if updated[0].Quantity != 1 {
		t.Fatalf("sku1 qty=%d want 1", updated[0].Quantity)
	}
	// 1*1000 + 1*500 + 750 delivery fee = 2250
	if total != 2250 {
		t.Fatalf("total=%d want 2250 (must preserve delivery fee)", total)
	}
}

func TestEarlyComplete_AggregatesDuplicateSKUsAcrossOrders(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supID := fmt.Sprintf("sup_ec_%d", suffix)
	whID := fmt.Sprintf("wh_ec_%d", suffix)
	drvID := fmt.Sprintf("drv_ec_%d", suffix)
	skuID := fmt.Sprintf("sku_ec_%d", suffix)
	ord1 := fmt.Sprintf("ord_ec1_%d", suffix)
	ord2 := fmt.Sprintf("ord_ec2_%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)

	items1, _ := json.Marshal([]LineItem{{SKU: skuID, Quantity: 3, UnitPrice: 1000}})
	items2, _ := json.Marshal([]LineItem{{SKU: skuID, Quantity: 7, UnitPrice: 1000}})

	muts := []*spanner.Mutation{
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supID,
			"WarehouseId":      whID,
			"ProductId":        skuID,
			"QuantityOnHand":   int64(100),
			"QuantityReserved": int64(20),
			"ReorderThreshold": int64(0),
			"UpdatedAt":        now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ord1, "SupplierId": supID, "RetailerId": "ret-1", "WarehouseId": whID,
			"DriverId": drvID, "Status": string(StatusPending), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson": items1, "TotalMinor": int64(3000), "Currency": "UZS", "Version": int64(1),
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ord2, "SupplierId": supID, "RetailerId": "ret-2", "WarehouseId": whID,
			"DriverId": drvID, "Status": string(StatusPending), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson": items2, "TotalMinor": int64(7000), "Currency": "UZS", "Version": int64(1),
			"CreatedAt": now, "UpdatedAt": now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed early-complete fixture: %v", err)
	}

	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Repo:          NewSpannerRepository(client),
		SpannerClient: client,
		Cache:         c,
		SupplierID:    supID,
		Log:           slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})
	if err := svc.StoreEarlyCompleteRequest(ctx, earlyCompleteRecord{
		DriverID:    drvID,
		WarehouseID: whID,
		OrderIDs:    []string{ord1, ord2},
		Reason:      "vehicle_breakdown",
	}); err != nil {
		t.Fatalf("StoreEarlyCompleteRequest: %v", err)
	}

	claims := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: supID}
	reqBody := fmt.Sprintf(`{"driver_id":%q,"action":"CANCEL"}`, drvID)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/route/approve-early-complete", strings.NewReader(reqBody))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	req.Header.Set("Idempotency-Key", "approve-ec-"+drvID)
	rr := httptest.NewRecorder()
	svc.HandleApproveEarlyComplete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("HandleApproveEarlyComplete status=%d body=%s", rr.Code, rr.Body.String())
	}

	row, err := client.Single().ReadRow(ctx, "SupplierInventoryV2", spanner.Key{supID, whID, skuID}, []string{"QuantityReserved"})
	if err != nil {
		t.Fatalf("read SupplierInventoryV2: %v", err)
	}
	var qr int64
	if err := row.Columns(&qr); err != nil {
		t.Fatalf("parse QuantityReserved: %v", err)
	}
	// 20 - (3 + 7) = 10; per-order overwrite without aggregation would leave 17 or 13.
	if qr != 10 {
		t.Fatalf("QuantityReserved=%d, want 10 after aggregating both orders", qr)
	}
}

func TestRefund_MixedCardAndCashSplitAndPendingResume(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	// 1. MIXED refund splits remaining across both CARD and CASH legs.
	mixOrderID := fmt.Sprintf("ord_rf_mix_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, mixOrderID, []map[string]any{
		capturedLeg("leg-card", string(MethodCard), 6000, "mix-card-"+mixOrderID),
		capturedLeg("leg-cash", string(MethodCash), 4000, "mix-cash-"+mixOrderID),
	})

	s := newMoneyPathGateService(client, nil)
	s.SetPaymentRefunder(stubPaymentRefunder{ref: "gp-mix-1"})
	resMix, err := s.InitiateRefund(ctx, RefundRequest{
		OrderID:        mixOrderID,
		AmountMinor:    10000,
		ActorID:        "a",
		IdempotencyKey: "idem-mix-" + mixOrderID,
	})
	if err != nil {
		t.Fatalf("MIXED refund: %v", err)
	}
	if resMix.Method != RefundMethodMixed || resMix.Status != RefundStatusCaptured {
		t.Fatalf("MIXED refund = %s/%s, want MIXED/CAPTURED", resMix.Method, resMix.Status)
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT AmountMinor, Status, IdempotencyKey FROM OrderPaymentLegs WHERE OrderId = @oid AND Method = @m`,
		Params: map[string]any{"oid": mixOrderID, "m": string(MethodRefund)},
	})
	defer iter.Stop()
	var cardRefunded, cashRefunded int64
	for {
		row, rerr := iter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr != nil {
			t.Fatalf("query MIXED legs: %v", rerr)
		}
		var amt int64
		var st, ik string
		_ = row.Columns(&amt, &st, &ik)
		if st != string(PaymentStatusCaptured) {
			t.Fatalf("MIXED refund leg %s status=%s want CAPTURED", ik, st)
		}
		if strings.HasPrefix(ik, "refund-card:") {
			cardRefunded += amt
		} else if strings.HasPrefix(ik, "refund-cash:") {
			cashRefunded += amt
		}
	}
	if cardRefunded != 6000 || cashRefunded != 4000 {
		t.Fatalf("MIXED refund legs card=%d cash=%d want 6000/4000", cardRefunded, cashRefunded)
	}

	// 2. Existing PENDING refund with idempotency key resumes and completes on retry.
	pendOrderID := fmt.Sprintf("ord_rf_pend_%d", time.Now().UnixNano())
	insertRefundableOrder(t, ctx, client, pendOrderID, []map[string]any{
		capturedLeg("leg-1", string(MethodCard), 10000, "pend-cap-"+pendOrderID),
	})
	pendRefundID := "ref_pend_" + uuid.NewString()[:8]
	pendIdem := "idem-pend-" + pendOrderID
	now := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Refunds", map[string]any{
			"RefundId": pendRefundID, "OrderId": pendOrderID, "SupplierId": "sup-refund", "RetailerId": "ret-refund",
			"AmountMinor": int64(4500), "Currency": "UZS", "ReasonCode": "DAMAGED",
			"Status": RefundStatusPending, "IdempotencyKey": pendIdem, "CreatedBy": "a",
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("OrderPaymentLegs", map[string]any{
			"OrderId": pendOrderID, "LegId": "leg-rf-pend", "Method": string(MethodRefund),
			"AmountMinor": int64(4500), "Status": string(PaymentStatusPending),
			"IdempotencyKey": "refund-card:" + pendRefundID, "CreatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed PENDING refund: %v", err)
	}

	s.SetPaymentRefunder(stubPaymentRefunder{ref: "gp-resume-1"})
	resResume, err := s.InitiateRefund(ctx, RefundRequest{
		OrderID:        pendOrderID,
		AmountMinor:    4500,
		ActorID:        "a",
		IdempotencyKey: pendIdem,
	})
	if err != nil {
		t.Fatalf("PENDING refund resume: %v", err)
	}
	if resResume.Status != RefundStatusCaptured || resResume.ProviderRef != "gp-resume-1" {
		t.Fatalf("resumed refund = %+v, want CAPTURED / gp-resume-1", resResume)
	}
}

func TestFinancialDrafts_PreCommitSoDBlocksWrite(t *testing.T) {
	ctx := context.Background()
	client := requireDraftSpannerClient(t, ctx)

	supplierA := "sup-sod-" + uuid.NewString()[:8]
	orderID := "ord-sod-" + uuid.NewString()[:8]
	svc := newDraftTestService(map[string]Order{
		orderID: {OrderID: orderID, SupplierID: supplierA, RetailerID: "ret-sod", Status: StatusCompleted},
	})
	svc.spannerClient = client

	opClaims := auth.Claims{Subject: "op-sod-1", Role: auth.RoleAdmin, SupplierID: supplierA}
	created, err := svc.CreateDraftCreditNote(ctx, opClaims, CreateDraftRequest{
		OrderID: orderID, ProposedAmountMinor: 15000, ReasonCode: "SHORT_DELIVERY",
	})
	if err != nil {
		t.Fatalf("CreateDraftCreditNote: %v", err)
	}

	sodErr := errors.New("external SoD service blocked review")
	svc.SetDraftSoDCheck(func(context.Context, DraftSoDCheckInput) error {
		return sodErr
	})

	revClaims := auth.Claims{Subject: "rev-sod-1", Role: auth.RoleAdmin, SupplierID: supplierA}
	_, err = svc.ReviewDraftCreditNote(ctx, revClaims, ReviewDraftRequest{DraftID: created.DraftID, Action: "APPROVE"})
	if !errors.Is(err, sodErr) {
		t.Fatalf("ReviewDraftCreditNote err=%v, want %v", err, sodErr)
	}

	row, rerr := client.Single().ReadRow(ctx, draftTableCreditNotes, spanner.Key{created.DraftID}, []string{"Status"})
	if rerr != nil {
		t.Fatalf("read DraftCreditNotes: %v", rerr)
	}
	var st string
	_ = row.Columns(&st)
	if st != "PENDING_APPROVAL" {
		t.Fatalf("DraftCreditNotes status=%q after pre-commit SoD error, want PENDING_APPROVAL", st)
	}
}

func TestApprovals_ReconstructUnifiedCheckoutReplay(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	repo := newMandateMemRepo()
	reader := newMapMandateReader()
	svc := newMandateTestService(repo, reader, func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) })
	ctx := context.Background()

	clientKey := "uc-replay-parent-1"
	for _, spec := range []struct {
		oid, sid string
		total    int64
	}{
		{"ord-uc-a", "sup-a", 6000},
		{"ord-uc-b", "sup-b", 7000},
	} {
		o := Order{
			OrderID:        spec.oid,
			ParentOrderID:  "par-uc-1",
			RetailerID:     "ret-1",
			SupplierID:     spec.sid,
			IdempotencyKey: legClientOrderIDWithRetry(clientKey, spec.sid, 0),
			TotalMinor:     spec.total,
			Currency:       "UZS",
		}
		_ = repo.CreateOrder(ctx, &o, nil, StockReservationOpts{})
		svc.recordOrderIdempotencyMemory(o)
	}

	env := ApprovalPayloadEnvelope{
		Kind: ApprovalPayloadUnifiedCheckout,
		UnifiedRequest: &UnifiedCheckoutRequest{
			RetailerID:    "ret-1",
			ClientOrderID: clientKey,
			Items: []UnifiedCheckoutLineItem{
				{SupplierID: "sup-a", SkuID: "SKU-A", Quantity: 1, UnitPrice: 6000},
				{SupplierID: "sup-b", SkuID: "SKU-B", Quantity: 1, UnitPrice: 7000},
			},
		},
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	rec := ApprovalRequestDTO{
		RequestID:       "req-uc-replay-1",
		RetailerID:      "ret-1",
		ClientOrderID:   clientKey,
		Status:          ApprovalStatusApproved,
		DecisionReason:  "approved",
		OrderPayload:    &env,
		OrderPayloadRaw: string(raw),
	}
	resp, ok, err := svc.reconstructApprovedReplay(ctx, rec)
	if err != nil || !ok {
		t.Fatalf("reconstructApprovedReplay ok=%v err=%v", ok, err)
	}
	if resp.UnifiedCheckout == nil || resp.UnifiedCheckout.Total != 13000 || len(resp.UnifiedCheckout.SupplierOrders) != 2 {
		t.Fatalf("unexpected UnifiedCheckout replay response: %+v", resp.UnifiedCheckout)
	}
}
