package retailer

import (
	"context"
	"encoding/csv"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// Wave C2.2 HQ REST — reads of RetailerHqSalesDaily / snapshots.
// Writers always-on (C2.1). Reads gated by HQ_ANALYTICS_ENABLED (default off → 404 honest).

func (s *Service) hqAnalyticsEnabled() bool {
	if s != nil && s.hqAnalyticsOverride != nil {
		return *s.hqAnalyticsOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("HQ_ANALYTICS_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (s *Service) hqAuth(w http.ResponseWriter, r *http.Request) (auth.Claims, string, bool) {
	if !s.hqAnalyticsEnabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
			"code":  "HQ_ANALYTICS_DISABLED",
		})
		return auth.Claims{}, "", false
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return auth.Claims{}, "", false
	}
	role := auth.EffectiveRetailerRole(claims)
	if role != "OWNER" && role != "ADMIN" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":  "forbidden",
			"detail": "hq_requires_owner_or_admin",
		})
		return auth.Claims{}, "", false
	}
	if !auth.HasRetailerPerm(claims, auth.PermReportsView) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermReportsView})
		return auth.Claims{}, "", false
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return auth.Claims{}, "", false
	}
	// REPORTS_PRO pack: auto-enable on first HQ use (same posture as reports_pro).
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackREPORTSPRO) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackREPORTSPRO, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}
	return claims, orgID, true
}

func hqDayFromRequest(r *http.Request, now time.Time) string {
	day := strings.TrimSpace(r.URL.Query().Get("day"))
	if day == "" {
		return hqDayUTC(now)
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return hqDayUTC(now)
	}
	return day
}

// HandleHqSummary serves GET /v1/retailer/hq/summary?day=YYYY-MM-DD
func (s *Service) HandleHqSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.hqAuth(w, r)
	if !ok {
		return
	}
	day := hqDayFromRequest(r, s.now())
	rows, err := s.listHqSalesRows(r.Context(), orgID, day, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hq_summary_failed"})
		return
	}
	var qtySold, qtyVoided, gross, net int64
	locs := map[string]struct{}{}
	skus := map[string]struct{}{}
	currency := "UZS"
	for _, row := range rows {
		qtySold += row.QtySold
		qtyVoided += row.QtyVoided
		gross += row.GrossMinor
		net += row.NetMinor
		locs[row.LocationID] = struct{}{}
		skus[row.SkuID] = struct{}{}
		if row.Currency != "" {
			currency = row.Currency
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":       orgID,
		"day":               day,
		"location_count":    len(locs),
		"sku_count":         len(skus),
		"qty_sold":          qtySold,
		"qty_voided":        qtyVoided,
		"gross_minor":       gross,
		"net_minor":         net,
		"currency":          currency,
		"honest_empty":      len(rows) == 0,
		"pack":              PackREPORTSPRO,
	})
}

// HandleHqSalesByLocation serves GET /v1/retailer/hq/sales-by-location?day=
func (s *Service) HandleHqSalesByLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.hqAuth(w, r)
	if !ok {
		return
	}
	day := hqDayFromRequest(r, s.now())
	rows, err := s.listHqSalesRows(r.Context(), orgID, day, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hq_by_location_failed"})
		return
	}
	type locRow struct {
		LocationID string `json:"location_id"`
		QtySold    int64  `json:"qty_sold"`
		QtyVoided  int64  `json:"qty_voided"`
		GrossMinor int64  `json:"gross_minor"`
		NetMinor   int64  `json:"net_minor"`
		Currency   string `json:"currency"`
	}
	byLoc := map[string]*locRow{}
	var orgNet int64
	for _, row := range rows {
		b := byLoc[row.LocationID]
		if b == nil {
			b = &locRow{LocationID: row.LocationID, Currency: row.Currency}
			if b.Currency == "" {
				b.Currency = "UZS"
			}
			byLoc[row.LocationID] = b
		}
		b.QtySold += row.QtySold
		b.QtyVoided += row.QtyVoided
		b.GrossMinor += row.GrossMinor
		b.NetMinor += row.NetMinor
		orgNet += row.NetMinor
	}
	items := make([]locRow, 0, len(byLoc))
	var sumNet int64
	for _, b := range byLoc {
		items = append(items, *b)
		sumNet += b.NetMinor
	}
	sort.Slice(items, func(i, j int) bool { return items[i].LocationID < items[j].LocationID })
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":    orgID,
		"day":            day,
		"items":          items,
		"org_net_minor":  orgNet,
		"sum_locations":  sumNet,
		"balanced":       sumNet == orgNet,
		"honest_empty":   len(items) == 0,
	})
}

