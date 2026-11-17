package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Refunds: full/partial reversal of captured money with provider confirmation.
//
// Invariants (Phase 1 money rules):
//   - capped at Σcaptured − Σrefunded, per original method (card/cash)
//   - reversal legs are explicit REFUND rows, never mutation of capture legs
//   - provider (card) refunds are confirmed synchronously; the refund and its
//     leg are only CAPTURED after the gateway confirms — same truthfulness rule
//     as the Phase-0 capture fix
//   - credit-paid portions are refused here: they reduce AR balances via the AR
//     adjustment path, not card/cash reversals
//   - fiscal corrective chain: on success the refund links the order's issued
//     receipt (CreditNotes.OriginalEhfId) and requests a corrective receipt
//     (CorrectiveEhfId) from providers that support it
//
// Idempotency: Refunds.IdempotencyKey unique index; REFUND leg keys are
// "refund-{card|cash}:{refundID}" so retries hit the unique leg index.

// PaymentRefunder performs a retry-safe provider reversal for captured card
// payments. Implementations must return success with the provider reference
// when the reversal already happened (repeat calls never double-refund).
type PaymentRefunder interface {
	RefundCardPayment(ctx context.Context, orderID string, amountMinor int64, currency string) (providerRef string, err error)
}

const (
	RefundStatusPending  = "PENDING"
	RefundStatusCaptured = "CAPTURED"
	RefundStatusFailed   = "FAILED"

	RefundMethodCard  = "CARD"
	RefundMethodCash  = "CASH"
	RefundMethodMixed = "MIXED"
)

var (
	ErrRefundExceedsCaptured = errors.New("refund exceeds captured minus refunded")
	ErrRefundCreditPortion   = errors.New("credit-paid portion refunds via AR adjustment, not card/cash reversal")
	ErrRefundOrderState      = errors.New("order is not in a refundable state")
	ErrRefundForbidden       = errors.New("forbidden: order belongs to another supplier")
)

type RefundRequest struct {
	OrderID         string
	AmountMinor     int64 // <=0 means full remaining refundable amount
	ReasonCode      string
	ReasonText      string
	ActorID         string
	ActorSupplierID string
	IdempotencyKey  string
}

type RefundResult struct {
	RefundID        string
	OrderID         string
	Status          string
	Method          string
	AmountMinor     int64
	Currency        string
	ProviderRef     string
	CreditNoteID    string
	CorrectiveEhfID string
}

func (s *Service) SetPaymentRefunder(r PaymentRefunder) {
	s.paymentRefunder = r
}

// refundableBalances computes per-method captured-minus-refunded balances.
// REFUND legs attribute via their idempotency key prefix (refund-card: / refund-cash:).
// PENDING refunds are deducted to prevent concurrent double-refund races.
func refundableBalances(rows func(func(method string, amount int64, status string, idemKey string) bool) error) (card, cash, credit int64, err error) {
	err = rows(func(method string, amount int64, status string, idemKey string) bool {
		switch method {
		case string(MethodCard):
			if status == string(PaymentStatusCaptured) {
				card += amount
			}
		case string(MethodCash):
			if status == string(PaymentStatusCaptured) {
				cash += amount
			}
		case string(MethodCredit):
			if status == string(PaymentStatusCaptured) {
				credit += amount
			}
		case string(MethodRefund):
			if status == string(PaymentStatusCaptured) || status == string(PaymentStatusPending) {
				switch {
				case strings.HasPrefix(idemKey, "refund-card:"):
					card -= amount
				case strings.HasPrefix(idemKey, "refund-cash:"):
					cash -= amount
				default:
					// Legacy/unknown refund attribution: subtract from the largest
					// remaining balance, never silently ignore the reversal.
					if card >= cash {
						card -= amount
					} else {
						cash -= amount
					}
				}
			}
		}
		return true
	})
	return card, cash, credit, err
}

