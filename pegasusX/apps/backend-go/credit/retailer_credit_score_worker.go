package credit

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// Worker computes and saves retailer credit risk scores.
type Worker struct {
	client *spanner.Client
}

// NewWorker initializes a credit score worker.
func NewWorker(client *spanner.Client) *Worker {
	return &Worker{client: client}
}

// RiskInputs feeds the multi-factor score formula.
type RiskInputs struct {
	PaymentRate    float64
	ClaimRate      float64
	ShopClosedRate float64
	VelocityScore  float64
	UtilisationBps int64
	AccountAgeDays float64
}

func calculateRiskScore(in RiskInputs) (int64, RiskTier, int64) {
	score := int64(100)
	if in.PaymentRate < 0.8 {
		score -= 20
	}
	if in.ClaimRate > 0.1 {
		score -= 30
	}
	if in.ShopClosedRate > 0.05 {
		score -= 15
	}
	if in.VelocityScore < 0.3 {
		score -= 10
	}
	if in.UtilisationBps > 8000 {
		score -= 15
	}
	if in.AccountAgeDays < 30 {
		score -= 10
	}
	if score < 0 {
		score = 0
	}

	var riskTier RiskTier
	switch {
	case score >= 80:
		riskTier = RiskTierLow
	case score >= 60:
		riskTier = RiskTierMedium
	default:
		riskTier = RiskTierHigh
	}

	limit := int64(1000000)
	switch riskTier {
	case RiskTierHigh:
		limit = 0
	case RiskTierMedium:
		limit = 500000
	}

	return score, riskTier, limit
}

// RunNightlyWorker recomputes retailer credit scores on a fixed interval.
func (w *Worker) RunNightlyWorker(ctx context.Context, interval time.Duration) {
	if w == nil || w.client == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	_ = w.RunBatch(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = w.RunBatch(ctx)
		}
	}
}

// RunBatch scores retailers with recent order activity.
func (w *Worker) RunBatch(ctx context.Context) error {
	if w == nil || w.client == nil {
		return nil
	}
	now := time.Now().UTC()
	windowStart := now.Add(-30 * 24 * time.Hour)

	iter := w.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DISTINCT RetailerId FROM Orders WHERE CreatedAt >= @start LIMIT 5000`,
		Params: map[string]any{"start": windowStart},
	})
	defer iter.Stop()

	var retailerIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var id string
		if err := row.Columns(&id); err != nil {
			return err
		}
		retailerIDs = append(retailerIDs, id)
	}

	for _, retailerID := range retailerIDs {
		inputs, err := w.loadRiskInputs(ctx, retailerID, windowStart, now)
		if err != nil {
			continue
		}
		if err := w.ComputeAndSave(ctx, retailerID, inputs, windowStart, now); err != nil {
			continue
		}
	}
	return nil
}

func queryInt64(ctx context.Context, client *spanner.Client, sql string, params map[string]any) int64 {
	if client == nil {
		return 0
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0
	}
	var v int64
	if err := row.Column(0, &v); err != nil {
		return 0
	}
	return v
}

func (w *Worker) loadRiskInputs(ctx context.Context, retailerID string, windowStart, windowEnd time.Time) (RiskInputs, error) {
	in := RiskInputs{
		PaymentRate:    1.0,
		ClaimRate:      0.0,
		ShopClosedRate: 0.0,
		VelocityScore:  0.5,
	}
	ro := w.client
	params := map[string]any{"rid": retailerID, "start": windowStart, "end": windowEnd}

	paid := queryInt64(ctx, ro, `
		SELECT COALESCE(SUM(CASE WHEN Status IN ('COMPLETED','DELIVERED_ON_CREDIT') THEN 1 ELSE 0 END), 0)
		FROM Orders WHERE RetailerId = @rid AND CreatedAt >= @start AND CreatedAt < @end`, params)
	total := queryInt64(ctx, ro, `
		SELECT COUNT(*) FROM Orders WHERE RetailerId = @rid AND CreatedAt >= @start AND CreatedAt < @end`, params)
	if total > 0 {
		in.PaymentRate = float64(paid) / float64(total)
	}

	claims := queryInt64(ctx, ro, `
		SELECT COUNT(*) FROM Claims c JOIN Orders o ON o.OrderId = c.OrderId
		WHERE o.RetailerId = @rid AND c.CreatedAt >= @start AND c.CreatedAt < @end`, params)
	delivered := queryInt64(ctx, ro, `
		SELECT COUNT(*) FROM Orders
		WHERE RetailerId = @rid AND Status = 'COMPLETED' AND CreatedAt >= @start AND CreatedAt < @end`, params)
	if delivered > 0 {
		in.ClaimRate = float64(claims) / float64(delivered)
	}

	shopClosed := queryInt64(ctx, ro, `
		SELECT COUNT(*) FROM Orders
		WHERE RetailerId = @rid AND ShopClosedAt IS NOT NULL
		  AND CreatedAt >= @start AND CreatedAt < @end`, params)
	if delivered > 0 {
		in.ShopClosedRate = float64(shopClosed) / float64(delivered)
	}

	orderCnt := queryInt64(ctx, ro, `
		SELECT COUNT(*) FROM Orders WHERE RetailerId = @rid AND CreatedAt >= @start`, params)
	in.VelocityScore = float64(orderCnt) / 30.0
	if in.VelocityScore > 1 {
		in.VelocityScore = 1
	}

	limitIter := w.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT CreditLimitMinor, CurrentBalanceMinor FROM RetailerCreditProfiles
		      WHERE RetailerId = @rid ORDER BY UpdatedAt DESC LIMIT 1`,
		Params: map[string]any{"rid": retailerID},
	})
	if row, err := limitIter.Next(); err == nil {
		var limit, balance int64
		if err := row.Columns(&limit, &balance); err == nil && limit > 0 {
			in.UtilisationBps = (balance * 10000) / limit
		}
	}
	limitIter.Stop()

	ageIter := w.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT MIN(CreatedAt) FROM Orders WHERE RetailerId = @rid`,
		Params: map[string]any{"rid": retailerID},
	})
	if row, err := ageIter.Next(); err == nil {
		var first time.Time
		if err := row.Column(0, &first); err == nil && !first.IsZero() {
			in.AccountAgeDays = windowEnd.Sub(first).Hours() / 24
		}
	}
	ageIter.Stop()

	return in, nil
}

// ComputeAndSaveRiskScore computes score from payment and claim rates (legacy helper).
func (w *Worker) ComputeAndSaveRiskScore(ctx context.Context, retailerID string, paymentRate float64, claimRate float64) error {
	inputs := RiskInputs{PaymentRate: paymentRate, ClaimRate: claimRate, VelocityScore: 0.5}
	now := time.Now().UTC()
	return w.ComputeAndSave(ctx, retailerID, inputs, now.Add(-30*24*time.Hour), now)
}

func (w *Worker) ComputeAndSave(ctx context.Context, retailerID string, inputs RiskInputs, windowStart, windowEnd time.Time) error {
	score, riskTier, limit := calculateRiskScore(inputs)

	factors := map[string]float64{
		"payment_rate":     inputs.PaymentRate,
		"claim_rate":       inputs.ClaimRate,
		"shop_closed_rate": inputs.ShopClosedRate,
		"velocity_score":   inputs.VelocityScore,
		"utilisation_bps":  float64(inputs.UtilisationBps),
		"account_age_days": inputs.AccountAgeDays,
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
		ComputedAt:          windowEnd,
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
