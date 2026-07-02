package planning

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ReplenishmentInsightWriteInput is one predictive or planning-driven replenishment row.
type ReplenishmentInsightWriteInput struct {
	InsightID       string
	WarehouseID     string
	ProductID       string
	SupplierID      string
	TargetFactoryID string
	SuggestedQty    int64
	UrgencyLevel    string
	ReasonCode      string
	Status          string
	DemandBreakdown string
}

// WriteReplenishmentInsightsWithOutbox inserts insights and emits REPLENISHMENT_INSIGHT_CREATED per row.
func WriteReplenishmentInsightsWithOutbox(ctx context.Context, client *spanner.Client, now time.Time, insights []ReplenishmentInsightWriteInput) error {
	if client == nil || len(insights) == 0 {
		return nil
	}
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &planningTxnBuffer{}
		mutations := make([]*spanner.Mutation, 0, len(insights))
		ts := now.UTC().Format(time.RFC3339Nano)
		for _, in := range insights {
			if in.InsightID == "" || in.WarehouseID == "" || in.ProductID == "" || in.SupplierID == "" {
				continue
			}
			status := in.Status
			if status == "" {
				status = "PENDING"
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId":         in.InsightID,
				"WarehouseId":       in.WarehouseID,
				"ProductId":         in.ProductID,
				"SupplierId":        in.SupplierID,
				"SuggestedQuantity": in.SuggestedQty,
				"UrgencyLevel":      in.UrgencyLevel,
				"ReasonCode":        in.ReasonCode,
				"Status":            status,
				"TargetFactoryId":   in.TargetFactoryID,
				"DemandBreakdown":   in.DemandBreakdown,
				"CreatedAt":         spanner.CommitTimestamp,
			}))
			payload := events.PlanningEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventReplenishmentInsightCreated,
					Timestamp: ts,
				},
				SupplierID:  in.SupplierID,
				WarehouseID: in.WarehouseID,
				FactoryID:   in.TargetFactoryID,
				InsightID:   in.InsightID,
				ProductID:   in.ProductID,
				BaselineQty: in.SuggestedQty,
				Action:      in.ReasonCode,
			}
			if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, in.SupplierID, events.TopicMain, payload); emitErr != nil {
				return emitErr
			}
		}
		if len(mutations) == 0 {
			return nil
		}
		mutations = append(mutations, planningOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}
