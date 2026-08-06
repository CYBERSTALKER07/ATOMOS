package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	segkafka "github.com/segmentio/kafka-go"
)

// BillingTierWorker consumes ORDER_FINALIZED events and routes them to the MeterWorker.
type BillingTierWorker struct {
	MeterWorker *billing.MeterWorker
}

// NewBillingTierWorker initializes a new BillingTierWorker.
func NewBillingTierWorker(meterWorker *billing.MeterWorker) *BillingTierWorker {
	return &BillingTierWorker{
		MeterWorker: meterWorker,
	}
}

// HandleEvent adapts BillingTierWorker to the Kafka EventHandler signature.
func (w *BillingTierWorker) HandleEvent(ctx context.Context, msg segkafka.Message) error {
	return w.HandleMessage(ctx, msg.Value)
}

// orderFinalizedBillingEvent matches the live ORDER_FINALIZED emit shape from order/service.go
// plus legacy amount / total_minor fields.
type orderFinalizedBillingEvent struct {
	Type        string `json:"type"`
	OrderID     string `json:"order_id"`
	SupplierID  string `json:"supplier_id"`
	AmountMinor int64  `json:"amount_minor"`
	TotalMinor  int64  `json:"total_minor"`
	Amount      float64 `json:"amount"`
	Total       struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"total"`
}

// HandleMessage processes incoming Kafka messages for the billing tier worker.
func (w *BillingTierWorker) HandleMessage(ctx context.Context, msg []byte) error {
	if w == nil || w.MeterWorker == nil {
		return nil
	}
	var event orderFinalizedBillingEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal billing event: %v", err)
		return err
	}

	if event.Type != events.EventOrderFinalized {
		return nil
	}
	orderID := strings.TrimSpace(event.OrderID)
	supplierID := strings.TrimSpace(event.SupplierID)
	if orderID == "" || supplierID == "" {
		log.Printf("billing ORDER_FINALIZED missing order_id or supplier_id; skipping")
		return nil
	}
	amount := billing.ResolveMeterAmountMajor(event.AmountMinor, event.Total.Amount, event.TotalMinor, event.Amount)
	return w.MeterWorker.ProcessOrderFinalized(ctx, orderID, amount, supplierID)
}
