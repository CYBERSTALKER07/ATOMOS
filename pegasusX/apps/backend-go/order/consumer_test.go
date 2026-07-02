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
	orders map[string]Order
}

func (s *consumerRepoStub) CreateOrder(context.Context, *Order, func(outbox.TxnBuffer) error) error {
	return nil
}
func (s *consumerRepoStub) UpdateOrder(context.Context, Order, []DeliveryProofArtifact, func(outbox.TxnBuffer) error) error {
	return nil
}
func (s *consumerRepoStub) GetOrder(_ context.Context, orderID string) (Order, bool, error) {
	o, ok := s.orders[orderID]
	return o, ok, nil
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
