package order

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/loyalty"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
)

type PaymentMethod string

const (
	MethodCard       PaymentMethod = "CARD"
	MethodCash       PaymentMethod = "CASH"
	MethodCredit     PaymentMethod = "CREDIT"
	MethodWallet     PaymentMethod = "WALLET"
	MethodRefund     PaymentMethod = "REFUND"
	MethodCreditNote PaymentMethod = "CREDIT_NOTE"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "PENDING"
	PaymentStatusAuthorized PaymentStatus = "AUTHORIZED"
	PaymentStatusCaptured   PaymentStatus = "CAPTURED"
	PaymentStatusFailed     PaymentStatus = "FAILED"
	PaymentStatusReversed   PaymentStatus = "REVERSED"
)

type PaymentLeg struct {
	OrderID        string             `spanner:"OrderId"`
	LegID          string             `spanner:"LegId"`
	Method         PaymentMethod      `spanner:"Method"`
	AmountMinor    int64              `spanner:"AmountMinor"`
	Status         PaymentStatus      `spanner:"Status"`
	IdempotencyKey string             `spanner:"IdempotencyKey"`
	ProviderRef    spanner.NullString `spanner:"ProviderRef"`
	CreatedAt      time.Time          `spanner:"CreatedAt"`
	CapturedAt     spanner.NullTime   `spanner:"CapturedAt"`
}

type SettlementException struct {
	OrderID     string             `spanner:"OrderId"`
	ExceptionID string             `spanner:"ExceptionId"`
	Type        string             `spanner:"Type"`
	AmountMinor int64              `spanner:"AmountMinor"`
	Status      string             `spanner:"Status"`
	Reason      spanner.NullString `spanner:"Reason"`
	CreatedBy   string             `spanner:"CreatedBy"`
	CreatedAt   time.Time          `spanner:"CreatedAt"`
}

// AssertMoneyCoversDelivery evaluates OrderPaymentLegs and exceptions vs order.DeliveredGross.
func (s *Service) AssertMoneyCoversDelivery(ctx context.Context, orderID string, proposedAmountMinor int64, proposedExceptionsMinor int64) error {
	delivered, err := s.getDeliveredGrossMinor(ctx, orderID)
	if err != nil {
		return err
	}

	paid, err := s.getCapturedPaymentMinor(ctx, orderID)
	if err != nil {
		return err
	}

	exceptions, err := s.getExceptionsTotalMinor(ctx, orderID)
	if err != nil {
		return err
	}

	totalCovered := paid + proposedAmountMinor + exceptions + proposedExceptionsMinor

	if totalCovered < delivered {
		return status.Errorf(codes.FailedPrecondition,
			"payment shortfall: delivered %d, covered %d", delivered, totalCovered)
	}
	return nil
}

// AssertMoneyCoversDeliveryTxn is the in-transaction variant (must not open Single() reads).
func (s *Service) AssertMoneyCoversDeliveryTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, proposedAmountMinor int64, proposedExceptionsMinor int64) error {
	orderRecord, ok, err := s.repo.GetOrderTxn(ctx, txn, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}
	delivered := deliveredGrossFromOrder(orderRecord)
	paid, err := s.getCapturedPaymentMinorTxn(ctx, txn, orderID)
	if err != nil {
		return err
	}
	exceptions, err := s.getExceptionsTotalMinorTxn(ctx, txn, orderID)
	if err != nil {
		return err
	}
	totalCovered := paid + proposedAmountMinor + exceptions + proposedExceptionsMinor
	if totalCovered < delivered {
		return status.Errorf(codes.FailedPrecondition,
			"payment shortfall: delivered %d, covered %d", delivered, totalCovered)
	}
	return nil
}

func sumAmountMinorRows(iter *spanner.RowIterator) (int64, error) {
	defer iter.Stop()
	var total int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var amt int64
		if err := row.Columns(&amt); err != nil {
			return 0, err
		}
		total += amt
	}
	return total, nil
}

// getExceptionsTotalMinor sums all settlement exceptions for the order.
func (s *Service) getExceptionsTotalMinor(ctx context.Context, orderID string) (int64, error) {
	if s.spannerClient == nil {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderSettlementExceptions WHERE OrderId = @order_id`,
		Params: map[string]interface{}{
			"order_id": orderID,
		},
	}
	total, err := sumAmountMinorRows(s.spannerClient.Single().Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query exceptions: %w", err)
	}
	return total, nil
}

func (s *Service) getExceptionsTotalMinorTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderSettlementExceptions WHERE OrderId = @order_id`,
		Params: map[string]interface{}{
			"order_id": orderID,
		},
	}
	total, err := sumAmountMinorRows(txn.Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query exceptions: %w", err)
	}
	return total, nil
}

