package planning

import (
	"context"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// BaselineWriteInput is one demand baseline upsert with optional confidence range.
type BaselineWriteInput struct {
	SupplierID     string
	WarehouseID    string
	ProductID      string
	ForecastDate   time.Time
	BaselineQty    int64
	LowUnits       int64
	HighUnits      int64
	Confidence     float64
	ConfidencePct  int64
	Source         string
	BaselineSource string
	BlockedReason  string
}

// WriteBaselineWithOutbox upserts DemandForecastBaseline and emits DEMAND_BASELINE_UPDATED in one txn.
func WriteBaselineWithOutbox(ctx context.Context, client *spanner.Client, now time.Time, in BaselineWriteInput) error {
	if client == nil {
		return nil
	}
	if in.ForecastDate.IsZero() {
		in.ForecastDate = now.UTC().Truncate(24 * time.Hour)
	}
	if in.LowUnits == 0 && in.HighUnits == 0 && in.BaselineQty > 0 {
		margin := int64(float64(in.BaselineQty) * 0.1)
		if margin < 1 {
			margin = 1
		}
		in.LowUnits = in.BaselineQty - margin
		in.HighUnits = in.BaselineQty + margin
	}
	if in.ConfidencePct == 0 && in.Confidence > 0 {
		in.ConfidencePct = int64(in.Confidence * 100)
	}
	if in.BaselineSource == "" {
		in.BaselineSource = in.Source
	}
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"SupplierId":     in.SupplierID,
			"ForecastDate":   civil.DateOf(in.ForecastDate.UTC()),
			"WarehouseId":    in.WarehouseID,
			"ProductId":      in.ProductID,
			"BaselineQty":    in.BaselineQty,
			"Confidence":     in.Confidence,
			"Source":         in.Source,
			"LowUnits":       in.LowUnits,
			"HighUnits":      in.HighUnits,
			"ConfidencePct":  in.ConfidencePct,
			"BaselineSource": in.BaselineSource,
			"BlockedReason":  in.BlockedReason,
			"CreatedAt":      spanner.CommitTimestamp,
		}
		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("DemandForecastBaseline", row)}
		payload := events.PlanningEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventDemandBaselineUpdated,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			SupplierID:     in.SupplierID,
			WarehouseID:    in.WarehouseID,
			ProductID:      in.ProductID,
			BaselineQty:    in.BaselineQty,
			LowUnits:       in.LowUnits,
			HighUnits:      in.HighUnits,
			Confidence:     in.Confidence,
			ConfidencePct:  in.ConfidencePct,
			BaselineSource: in.BaselineSource,
			BlockedReason:  in.BlockedReason,
			Action:         in.Source,
		}
		buf := &planningTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, in.SupplierID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, planningOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}
