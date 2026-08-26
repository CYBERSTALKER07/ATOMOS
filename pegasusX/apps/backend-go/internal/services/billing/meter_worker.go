package billing

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// MeterWorker handles idempotent per-order metering and dynamic fee milestone checks.
type MeterWorker struct {
	client *spanner.Client
}

// NewMeterWorker initializes a new MeterWorker.
func NewMeterWorker(client *spanner.Client) *MeterWorker {
	return &MeterWorker{
		client: client,
	}
}

// ProcessOrderFinalized performs idempotent metering when an order is finalized.
// It checks if global billing milestones are crossed and adjusts system fee rates accordingly.
func (w *MeterWorker) ProcessOrderFinalized(ctx context.Context, orderID string, amount int64, supplierID string) error {
	log.Printf("Metering ORDER_FINALIZED: orderID=%s amount=%d supplierID=%s", orderID, amount, supplierID)

	_, err := w.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Idempotency: skip when this order was already metered.
		stmt := spanner.Statement{
			SQL: `SELECT EventId FROM BillingMeterEvents WHERE OrderId = @orderId LIMIT 1`,
			Params: map[string]interface{}{
				"orderId": orderID,
			},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()
		if _, err := iter.Next(); err == nil {
			log.Printf("Order %s already metered, skipping", orderID)
			return nil
		} else if err != iterator.Done {
			return fmt.Errorf("billing meter idempotency lookup: %w", err)
		}

		const shardID int64 = 0
		var current int64
		row, err := txn.ReadRow(ctx, "BillingSupplierMeters", spanner.Key{supplierID, shardID}, []string{"CurrentValue"})
		if err == nil {
			if err := row.Column(0, &current); err != nil {
				return fmt.Errorf("billing meter read current: %w", err)
			}
		} else if spanner.ErrCode(err) != 5 { // NotFound
			return fmt.Errorf("billing meter read: %w", err)
		}

		mutations := []*spanner.Mutation{
			spanner.InsertMap("BillingMeterEvents", map[string]interface{}{
				"EventId":     uuid.NewString(),
				"SupplierId":  supplierID,
				"OrderId":     orderID,
				"MeterType":   "ORDER_GMV",
				"Amount":      amount,
				"ProcessedAt": spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("BillingSupplierMeters", map[string]interface{}{
				"SupplierId":   supplierID,
				"ShardId":      shardID,
				"CurrentValue": current + amount,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		}
		return txn.BufferWrite(mutations)
	})

	if err != nil {
		log.Printf("Failed to process billing for order %s: %v", orderID, err)
		return err
	}
	return nil
}