func deliveredGrossFromOrder(orderRecord Order) int64 {
	var gross int64
	for _, item := range orderRecord.LineItems {
		if item.OffloadStatus == "PARTIAL" || item.DeliveredQty > 0 {
			gross += item.DeliveredQty * item.UnitPrice
		} else if item.OffloadStatus == "RETURNED" || item.OffloadStatus == "FULL_RETURN" {
			// Delivered 0
		} else {
			gross += item.Quantity * item.UnitPrice
		}
	}
	return gross
}

// getDeliveredGrossMinor calculates the delivered gross using only lines with DeliveredQty > 0
// if there is a partial delivery. Otherwise it uses Quantity.
func (s *Service) getDeliveredGrossMinor(ctx context.Context, orderID string) (int64, error) {
	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return 0, fmt.Errorf("failed to get order: %w", err)
	}
	if !ok {
		return 0, fmt.Errorf("order not found: %s", orderID)
	}
	return deliveredGrossFromOrder(orderRecord), nil
}

// getCapturedPaymentMinor sums all CAPTURED payment legs for the order.
func (s *Service) getCapturedPaymentMinor(ctx context.Context, orderID string) (int64, error) {
	if s.spannerClient == nil {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderPaymentLegs WHERE OrderId = @order_id AND Status = @status`,
		Params: map[string]interface{}{
			"order_id": orderID,
			"status":   string(PaymentStatusCaptured),
		},
	}
	total, err := sumAmountMinorRows(s.spannerClient.Single().Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query payment legs: %w", err)
	}
	return total, nil
}

func (s *Service) getCapturedPaymentMinorTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderPaymentLegs WHERE OrderId = @order_id AND Status = @status`,
		Params: map[string]interface{}{
			"order_id": orderID,
			"status":   string(PaymentStatusCaptured),
		},
	}
	total, err := sumAmountMinorRows(txn.Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query payment legs: %w", err)
	}
	return total, nil
}

// RecordPaymentLeg writes a payment leg to the database within a transaction.
func (s *Service) RecordPaymentLeg(ctx context.Context, txn *spanner.ReadWriteTransaction, leg PaymentLeg) error {
	m, err := spanner.InsertStruct("OrderPaymentLegs", leg)
	if err != nil {
		return err
	}
	muts := []*spanner.Mutation{m}

	if leg.Status == PaymentStatusCaptured {
		if s.commissionResolver == nil {
			return fmt.Errorf("commission resolver not configured")
		}
		
		// Fetch order for SupplierID and Currency
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{leg.OrderID}, []string{"SupplierId", "Currency"})
		if err != nil {
			return fmt.Errorf("failed to read order %s for settlement slice: %w", leg.OrderID, err)
		}
		var supplierID, currency string
		if err := row.Columns(&supplierID, &currency); err != nil {
			return err
		}

		amount := leg.AmountMinor
		if leg.Method == MethodRefund {
			amount = -amount
		}

		_, sliceM, err := payout.GenerateSettlementSlice(ctx, s.commissionResolver, s.newID(), leg.OrderID, supplierID, leg.LegID, amount, currency, leg.CapturedAt.Time)
		if err != nil {
			return err
		}
		muts = append(muts, sliceM)
	}

	return txn.BufferWrite(muts)
}

// RecordSettlementException writes a settlement exception to the database within a transaction.
func (s *Service) RecordSettlementException(ctx context.Context, txn *spanner.ReadWriteTransaction, ex SettlementException) error {
	m, err := spanner.InsertStruct("OrderSettlementExceptions", ex)
	if err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// recordPaymentLegStandalone inserts a payment leg outside an order-update
// transaction (e.g. the deferred-payment sweeper recording a confirmed capture).
func (s *Service) recordPaymentLegStandalone(ctx context.Context, leg PaymentLeg) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner client not configured")
	}
	m, err := spanner.InsertStruct("OrderPaymentLegs", leg)
	if err != nil {
		return err
	}
	_, err = s.spannerClient.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// latestCardPaymentLeg returns the most recent CARD payment leg for the order.
