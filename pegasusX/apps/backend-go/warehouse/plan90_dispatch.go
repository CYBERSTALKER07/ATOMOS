package warehouse

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"google.golang.org/api/iterator"
)

func (s *Service) demandBaselineByProduct(ctx context.Context, supplierID, warehouseID string, forecastDays int) map[string]int64 {
	out := map[string]int64{}
	if s == nil || s.spannerClient == nil {
		return out
	}
	if forecastDays <= 0 {
		forecastDays = 7
	}
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -forecastDays+1)
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, SUM(BaselineQty) FROM DemandForecastBaseline
		      WHERE SupplierId = @sid AND WarehouseId = @wh
		        AND ForecastDate BETWEEN @start AND @end
		      GROUP BY ProductId`,
		Params: map[string]any{"sid": supplierID, "wh": warehouseID, "start": start, "end": end},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var pid string
		var qty int64
		if err := row.Columns(&pid, &qty); err == nil {
			out[pid] = qty
		}
	}
	return out
}

func (s *Service) loadActiveZoneOverrides(ctx context.Context, supplierID string) ([]dispatch.ZoneOverride, error) {
	if s == nil || s.spannerClient == nil || supplierID == "" {
		return nil, nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OverrideId, SupplierId, COALESCE(WarehouseId,''), Action, PolygonGeoJSON
		      FROM ControlTowerZoneOverrides
		      WHERE SupplierId = @sid AND IsActive = true AND TtlExpiresAt > CURRENT_TIMESTAMP()`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	var out []dispatch.ZoneOverride
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ov dispatch.ZoneOverride
		if err := row.Columns(&ov.OverrideID, &ov.SupplierID, &ov.WarehouseID, &ov.Action, &ov.Polygon); err != nil {
			continue
		}
		out = append(out, ov)
	}
	return out, nil
}

func (s *Service) applyZoneOverridesToDispatchRows(ctx context.Context, supplierID string, rows []dispatch.DispatchableOrder) ([]dispatch.DispatchableOrder, []map[string]any) {
	overrides, err := s.loadActiveZoneOverrides(ctx, supplierID)
	if err != nil || len(overrides) == 0 {
		return rows, nil
	}
	return dispatch.ApplyZoneOverrides(rows, overrides)
}

// seedDemandBaselineFromInsights writes baseline rows when insights exist (one-number path).
func (s *Service) seedDemandBaselineFromInsights(ctx context.Context, supplierID, warehouseID string) {
	if s == nil || s.spannerClient == nil {
		return
	}
	insights := s.replenishmentInsightsByProduct(ctx, warehouseID)
	adjustments := s.demandAdjustmentsByProduct(ctx, warehouseID)
	if len(insights) == 0 && len(adjustments) == 0 {
		return
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for productID, insight := range insights {
		qty := insight.ReorderQuantity
		burn := insight.AvgDailyVelocity
		if adj, hasAdj := adjustments[productID]; hasAdj && adj > 0 {
			burn = adj
		}
		if qty <= 0 {
			qty = int64(burn * 7)
		}
		_, _ = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.InsertOrUpdateMap("DemandForecastBaseline", map[string]any{
					"SupplierId":   supplierID,
					"ForecastDate": today,
					"WarehouseId":  warehouseID,
					"ProductId":    productID,
					"BaselineQty":  qty,
					"Confidence":   0.75,
					"Source":       strings.TrimSpace(insight.Urgency),
					"CreatedAt":    spanner.CommitTimestamp,
				}),
			})
		})
	}
}
