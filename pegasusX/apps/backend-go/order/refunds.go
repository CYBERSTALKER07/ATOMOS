package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

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

	RefundMethodCard = "CARD"
	RefundMethodCash = "CASH"
)

var (
	ErrRefundExceedsCaptured = errors.New("refund exceeds captured minus refunded")
	ErrRefundCreditPortion   = errors.New("credit-paid portion refunds via AR adjustment, not card/cash reversal")
	ErrRefundOrderState      = errors.New("order is not in a refundable state")
)

type RefundRequest struct {
	OrderID        string
	AmountMinor    int64 // <=0 means full remaining refundable amount
	ReasonCode     string
	ReasonText     string
	ActorID        string
	IdempotencyKey string
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
func refundableBalances(rows func(func(method string, amount int64, status string, idemKey string) bool) error) (card, cash, credit int64, err error) {
	err = rows(func(method string, amount int64, status string, idemKey string) bool {
		if status != string(PaymentStatusCaptured) {
			return true
		}
		switch method {
		case string(MethodCard):
			card += amount
		case string(MethodCash):
			cash += amount
		case string(MethodCredit):
			credit += amount
		case string(MethodRefund):
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

	// Replay: same idempotency key returns the stored refund.
	if existing, found, err := s.refundByIdempotencyKey(ctx, idemKey); err != nil {
		return RefundResult{}, err
	} else if found {
		return existing, nil
	}

	refundID := s.newID()
	now := s.now().UTC()

	var orderRow Order
	var cardBal, cashBal, creditBal int64
	var method string
	var amount int64

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"OrderId", "SupplierId", "RetailerId", "Status", "TotalMinor", "Currency"})
		if err != nil {
			return fmt.Errorf("order %s: %w", orderID, err)
		}
		var status string
		if err := row.Columns(&orderRow.OrderID, &orderRow.SupplierID, &orderRow.RetailerID, &status, &orderRow.TotalMinor, &orderRow.Currency); err != nil {
			return err
		}
		switch Status(status) {
		case StatusFiscalizing, StatusDeliveredOnCredit, StatusPendingCashCollection, StatusFiscalFailed, StatusCompleted:
		default:
			return fmt.Errorf("%w: %s", ErrRefundOrderState, status)
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
		switch {
		case amount <= cardBal:
			method = RefundMethodCard
		case amount <= cashBal:
			method = RefundMethodCash
		case amount <= cardBal+cashBal+creditBal:
			return fmt.Errorf("%w (remaining credit-covered: %d)", ErrRefundCreditPortion, creditBal)
		default:
			return fmt.Errorf("%w: requested %d, refundable card=%d cash=%d", ErrRefundExceedsCaptured, amount, cardBal, cashBal)
		}

		legKeyPrefix := "refund-card:"
		if method == RefundMethodCash {
			legKeyPrefix = "refund-cash:"
		}
		leg := PaymentLeg{
			OrderID:        orderID,
			LegID:          s.newID(),
			Method:         MethodRefund,
			AmountMinor:    amount,
			Status:         PaymentStatusPending,
			IdempotencyKey: legKeyPrefix + refundID,
			CreatedAt:      now,
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
			spanner.InsertMap("OrderPaymentLegs", map[string]any{
				"OrderId":        leg.OrderID,
				"LegId":          leg.LegID,
				"Method":         string(leg.Method),
				"AmountMinor":    leg.AmountMinor,
				"Status":         string(leg.Status),
				"IdempotencyKey": leg.IdempotencyKey,
				"CreatedAt":      now,
			}),
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

	// Synchronous confirmation — ledger never asserts money that did not move.
	var providerRef string
	var confirmErr error
	switch method {
	case RefundMethodCard:
		if s.paymentRefunder == nil {
			confirmErr = fmt.Errorf("card refund configured against order %s but no payment refunder wired", orderID)
			break
		}
		callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		providerRef, confirmErr = s.paymentRefunder.RefundCardPayment(callCtx, orderID, amount, orderRow.Currency)
		cancel()
	case RefundMethodCash:
		providerRef = "cash-ledger-reversal"
	}

	finalStatus := RefundStatusCaptured
	var outboxType string
	if confirmErr != nil {
		finalStatus = RefundStatusFailed
		outboxType = events.EventRefundFailed
	} else {
		outboxType = events.EventRefundSucceeded
	}
	if err := s.finalizeRefund(ctx, refundID, orderID, method, amount, finalStatus, providerRef, outboxType, orderRow, confirmErr); err != nil {
		return RefundResult{}, fmt.Errorf("refund %s confirmed as %s but finalization failed: %w", refundID, finalStatus, err)
	}
	result.Status = finalStatus
	result.ProviderRef = providerRef
	if confirmErr != nil {
		return result, confirmErr
	}

	// Fiscal corrective chain (post-success, never blocks the confirmed refund).
	cnID, correctiveID, cnErr := s.recordRefundCreditNote(ctx, refundID, orderRow, amount, reasonCode, actor)
	if cnErr != nil {
		s.log.ErrorContext(ctx, "refund credit note / corrective chain failed",
			"refund_id", refundID, "order_id", orderID, "err", cnErr)
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
		legKeyPrefix := "refund-card:"
		if method == RefundMethodCash {
			legKeyPrefix = "refund-cash:"
		}
		// Locate the PENDING reversal leg by its deterministic idempotency key.
		legRows := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT LegId FROM OrderPaymentLegs WHERE OrderId = @oid AND IdempotencyKey = @key`,
			Params: map[string]any{"oid": orderID, "key": legKeyPrefix + refundID},
		})
		defer legRows.Stop()
		r, err := legRows.Next()
		if err != nil {
			return fmt.Errorf("pending refund leg missing: %w", err)
		}
		var legID string
		if err := r.Columns(&legID); err != nil {
			return err
		}
		muts := []*spanner.Mutation{
			spanner.UpdateMap("Refunds", map[string]any{
				"RefundId":    refundID,
				"Status":      updates["Status"],
				"ProviderRef": updates["ProviderRef"],
				"UpdatedAt":   updates["UpdatedAt"],
			}),
			spanner.UpdateMap("OrderPaymentLegs", map[string]any{
				"OrderId":     orderID,
				"LegId":       legID,
				"Status":      legStatus,
				"ProviderRef": spanner.NullString{StringVal: providerRef, Valid: providerRef != ""},
				"CapturedAt":  spanner.NullTime{Time: now, Valid: status == RefundStatusCaptured},
			}),
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
	_, err = s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("CreditNotes", map[string]any{
			"CreditNoteId":  creditNoteID,
			"OrderId":       orderRow.OrderID,
			"Type":          "REFUND",
			"Status":        "ISSUED",
			"ReasonCode":    reasonCode,
			"ReasonText":    spanner.NullString{StringVal: "refund_id=" + refundID, Valid: true},
			"TotalNetMinor": amount,
			"TotalVatMinor": int64(0),
			"TotalGrossMinor": amount,
			"OriginalEhfId": spanner.NullString{StringVal: originalEhf, Valid: originalEhf != ""},
			"CreatedBy":     actor,
			"CreatedAt":     cnNow,
			"IssuedAt":      cnNow,
		}),
		spanner.UpdateMap("Refunds", map[string]any{
			"RefundId":     refundID,
			"CreditNoteId": spanner.NullString{StringVal: creditNoteID, Valid: true},
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
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

func (s *Service) refundByIdempotencyKey(ctx context.Context, key string) (RefundResult, bool, error) {
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT RefundId, OrderId, Status, AmountMinor, Currency, ProviderRef FROM Refunds WHERE IdempotencyKey = @key`,
		Params: map[string]any{"key": key},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return RefundResult{}, false, nil
	}
	if err != nil {
		return RefundResult{}, false, err
	}
	var res RefundResult
	var ref spanner.NullString
	if err := row.Columns(&res.RefundID, &res.OrderID, &res.Status, &res.AmountMinor, &res.Currency, &ref); err != nil {
		return RefundResult{}, false, err
	}
	res.ProviderRef = ref.StringVal
	return res, true, nil
}