// InitiateRefund starts a refund: validates the cap, records the PENDING
// refund + reversal leg, then confirms synchronously against the gateway
// (card) or the cash ledger. Returns the final refund state.
func (s *Service) InitiateRefund(ctx context.Context, req RefundRequest) (RefundResult, error) {
	if s == nil || s.spannerClient == nil {
		return RefundResult{}, fmt.Errorf("refund service unavailable")
	}
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RefundResult{}, fmt.Errorf("order_id required")
	}
	reasonCode := strings.TrimSpace(req.ReasonCode)
	if reasonCode == "" {
		reasonCode = "CUSTOMER_RETURN"
	}
	actor := strings.TrimSpace(req.ActorID)
	if actor == "" {
		actor = "system"
	}
	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey == "" {
		idemKey = fmt.Sprintf("refund-%s-%d-%s", orderID, req.AmountMinor, reasonCode)
	}

	// Replay: same idempotency key returns completed refund or resumes a stuck PENDING attempt.
	if existing, cardPending, cashPending, found, err := s.refundByIdempotencyKey(ctx, idemKey); err != nil {
		return RefundResult{}, err
	} else if found {
		if existing.Status != RefundStatusPending {
			return existing, nil
		}
		var orderRow Order
		row, rerr := s.spannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{existing.OrderID}, []string{"OrderId", "SupplierId", "RetailerId", "Currency"})
		if rerr != nil {
			return RefundResult{}, fmt.Errorf("load order for pending refund resume: %w", rerr)
		}
		if cerr := row.Columns(&orderRow.OrderID, &orderRow.SupplierID, &orderRow.RetailerID, &orderRow.Currency); cerr != nil {
			return RefundResult{}, cerr
		}
		if cur, curErr := auth.CoalesceCurrency(ctx, orderRow.SupplierID, orderRow.Currency); curErr == nil {
			orderRow.Currency = cur
		}
		return s.confirmAndFinalizeRefund(ctx, existing, orderRow, cardPending, cashPending, reasonCode, actor)
	}

	refundID := s.newID()
	now := s.now().UTC()

	var orderRow Order
	var cardBal, cashBal, creditBal int64
	var method string
	var amount, cardAmt, cashAmt int64

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"OrderId", "SupplierId", "RetailerId", "Status", "TotalMinor", "Currency"})
		if err != nil {
			return fmt.Errorf("order %s: %w", orderID, err)
		}
		var orderStatus string
		if err := row.Columns(&orderRow.OrderID, &orderRow.SupplierID, &orderRow.RetailerID, &orderStatus, &orderRow.TotalMinor, &orderRow.Currency); err != nil {
			return err
		}
		if strings.TrimSpace(req.ActorSupplierID) != "" && orderRow.SupplierID != "" && req.ActorSupplierID != orderRow.SupplierID {
			return ErrRefundForbidden
		}
		cur, curErr := auth.CoalesceCurrency(ctx, orderRow.SupplierID, orderRow.Currency)
		if curErr != nil {
			return curErr
		}
		orderRow.Currency = cur
		switch Status(orderStatus) {
		case StatusFiscalizing, StatusDeliveredOnCredit, StatusPendingCashCollection, StatusFiscalFailed, StatusCompleted:
		default:
			return fmt.Errorf("%w: %s", ErrRefundOrderState, orderStatus)
		}

		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT Method, AmountMinor, Status, IdempotencyKey FROM OrderPaymentLegs WHERE OrderId = @oid`,
			Params: map[string]any{"oid": orderID},
		})
		defer iter.Stop()
		readRows := func(yield func(string, int64, string, string) bool) error {
			for {
				r, err := iter.Next()
				if err == iterator.Done {
					return nil
				}
				if err != nil {
					return err
				}
				var m, st, ik string
				var amt int64
				if err := r.Columns(&m, &amt, &st, &ik); err != nil {
					return err
				}
				if !yield(m, amt, st, ik) {
					return nil
				}
			}
		}
		var err2 error
		cardBal, cashBal, creditBal, err2 = refundableBalances(readRows)
		if err2 != nil {
			return err2
		}

		amount = req.AmountMinor
		if amount <= 0 {
			amount = cardBal + cashBal
		}
		if amount <= 0 {
			return fmt.Errorf("%w: nothing refundable on order %s", ErrRefundExceedsCaptured, orderID)
		}
		totalRefundable := cardBal + cashBal
		if amount > totalRefundable {
			if creditBal > 0 && amount <= totalRefundable+creditBal {
				return fmt.Errorf("%w (remaining credit-covered: %d)", ErrRefundCreditPortion, creditBal)
			}
			return fmt.Errorf("%w: requested %d, refundable card=%d cash=%d", ErrRefundExceedsCaptured, amount, cardBal, cashBal)
		}

		cardAmt = amount
		if cardAmt > cardBal {
			cardAmt = cardBal
		}
		if cardAmt < 0 {
			cardAmt = 0
		}
		cashAmt = amount - cardAmt
		switch {
		case cardAmt > 0 && cashAmt > 0:
			method = RefundMethodMixed
		case cardAmt > 0:
			method = RefundMethodCard
		default:
			method = RefundMethodCash
		}

		muts := []*spanner.Mutation{
			spanner.InsertMap("Refunds", map[string]any{
				"RefundId":       refundID,
				"OrderId":        orderID,
				"SupplierId":     orderRow.SupplierID,
				"RetailerId":     orderRow.RetailerID,
				"AmountMinor":    amount,
				"Currency":       orderRow.Currency,
				"ReasonCode":     reasonCode,
				"ReasonText":     spanner.NullString{StringVal: req.ReasonText, Valid: strings.TrimSpace(req.ReasonText) != ""},
				"Status":         RefundStatusPending,
				"IdempotencyKey": idemKey,
				"CreatedBy":      actor,
				"CreatedAt":      spanner.CommitTimestamp,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}
		if cardAmt > 0 {
			muts = append(muts, spanner.InsertMap("OrderPaymentLegs", map[string]any{
				"OrderId":        orderID,
				"LegId":          s.newID(),
				"Method":         string(MethodRefund),
				"AmountMinor":    cardAmt,
				"Status":         string(PaymentStatusPending),
				"IdempotencyKey": "refund-card:" + refundID,
				"CreatedAt":      now,
			}))
		}
		if cashAmt > 0 {
			muts = append(muts, spanner.InsertMap("OrderPaymentLegs", map[string]any{
				"OrderId":        orderID,
				"LegId":          s.newID(),
				"Method":         string(MethodRefund),
				"AmountMinor":    cashAmt,
				"Status":         string(PaymentStatusPending),
				"IdempotencyKey": "refund-cash:" + refundID,
				"CreatedAt":      now,
			}))
		}
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.FinanceEvent{
			BaseEvent:       events.BaseEvent{Type: events.EventRefundRequested, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:         orderID,
			SupplierID:      orderRow.SupplierID,
			RetailerID:      orderRow.RetailerID,
			Status:          RefundStatusPending,
			ExecutionAction: "REFUND",
			AmountMinor:     amount,
			Currency:        orderRow.Currency,
			TransactionID:   refundID,
			Source:          "order.refund:" + method,
		}); err != nil {
			return err
		}
		muts = append(muts, bufferedOutboxMutations(buf, now)...)
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return RefundResult{}, err
	}

	result := RefundResult{
		RefundID: refundID, OrderID: orderID, Method: method,
		AmountMinor: amount, Currency: orderRow.Currency, Status: RefundStatusPending,
	}
	return s.confirmAndFinalizeRefund(ctx, result, orderRow, cardAmt, cashAmt, reasonCode, actor)
}

func (s *Service) confirmAndFinalizeRefund(ctx context.Context, result RefundResult, orderRow Order, cardAmt, cashAmt int64, reasonCode, actor string) (RefundResult, error) {
	// Synchronous confirmation — ledger never asserts money that did not move.
	var providerRef string
	var confirmErr error
	if cardAmt > 0 {
		if s.paymentRefunder == nil {
			confirmErr = fmt.Errorf("card refund configured against order %s but no payment refunder wired", result.OrderID)
		} else {
			callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			providerRef, confirmErr = s.paymentRefunder.RefundCardPayment(callCtx, result.OrderID, cardAmt, orderRow.Currency)
			cancel()
		}
	} else if cashAmt > 0 {
		providerRef = "cash-ledger-reversal"
	}
	if cardAmt > 0 && cashAmt > 0 && confirmErr == nil && providerRef != "" {
		providerRef += "+cash-ledger-reversal"
	}

	finalStatus := RefundStatusCaptured
	var outboxType string
	if confirmErr != nil {
		finalStatus = RefundStatusFailed
		outboxType = events.EventRefundFailed
	} else {
		outboxType = events.EventRefundSucceeded
	}
	if err := s.finalizeRefund(ctx, result.RefundID, result.OrderID, result.Method, result.AmountMinor, finalStatus, providerRef, outboxType, orderRow, confirmErr); err != nil {
		return RefundResult{}, fmt.Errorf("refund %s confirmed as %s but finalization failed: %w", result.RefundID, finalStatus, err)
	}
	result.Status = finalStatus
	result.ProviderRef = providerRef
	if confirmErr != nil {
		return result, confirmErr
	}

	// Fiscal corrective chain (post-success, never blocks the confirmed refund).
	cnID, correctiveID, cnErr := s.recordRefundCreditNote(ctx, result.RefundID, orderRow, result.AmountMinor, reasonCode, actor)
	if cnErr != nil {
		s.log.ErrorContext(ctx, "refund credit note / corrective chain failed",
			"refund_id", result.RefundID, "order_id", result.OrderID, "err", cnErr)
	} else {
		result.CreditNoteID = cnID
		result.CorrectiveEhfID = correctiveID
	}
	return result, nil
}

func (s *Service) finalizeRefund(ctx context.Context, refundID, orderID, method string, amount int64, status, providerRef, outboxType string, orderRow Order, confirmErr error) error {
	now := s.now().UTC()
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		updates := map[string]any{
			"Status":    status,
			"UpdatedAt": spanner.CommitTimestamp,
		}
		if providerRef != "" {
			updates["ProviderRef"] = spanner.NullString{StringVal: providerRef, Valid: true}
		}
		legStatus := string(PaymentStatusCaptured)
		if status == RefundStatusFailed {
			legStatus = string(PaymentStatusFailed)
		}
		// Locate all PENDING reversal legs for this refund (single or MIXED split).
		legKeys := []string{"refund-card:" + refundID, "refund-cash:" + refundID}
		legRows := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT LegId FROM OrderPaymentLegs WHERE OrderId = @oid AND IdempotencyKey IN UNNEST(@keys)`,
			Params: map[string]any{"oid": orderID, "keys": legKeys},
		})
		defer legRows.Stop()
		var legIDs []string
		for {
			r, err := legRows.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("query pending refund legs: %w", err)
			}
			var lid string
			if err := r.Columns(&lid); err != nil {
				return err
			}
			legIDs = append(legIDs, lid)
		}
		if len(legIDs) == 0 {
			return fmt.Errorf("pending refund leg missing for refund %s", refundID)
		}
		muts := []*spanner.Mutation{
			spanner.UpdateMap("Refunds", map[string]any{
				"RefundId":    refundID,
				"Status":      updates["Status"],
				"ProviderRef": updates["ProviderRef"],
				"UpdatedAt":   updates["UpdatedAt"],
			}),
		}
		for _, lid := range legIDs {
			muts = append(muts, spanner.UpdateMap("OrderPaymentLegs", map[string]any{
				"OrderId":     orderID,
				"LegId":       lid,
				"Status":      legStatus,
				"ProviderRef": spanner.NullString{StringVal: providerRef, Valid: providerRef != ""},
				"CapturedAt":  spanner.NullTime{Time: now, Valid: status == RefundStatusCaptured},
			}))
		}
		evt := events.FinanceEvent{
			BaseEvent:         events.BaseEvent{Type: outboxType, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:           orderID,
			SupplierID:        orderRow.SupplierID,
			RetailerID:        orderRow.RetailerID,
			Status:            status,
			ExecutionAction:   "REFUND",
			ProviderReference: providerRef,
			AmountMinor:       amount,
			Currency:          orderRow.Currency,
			TransactionID:     refundID,
			Source:            "order.refund:" + method,
		}
		if confirmErr != nil {
			evt.Source += ": " + confirmErr.Error()
		}
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, evt); err != nil {
			return err
		}
		muts = append(muts, bufferedOutboxMutations(buf, now)...)
		return txn.BufferWrite(muts)
	})
	return err
}

