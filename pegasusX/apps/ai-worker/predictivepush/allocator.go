package predictivepush

import (
	"context"
	"encoding/json"

	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

type Allocator struct {
	spannerClient *spanner.Client
	locator       *Locator
}

func NewAllocator(client *spanner.Client, locator *Locator) *Allocator {
	return &Allocator{
		spannerClient: client,
		locator:       locator,
	}
}

// Allocate proactively generates ReplenishmentInsights for the predicted demand events.
func (a *Allocator) Allocate(ctx context.Context, events []*DemandEvent) error {
	var mutations []*spanner.Mutation

	for _, event := range events {
		// 1. Find the nearest warehouse
		warehouse, err := a.locator.FindNearestWarehouse(ctx, event.RetailerId, event.SupplierId)
		if err != nil {
			return err
		}
		if warehouse == nil {
			continue // No warehouse found, skip
		}

		// 2. Generate a ReplenishmentInsight
		insightId := uuid.New().String()
		
		demandBreakdown, _ := json.Marshal(map[string]interface{}{
			"retailerId":   event.RetailerId,
			"targetDate":   event.TargetDate.Format("2006-01-02"),
			"confidence":   event.Confidence,
			"patternDays":  event.PatternDays,
			"predictedQty": event.Quantity,
		})

		mutation := spanner.Insert(
			"ReplenishmentInsights",
			[]string{
				"InsightId",
				"WarehouseId",
				"ProductId",
				"SupplierId",
				"SuggestedQuantity",
				"UrgencyLevel",
				"ReasonCode",
				"Status",
				"TargetFactoryId",
				"DemandBreakdown",
				"CreatedAt",
			},
			[]interface{}{
				insightId,
				warehouse.WarehouseId,
				event.ProductId,
				event.SupplierId,
				event.Quantity,         // Proactively push the expected quantity
				"PROACTIVE",            // Special urgency level
				"PREDICTIVE_PUSH",      // Identifies the AI agent action
				"PENDING",              // To be reviewed by human or automated auto-approver
				warehouse.PrimaryFactoryId,
				string(demandBreakdown),
				spanner.CommitTimestamp,
			},
		)
		
		mutations = append(mutations, mutation)
	}

	if len(mutations) > 0 {
		_, err := a.spannerClient.Apply(ctx, mutations)
		if err != nil {
			return err
		}
	}
	return a.writeDemandBaselines(ctx, events)
}

func (a *Allocator) writeDemandBaselines(ctx context.Context, events []*DemandEvent) error {
	if a == nil || a.spannerClient == nil || len(events) == 0 {
		return nil
	}
	day := time.Now().UTC().Truncate(24 * time.Hour)
	var mutations []*spanner.Mutation
	for _, event := range events {
		if event == nil || event.SupplierId == "" || event.ProductId == "" {
			continue
		}
		warehouse, err := a.locator.FindNearestWarehouse(ctx, event.RetailerId, event.SupplierId)
		if err != nil || warehouse == nil {
			continue
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("DemandForecastBaseline", map[string]any{
			"SupplierId":   event.SupplierId,
			"ForecastDate": day,
			"WarehouseId":  warehouse.WarehouseId,
			"ProductId":    event.ProductId,
			"BaselineQty":  event.Quantity,
			"Confidence":   event.Confidence,
			"Source":       "predictive_push",
			"CreatedAt":    spanner.CommitTimestamp,
		}))
	}
	if len(mutations) == 0 {
		return nil
	}
	_, err := a.spannerClient.Apply(ctx, mutations)
	return err
}
