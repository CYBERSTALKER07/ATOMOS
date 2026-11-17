package order

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

// coveringExceptionsSQL selects only settlement exceptions that account for an
// order-level shortfall (an acknowledged receivable recorded at settlement time).
// Excluded on purpose:
//   - CARD_CAPTURE_FAILED: a retryable capture failure — the FAILED card leg is
//     re-captured by settleOutstandingCardPayment, so counting it would let an
//     unpaid order pass AssertMoneyCoversDelivery and double-count on success.
//   - CASH_DISCREPANCY: a route-level cash-recon variance attached to one
//     arbitrary order on the route; it is not money owed on that order.
const coveringExceptionsSQL = `SELECT AmountMinor FROM OrderSettlementExceptions
	WHERE OrderId = @order_id AND Type IN UNNEST(@covering_types)`

// coveringExceptionTypes are the settlement exception types that represent an
// acknowledged order-level shortfall.
var coveringExceptionTypes = []string{"CASH_SHORTFALL", "FORCE_COMPLETE_SHORTFALL"}

func coveringExceptionsStatement(orderID string) spanner.Statement {
	return spanner.Statement{
		SQL: coveringExceptionsSQL,
		Params: map[string]interface{}{
			"order_id":       orderID,
			"covering_types": coveringExceptionTypes,
		},
	}
}

// getExceptionsTotalMinor sums the shortfall-covering settlement exceptions for the order.
func (s *Service) getExceptionsTotalMinor(ctx context.Context, orderID string) (int64, error) {
	if s.spannerClient == nil {
		return 0, nil
	}
	stmt := coveringExceptionsStatement(orderID)
	total, err := sumAmountMinorRows(s.spannerClient.Single().Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query exceptions: %w", err)
	}
	return total, nil
}

func (s *Service) getExceptionsTotalMinorTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (int64, error) {
	stmt := coveringExceptionsStatement(orderID)
	total, err := sumAmountMinorRows(txn.Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query exceptions: %w", err)
	}
	return total, nil
}

// deliveredGrossFromOrder returns the money owed for what was actually handed
// over: delivered units at their unit price, plus the delivery fee when at
// least one unit was delivered. Lines without an offload outcome are billed at
// the ordered quantity; lines offloaded as NONE / RETURNED bill nothing.
// When an order carries no itemized lines (header-only order with TotalMinor > 0),
// fall back to TotalMinor so unpaid or credit-leave orders never settle as 0.
func deliveredGrossFromOrder(orderRecord Order) int64 {
	if len(orderRecord.LineItems) == 0 && orderRecord.TotalMinor > 0 {
		return orderRecord.TotalMinor
	}
	var gross int64
	for _, item := range orderRecord.LineItems {
		gross += deliveredLineQty(item) * item.UnitPrice
	}
	if gross > 0 && orderRecord.DeliveryFeeMinor > 0 {
		gross += orderRecord.DeliveryFeeMinor
	}
	return gross
}