// recordRefundCreditNote links the refund to the order's issued fiscal receipt
// and requests the corrective receipt when the provider supports it.
func (s *Service) recordRefundCreditNote(ctx context.Context, refundID string, orderRow Order, amount int64, reasonCode, actor string) (creditNoteID, correctiveID string, err error) {
	originalEhf := ""
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FiscalReceiptId FROM OrderFiscalReceipts
		      WHERE OrderId = @oid AND Status = @st AND FiscalReceiptId IS NOT NULL
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"oid": orderRow.OrderID, "st": FiscalAttemptSuccess},
	})
	row, qErr := iter.Next()
	iter.Stop()
	if qErr == nil {
		var id spanner.NullString
		if cErr := row.Columns(&id); cErr == nil {
			originalEhf = id.StringVal
		}
	} else if qErr != iterator.Done {
		return "", "", fmt.Errorf("read fiscal receipts: %w", qErr)
	}

	creditNoteID = s.newID()
	// CreditNotes.CreatedAt/IssuedAt predate allow_commit_timestamp (DDL has no
	// OPTIONS) — wall clock here, consistent with the creditnote repository.
	cnNow := s.now().UTC()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("CreditNotes", map[string]any{
				"CreditNoteId":    creditNoteID,
				"OrderId":         orderRow.OrderID,
				"Type":            "REFUND",
				"Status":          "ISSUED",
				"ReasonCode":      reasonCode,
				"ReasonText":      spanner.NullString{StringVal: "refund_id=" + refundID, Valid: true},
				"TotalNetMinor":   amount,
				"TotalVatMinor":   int64(0),
				"TotalGrossMinor": amount,
				"OriginalEhfId":   spanner.NullString{StringVal: originalEhf, Valid: originalEhf != ""},
				"CreatedBy":       actor,
				"CreatedAt":       cnNow,
				"IssuedAt":        cnNow,
			}),
			spanner.UpdateMap("Refunds", map[string]any{
				"RefundId":     refundID,
				"CreditNoteId": spanner.NullString{StringVal: creditNoteID, Valid: true},
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		return "", "", fmt.Errorf("persist credit note: %w", err)
	}

	if originalEhf == "" {
		// No issued receipt to correct (order completed pre-fiscal or fiscal
		// skipped). The credit note stands alone; nothing else to do.
		return creditNoteID, "", nil
	}
	cp, ok := s.fiscalProvider().(CorrectiveFiscalProvider)
	if !ok {
		// Honest gap: provider cannot issue corrective receipts. Credit note
		// carries OriginalEhfId so the corrective can be issued manually.
		s.log.WarnContext(ctx, "fiscal provider has no corrective receipt support",
			"order_id", orderRow.OrderID, "original_ehf", originalEhf)
		return creditNoteID, "", nil
	}
	res, cErr := cp.CreateCorrectiveReceipt(ctx, FiscalCorrectiveRequest{
		AttemptID:         "corr-" + refundID,
		OrderID:           orderRow.OrderID,
		SupplierID:        orderRow.SupplierID,
		RetailerID:        orderRow.RetailerID,
		OriginalReceiptID: originalEhf,
		AmountMinor:       amount,
		Currency:          orderRow.Currency,
		ReasonCode:        reasonCode,
	})
	if cErr != nil {
		return creditNoteID, "", fmt.Errorf("corrective receipt: %w", cErr)
	}
	correctiveID = res.FiscalReceiptID
	now := s.now().UTC()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.UpdateMap("Refunds", map[string]any{
				"RefundId":        refundID,
				"CorrectiveEhfId": spanner.NullString{StringVal: correctiveID, Valid: true},
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
			spanner.UpdateMap("CreditNotes", map[string]any{
				"CreditNoteId":    creditNoteID,
				"CorrectiveEhfId": spanner.NullString{StringVal: correctiveID, Valid: true},
			}),
		}
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderRow.OrderID, events.TopicMain, events.FiscalReceiptEvent{
			BaseEvent:       events.BaseEvent{Type: events.EventFiscalCorrectiveRequested, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:         orderRow.OrderID,
			AttemptID:       "corr-" + refundID,
			SupplierID:      orderRow.SupplierID,
			RetailerID:      orderRow.RetailerID,
			AmountMinor:     amount,
			Currency:        orderRow.Currency,
			Status:          "SUCCESS",
			FiscalReceiptID: correctiveID,
		}); err != nil {
			return err
		}
		muts = append(muts, bufferedOutboxMutations(buf, now)...)
		return txn.BufferWrite(muts)
	})
	return creditNoteID, correctiveID, err
}