func (s *Service) latestCardPaymentLeg(ctx context.Context, orderID string) (PaymentLeg, bool, error) {
	if s.spannerClient == nil {
		return PaymentLeg{}, false, fmt.Errorf("spanner client not configured")
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, LegId, Method, AmountMinor, Status, IdempotencyKey, ProviderRef, CreatedAt, CapturedAt
		      FROM OrderPaymentLegs
		      WHERE OrderId = @order_id AND Method = @method
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"order_id": orderID,
			"method":   string(MethodCard),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return PaymentLeg{}, false, nil
	}
	if err != nil {
		return PaymentLeg{}, false, fmt.Errorf("query card payment leg: %w", err)
	}
	var leg PaymentLeg
	if err := row.ToStruct(&leg); err != nil {
		return PaymentLeg{}, false, fmt.Errorf("decode card payment leg: %w", err)
	}
	return leg, true, nil
}

// bufferedOutboxMutations converts buffered outbox events into OutboxEvents mutations.
// CreatedAt is the commit timestamp: row creation time is commit time, and
// wall-clock writes fail under any client/host clock skew.
func bufferedOutboxMutations(buf *spannerTxnBuffer, _ time.Time) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(buf.events))
	for _, e := range buf.events {
		mutations = append(mutations, outboxEventMutation(e))
	}
	return mutations
}

// finalizeCardSettlement emits PAYMENT_CLEARED and FISCAL_RECEIPT_REQUESTED in one
// transaction. When leg is non-nil the same commit marks it CAPTURED with the
// provider reference, so a card leg is only ever CAPTURED after provider confirmation.
func (s *Service) finalizeCardSettlement(ctx context.Context, orderRecord Order, leg *PaymentLeg, attemptRow FiscalReceiptRow, providerRef string) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner client not configured")
	}
	now := s.now().UTC()
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := emitPaymentCleared(ctx, buf, orderRecord, string(MethodCard)); err != nil {
			return err
		}
		if err := emitFiscalReceiptRequested(ctx, buf, attemptRow); err != nil {
			return err
		}
		mutations := bufferedOutboxMutations(buf, now)
		if leg != nil {
			mutations = append(mutations, spanner.UpdateMap("OrderPaymentLegs", map[string]any{
				"OrderId":     leg.OrderID,
				"LegId":       leg.LegID,
				"Status":      string(PaymentStatusCaptured),
				"ProviderRef": nullableString(providerRef),
				"CapturedAt":  now,
			}))
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}
		// G1-A2: clear credit balance only after card money is CAPTURED (same txn).
		// No-op when order was never credit-left (no CONVERTED reservation).
		if s.credit != nil && orderRecord.TotalMinor > 0 {
			if err := s.credit.ClearBalanceInTxn(ctx, txn,
				orderRecord.RetailerID, orderRecord.SupplierID, orderRecord.OrderID, orderRecord.TotalMinor); err != nil {
				return err
			}
		}
		// G1.A twin for card: AR pay-down when invoice exists (same txn as capture).
		if s.ar != nil && orderRecord.TotalMinor > 0 {
			amt := orderRecord.TotalMinor
			if leg != nil && leg.AmountMinor > 0 {
				amt = leg.AmountMinor
			}
			if err := s.ar.RecordPaymentForOrderInTxn(ctx, txn, orderRecord.OrderID, amt,
				"ar-card-settle-"+orderRecord.OrderID, orderRecord.Currency); err != nil {
				return err
			}
		}
		amt := orderRecord.TotalMinor
		if leg != nil && leg.AmountMinor > 0 {
			amt = leg.AmountMinor
		}
		return loyalty.EarnInTxn(ctx, txn, nil, orderRecord.SupplierID, orderRecord.RetailerID, orderRecord.OrderID, amt)
	})
	return err
}

// failCardCapture marks a card leg FAILED and records an open settlement exception
// so a capture shortfall is visible to ops and finance instead of being absorbed.
func (s *Service) failCardCapture(ctx context.Context, orderRecord Order, leg PaymentLeg, cause error) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner client not configured")
	}
	now := s.now().UTC()
	reason := "card capture failed"
	if cause != nil {
		reason = cause.Error()
	}
	if len(reason) > 512 {
		reason = reason[:512]
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.FinanceEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventPaymentFailed, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:     orderRecord.OrderID,
			SupplierID:  orderRecord.SupplierID,
			RetailerID:  orderRecord.RetailerID,
			Status:      "CARD_CAPTURE_FAILED",
			AmountMinor: leg.AmountMinor,
			Currency:    orderRecord.Currency,
			Source:      "order.card_capture",
		}); err != nil {
			return err
		}
		mutations := bufferedOutboxMutations(buf, now)
		mutations = append(mutations,
			spanner.UpdateMap("OrderPaymentLegs", map[string]any{
				"OrderId": leg.OrderID,
				"LegId":   leg.LegID,
				"Status":  string(PaymentStatusFailed),
			}),
			spanner.InsertMap("OrderSettlementExceptions", map[string]any{
				"OrderId":     orderRecord.OrderID,
				"ExceptionId": s.newID(),
				"Type":        "CARD_CAPTURE_FAILED",
				"AmountMinor": leg.AmountMinor,
				"Status":      "OPEN",
				"Reason":      reason,
				"CreatedBy":   "system",
				"CreatedAt":   now,
			}),
		)
		return txn.BufferWrite(mutations)
	})
	return err
}

