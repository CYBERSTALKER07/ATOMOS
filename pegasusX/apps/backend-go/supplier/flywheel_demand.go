package supplier

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// FlywheelDemandItem is one STORE_POS DEMAND_SIGNAL feed row for supplier UI.
// Distinct from planning DemandSignals (weather/holiday multipliers).
type FlywheelDemandItem struct {
	SignalID   string `json:"signal_id"`
	SupplierID string `json:"supplier_id,omitempty"`
	RetailerID string `json:"retailer_id"`
	LocationID string `json:"location_id,omitempty"`
	SKU        string `json:"sku"`
	Day        string `json:"day"`
	QtyDelta   int64  `json:"qty_delta"`
	NetSold    int64  `json:"net_sold"`
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// HandleAnalyticsDemandFlywheel serves GET /v1/supplier/analytics/demand/flywheel
// Query: days (default 7, max 90), limit (default 100, max 500), sku= optional filter.
func (s *Service) HandleAnalyticsDemandFlywheel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	if strings.TrimSpace(sid) == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	days := 7
	if v := strings.TrimSpace(r.URL.Query().Get("days")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	if days > 90 {
		days = 90
	}
	limit := 100
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	skuFilter := strings.TrimSpace(r.URL.Query().Get("sku"))

	items, err := s.listFlywheelDemand(r.Context(), sid, days, limit, skuFilter)
	if err != nil {
		if s.log != nil {
			s.log.Warn("flywheel demand feed list failed", "supplier_id", sid, "err", err)
		}
		// Pre-migration or Spanner issue → honest empty (not 500) so UI degrades cleanly.
		writeJSON(w, http.StatusOK, map[string]any{
			"source":      "STORE_POS",
			"description": "Retailer POS sell-through flywheel (DEMAND_SIGNAL). Distinct from planning demand signals.",
			"items":       []FlywheelDemandItem{},
			"days":        days,
			"count":       0,
			"feed_error":  "unavailable",
		})
		return
	}
	if items == nil {
		items = []FlywheelDemandItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"source":      "STORE_POS",
		"description": "Retailer POS sell-through flywheel (DEMAND_SIGNAL). Distinct from planning demand signals.",
		"items":       items,
		"days":        days,
		"count":       len(items),
	})
}

func (s *Service) listFlywheelDemand(ctx context.Context, supplierID string, days, limit int, skuFilter string) ([]FlywheelDemandItem, error) {
	if s.portalSpanner == nil {
		return []FlywheelDemandItem{}, nil
	}
	nowFn := s.now
	if nowFn == nil {
		nowFn = time.Now
	}
	since := nowFn().UTC().AddDate(0, 0, -days)
	sql := `
		SELECT SignalId, COALESCE(SupplierId, ''), RetailerId, COALESCE(LocationId, ''),
		       SkuId, Day, QtyDelta, NetSold, Kind, Source, CreatedAt
		FROM FlywheelDemandFeed
		WHERE SupplierId = @sid
		  AND CreatedAt >= @since`
	params := map[string]any{
		"sid":   supplierID,
		"since": since,
	}
	if skuFilter != "" {
		sql += ` AND SkuId = @sku`
		params["sku"] = skuFilter
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @limit`
	params["limit"] = int64(limit)

	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var out []FlywheelDemandItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var item FlywheelDemandItem
		var day civil.Date
		var created time.Time
		if err := row.Columns(
			&item.SignalID, &item.SupplierID, &item.RetailerID, &item.LocationID,
			&item.SKU, &day, &item.QtyDelta, &item.NetSold, &item.Kind, &item.Source, &created,
		); err != nil {
			return nil, err
		}
		item.Day = time.Date(day.Year, day.Month, day.Day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		if !created.IsZero() {
			item.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		}
		if item.Source == "" {
			item.Source = "STORE_POS"
		}
		out = append(out, item)
	}
	return out, nil
}