func (s *Service) refundByIdempotencyKey(ctx context.Context, key string) (res RefundResult, cardAmt, cashAmt int64, found bool, err error) {
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT RefundId, OrderId, Status, AmountMinor, Currency, ProviderRef FROM Refunds WHERE IdempotencyKey = @key`,
		Params: map[string]any{"key": key},
	})
	defer iter.Stop()
	row, qerr := iter.Next()
	if qerr == iterator.Done {
		return RefundResult{}, 0, 0, false, nil
	}
	if qerr != nil {
		return RefundResult{}, 0, 0, false, qerr
	}
	var ref spanner.NullString
	if err := row.Columns(&res.RefundID, &res.OrderID, &res.Status, &res.AmountMinor, &res.Currency, &ref); err != nil {
		return RefundResult{}, 0, 0, false, err
	}
	res.ProviderRef = ref.StringVal
	legIter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT AmountMinor, IdempotencyKey FROM OrderPaymentLegs WHERE OrderId = @oid AND IdempotencyKey IN UNNEST(@keys)`,
		Params: map[string]any{"oid": res.OrderID, "keys": []string{"refund-card:" + res.RefundID, "refund-cash:" + res.RefundID}},
	})
	defer legIter.Stop()
	for {
		lr, lerr := legIter.Next()
		if lerr == iterator.Done {
			break
		}
		if lerr != nil {
			return RefundResult{}, 0, 0, false, lerr
		}
		var amt int64
		var ik string
		if err := lr.Columns(&amt, &ik); err != nil {
			return RefundResult{}, 0, 0, false, err
		}
		if strings.HasPrefix(ik, "refund-card:") {
			cardAmt += amt
		} else if strings.HasPrefix(ik, "refund-cash:") {
			cashAmt += amt
		}
	}
	switch {
	case cardAmt > 0 && cashAmt > 0:
		res.Method = RefundMethodMixed
	case cardAmt > 0:
		res.Method = RefundMethodCard
	case cashAmt > 0:
		res.Method = RefundMethodCash
	}
	return res, cardAmt, cashAmt, true, nil
}
