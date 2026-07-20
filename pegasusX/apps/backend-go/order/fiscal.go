package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Fiscal attempt / order denorm statuses (app-enforced).
const (
	FiscalAttemptPending      = "PENDING"
	FiscalAttemptSuccess      = "SUCCESS"
	FiscalAttemptFailed       = "FAILED"
	FiscalAttemptForceSkipped = "FORCE_SKIPPED"

	FiscalStatusNone         = "NONE"
	FiscalStatusPending      = "PENDING"
	FiscalStatusSuccess      = "SUCCESS"
	FiscalStatusFailed       = "FAILED"
	FiscalStatusForceSkipped = "FORCE_SKIPPED"

	FiscalProviderFake    = "FAKE"
	FiscalProviderMySoliq = "MY_SOLIQ"

	// FiscalOFDTimeout is the hard timeout per OFD call (P0 T8/T9).
	FiscalOFDTimeout = 8 * time.Second
	// FiscalMaxFailedAttempts before order stays in FISCAL_FAILED requiring retry/force.
	FiscalMaxFailedAttempts = 3
)

// Force-complete reason codes (audited enum — ADR-009 Phase 4).
const (
	ForceReasonOFDDown       = "OFD_DOWN"
	ForceReasonOFDTimeout    = "OFD_TIMEOUT"
	ForceReasonOpsEscalation = "OPS_ESCALATION"
	ForceReasonTaxExempt     = "TAX_EXEMPT_POLICY"
	ForceReasonOther         = "OTHER"
)

// ValidForceReasonCodes is the closed set accepted by force-complete.
var ValidForceReasonCodes = map[string]struct{}{
	ForceReasonOFDDown:       {},
	ForceReasonOFDTimeout:    {},
	ForceReasonOpsEscalation: {},
	ForceReasonTaxExempt:     {},
	ForceReasonOther:         {},
}

func NormalizeForceReasonCode(code string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if c == "" {
		return "", ErrForceReasonRequired
	}
	if _, ok := ValidForceReasonCodes[c]; !ok {
		return "", fmt.Errorf("%w: %s", ErrForceReasonInvalid, code)
	}
	return c, nil
}

// isTerminalMoneyStatus blocks late webhooks inventing new money (P0 T7).
func isTerminalMoneyStatus(st Status) bool {
	switch st {
	case StatusCompleted, StatusCancelled, StatusReconciliationRequired:
		return true
	default:
		return false
	}
}