// settleOutstandingCardPayment performs the deferred card settlement for an order
// entering FISCALIZING: captures any outstanding (PENDING/FAILED) card leg with the
// provider and only then emits the fiscal receipt request. When no card leg exists,
// delivery is already covered by other tender and fiscal is requested directly.
func (s *Service) settleOutstandingCardPayment(ctx context.Context, orderRecord Order) error {
	if s.spannerClient == nil {
		return nil
	}
	leg, found, err := s.latestCardPaymentLeg(ctx, orderRecord.OrderID)
	if err != nil {
		return err
	}
	if found && (leg.Status == PaymentStatusPending || leg.Status == PaymentStatusFailed) {
		if s.paymentCapturer == nil {
			return fmt.Errorf("card capture outstanding for order %s but payment capturer is not configured", orderRecord.OrderID)
		}
		captureCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		providerRef, capErr := s.paymentCapturer.CaptureCardPayment(captureCtx, orderRecord.OrderID, leg.AmountMinor, orderRecord.Currency)
		cancel()
		if capErr != nil {
			if ferr := s.failCardCapture(ctx, orderRecord, leg, capErr); ferr != nil {
				s.log.Error("persist card capture failure failed", "order_id", orderRecord.OrderID, "err", ferr)
			}
			return fmt.Errorf("card capture failed: %w", capErr)
		}
		attemptRow, rowErr := s.resolveLatestFiscalAttemptRow(ctx, orderRecord)
		if rowErr != nil {
			return fmt.Errorf("card capture succeeded but fiscal attempt unavailable: %w", rowErr)
		}
		if cerr := s.finalizeCardSettlement(ctx, orderRecord, &leg, attemptRow, providerRef); cerr != nil {
			// Money moved but the confirmation commit failed. The leg stays PENDING;
			// the next CompleteOrder retry short-circuits at the provider
			// (already-captured) and re-confirms here.
			return fmt.Errorf("card capture confirmation failed: %w", cerr)
		}
		return nil
	}
	if found {
		// A CAPTURED card leg implies its settlement events were committed atomically.
		return nil
	}
	// No card tender for this order: request fiscalization when it has not started.
	if orderRecord.Status == StatusFiscalizing && orderRecord.FiscalStatus == FiscalStatusPending {
		attemptRow, rowErr := s.resolveLatestFiscalAttemptRow(ctx, orderRecord)
		if rowErr != nil {
			return rowErr
		}
		if attemptRow.Status != FiscalAttemptPending {
			return nil
		}
		return s.finalizeCardSettlement(ctx, orderRecord, nil, attemptRow, "")
	}
	return nil
}

// resolveLatestFiscalAttemptRow returns the pending fiscal attempt row prepared by
// the completing transition, loading it from the repository when the in-memory
// order snapshot does not carry it (idempotent retry path).
func (s *Service) resolveLatestFiscalAttemptRow(ctx context.Context, orderRecord Order) (FiscalReceiptRow, error) {
	for _, row := range orderRecord.PendingFiscalReceipts {
		if row.AttemptID != "" {
			return row, nil
		}
	}
	if orderRecord.LatestFiscalAttemptID == "" {
		return FiscalReceiptRow{}, fmt.Errorf("no fiscal attempt pending for order %s", orderRecord.OrderID)
	}
	row, found, err := s.repo.GetFiscalAttempt(ctx, orderRecord.OrderID, orderRecord.LatestFiscalAttemptID)
	if err != nil {
		return FiscalReceiptRow{}, err
	}
	if !found {
		return FiscalReceiptRow{}, fmt.Errorf("fiscal attempt %s not found", orderRecord.LatestFiscalAttemptID)
	}
	return row, nil
}
