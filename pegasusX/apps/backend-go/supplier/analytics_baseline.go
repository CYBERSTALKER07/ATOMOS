package supplier

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (s *Service) mergeDemandBaselineItems(ctx context.Context, supplierID string, now time.Time, items []demandSummaryItem, source *string) []demandSummaryItem {
	if s == nil || s.portalSpanner == nil || supplierID == "" {
		return items
	}
	day := now.UTC().Truncate(24 * time.Hour)
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT b.ProductId, COALESCE(p.Name, b.ProductId), SUM(b.BaselineQty), AVG(b.Confidence)
		      FROM DemandForecastBaseline b
		      LEFT JOIN Products p ON b.ProductId = p.ProductId
		      WHERE b.SupplierId = @sid AND b.ForecastDate = @day
		      GROUP BY b.ProductId, p.Name
		      ORDER BY SUM(b.BaselineQty) DESC
		      LIMIT 100`,
		Params: map[string]any{"sid": supplierID, "day": day},
	})
	defer iter.Stop()

	seen := map[string]struct{}{}
	for _, item := range items {
		seen[item.SkuID] = struct{}{}
	}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return items
		}
		var sku, name string
		var qty int64
		var confidence float64
		if err := row.Columns(&sku, &name, &qty, &confidence); err != nil {
			continue
		}
		if _, ok := seen[sku]; ok {
			continue
		}
		seen[sku] = struct{}{}
		if source != nil {
			if *source == "" {
				*source = "demand_forecast_baseline"
			} else if *source != "demand_forecast_baseline" {
				*source = "mixed"
			}
		}
		items = append(items, demandSummaryItem{
			SkuID:       sku,
			ProductName: name,
			TotalQty:    qty,
		})
	}
	return items
}