// FiscalReceiptRow is one immutable OFD attempt (supplier-scoped leg).
type FiscalReceiptRow struct {
	OrderID           string
	AttemptID         string
	SupplierID        string
	RetailerID        string
	Provider          string
	Status            string
	FiscalReceiptID   string
	FiscalQR          string
	AmountMinor       int64
	Currency          string
	PaymentMethod     string
	ProviderPayload   []byte
	ErrorCode         string
	ErrorMessage      string
	ReasonCode        string
	ActorID           string
	TraceID           string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// FiscalProvider calls the OFD / tax receipt API asynchronously from a worker.
type FiscalProvider interface {
	CreateReceipt(ctx context.Context, req FiscalCreateRequest) (FiscalCreateResult, error)
}

// FiscalCreateRequest is the provider input (integer Tiyin only).
type FiscalCreateRequest struct {
	AttemptID     string
	OrderID       string
	SupplierID    string
	RetailerID    string
	AmountMinor   int64
	Currency      string
	PaymentMethod string
	LineItems     []LineItem
}

// FiscalCreateResult is a successful OFD response.
type FiscalCreateResult struct {
	FiscalReceiptID string
	FiscalQR        string
	RawPayload      []byte
}

// FakeFiscalProvider succeeds unless OrderID contains "fiscal-fail" (SSMR hook).
type FakeFiscalProvider struct{}

func (FakeFiscalProvider) CreateReceipt(_ context.Context, req FiscalCreateRequest) (FiscalCreateResult, error) {
	if strings.Contains(strings.ToLower(req.OrderID), "fiscal-fail") {
		return FiscalCreateResult{}, fmt.Errorf("fake_ofd_rejected: order_id=%s", req.OrderID)
	}
	id := "FAKE-RCPT-" + req.AttemptID
	if len(id) > 64 {
		id = id[:64]
	}
	return FiscalCreateResult{
		FiscalReceiptID: id,
		FiscalQR:        "https://fake.ofd.local/qr/" + req.AttemptID,
		RawPayload:      []byte(`{"provider":"FAKE","ok":true}`),
	}, nil
}

func defaultFiscalProvider() FiscalProvider {
	// Real MY_SOLIQ adapter ships behind FISCAL_PROVIDER env later.
	return FakeFiscalProvider{}
}

func (s *Service) fiscalProvider() FiscalProvider {
	// Future: s.ofd != nil → return s.ofd
	return defaultFiscalProvider()
}

// newFiscalPendingRow builds a PENDING supplier-leg attempt for capture txn.
// amountMinor is the fiscalized cash/card amount (received cash or order total).
func (s *Service) newFiscalPendingRow(orderRecord Order, paymentMethod, attemptID string, amountMinor int64) FiscalReceiptRow {
	now := time.Now().UTC()
	if s != nil && s.now != nil {
		now = s.now().UTC()
	}
	if attemptID == "" {
		if s != nil && s.newID != nil {
			attemptID = s.newID()
		} else {
			attemptID = defaultOrderID()
		}
	}
	currency := strings.TrimSpace(orderRecord.Currency)
	if currency == "" {
		currency = "UZS"
	}
	if amountMinor < 0 {
		amountMinor = 0
	}
	if amountMinor == 0 {
		amountMinor = orderRecord.TotalMinor
	}
	return FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      FiscalProviderFake,
		Status:        FiscalAttemptPending,
		AmountMinor:   amountMinor,
		Currency:      currency,
		PaymentMethod: paymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func emitCashVariance(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, driverID string, expected, received int64, note string) error {
	currency := strings.TrimSpace(orderRecord.Currency)
	if currency == "" {
		currency = "UZS"
	}
	shortfall := expected - received
	if shortfall < 0 {
		shortfall = 0
	}
	overage := received - expected
	if overage < 0 {
		overage = 0
	}
	if shortfall == 0 && overage == 0 {
		return nil
	}
	eventType := events.EventCashShortfall
	if overage > 0 {
		eventType = events.EventCashOverage
	}
	ts := orderRecord.UpdatedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CashVarianceEvent{
		BaseEvent:      events.BaseEvent{Type: eventType, Timestamp: ts.Format(time.RFC3339Nano)},
		OrderID:        orderRecord.OrderID,
		SupplierID:     orderRecord.SupplierID,
		RetailerID:     orderRecord.RetailerID,
		DriverID:       driverID,
		ExpectedMinor:  expected,
		ReceivedMinor:  received,
		ShortfallMinor: shortfall,
		OverageMinor:   overage,
		Currency:       currency,
		Note:           strings.TrimSpace(note),
	})
}

func emitFiscalReceiptRequested(ctx context.Context, txn outbox.TxnBuffer, row FiscalReceiptRow) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, row.OrderID, events.TopicMain, events.FiscalReceiptEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventFiscalReceiptRequested, Timestamp: row.CreatedAt.Format(time.RFC3339Nano)},
		OrderID:       row.OrderID,
		AttemptID:     row.AttemptID,
		SupplierID:    row.SupplierID,
		RetailerID:    row.RetailerID,
		AmountMinor:   row.AmountMinor,
		Currency:      row.Currency,
		PaymentMethod: row.PaymentMethod,
		Provider:      row.Provider,
		Status:        row.Status,
		TraceID:       row.TraceID,
	})
}