// HandleHqSalesBySku serves GET /v1/retailer/hq/sales-by-sku?day=&location_id=
func (s *Service) HandleHqSalesBySku(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.hqAuth(w, r)
	if !ok {
		return
	}
	day := hqDayFromRequest(r, s.now())
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	rows, err := s.listHqSalesRows(r.Context(), orgID, day, locID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hq_by_sku_failed"})
		return
	}
	type skuRow struct {
		SkuID      string `json:"sku_id"`
		LocationID string `json:"location_id,omitempty"`
		QtySold    int64  `json:"qty_sold"`
		QtyVoided  int64  `json:"qty_voided"`
		GrossMinor int64  `json:"gross_minor"`
		NetMinor   int64  `json:"net_minor"`
		Currency   string `json:"currency"`
		IsLocal    bool   `json:"is_local"`
	}
	// Aggregate by SKU across locations when location filter empty
	if locID == "" {
		agg := map[string]*skuRow{}
		for _, row := range rows {
			b := agg[row.SkuID]
			if b == nil {
				b = &skuRow{SkuID: row.SkuID, Currency: row.Currency, IsLocal: IsLocalSKU(row.SkuID)}
				if b.Currency == "" {
					b.Currency = "UZS"
				}
				agg[row.SkuID] = b
			}
			b.QtySold += row.QtySold
			b.QtyVoided += row.QtyVoided
			b.GrossMinor += row.GrossMinor
			b.NetMinor += row.NetMinor
		}
		items := make([]skuRow, 0, len(agg))
		for _, b := range agg {
			items = append(items, *b)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].NetMinor > items[j].NetMinor })
		writeJSON(w, http.StatusOK, map[string]any{
			"retailer_id":  orgID,
			"day":          day,
			"items":        items,
			"honest_empty": len(items) == 0,
		})
		return
	}
	items := make([]skuRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, skuRow{
			SkuID: row.SkuID, LocationID: row.LocationID,
			QtySold: row.QtySold, QtyVoided: row.QtyVoided,
			GrossMinor: row.GrossMinor, NetMinor: row.NetMinor,
			Currency: row.Currency, IsLocal: IsLocalSKU(row.SkuID),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].NetMinor > items[j].NetMinor })
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":  orgID,
		"day":          day,
		"location_id":  locID,
		"items":        items,
		"honest_empty": len(items) == 0,
	})
}

// HandleHqShrinkage serves GET /v1/retailer/hq/shrinkage?day=
// Honest proxy: void qty/value from HQ daily (not full inventory cycle count).
func (s *Service) HandleHqShrinkage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.hqAuth(w, r)
	if !ok {
		return
	}
	day := hqDayFromRequest(r, s.now())
	rows, err := s.listHqSalesRows(r.Context(), orgID, day, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hq_shrinkage_failed"})
		return
	}
	var voidQty, voidMinor, gross, net int64
	for _, row := range rows {
		voidQty += row.QtyVoided
		// void value approximated as gross - net when voids recorded
		voidMinor += row.GrossMinor - row.NetMinor
		if voidMinor < 0 {
			// if net went negative from partial voids across days, clamp display
			voidMinor = 0
		}
		gross += row.GrossMinor
		net += row.NetMinor
	}
	// Prefer explicit void accounting: sum of line voids = gross - net when only voids reduce net
	if voidMinor == 0 && gross > net {
		voidMinor = gross - net
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":        orgID,
		"day":                day,
		"void_qty":           voidQty,
		"void_value_minor":   voidMinor,
		"gross_minor":        gross,
		"net_minor":          net,
		"method":             "hq_void_proxy",
		"note":               "Shrinkage here is POS void rollup from HQ daily, not full cycle-count loss.",
		"honest_empty":       len(rows) == 0,
	})
}

// HandleHqExport serves GET /v1/retailer/hq/export?day=&format=csv
func (s *Service) HandleHqExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.hqAuth(w, r)
	if !ok {
		return
	}
	day := hqDayFromRequest(r, s.now())
	rows, err := s.listHqSalesRows(r.Context(), orgID, day, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hq_export_failed"})
		return
	}
	if wantsCSV(r) || strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="hq-sales-`+day+`.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"day", "location_id", "sku_id", "qty_sold", "qty_voided", "gross_minor", "net_minor", "currency"})
		for _, row := range rows {
			_ = cw.Write([]string{
				row.Day, row.LocationID, row.SkuID,
				strconv.FormatInt(row.QtySold, 10),
				strconv.FormatInt(row.QtyVoided, 10),
				strconv.FormatInt(row.GrossMinor, 10),
				strconv.FormatInt(row.NetMinor, 10),
				row.Currency,
			})
		}
		cw.Flush()
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":  orgID,
		"day":          day,
		"items":        rows,
		"honest_empty": len(rows) == 0,
	})
}

// listHqSalesRows returns all SKU rows for org+day (optional location filter).
func (s *Service) listHqSalesRows(ctx context.Context, retailerID, day, locationID string) ([]HqSalesDayDTO, error) {
	retailerID = strings.TrimSpace(retailerID)
	day = strings.TrimSpace(day)
	locationID = strings.TrimSpace(locationID)
	if retailerID == "" || day == "" {
		return nil, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []HqSalesDayDTO
		for k, v := range s.hqSalesDaily {
			if k.RetailerID != retailerID || k.Day != day {
				continue
			}
			if locationID != "" && k.LocationID != locationID {
				continue
			}
			out = append(out, v)
		}
		return out, nil
	}
	sql := `SELECT RetailerId, LocationId, CAST(Day AS STRING), SkuId, QtySold, QtyVoided, GrossMinor, NetMinor, Currency
		FROM RetailerHqSalesDaily WHERE RetailerId = @rid AND Day = @day`
	params := map[string]any{"rid": retailerID, "day": parseHQDay(day)}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []HqSalesDayDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerHqSalesDaily") {
				return nil, nil
			}
			return nil, err
		}
		var d HqSalesDayDTO
		var dayStr string
		if err := row.Columns(&d.RetailerID, &d.LocationID, &dayStr, &d.SkuID, &d.QtySold, &d.QtyVoided, &d.GrossMinor, &d.NetMinor, &d.Currency); err != nil {
			return nil, err
		}
		d.Day = dayStr
		if len(d.Day) > 10 {
			d.Day = d.Day[:10]
		}
		out = append(out, d)
	}
	return out, nil
}
