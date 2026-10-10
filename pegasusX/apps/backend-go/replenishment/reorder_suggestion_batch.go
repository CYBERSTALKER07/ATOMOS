package replenishment

import (
	"context"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const batchLeadTimeDays = 2.0

// retailerSafetyStock returns the SS component for retailer reorder suggestions.
// When v2 is off, callers use demand·0.15 instead.
func retailerSafetyStock(dBar, sigmaD, leadDays, leadSigma, serviceLevel float64, sigmaDAssumed, leadSigmaAssumed bool) float64 {
	if dBar < 0 {
		dBar = 0
	}
	if sigmaD <= 0 {
		sigmaD = math.Max(dBar*0.25, 1)
		sigmaDAssumed = true
	}
	if leadDays <= 0 {
		leadDays = batchLeadTimeDays
	}
	if leadSigma < 0 {
		leadSigma = 1.0
	}
	if serviceLevel <= 0 {
		serviceLevel = 0.98
	}
	return SafetyStockUnits(SafetyStockInputs{
		DBar:             dBar,
		SigmaD:           sigmaD,
		SigmaDAssumed:    sigmaDAssumed,
		L:                leadDays,
		SigmaL:           leadSigma,
		LeadSigmaAssumed: leadSigmaAssumed,
		ServiceLevel:     serviceLevel,
	})
}

// RunBatch builds reorder suggestions from today's demand adjustments for one supplier.
// L3.3: merges POS sell-through velocity (max with base) and tags sources.
func (w *ReorderSuggestionWorker) RunBatch(ctx context.Context, supplierID string) error {
	if w == nil || w.Spanner == nil {
		return nil
	}
	now := w.Now()
	today := civil.DateOf(now)
	tomorrow := today.AddDays(1)

	leadDays := batchLeadTimeDays
	leadSigma := 1.0
	serviceLevel := 0.98
	leadAssumed := true
	useV2 := SafetyStockV2Enabled()
	if useV2 && strings.TrimSpace(supplierID) != "" {
		if policy, err := LoadPolicy(ctx, w.Spanner, supplierID); err == nil {
			if policy.LeadTimeDays > 0 {
				leadDays = float64(policy.LeadTimeDays)
			}
			if policy.LeadTimeSigmaDays >= 0 {
				leadSigma = policy.LeadTimeSigmaDays
			}
			if policy.TargetServiceLevel > 0 {
				serviceLevel = policy.TargetServiceLevel
			}
		}
		if obs, err := ObservedLeadStats(ctx, w.Spanner, supplierID); err == nil && !obs.Assumed {
			leadDays = obs.MeanDays
			if leadDays < 1 {
				leadDays = 1
			}
			leadSigma = obs.SigmaDays
			leadAssumed = false
		}
	}

	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT da.RetailerId, da.Sku, da.AdjustedDemand, da.BaseVelocity, da.FactorsJson
			FROM DemandAdjustments da
			WHERE da.Date = @today
			LIMIT 2000`,
		Params: map[string]any{"today": today},
	})
	defer iter.Stop()

	type row struct {
		retailerID     string
		sku            string
		adjustedDemand float64
		baseVelocity   float64
		factorsJSON    spanner.NullString
	}
	var rows []row
	for {
		r, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var item row
		var factors spanner.NullJSON
		// FactorsJson is JSON type in Spanner — try NullJSON then string.
		if err := r.Columns(&item.retailerID, &item.sku, &item.adjustedDemand, &item.baseVelocity, &factors); err != nil {
			// Fallback without BaseVelocity/Factors for older schemas
			if err2 := r.Columns(&item.retailerID, &item.sku, &item.adjustedDemand); err2 != nil {
				return err
			}
		} else if factors.Valid {
			if s, ok := factors.Value.(string); ok {
				item.factorsJSON = spanner.NullString{StringVal: s, Valid: true}
			} else {
				b, _ := factors.MarshalJSON()
				item.factorsJSON = spanner.NullString{StringVal: string(b), Valid: true}
			}
		}
		rows = append(rows, item)
	}

	for _, item := range rows {
		// Skip pure local SKUs from supplier-facing suggestions (L6 guard).
		if strings.HasPrefix(strings.ToLower(item.sku), "local:") {
			continue
		}

		factors := ParseFactorsJSON("")
		if item.factorsJSON.Valid {
			factors = ParseFactorsJSON(item.factorsJSON.StringVal)
		}
		base, factorF := StripSellThroughFactor(item.adjustedDemand, item.baseVelocity, factors)

		stUnits, stDays, usedRollup := w.sellThroughUnits(ctx, item.retailerID, item.sku, today, DefaultSellThroughDays)
		if !usedRollup && factorF > 0 {
			// Fallback: same-day factor only (before multi-day rollup history exists).
			stUnits = factorF
			stDays = 1
		}

		demand, sources := MergeDemandVelocities(base, stUnits, stDays)
		stVel := 0.0
		if stDays > 0 && stUnits > 0 {
			stVel = stUnits / float64(stDays)
		}

		inFlight := w.inFlightQty(ctx, item.retailerID, item.sku)
		var safety float64
		if useV2 {
			sigmaD, _, ok, _ := ResidualSigmaD(ctx, w.Spanner, supplierID, "", item.sku)
			sigmaAssumed := !ok
			if !ok {
				sigmaD = math.Max(demand*0.25, 1)
			}
			safety = retailerSafetyStock(demand, sigmaD, leadDays, leadSigma, serviceLevel, sigmaAssumed, leadAssumed)
		} else {
			safety = demand * 0.15
		}
		suggestion := ReorderSuggestion{
			RetailerId:      item.retailerID,
			Sku:             item.sku,
			AdjustedDemand:  demand,
			CurrentStock:    w.retailerStockEstimate(ctx, item.retailerID, item.sku),
			InFlightQty:     inFlight,
			SafetyStock:     safety,
			SuggestedByDate: tomorrow,
			Status:          SuggestionStatusOpen,
			Sources:         sources,
			SellThroughVel:  stVel,
			BaseDemand:      base,
		}
		effLead := leadDays
		if !useV2 {
			effLead = batchLeadTimeDays
		}
		if err := w.ProcessSuggestion(ctx, supplierID, suggestion, effLead); err != nil {
			continue
		}
	}
	return nil
}

// sellThroughUnits returns net sold (QtySold-QtyVoided) over the last windowDays ending today.
// usedRollup is true when at least one RetailerSellThroughDaily row was found.
func (w *ReorderSuggestionWorker) sellThroughUnits(ctx context.Context, retailerID, sku string, today civil.Date, windowDays int) (units float64, days int, usedRollup bool) {
	if windowDays <= 0 {
		windowDays = DefaultSellThroughDays
	}
	if w == nil || w.Spanner == nil {
		return 0, windowDays, false
	}
	from := today.AddDays(-(windowDays - 1))
	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT COALESCE(SUM(QtySold - QtyVoided), 0), COUNT(*)
			FROM RetailerSellThroughDaily
			WHERE RetailerId = @rid AND SkuId = @sku
			  AND Day >= @from AND Day <= @to`,
		Params: map[string]any{
			"rid":  retailerID,
			"sku":  sku,
			"from": from,
			"to":   today,
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, windowDays, false
	}
	var sum int64
	var cnt int64
	if err := row.Columns(&sum, &cnt); err != nil {
		return 0, windowDays, false
	}
	if cnt == 0 {
		return 0, windowDays, false
	}
	if sum < 0 {
		sum = 0
	}
	return float64(sum), windowDays, true
}

func (w *ReorderSuggestionWorker) inFlightQty(ctx context.Context, retailerID, sku string) int64 {
	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT COALESCE(SUM(l.OrderedQty - l.DeliveredQty), 0)
			FROM Orders o
			JOIN OrderLines l ON l.OrderId = o.OrderId
			WHERE o.RetailerId = @rid AND l.Sku = @sku
			  AND o.Status IN ('PENDING','LOADED','IN_TRANSIT','ARRIVED')`,
		Params: map[string]any{"rid": retailerID, "sku": sku},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0
	}
	var qty int64
	if err := row.Column(0, &qty); err != nil {
		return 0
	}
	return qty
}

func (w *ReorderSuggestionWorker) retailerStockEstimate(ctx context.Context, retailerID, sku string) int64 {
	// Prefer Retail OS Phase 3 store ledger (sum OnHand across bins/locations).
	if w.Spanner != nil {
		iter := w.Spanner.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT COALESCE(SUM(OnHand), 0) FROM RetailerStockBalances
				WHERE RetailerId = @rid AND Sku = @sku`,
			Params: map[string]any{"rid": retailerID, "sku": sku},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			var qty int64
			if err := row.Column(0, &qty); err == nil && qty > 0 {
				return qty
			}
			// qty == 0 may mean no ledger rows OR truly empty — try legacy only when no rows.
		}
		// Detect whether any balance row exists.
		iter2 := w.Spanner.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT 1 FROM RetailerStockBalances WHERE RetailerId = @rid AND Sku = @sku LIMIT 1`,
			Params: map[string]any{"rid": retailerID, "sku": sku},
		})
		defer iter2.Stop()
		if _, err := iter2.Next(); err == nil {
			// Ledger exists (possibly zero) — trust it.
			iter3 := w.Spanner.Single().Query(ctx, spanner.Statement{
				SQL: `SELECT COALESCE(SUM(OnHand), 0) FROM RetailerStockBalances
					WHERE RetailerId = @rid AND Sku = @sku`,
				Params: map[string]any{"rid": retailerID, "sku": sku},
			})
			defer iter3.Stop()
			if row, err := iter3.Next(); err == nil {
				var qty int64
				_ = row.Column(0, &qty)
				return qty
			}
			return 0
		}
	}
	// Legacy estimate: last completed line delivered qty (pre store-stock).
	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT COALESCE(l.DeliveredQty, 0)
			FROM Orders o
			JOIN OrderLines l ON l.OrderId = o.OrderId
			WHERE o.RetailerId = @rid AND l.Sku = @sku AND o.Status = 'COMPLETED'
			ORDER BY o.UpdatedAt DESC
			LIMIT 1`,
		Params: map[string]any{"rid": retailerID, "sku": sku},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0
	}
	var qty int64
	if err := row.Column(0, &qty); err != nil {
		return 0
	}
	return qty
}

// RunBatchAllSuppliers runs reorder suggestions for every supplier with recent order activity.
func (w *ReorderSuggestionWorker) RunBatchAllSuppliers(ctx context.Context) error {
	if w == nil || w.Spanner == nil {
		return nil
	}
	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DISTINCT SupplierId FROM Orders
		      WHERE UpdatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 DAY)
		        AND SupplierId IS NOT NULL`,
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var supplierID string
		if err := row.Column(0, &supplierID); err != nil || strings.TrimSpace(supplierID) == "" {
			continue
		}
		_ = w.RunBatch(ctx, supplierID)
	}
	return nil
}

// RunBatchWorker periodically refreshes reorder suggestions after demand sensing.
func (w *ReorderSuggestionWorker) RunBatchWorker(ctx context.Context, supplierID string, interval time.Duration) {
	if w == nil || w.Spanner == nil {
		return
	}
	if interval <= 0 {
		interval = 12 * time.Hour
	}
	_ = w.RunBatch(ctx, supplierID)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = w.RunBatch(ctx, supplierID)
		}
	}
}
