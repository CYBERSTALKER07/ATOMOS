package planning

import (
	"context"
	"errors"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	SparsityBlockedReason    = "insufficient_history"
	SparsityEarlySignalCap   = 60
	SparsityMinCompletedOrders = 2
)

// SparsityResult describes whether forecasting is allowed for a retailer.
type SparsityResult struct {
	Allowed           bool   `json:"allowed"`
	CompletedOrders   int64  `json:"completed_orders"`
	ConfidenceCapPct  int64  `json:"confidence_cap_pct,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
	Label             string `json:"label"`
}

// CanForecast checks the zero/one order rule for a retailer.
func CanForecast(ctx context.Context, client *spanner.Client, retailerID string) (SparsityResult, error) {
	out := SparsityResult{Allowed: true, Label: "standard"}
	if client == nil || retailerID == "" {
		return out, nil
	}
	count, err := completedOrderCount(ctx, client, retailerID)
	if err != nil {
		return out, err
	}
	out.CompletedOrders = count
	switch {
	case count < SparsityMinCompletedOrders:
		out.Allowed = false
		out.BlockedReason = SparsityBlockedReason
		out.Label = "insufficient_history"
	case count < 10:
		out.Allowed = true
		out.ConfidenceCapPct = SparsityEarlySignalCap
		out.Label = "early_signal"
	default:
		out.Label = "standard"
	}
	return out, nil
}

func completedOrderCount(ctx context.Context, client *spanner.Client, retailerID string) (int64, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders
		      WHERE RetailerId = @rid AND Status = 'COMPLETED'`,
		Params: map[string]any{"rid": retailerID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ApplyConfidenceCap clamps confidence when sparsity gate limits apply.
func ApplyConfidenceCap(result SparsityResult, confidencePct int64) int64 {
	if !result.Allowed {
		return 0
	}
	if result.ConfidenceCapPct > 0 && confidencePct > result.ConfidenceCapPct {
		return result.ConfidenceCapPct
	}
	return confidencePct
}
