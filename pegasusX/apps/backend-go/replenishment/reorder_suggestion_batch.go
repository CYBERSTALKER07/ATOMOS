package replenishment

import (
	"context"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"strings"
	"google.golang.org/api/iterator"
)

const batchLeadTimeDays = 2.0

// RunBatch builds reorder suggestions from today's demand adjustments for one supplier.
func (w *ReorderSuggestionWorker) RunBatch(ctx context.Context, supplierID string) error {
	if w == nil || w.Spanner == nil {
		return nil
	}
	now := w.Now()
	today := civil.DateOf(now)
	tomorrow := today.AddDays(1)

	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT da.RetailerId, da.Sku, da.AdjustedDemand
			FROM DemandAdjustments da
			WHERE da.Date = @today
			LIMIT 2000`,
		Params: map[string]any{"today": today},
	})
	defer iter.Stop()

	type row struct {
		retailerID string
		sku        string
		demand     float64
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
		if err := r.Columns(&item.retailerID, &item.sku, &item.demand); err != nil {
			return err
		}
		rows = append(rows, item)
	}

	for _, item := range rows {
		inFlight := w.inFlightQty(ctx, item.retailerID, item.sku)
		safety := item.demand * 0.15
		suggestion := ReorderSuggestion{
			RetailerId:      item.retailerID,
			Sku:             item.sku,
			AdjustedDemand:  item.demand,
			CurrentStock:    w.retailerStockEstimate(ctx, item.retailerID, item.sku),
			InFlightQty:     inFlight,
			SafetyStock:     safety,
			SuggestedByDate: tomorrow,
			Status:          SuggestionStatusOpen,
		}
		if err := w.ProcessSuggestion(ctx, supplierID, suggestion, batchLeadTimeDays); err != nil {
			continue
		}
	}
	return nil
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
