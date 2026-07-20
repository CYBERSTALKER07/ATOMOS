package order

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	kafka "github.com/segmentio/kafka-go"
)

type consumerRepoStub struct {
	orders  map[string]Order
	updated []Order
}

func (s *consumerRepoStub) CreateOrder(context.Context, *Order, func(outbox.TxnBuffer) error) error {
	return nil
}
func (s *consumerRepoStub) UpdateOrder(_ context.Context, o Order, _ []DeliveryProofArtifact, _ func(outbox.TxnBuffer) error) error {
	s.updated = append(s.updated, o)
	return nil
}
func (s *consumerRepoStub) GetOrder(_ context.Context, orderID string) (Order, bool, error) {
	o, ok := s.orders[orderID]
	return o, ok, nil
}
func (s *consumerRepoStub) GetFiscalAttempt(context.Context, string, string) (FiscalReceiptRow, bool, error) {
	return FiscalReceiptRow{}, false, nil
}
func (s *consumerRepoStub) CountFiscalAttemptsByStatus(context.Context, string, string) (int64, error) {
	return 0, nil
}
func (s *consumerRepoStub) ListRetailerOrders(context.Context, string, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListWarehouseOrdersByDeliveryWindow(context.Context, string, time.Time, time.Time, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListDueAutoConfirmOrders(context.Context, time.Time, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListManifestOrders(context.Context, string) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListWarehousePreorders(context.Context, string, int, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListOrdersForStockCommitment(context.Context, string, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListBackorderedOrders(context.Context, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) ClearBackorder(context.Context, string, func(outbox.TxnBuffer) error) error {
	return nil
}
func (s *consumerRepoStub) ListOrdersByStatus(context.Context, string, string, int) ([]Order, error) {
	return nil, nil
}
func (s *consumerRepoStub) CreateConditionReport(context.Context, ConditionReport, func(outbox.TxnBuffer) error) error {
	return nil
}

func (s *consumerRepoStub) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	return nil, nil
}
func (s *consumerRepoStub) ListConditionReports(context.Context, string) ([]ConditionReport, error) {
	return nil, nil
}

func TestEventConsumer_PaymentFailedAwaitingPayment(t *testing.T) {
	repo := &consumerRepoStub{orders: map[string]Order{
		"ord-1": {OrderID: "ord-1", RetailerID: "ret-1", SupplierID: "sup-1", Status: StatusAwaitingPayment},
	}}
	svc := &Service{repo: repo, log: slog.Default()}
	consumer := NewEventConsumer(svc, slog.Default())

	payload, _ := json.Marshal(events.FinanceEvent{
		BaseEvent: events.BaseEvent{Type: events.EventPaymentFailed},
		OrderID:   "ord-1",
		Gateway:   "STRIPE",
		Source:    "test",
	})
	if err := consumer.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
}

func TestEventConsumer_PaymentFailedSkipsCompleted(t *testing.T) {
	repo := &consumerRepoStub{orders: map[string]Order{
		"ord-1": {OrderID: "ord-1", Status: StatusCompleted},
	}}
	svc := &Service{repo: repo, log: slog.Default()}
	consumer := NewEventConsumer(svc, slog.Default())

	payload, _ := json.Marshal(events.FinanceEvent{
		BaseEvent: events.BaseEvent{Type: events.EventPaymentFailed},
		OrderID:   "ord-1",
	})
	if err := consumer.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
}

func TestEventConsumer_PaymentClearedOnCancelledOrderGoesToReconciliation(t *testing.T) {
	repo := &consumerRepoStub{orders: map[string]Order{
		"ord-1": {OrderID: "ord-1", RetailerID: "ret-1", SupplierID: "sup-1", Status: StatusCancelled},
	}}
	svc := &Service{repo: repo, log: slog.Default(), now: time.Now}
	consumer := NewEventConsumer(svc, slog.Default())

	payload, _ := json.Marshal(map[string]any{
		"type":     events.EventPaymentCleared,
		"order_id": "ord-1",
		"gateway":  "GLOBALPAY",
	})
	if err := consumer.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 UpdateOrder call, got %d", len(repo.updated))
	}
	if repo.updated[0].Status != StatusReconciliationRequired {
		t.Fatalf("expected RECONCILIATION_REQUIRED, got %s", repo.updated[0].Status)
	}
}

func TestEventConsumer_PaymentClearedAwaitingKeepsReadVersion(t *testing.T) {
	repo := &consumerRepoStub{orders: map[string]Order{
		"ord-1": {OrderID: "ord-1", RetailerID: "ret-1", SupplierID: "sup-1", Status: StatusAwaitingPayment, Version: 7},
	}}
	svc := &Service{repo: repo, log: slog.Default(), now: time.Now}
	if err := svc.SettleExternalPayment(context.Background(), "ord-1", "GLOBALPAY"); err != nil {
		t.Fatalf("SettleExternalPayment: %v", err)
	}
	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 UpdateOrder call, got %d", len(repo.updated))
	}
	// UpdateOrder performs optimistic concurrency against the stored version and
	// increments it itself; the service must pass through the version it read.
	if repo.updated[0].Version != 7 {
		t.Fatalf("expected read version 7 passed to UpdateOrder, got %d", repo.updated[0].Version)
	}
	// ADR-009: external clear enters FISCALIZING (not COMPLETED until OFD success).
	if repo.updated[0].Status != StatusFiscalizing {
		t.Fatalf("expected FISCALIZING, got %s", repo.updated[0].Status)
	}
	if len(repo.updated[0].PendingFiscalReceipts) != 1 {
		t.Fatalf("expected 1 pending fiscal receipt, got %d", len(repo.updated[0].PendingFiscalReceipts))
	}
}

func TestEventConsumer_DeliveryDisputed(t *testing.T) {
	repo := &consumerRepoStub{orders: map[string]Order{
		"ord-1": {OrderID: "ord-1", RetailerID: "ret-1", SupplierID: "sup-1", Status: StatusCompleted},
	}}
	svc := &Service{repo: repo, log: slog.Default()}
	consumer := NewEventConsumer(svc, slog.Default())

	payload, _ := json.Marshal(events.OrderEvent{
		BaseEvent: events.BaseEvent{Type: events.EventDeliveryDisputed},
		OrderID:   "ord-1",
		Reason:    "chargeback_recorded",
		Action:    "payment.chargeback",
	})
	if err := consumer.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
}