func emitFiscalReceiptSucceeded(ctx context.Context, txn outbox.TxnBuffer, row FiscalReceiptRow) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, row.OrderID, events.TopicMain, events.FiscalReceiptEvent{
		BaseEvent:       events.BaseEvent{Type: events.EventFiscalReceiptSucceeded, Timestamp: row.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:         row.OrderID,
		AttemptID:       row.AttemptID,
		SupplierID:      row.SupplierID,
		RetailerID:      row.RetailerID,
		AmountMinor:     row.AmountMinor,
		Currency:        row.Currency,
		PaymentMethod:   row.PaymentMethod,
		Provider:        row.Provider,
		Status:          row.Status,
		FiscalReceiptID: row.FiscalReceiptID,
		FiscalQR:        row.FiscalQR,
		TraceID:         row.TraceID,
	})
}

func emitFiscalReceiptFailed(ctx context.Context, txn outbox.TxnBuffer, row FiscalReceiptRow) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, row.OrderID, events.TopicMain, events.FiscalReceiptEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventFiscalReceiptFailed, Timestamp: row.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:       row.OrderID,
		AttemptID:     row.AttemptID,
		SupplierID:    row.SupplierID,
		RetailerID:    row.RetailerID,
		AmountMinor:   row.AmountMinor,
		Currency:      row.Currency,
		PaymentMethod: row.PaymentMethod,
		Provider:      row.Provider,
		Status:        row.Status,
		ErrorCode:     row.ErrorCode,
		ErrorMessage:  row.ErrorMessage,
		TraceID:       row.TraceID,
	})
}

func emitOrderForceCompleted(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, reasonCode, actorID string) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.OrderForceCompletedEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventOrderForceCompleted, Timestamp: orderRecord.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:    orderRecord.OrderID,
		SupplierID: orderRecord.SupplierID,
		RetailerID: orderRecord.RetailerID,
		ReasonCode: reasonCode,
		ActorID:    actorID,
	})
}

// emitPaymentCaptureFiscal wires PAYMENT_CLEARED + FISCAL_RECEIPT_REQUESTED for a capture txn.
// Does NOT emit ORDER_FINALIZED (ADR-009).
func emitPaymentCaptureFiscal(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, row FiscalReceiptRow, paymentMethod string) error {
	if err := emitPaymentCleared(ctx, txn, orderRecord, paymentMethod); err != nil {
		return err
	}
	return emitFiscalReceiptRequested(ctx, txn, row)
}

