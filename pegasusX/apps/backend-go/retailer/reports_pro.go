package retailer

import (
	"context"
	"encoding/csv"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type salesBucket struct {
	Key        string `json:"key"`
	SalesMinor int64  `json:"sales_minor"`
	SaleCount  int    `json:"sale_count"`
	Units      int64  `json:"units"`
}

// HandleReportsSummary serves GET /v1/retailer/reports/summary
func (s *Service) HandleReportsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, orgID, ok := s.reportsAuth(w, r)
	if !ok {
		return
	}
	from, to := s.reportWindow(r)
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))

	salesMinor, saleCount, topSKUs := s.aggregateSales(orgID, locID, from, to)
	onHandSKUs, lowStock := s.aggregateInventory(r.Context(), orgID, locID)
	openVariances := s.countClosedShiftVariances(r.Context(), orgID, locID)

	writeJSON(w, http.StatusOK, map[string]any{
		"from":              from.UTC().Format(time.RFC3339),
		"to":                to.UTC().Format(time.RFC3339),
		"location_id":       locID,
		"sales_minor":       salesMinor,
		"sale_count":        saleCount,
		"on_hand_sku_count": onHandSKUs,
		"low_stock_count":   lowStock,
		"open_variances":    openVariances,
		"top_skus":          topSKUs,
		"pack":              PackREPORTSPRO,
		"actor":             auth.ResolveRetailerUserID(claims),
	})
}

// HandleReportsSales serves GET /v1/retailer/reports/sales
func (s *Service) HandleReportsSales(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.reportsAuth(w, r)
	if !ok {
		return
	}
	from, to := s.reportWindow(r)
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	groupBy := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("group_by")))
	if groupBy == "" {
		groupBy = "sku"
	}

	buckets := map[string]*salesBucket{}
	s.mu.RLock()
	for _, sale := range s.posSales {
		if sale.RetailerID != orgID || sale.Status != "COMPLETED" {
			continue
		}
		if locID != "" && sale.LocationID != locID {
			continue
		}
		t, okT := parseReportTime(sale.CreatedAt)
		if !okT || t.Before(from) || !t.Before(to) {
			continue
		}
		switch groupBy {
		case "hour":
			key := t.UTC().Format("2006-01-02T15")
			b := ensureBucket(buckets, key)
			b.SalesMinor += sale.TotalMinor
			b.SaleCount++
			for _, line := range sale.Lines {
				b.Units += line.Qty
			}
		case "cashier":
			key := sale.CashierUserID
			if key == "" {
				key = "unknown"
			}
			b := ensureBucket(buckets, key)
			b.SalesMinor += sale.TotalMinor
			b.SaleCount++
		default:
			for _, line := range sale.Lines {
				b := ensureBucket(buckets, line.Sku)
				lt := line.LineTotalMinor
				if lt == 0 {
					lt = line.Qty * line.UnitPriceMinor
				}
				b.SalesMinor += lt
				b.Units += line.Qty
				b.SaleCount++
			}
		}
	}
	s.mu.RUnlock()

	items := make([]salesBucket, 0, len(buckets))
	for _, b := range buckets {
		items = append(items, *b)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SalesMinor > items[j].SalesMinor })

	if wantsCSV(r) {
		writeSalesCSV(w, items)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from":     from.UTC().Format(time.RFC3339),
		"to":       to.UTC().Format(time.RFC3339),
		"group_by": groupBy,
		"items":    items,
	})
}

// HandleReportsInventory serves GET /v1/retailer/reports/inventory
func (s *Service) HandleReportsInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.reportsAuth(w, r)
	if !ok {
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	balances, _ := s.listStockBalances(r.Context(), orgID, locID)
	movements, _ := s.listStockMovements(r.Context(), orgID, locID, "", 500)

	var totalOnHand int64
	type skuRow struct {
		Sku      string `json:"sku"`
		OnHand   int64  `json:"on_hand"`
		StockBin string `json:"stock_bin"`
	}
	rows := make([]skuRow, 0, len(balances))
	for _, b := range balances {
		totalOnHand += b.OnHand
		rows = append(rows, skuRow{Sku: b.Sku, OnHand: b.OnHand, StockBin: b.StockBin})
	}
	byType := map[string]int64{}
	var shrink int64
	for _, m := range movements {
		byType[m.MovementType] += m.Qty
		if (m.MovementType == MoveAdjust || m.MovementType == MoveCountVariance) && m.Qty < 0 {
			shrink += -m.Qty
		}
	}
	if wantsCSV(r) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="inventory.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"sku", "stock_bin", "on_hand"})
		for _, row := range rows {
			_ = cw.Write([]string{row.Sku, row.StockBin, strconv.FormatInt(row.OnHand, 10)})
		}
		cw.Flush()
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"location_id":      locID,
		"total_on_hand":    totalOnHand,
		"sku_bin_rows":     len(rows),
		"balances":         rows,
		"movement_by_type": byType,
		"shrink_units":     shrink,
	})
}

