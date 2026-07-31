package kafka

import (
	"context"
	"encoding/json"
	"log"

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

// HandleMessage processes incoming Kafka messages for the billing tier worker.
func (w *BillingTierWorker) HandleMessage(ctx context.Context, msg []byte) error {
	if w == nil || w.MeterWorker == nil {
		return nil
	}
	var event struct {
		Type       string  `json:"type"`
		OrderID    string  `json:"order_id"`
		Amount     float64 `json:"amount"`
		TotalMinor int64   `json:"total_minor"`
		SupplierID string  `json:"supplier_id"`
	}

	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal billing event: %v", err)
		return err
	}

	if event.Type != "ORDER_FINALIZED" {
		return nil
	}
	amount := event.Amount
	if amount == 0 && event.TotalMinor > 0 {
		amount = float64(event.TotalMinor) / 100.0
	}
	return w.MeterWorker.ProcessOrderFinalized(ctx, event.OrderID, amount, event.SupplierID)
}
