package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
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

// HandleMessage processes incoming Kafka messages for the billing tier worker.
func (w *BillingTierWorker) HandleMessage(ctx context.Context, msg []byte) error {
	var event struct {
		Type       string  `json:"type"`
		OrderID    string  `json:"order_id"`
		Amount     float64 `json:"amount"`
		SupplierID string  `json:"supplier_id"`
	}

	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal billing event: %v", err)
		return err
	}

	if event.Type == "ORDER_FINALIZED" {
		return w.MeterWorker.ProcessOrderFinalized(ctx, event.OrderID, event.Amount, event.SupplierID)
	}

	return nil
}
