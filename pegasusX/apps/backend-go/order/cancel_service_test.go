package order

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type mockCreditRepoForCancel struct {
	releaseErr error
}

func (m *mockCreditRepoForCancel) GetProfile(ctx context.Context, retailerID, supplierID string) (credit.Profile, bool, error) {
	return credit.Profile{}, false, nil
}
func (m *mockCreditRepoForCancel) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]credit.Profile, error) {
	return nil, nil
}
func (m *mockCreditRepoForCancel) UpsertProfile(ctx context.Context, p credit.Profile, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockCreditRepoForCancel) AdjustBalance(ctx context.Context, retailerID, supplierID string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockCreditRepoForCancel) GetScoresForRetailers(ctx context.Context, retailerIDs []string) (map[string]credit.RetailerCreditScore, error) {
	return nil, nil
}
func (m *mockCreditRepoForCancel) ReserveOrder(ctx context.Context, res credit.OrderReservation, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockCreditRepoForCancel) ReleaseOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	return m.releaseErr
}
func (m *mockCreditRepoForCancel) ConvertOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockCreditRepoForCancel) GetOrderReservation(ctx context.Context, orderID string) (credit.OrderReservation, bool, error) {
	return credit.OrderReservation{}, false, nil
}
func (m *mockCreditRepoForCancel) ListDelinquentCandidates(ctx context.Context, limit int) ([]credit.Profile, error) {
	return nil, nil
}

func TestCancelOrderWithReason_CreditReleaseFailureEnqueuesCompensation(t *testing.T) {
	creditRepo := &mockCreditRepoForCancel{
		releaseErr: errors.New("simulated credit spanner timeout"),
	}
	creditSvc := credit.NewService(creditRepo)

	orderRepo := &testRepo{}
	svc := NewService(ServiceConfig{
		Repo:         orderRepo,
		Credit:       creditSvc,
		SupplierID:   "sup-cancel-1",
		SupplierName: "Cancel Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	var capturedEvents []outbox.Event
	CompensationOutboxHook = func(ctx context.Context, e outbox.Event) error {
		capturedEvents = append(capturedEvents, e)
		return nil
	}
	defer func() { CompensationOutboxHook = nil }()

	current := &Order{
		OrderID:            "ord-cancel-credit-1",
		SupplierID:         "sup-cancel-1",
		RetailerID:         "ret-cancel-1",
		Status:             StatusPending,
		ConfirmationStatus: ConfirmationStatusConfirmed,
		TotalMinor:         50000,
		Currency:           "UZS",
		Version:            1,
		UpdatedAt:          time.Now().UTC(),
	}

	ctx := context.Background()
	resp, err := svc.cancelOrderWithReason(ctx, current, "retailer-1", "RETAILER", "customer_requested_cancel", events.EventPreOrderCancelled)
	if err != nil {
		t.Fatalf("expected cancelOrderWithReason to succeed, got %v", err)
	}
	if resp.Status != StatusCancelled {
		t.Errorf("expected response status CANCELLED, got %s", resp.Status)
	}
	if current.Status != StatusCancelled {
		t.Errorf("expected order status CANCELLED, got %s", current.Status)
	}

	// Verify compensating outbox event was enqueued
	if len(capturedEvents) != 1 {
		t.Fatalf("expected 1 compensating outbox event, got %d", len(capturedEvents))
	}
	evt := capturedEvents[0]
	if evt.AggregateID != "ord-cancel-credit-1" {
		t.Errorf("expected AggregateID 'ord-cancel-credit-1', got %s", evt.AggregateID)
	}
	if evt.SupplierID != "sup-cancel-1" {
		t.Errorf("expected SupplierID 'sup-cancel-1', got %s", evt.SupplierID)
	}

	var payload CreditReleaseCompensationPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}
	if payload.OrderID != "ord-cancel-credit-1" {
		t.Errorf("expected payload OrderID 'ord-cancel-credit-1', got %s", payload.OrderID)
	}
	if payload.SupplierID != "sup-cancel-1" {
		t.Errorf("expected payload SupplierID 'sup-cancel-1', got %s", payload.SupplierID)
	}
	if payload.RetailerID != "ret-cancel-1" {
		t.Errorf("expected payload RetailerID 'ret-cancel-1', got %s", payload.RetailerID)
	}
	if payload.Reason != "customer_requested_cancel" {
		t.Errorf("expected payload Reason 'customer_requested_cancel', got %s", payload.Reason)
	}
	if payload.Error != "simulated credit spanner timeout" {
		t.Errorf("expected payload Error 'simulated credit spanner timeout', got %s", payload.Error)
	}
}

func TestCancelOrderWithReason_CreditReleaseSuccessNoCompensation(t *testing.T) {
	creditRepo := &mockCreditRepoForCancel{
		releaseErr: nil, // success
	}
	creditSvc := credit.NewService(creditRepo)

	orderRepo := &testRepo{}
	svc := NewService(ServiceConfig{
		Repo:         orderRepo,
		Credit:       creditSvc,
		SupplierID:   "sup-cancel-2",
		SupplierName: "Cancel Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	var capturedEvents []outbox.Event
	CompensationOutboxHook = func(ctx context.Context, e outbox.Event) error {
		capturedEvents = append(capturedEvents, e)
		return nil
	}
	defer func() { CompensationOutboxHook = nil }()

	current := &Order{
		OrderID:            "ord-cancel-credit-2",
		SupplierID:         "sup-cancel-2",
		RetailerID:         "ret-cancel-2",
		Status:             StatusPending,
		ConfirmationStatus: ConfirmationStatusConfirmed,
		TotalMinor:         20000,
		Currency:           "UZS",
		Version:            1,
		UpdatedAt:          time.Now().UTC(),
	}

	ctx := context.Background()
	resp, err := svc.cancelOrderWithReason(ctx, current, "retailer-2", "RETAILER", "not_needed_anymore", events.EventPreOrderCancelled)
	if err != nil {
		t.Fatalf("expected cancelOrderWithReason to succeed, got %v", err)
	}
	if resp.Status != StatusCancelled {
		t.Errorf("expected response status CANCELLED, got %s", resp.Status)
	}

	// Verify no compensating outbox event was enqueued
	if len(capturedEvents) != 0 {
		t.Errorf("expected 0 compensating outbox events on success, got %d", len(capturedEvents))
	}
}
