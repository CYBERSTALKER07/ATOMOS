package predictivepush

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

type Auditor struct {
	spannerClient *spanner.Client
}

func NewAuditor(client *spanner.Client) *Auditor {
	return &Auditor{
		spannerClient: client,
	}
}

// LogPredictions saves the predictions to the AIPredictions table for future auditing and score tracking.
func (a *Auditor) LogPredictions(ctx context.Context, events []*DemandEvent) error {
	var mutations []*spanner.Mutation

	for _, event := range events {
		predictionId := uuid.New().String()
		
		predictionData, _ := json.Marshal(map[string]interface{}{
			"retailerId":  event.RetailerId,
			"productId":   event.ProductId,
			"targetDate":  event.TargetDate.Format("2006-01-02"),
			"patternDays": event.PatternDays,
			"quantity":    event.Quantity,
		})

		mutation := spanner.Insert(
			"AIPredictions",
			[]string{
				"PredictionId",
				"AggregateId",
				"AggregateType",
				"SupplierId",
				"PredictionData",
				"Score",
				"Status",
				"CreatedAt",
				"UpdatedAt",
			},
			[]interface{}{
				predictionId,
				event.RetailerId + ":" + event.ProductId, // Combined key for Retailer/Product
				"RETAILER_PRODUCT",
				event.SupplierId,
				[]byte(predictionData), // Stored as BYTES(MAX) in Spanner
				event.Confidence,
				"PENDING_VALIDATION", // Status will be updated when actual order comes in
				spanner.CommitTimestamp,
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