// deliveredLineQty resolves the billable delivered quantity for one line.
func deliveredLineQty(item LineItem) int64 {
	switch item.OffloadStatus {
	case OffloadStatusPartial, OffloadStatusFull:
		if item.DeliveredQty < 0 {
			return 0
		}
		return item.DeliveredQty
	case OffloadStatusNone, OffloadStatusReturned, "FULL_RETURN":
		return 0
	}
	if item.DeliveredQty > 0 {
		return item.DeliveredQty
	}
	return item.Quantity
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

func capturedPaymentStatement(orderID string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderPaymentLegs 
		      WHERE OrderId = @order_id 
		        AND Status = @status 
		        AND Method != @credit_method 
		        AND IdempotencyKey NOT LIKE 'credit-leave-%'`,
		Params: map[string]interface{}{
			"order_id":      orderID,
			"status":        string(PaymentStatusCaptured),
			"credit_method": string(MethodCredit),
		},
	}
}

// getCapturedPaymentMinor sums all non-credit CAPTURED payment legs for the order.
func (s *Service) getCapturedPaymentMinor(ctx context.Context, orderID string) (int64, error) {
	if s.spannerClient == nil {
		return 0, nil
	}
	stmt := capturedPaymentStatement(orderID)
	total, err := sumAmountMinorRows(s.spannerClient.Single().Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query payment legs: %w", err)
	}
	return total, nil
}

func (s *Service) getCapturedPaymentMinorTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (int64, error) {
	stmt := capturedPaymentStatement(orderID)
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

		sliceTime := leg.CapturedAt.Time
		if !leg.CapturedAt.Valid || sliceTime.IsZero() {
			if !leg.CreatedAt.IsZero() {
				sliceTime = leg.CreatedAt
			} else {
				sliceTime = time.Now().UTC()
			}
		}

		_, sliceM, err := payout.GenerateSettlementSlice(ctx, s.commissionResolver, s.newID(), leg.OrderID, supplierID, leg.LegID, amount, currency, sliceTime)
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

// bufferCardSettlementEvents buffers outbox events for card settlement:
// - Emits PAYMENT_CLEARED only when leg != nil && leg.AmountMinor > 0 (real card tender collected).
// - Always emits FISCAL_RECEIPT_REQUESTED for attemptRow.
// Never emits PAYMENT_CLEARED when leg is nil or zero.
func bufferCardSettlementEvents(ctx context.Context, buf outbox.TxnBuffer, orderRecord Order, leg *PaymentLeg, attemptRow FiscalReceiptRow) error {
	if leg != nil && leg.AmountMinor > 0 {
		if err := emitPaymentCleared(ctx, buf, orderRecord, string(MethodCard)); err != nil {
			return err
		}
	}
	return emitFiscalReceiptRequested(ctx, buf, attemptRow)
}

// shouldClearCreditAndAR verifies whether a payment leg represents verified positive tender
// that permits clearing credit balance and paying down accounts receivable.
// Returns false for nil legs, zero amounts, or negative amounts.
func shouldClearCreditAndAR(leg *PaymentLeg) bool {
	return leg != nil && leg.AmountMinor > 0
}

// checkSettlementTenderSufficiency verifies that paid captured tender plus approved
// settlement exceptions fully cover the delivered gross amount.
// Returns an error if paid + exceptions < delivered.
func checkSettlementTenderSufficiency(orderID string, paid, exceptions, delivered int64) error {
	if paid+exceptions < delivered {
		return fmt.Errorf("cannot settle order %s without verified payment receipt: paid (%d) + exceptions (%d) < delivered (%d)",
			orderID, paid, exceptions, delivered)
	}
	return nil
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
		if err := bufferCardSettlementEvents(ctx, buf, orderRecord, leg, attemptRow); err != nil {
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
		// G1-A2: clear credit balance only after real card money is CAPTURED (same txn).
		// Never forgive credit balance or clear AR when leg is nil or zero.
		if shouldClearCreditAndAR(leg) {
			if s.credit != nil {
				if err := s.credit.ClearBalanceInTxn(ctx, txn,
					orderRecord.RetailerID, orderRecord.SupplierID, orderRecord.OrderID, leg.AmountMinor); err != nil {
					return err
				}
			}
			// G1.A twin for card: AR pay-down when invoice exists (same txn as capture).
			if s.ar != nil {
				mode := "CARD"
				if m := normalizeOrderGLPaymentMode(string(leg.Method)); m != "" && m != "CREDIT" {
					mode = m
				}
				if err := s.ar.RecordPaymentForOrderWithModeInTxn(ctx, txn, orderRecord.OrderID, leg.AmountMinor,
					mode, "ar-card-settle-"+orderRecord.OrderID, orderRecord.Currency); err != nil {
					return err
				}
			}
			return loyalty.EarnInTxn(ctx, txn, nil, orderRecord.SupplierID, orderRecord.RetailerID, orderRecord.OrderID, leg.AmountMinor)
		}
		return nil
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
				"OrderId":     leg.OrderID,
				"LegId":       leg.LegID,
				"Status":      string(PaymentStatusFailed),
				"ProviderRef": spanner.NullString{},
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
		// When spanner client is not configured (e.g. unit test harness), enforce
		// tender sufficiency check for orders entering fiscalization.
		if orderRecord.Status == StatusFiscalizing && orderRecord.FiscalStatus == FiscalStatusPending {
			delivered := deliveredGrossFromOrder(orderRecord)
			return checkSettlementTenderSufficiency(orderRecord.OrderID, 0, 0, delivered)
		}
		return nil
	}
	leg, found, err := s.latestCardPaymentLeg(ctx, orderRecord.OrderID)
	if err != nil {
		return err
	}
	if found {
		var alreadyCaptured bool
		_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, rerr := txn.ReadRow(ctx, "OrderPaymentLegs", spanner.Key{leg.OrderID, leg.LegID}, []string{"Status", "ProviderRef"})
			if rerr != nil {
				return rerr
			}
			var curStatus string
			var curRef spanner.NullString
			if err := row.Columns(&curStatus, &curRef); err != nil {
				return err
			}
			if curStatus == string(PaymentStatusCaptured) {
				alreadyCaptured = true
				return nil
			}
			if curStatus != string(PaymentStatusPending) && curStatus != string(PaymentStatusFailed) {
				return fmt.Errorf("card leg %s in non-capturable state %s", leg.LegID, curStatus)
			}
			if curRef.Valid && strings.HasPrefix(curRef.StringVal, "claiming:") {
				parts := strings.Split(curRef.StringVal, ":")
				if len(parts) == 3 {
					if ts, perr := strconv.ParseInt(parts[2], 10, 64); perr == nil && s.now().Unix()-ts < 60 {
						return fmt.Errorf("card capture already in progress for order %s", orderRecord.OrderID)
					}
				}
			}
			claimVal := fmt.Sprintf("claiming:%s:%d", s.newID(), s.now().Unix())
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.UpdateMap("OrderPaymentLegs", map[string]any{
					"OrderId":     leg.OrderID,
					"LegId":       leg.LegID,
					"Status":      string(PaymentStatusPending),
					"ProviderRef": spanner.NullString{StringVal: claimVal, Valid: true},
				}),
			})
		})
		if err != nil {
			return err
		}
		if alreadyCaptured {
			return nil
		}
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
			return fmt.Errorf("card capture confirmation failed: %w", cerr)
		}
		return nil
	}
	// No card tender for this order: verify delivery is covered by real captured tender before requesting fiscalization.
	if orderRecord.Status == StatusFiscalizing && orderRecord.FiscalStatus == FiscalStatusPending {
		paid, err := s.getCapturedPaymentMinor(ctx, orderRecord.OrderID)
		if err != nil {
			return err
		}
		exceptions, err := s.getExceptionsTotalMinor(ctx, orderRecord.OrderID)
		if err != nil {
			return err
		}
		delivered, err := s.getDeliveredGrossMinor(ctx, orderRecord.OrderID)
		if err != nil {
			return err
		}
		if err := checkSettlementTenderSufficiency(orderRecord.OrderID, paid, exceptions, delivered); err != nil {
			return err
		}

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
