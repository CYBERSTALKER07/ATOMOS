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
)

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
func (s *Service) newFiscalPendingRow(orderRecord Order, paymentMethod, attemptID string) FiscalReceiptRow {
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
	return FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      FiscalProviderFake,
		Status:        FiscalAttemptPending,
		AmountMinor:   orderRecord.TotalMinor,
		Currency:      currency,
		PaymentMethod: paymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
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
		return nil // idempotent
	}
	if orderRecord.Status != StatusFiscalizing && orderRecord.Status != StatusFiscalFailed {
		s.log.InfoContext(ctx, "skip fiscal worker; order not fiscalizing",
			"order_id", orderID, "status", orderRecord.Status)
		return nil
	}

	// Prefer in-memory pending row fields if present; otherwise reconstruct from order.
	req := FiscalCreateRequest{
		AttemptID:     attemptID,
		OrderID:       orderRecord.OrderID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		AmountMinor:   orderRecord.TotalMinor,
		Currency:      orderRecord.Currency,
		PaymentMethod: "CASH",
		LineItems:     orderRecord.LineItems,
	}
	if req.Currency == "" {
		req.Currency = "UZS"
	}

	result, provErr := s.fiscalProvider().CreateReceipt(ctx, req)
	now := s.now().UTC()
	previousStatus := orderRecord.Status

	row := FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      FiscalProviderFake,
		AmountMinor:   orderRecord.TotalMinor,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if provErr != nil {
		row.Status = FiscalAttemptFailed
		row.ErrorCode = "OFD_ERROR"
		row.ErrorMessage = provErr.Error()
		orderRecord.Status = StatusFiscalFailed
		orderRecord.FiscalStatus = FiscalStatusFailed
		orderRecord.LatestFiscalAttemptID = attemptID
		orderRecord.UpdatedAt = now
		// Advance existing PENDING attempt (no second insert).
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

	attemptID := s.newID()
	row := s.newFiscalPendingRow(orderRecord, "CASH", attemptID)
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
func (s *Service) ForceCompleteOrder(ctx context.Context, claims auth.Claims, orderID, reasonCode string) (CollectCashResponse, error) {
	orderID = strings.TrimSpace(orderID)
	reasonCode = strings.TrimSpace(reasonCode)
	if orderID == "" {
		return CollectCashResponse{}, errors.New("order_id required")
	}
	if reasonCode == "" {
		return CollectCashResponse{}, ErrForceReasonRequired
	}
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin:
	default:
		return CollectCashResponse{}, ErrForceCompleteForbidden
	}

	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return CollectCashResponse{}, err
	}
	if !ok {
		return CollectCashResponse{}, ErrOrderNotFound
	}
	if orderRecord.Status == StatusCompleted {
		return CollectCashResponse{
			OrderID: orderRecord.OrderID,
			State:   orderRecord.Status,
			Amount:  orderRecord.TotalMinor,
			Currency: orderRecord.Currency,
			Message: "Order already completed",
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
		ReasonCode:    reasonCode,
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
			Reason:         "force_complete:" + reasonCode,
			ActorID:        claims.Subject,
		}); err != nil {
			return err
		}
		if err := emitOrderForceCompleted(ctx, txn, orderRecord, reasonCode, claims.Subject); err != nil {
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