// HandleReportsShifts serves GET /v1/retailer/reports/shifts
func (s *Service) HandleReportsShifts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, orgID, ok := s.reportsAuth(w, r)
	if !ok {
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	items, _ := s.listShifts(r.Context(), orgID, locID, 100)
	type row struct {
		ShiftID       string `json:"shift_id"`
		Status        string `json:"status"`
		LocationID    string `json:"location_id"`
		VarianceMinor *int64 `json:"variance_minor,omitempty"`
		OpenedAt      string `json:"opened_at,omitempty"`
		ClosedAt      string `json:"closed_at,omitempty"`
	}
	out := make([]row, 0, len(items))
	for _, sh := range items {
		out = append(out, row{
			ShiftID: sh.ShiftID, Status: sh.Status, LocationID: sh.LocationID,
			VarianceMinor: sh.VarianceMinor, OpenedAt: sh.OpenedAt, ClosedAt: sh.ClosedAt,
		})
	}
	if wantsCSV(r) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="shifts.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"shift_id", "status", "location_id", "variance_minor", "opened_at", "closed_at"})
		for _, rrow := range out {
			v := ""
			if rrow.VarianceMinor != nil {
				v = strconv.FormatInt(*rrow.VarianceMinor, 10)
			}
			_ = cw.Write([]string{rrow.ShiftID, rrow.Status, rrow.LocationID, v, rrow.OpenedAt, rrow.ClosedAt})
		}
		cw.Flush()
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// HandleReportsExport serves GET /v1/retailer/reports/export?report=&format=csv
func (s *Service) HandleReportsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	q := r.URL.Query()
	q.Set("format", "csv")
	r.URL.RawQuery = q.Encode()
	report := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("report")))
	switch report {
	case "inventory":
		s.HandleReportsInventory(w, r)
	case "shifts":
		s.HandleReportsShifts(w, r)
	case "sales", "":
		s.HandleReportsSales(w, r)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown_report"})
	}
}

func (s *Service) reportsAuth(w http.ResponseWriter, r *http.Request) (auth.Claims, string, bool) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermReportsView) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermReportsView})
		return auth.Claims{}, "", false
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return auth.Claims{}, "", false
	}
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackREPORTSPRO) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackREPORTSPRO, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}
	return claims, orgID, true
}

func (s *Service) reportWindow(r *http.Request) (time.Time, time.Time) {
	to := s.now().UTC()
	from := to.Add(-7 * 24 * time.Hour)
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t.UTC()
		} else if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			from = t.UTC()
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t.UTC()
		} else if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			to = t.UTC()
		}
	}
	return from, to
}

func parseReportTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func wantsCSV(r *http.Request) bool {
	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		return true
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/csv")
}

func ensureBucket(m map[string]*salesBucket, key string) *salesBucket {
	b := m[key]
	if b == nil {
		b = &salesBucket{Key: key}
		m[key] = b
	}
	return b
}

func writeSalesCSV(w http.ResponseWriter, items []salesBucket) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="sales.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"key", "sales_minor", "sale_count", "units"})
	for _, b := range items {
		_ = cw.Write([]string{b.Key, strconv.FormatInt(b.SalesMinor, 10), strconv.Itoa(b.SaleCount), strconv.FormatInt(b.Units, 10)})
	}
	cw.Flush()
}

func (s *Service) aggregateSales(orgID, locID string, from, to time.Time) (salesMinor int64, saleCount int, top []map[string]any) {
	type skuAgg struct {
		minor int64
		units int64
	}
	bySku := map[string]*skuAgg{}
	s.mu.RLock()
	for _, sale := range s.posSales {
		if sale.RetailerID != orgID || sale.Status != "COMPLETED" {
			continue
		}
		if locID != "" && sale.LocationID != locID {
			continue
		}
		t, okT := parseReportTime(sale.CreatedAt)
		if !okT || t.Before(from) || !t.Before(to) {
			continue
		}
		salesMinor += sale.TotalMinor
		saleCount++
		for _, line := range sale.Lines {
			a := bySku[line.Sku]
			if a == nil {
				a = &skuAgg{}
				bySku[line.Sku] = a
			}
			lt := line.LineTotalMinor
			if lt == 0 {
				lt = line.Qty * line.UnitPriceMinor
			}
			a.minor += lt
			a.units += line.Qty
		}
	}
	s.mu.RUnlock()
	type pair struct {
		sku string
		a   *skuAgg
	}
	var pairs []pair
	for sku, a := range bySku {
		pairs = append(pairs, pair{sku, a})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].a.minor > pairs[j].a.minor })
	for i, p := range pairs {
		if i >= 10 {
			break
		}
		top = append(top, map[string]any{"sku": p.sku, "sales_minor": p.a.minor, "units": p.a.units})
	}
	return salesMinor, saleCount, top
}

func (s *Service) aggregateInventory(ctx context.Context, orgID, locID string) (skuCount int, lowStock int) {
	balances, _ := s.listStockBalances(ctx, orgID, locID)
	seen := map[string]bool{}
	for _, b := range balances {
		if !seen[b.Sku] {
			seen[b.Sku] = true
			skuCount++
		}
		if b.OnHand > 0 && b.OnHand <= 5 {
			lowStock++
		}
	}
	return skuCount, lowStock
}

func (s *Service) countClosedShiftVariances(ctx context.Context, orgID, locID string) int {
	items, _ := s.listShifts(ctx, orgID, locID, 100)
	n := 0
	for _, sh := range items {
		if sh.Status == ShiftClosed && sh.VarianceMinor != nil && abs64(*sh.VarianceMinor) > 0 {
			n++
		}
	}
	return n
}
