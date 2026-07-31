package replenishment

import (
	"context"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type ReorderSuggestionWorker struct {
	Spanner               *spanner.Client
	Now                   func() time.Time
	EchelonTargetsEnabled bool
}

func NewReorderSuggestionWorker(client *spanner.Client) *ReorderSuggestionWorker {
	return &ReorderSuggestionWorker{
		Spanner: client,
		Now:     func() time.Time { return time.Now().UTC() },
	}
}

// ComputeSuggestedQty calculates the suggested order quantity.
// SuggestedQty = max(0, (AdjustedDemandPerDay * LeadTimeDays + SafetyStock) - CurrentStock - InFlightQty)
func ComputeSuggestedQty(adjustedDemandPerDay float64, leadTimeDays float64, safetyStock float64, currentStock int64, inFlightQty int64) int64 {
	target := (adjustedDemandPerDay * leadTimeDays) + safetyStock
	effectiveStock := float64(currentStock + inFlightQty)
	suggested := math.Ceil(target - effectiveStock)
	if suggested < 0 {
		return 0
	}
	return int64(suggested)
}

func (w *ReorderSuggestionWorker) ProcessSuggestion(ctx context.Context, supplierID string, s ReorderSuggestion, leadTimeDays float64) error {
	s.SuggestedQty = ComputeSuggestedQty(s.AdjustedDemand, leadTimeDays, s.SafetyStock, s.CurrentStock, s.InFlightQty)
	if w.EchelonTargetsEnabled && w.Spanner != nil && strings.TrimSpace(supplierID) != "" {
		if targetQty, ok := w.maxEchelonTargetQty(ctx, supplierID, s.Sku); ok {
			effective := float64(s.CurrentStock + s.InFlightQty)
			if qty := SuggestedQtyFromTarget(targetQty, effective); qty > 0 {
				s.SuggestedQty = qty
			}
		}
	}
	s.ComputedAt = w.Now()

	_, err := w.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("ReorderSuggestions", map[string]any{
				"RetailerId":      s.RetailerId,
				"Sku":             s.Sku,
				"SuggestedQty":    s.SuggestedQty,
				"AdjustedDemand":  s.AdjustedDemand,
				"CurrentStock":    s.CurrentStock,
				"InFlightQty":     s.InFlightQty,
				"SafetyStock":     s.SafetyStock,
				"SuggestedByDate": s.SuggestedByDate,
				"ComputedAt":      s.ComputedAt,
				"Status":          s.Status,
			}),
		}

		payload := map[string]any{
			"Type":         "reorder.suggestion.updated",
			"Timestamp":    s.ComputedAt.Format(time.RFC3339Nano),
			"RetailerId":   s.RetailerId,
			"Sku":          s.Sku,
			"SuggestedQty": s.SuggestedQty,
		}

		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, s.RetailerId, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}

		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})

	return err
}

func (w *ReorderSuggestionWorker) maxEchelonTargetQty(ctx context.Context, supplierID, sku string) (int64, bool) {
	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT MAX(TargetQty) FROM EchelonTargets
		      WHERE SupplierId = @sid AND Sku = @sku AND Echelon = @echelon`,
		Params: map[string]any{"sid": supplierID, "sku": sku, "echelon": echelonForward},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, false
	}
	var max spanner.NullInt64
	if err := row.Column(0, &max); err != nil || !max.Valid || max.Int64 <= 0 {
		return 0, false
	}
	return max.Int64, true
}