// ApplyFiscalWorkerResult is invoked by the order event consumer on FISCAL_RECEIPT_REQUESTED.
// P0 guarantees:
//   - Idempotent: SUCCESS attempt never re-calls OFD
//   - Hard timeout per OFD call
//   - After max failed attempts, order is FISCAL_FAILED
func (s *Service) ApplyFiscalWorkerResult(ctx context.Context, orderID, attemptID string) error {
	orderID = strings.TrimSpace(orderID)
	attemptID = strings.TrimSpace(attemptID)
	if orderID == "" || attemptID == "" {
		return nil
	}

	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOrderNotFound
	}
	if orderRecord.Status == StatusCompleted {
		return nil // terminal — never re-fiscalize
	}
	if orderRecord.Status != StatusFiscalizing && orderRecord.Status != StatusFiscalFailed {
		s.log.InfoContext(ctx, "skip fiscal worker; order not fiscalizing",
			"order_id", orderID, "status", orderRecord.Status)
		return nil
	}

	// Idempotency: if attempt already SUCCESS, complete order if needed without OFD.
	if existing, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, attemptID); gErr == nil && found {
		if existing.Status == FiscalAttemptSuccess {
			if orderRecord.Status != StatusCompleted {
				return s.completeOrderFromExistingFiscalSuccess(ctx, orderRecord, existing)
			}
			return nil
		}
		if existing.Status == FiscalAttemptForceSkipped {
			return nil
		}
	}

	amountMinor := orderRecord.TotalMinor
	payMethod := "CASH"
	if existing, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, attemptID); gErr == nil && found {
		if existing.AmountMinor > 0 {
			amountMinor = existing.AmountMinor
		}
		if existing.PaymentMethod != "" {
			payMethod = existing.PaymentMethod
		}
	}

	req := FiscalCreateRequest{
		AttemptID:     attemptID,
		OrderID:       orderRecord.OrderID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		AmountMinor:   amountMinor,
		Currency:      orderRecord.Currency,
		PaymentMethod: payMethod,
		LineItems:     orderRecord.LineItems,
	}
	if req.Currency == "" {
		req.Currency = "UZS"
	}

	ofdCtx, cancel := context.WithTimeout(ctx, FiscalOFDTimeout)
	defer cancel()
	result, provErr := s.fiscalProvider().CreateReceipt(ofdCtx, req)
	if provErr == nil && ofdCtx.Err() != nil {
		provErr = ofdCtx.Err()
	}
	if provErr != nil && errors.Is(ofdCtx.Err(), context.DeadlineExceeded) {
		provErr = fmt.Errorf("ofd_timeout: %w", ofdCtx.Err())
	}

	now := s.now().UTC()
	previousStatus := orderRecord.Status

	row := FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      FiscalProviderFake,
		AmountMinor:   amountMinor,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if provErr != nil {
		row.Status = FiscalAttemptFailed
		if errors.Is(ofdCtx.Err(), context.DeadlineExceeded) || strings.Contains(provErr.Error(), "ofd_timeout") {
			row.ErrorCode = "OFD_TIMEOUT"
		} else {
			row.ErrorCode = "OFD_ERROR"
		}
		row.ErrorMessage = provErr.Error()
		orderRecord.Status = StatusFiscalFailed
		orderRecord.FiscalStatus = FiscalStatusFailed
		orderRecord.LatestFiscalAttemptID = attemptID
		orderRecord.UpdatedAt = now
		orderRecord.FiscalReceiptUpdate = &row

		err = s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
			if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
				Order:          orderRecord,
				PreviousStatus: previousStatus,
				Reason:         "fiscal_failed",
			}); err != nil {
				return err
			}
			return emitFiscalReceiptFailed(ctx, txn, row)
		})
		if err != nil {
			return fmt.Errorf("persist fiscal failure: %w", err)
		}
		s.afterOrderMutation(ctx, orderRecord)
		return nil
	}

	row.Status = FiscalAttemptSuccess
	row.FiscalReceiptID = result.FiscalReceiptID
	row.FiscalQR = result.FiscalQR
	row.ProviderPayload = result.RawPayload
	orderRecord.Status = StatusCompleted
	orderRecord.FiscalStatus = FiscalStatusSuccess
	orderRecord.LatestFiscalReceiptID = result.FiscalReceiptID
	orderRecord.LatestFiscalAttemptID = attemptID
	orderRecord.FiscalizedAt = &now
	orderRecord.UpdatedAt = now
	orderRecord.FiscalReceiptUpdate = &row

	err = s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "fiscal_succeeded",
		}); err != nil {
			return err
		}
		if err := emitFiscalReceiptSucceeded(ctx, txn, row); err != nil {
			return err
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return fmt.Errorf("persist fiscal success: %w", err)
	}
	s.afterOrderMutation(ctx, orderRecord)
	if orderRecord.ManifestID != "" {
		if mErr := s.tryCompleteManifest(ctx, orderRecord.ManifestID); mErr != nil {
			s.log.ErrorContext(ctx, "manifest complete after fiscal failed", "manifest_id", orderRecord.ManifestID, "err", mErr)
		}
	}
	return nil
}

