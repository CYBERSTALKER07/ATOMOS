package predictivepush

import (
	"context"
	"encoding/json"


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
		return err
	}

	return nil
}
