package credit

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

// Worker computes and saves retailer credit risk scores.
type Worker struct {
	client *spanner.Client
}

// NewWorker initializes a credit score worker.
func NewWorker(client *spanner.Client) *Worker {
	return &Worker{client: client}
}

// calculateRiskScore extracts the pure logic for computing the score, tier, and limit.
func calculateRiskScore(paymentRate float64, claimRate float64) (int64, RiskTier, int64) {
	score := int64(100)
	if paymentRate < 0.8 {
		score -= 20
	}
	if claimRate > 0.1 {
		score -= 30
	}

	var riskTier RiskTier
	if score >= 80 {
		riskTier = RiskTierLow
	} else if score >= 60 {
		riskTier = RiskTierMedium
	} else {
		riskTier = RiskTierHigh
	}

	limit := int64(1000000)
	if riskTier == RiskTierHigh {
		limit = 0
	} else if riskTier == RiskTierMedium {
		limit = 500000
	}

	return score, riskTier, limit
}

// ComputeAndSaveRiskScore computes a basic risk score for a retailer and saves it,
// emitting an outbox event.
func (w *Worker) ComputeAndSaveRiskScore(ctx context.Context, retailerID string, paymentRate float64, claimRate float64) error {
	score, riskTier, limit := calculateRiskScore(paymentRate, claimRate)

	now := time.Now().UTC()
	windowStart := now.Add(-30 * 24 * time.Hour)
	windowEnd := now

	factors := map[string]float64{
		"payment_rate": paymentRate,
		"claim_rate":   claimRate,
	}
	factorsBytes, _ := json.Marshal(factors)

	rcs := RetailerCreditScore{
		RetailerID:          retailerID,
		Score:               score,
		RiskTier:            riskTier,
		SuggestedLimitMinor: limit,
		FactorsJSON:         string(factorsBytes),
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
		ComputedAt:          now,
	}

	return spannerutils.RunReadWriteTransaction(ctx, w.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerCreditScores", map[string]any{
				"RetailerId":          rcs.RetailerID,
				"Score":               rcs.Score,
				"RiskTier":            string(rcs.RiskTier),
				"SuggestedLimitMinor": rcs.SuggestedLimitMinor,
				"FactorsJson":         spanner.NullJSON{Value: factorsBytes, Valid: true},
				"WindowStart":         rcs.WindowStart,
				"WindowEnd":           rcs.WindowEnd,
				"ComputedAt":          spanner.CommitTimestamp,
			}),
		}

		payload, err := json.Marshal(rcs)
		if err != nil {
			return err
		}

		eventID := uuid.New().String()
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
			"EventId":       eventID,
			"AggregateType": "RetailerCreditScore",
			"AggregateId":   retailerID,
			"TopicName":     "credit.score.updated",
			"Payload":       payload,
			"CreatedAt":     spanner.CommitTimestamp,
			"PublishedAt":   nil,
		}))

		return txn.BufferWrite(mutations)
	})
}