func (s *Service) completeOrderFromExistingFiscalSuccess(ctx context.Context, orderRecord Order, existing FiscalReceiptRow) error {
	if orderRecord.Status == StatusCompleted {
		return nil
	}
	now := s.now().UTC()
	previousStatus := orderRecord.Status
	orderRecord.Status = StatusCompleted
	orderRecord.FiscalStatus = FiscalStatusSuccess
	orderRecord.LatestFiscalReceiptID = existing.FiscalReceiptID
	orderRecord.LatestFiscalAttemptID = existing.AttemptID
	orderRecord.FiscalizedAt = &now
	orderRecord.UpdatedAt = now
	err := s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "fiscal_succeeded_idempotent",
		}); err != nil {
			return err
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return err
	}
	s.afterOrderMutation(ctx, orderRecord)
	return nil
}

// RetryFiscal creates a new attempt when order is FISCAL_FAILED.
func (s *Service) RetryFiscal(ctx context.Context, claims auth.Claims, orderID string) (CollectCashResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return CollectCashResponse{}, errors.New("order_id required")
	}
	switch claims.Role {
	case auth.RoleDriver, auth.RoleAdmin, auth.RoleWarehouseAdmin:
	default:
		return CollectCashResponse{}, ErrOrderForbidden
	}

	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return CollectCashResponse{}, err
	}
	if !ok {
		return CollectCashResponse{}, ErrOrderNotFound
	}
	if orderRecord.Status != StatusFiscalFailed {
		return CollectCashResponse{}, ErrFiscalNotFailed
	}
	if claims.Role == auth.RoleDriver {
		if strings.TrimSpace(orderRecord.DriverID) == "" || strings.TrimSpace(orderRecord.DriverID) != strings.TrimSpace(claims.Subject) {
			return CollectCashResponse{}, ErrOrderForbidden
		}
	}

	// Retry budget: soft signal only (order already FISCAL_FAILED).
	if failedN, cErr := s.repo.CountFiscalAttemptsByStatus(ctx, orderID, FiscalAttemptFailed); cErr == nil && failedN >= FiscalMaxFailedAttempts {
		s.log.InfoContext(ctx, "fiscal retry after max failures", "order_id", orderID, "failed_attempts", failedN)
	}
	// Preserve amount + payment method from last attempt (cash shortfall / card path).
	amountMinor := orderRecord.TotalMinor
	payMethod := "CASH"
	if orderRecord.LatestFiscalAttemptID != "" {
		if prev, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, orderRecord.LatestFiscalAttemptID); gErr == nil && found {
			if prev.AmountMinor > 0 {
				amountMinor = prev.AmountMinor
			}
			if strings.TrimSpace(prev.PaymentMethod) != "" && prev.PaymentMethod != "FORCE" {
				payMethod = prev.PaymentMethod
			}
		}
	}
	attemptID := s.newID()
	row := s.newFiscalPendingRow(orderRecord, payMethod, attemptID, amountMinor)
	previousStatus := orderRecord.Status
	orderRecord.Status = StatusFiscalizing
	orderRecord.FiscalStatus = FiscalStatusPending
	orderRecord.LatestFiscalAttemptID = attemptID
	orderRecord.UpdatedAt = s.now().UTC()
	orderRecord.PendingFiscalReceipts = []FiscalReceiptRow{row}

	err = s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Claims:         claims,
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "fiscal_retry",
			ActorID:        claims.Subject,
		}); err != nil {
			return err
		}
		return emitFiscalReceiptRequested(ctx, txn, row)
	})
	if err != nil {
		return CollectCashResponse{}, err
	}
	s.afterOrderMutation(ctx, orderRecord)

	return CollectCashResponse{
		OrderID:   orderRecord.OrderID,
		State:     orderRecord.Status,
		Amount:    orderRecord.TotalMinor,
		Currency:  orderRecord.Currency,
		Message:   "Fiscal retry requested.",
		AttemptID: attemptID,
	}, nil
}

// ForceCompleteOrder completes with FORCE_SKIPPED fiscal audit (ADMIN / WAREHOUSE_ADMIN).
// Rejects when fiscal already SUCCESS (P0 R1/R2).
func (s *Service) ForceCompleteOrder(ctx context.Context, claims auth.Claims, orderID, reasonCode string) (CollectCashResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return CollectCashResponse{}, errors.New("order_id required")
	}
	normalizedReason, err := NormalizeForceReasonCode(reasonCode)
	if err != nil {
		return CollectCashResponse{}, err
	}
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin:
	default:
		return CollectCashResponse{}, ErrForceCompleteForbidden
	}

	orderRecord, ok, getErr := s.repo.GetOrder(ctx, orderID)
	if getErr != nil {
		return CollectCashResponse{}, getErr
	}
	if !ok {
		return CollectCashResponse{}, ErrOrderNotFound
	}
	// Never force-skip when a real fiscal SUCCESS exists.
	if orderRecord.FiscalStatus == FiscalStatusSuccess {
		return CollectCashResponse{}, ErrFiscalAlreadySucceeded
	}
	if orderRecord.LatestFiscalAttemptID != "" {
		if existing, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, orderRecord.LatestFiscalAttemptID); gErr == nil && found && existing.Status == FiscalAttemptSuccess {
			return CollectCashResponse{}, ErrFiscalAlreadySucceeded
		}
	}
	if orderRecord.Status == StatusCompleted && orderRecord.FiscalStatus != FiscalStatusForceSkipped && orderRecord.FiscalStatus != FiscalStatusFailed && orderRecord.FiscalStatus != FiscalStatusPending {
		// Completed with SUCCESS-like fiscal — refuse silent force.
		if orderRecord.FiscalStatus == FiscalStatusSuccess || orderRecord.LatestFiscalReceiptID != "" {
			return CollectCashResponse{}, ErrFiscalAlreadySucceeded
		}
		return CollectCashResponse{
			OrderID:  orderRecord.OrderID,
			State:    orderRecord.Status,
			Amount:   orderRecord.TotalMinor,
			Currency: orderRecord.Currency,
			Message:  "Order already completed",
		}, nil
	}
	if orderRecord.Status != StatusFiscalFailed && orderRecord.Status != StatusFiscalizing {
		return CollectCashResponse{}, fmt.Errorf("%w: must be FISCAL_FAILED or FISCALIZING (current %s)", ErrInvalidStatusTransition, orderRecord.Status)
	}

	now := s.now().UTC()
	attemptID := s.newID()
	row := FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      FiscalProviderFake,
		Status:        FiscalAttemptForceSkipped,
		AmountMinor:   orderRecord.TotalMinor,
		Currency:      orderRecord.Currency,
		PaymentMethod: "FORCE",
		ReasonCode:    normalizedReason,
		ActorID:       claims.Subject,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if row.Currency == "" {
		row.Currency = "UZS"
	}

	previousStatus := orderRecord.Status
	orderRecord.Status = StatusCompleted
	orderRecord.FiscalStatus = FiscalStatusForceSkipped
	orderRecord.LatestFiscalAttemptID = attemptID
	orderRecord.FiscalizedAt = &now
	orderRecord.UpdatedAt = now
	orderRecord.PendingFiscalReceipts = []FiscalReceiptRow{row}

	err = s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Claims:         claims,
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "force_complete:" + normalizedReason,
			ActorID:        claims.Subject,
		}); err != nil {
			return err
		}
		if err := emitOrderForceCompleted(ctx, txn, orderRecord, normalizedReason, claims.Subject); err != nil {
			return err
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return CollectCashResponse{}, err
	}
	s.afterOrderMutation(ctx, orderRecord)
	if orderRecord.ManifestID != "" {
		_ = s.tryCompleteManifest(ctx, orderRecord.ManifestID)
	}

	return CollectCashResponse{
		OrderID:   orderRecord.OrderID,
		State:     orderRecord.Status,
		Amount:    orderRecord.TotalMinor,
		Currency:  orderRecord.Currency,
		Message:   "Force-completed with fiscal FORCE_SKIPPED.",
		AttemptID: attemptID,
	}, nil
}
