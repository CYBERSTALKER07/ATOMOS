package supplier

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
)

type analyticsVelocityPoint struct {
	Date             string `json:"date"`
	OrdersCreated    int    `json:"orders_created"`
	OrdersCompleted  int    `json:"orders_completed"`
}

type analyticsVelocityResponse struct {
	PeriodDays int                      `json:"period_days"`
	Points     []analyticsVelocityPoint `json:"points"`
	GeneratedAt string                  `json:"generated_at"`
}

type analyticsRevenuePoint struct {
	Date         string `json:"date"`
	RevenueMinor int64  `json:"revenue_minor"`
}

type analyticsRevenueResponse struct {
	Currency    string                  `json:"currency"`
	TotalMinor  int64                   `json:"total_minor"`
	Series      []analyticsRevenuePoint `json:"series"`
	GeneratedAt string                  `json:"generated_at"`
}

type demandSummaryItem struct {
	SkuID         string `json:"sku_id"`
	ProductName   string `json:"product_name"`
	TotalQty      int64  `json:"total_qty"`
	RetailerCount int64  `json:"retailer_count"`
}

type demandSummaryResponse struct {
	TotalRetailers  int64                        `json:"total_retailers"`
	TotalPallets    int64                        `json:"total_pallets"`
	TotalValue      int64                        `json:"total_value"`
	PredictionCount int64                        `json:"prediction_count"`
	Items           []demandSummaryItem          `json:"items"`
	GeneratedAt     string                       `json:"generated_at"`
	BaselineSource  string                       `json:"baseline_source,omitempty"`
	Granularity     string                       `json:"granularity,omitempty"`
	Confidence      *planning.ForecastConfidence `json:"confidence,omitempty"`
}

type demandHistoryPoint struct {
	Date          string `json:"date"`
	Predicted     int64  `json:"predicted"`
	Actual        int64  `json:"actual"`
	PredictedQty  int64  `json:"predicted_qty"`
	ActualQty     int64  `json:"actual_qty"`
}

type demandUpcomingRow struct {
	Date          string `json:"date"`
	RetailerName  string `json:"retailer_name"`
	SkuID         string `json:"sku_id"`
	ProductName   string `json:"product_name"`
	PredictedQty  int64  `json:"predicted_qty"`
}

type demandHistoryResponse struct {
	TimeSeries []demandHistoryPoint `json:"time_series"`
	Upcoming   []demandUpcomingRow  `json:"upcoming"`
}

type demandAccuracySeriesRow struct {
	Date           string  `json:"date"`
	WarehouseID    string  `json:"warehouse_id"`
	ProductID      string  `json:"product_id"`
	ForecastQty    int64   `json:"forecast_qty"`
	ActualQty      int64   `json:"actual_qty"`
	Wape7          float64 `json:"wape_7"`
	Wape28         float64 `json:"wape_28"`
	Bias7          float64 `json:"bias_7"`
	Bias28         float64 `json:"bias_28"`
	TrackingSignal float64 `json:"tracking_signal"`
	AlertTs        bool    `json:"alert_ts"`
}

type demandAccuracyResponse struct {
	Enabled        bool                      `json:"enabled"`
	PeriodDays     int                       `json:"period_days"`
	AsOf           string                    `json:"as_of,omitempty"`
	Wape7          float64                   `json:"wape_7"`
	Wape28         float64                   `json:"wape_28"`
	Bias7          float64                   `json:"bias_7"`
	Bias28         float64                   `json:"bias_28"`
	TrackingSignal float64                   `json:"tracking_signal"`
	SampleDays7    int64                     `json:"sample_days_7"`
	SampleDays28   int64                     `json:"sample_days_28"`
	AlertCount     int                       `json:"alert_count"`
	ForecastUnits  int64                     `json:"forecast_units"`
	ActualUnits    int64                     `json:"actual_units"`
	Series         []demandAccuracySeriesRow `json:"series"`
	GeneratedAt    string                    `json:"generated_at"`
}

// HandleAnalyticsVelocity serves GET /v1/supplier/analytics/velocity.
func (s *Service) HandleAnalyticsVelocity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	now := s.now().UTC()
	orders, err := s.listSupplierOrders(r.Context(), s.scopedSupplierID(r), "", "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_orders_failed"})
		return
	}
	writeJSON(w, http.StatusOK, buildVelocityResponse(orders, now, 7))
}

// HandleAnalyticsRevenue serves GET /v1/supplier/analytics/revenue.
func (s *Service) HandleAnalyticsRevenue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	now := s.now().UTC()
	orders, err := s.listSupplierOrders(r.Context(), s.scopedSupplierID(r), "", "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_orders_failed"})
		return
	}
	cur, err := auth.CoalesceCurrency(r.Context(), s.scopedSupplierID(r), s.currency)
	if err != nil {
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, buildRevenueResponse(orders, cur, now, 30))
}

// HandleAnalyticsDemandToday serves GET /v1/supplier/analytics/demand/today.
func (s *Service) HandleAnalyticsDemandToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	now := s.now().UTC()
	query := parseDemandAnalyticsQuery(r)
	resp, err := s.buildDemandToday(r.Context(), sid, now, query)
	if err != nil {
		if s.log != nil {
			s.log.Warn("build demand today failed", "supplier_id", sid, "err", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_demand_summary_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleAnalyticsDemandHistory serves GET /v1/supplier/analytics/demand/history.
// Series = DemandForecastBaseline units vs completed LineItemsJson units (same grain rolled to day).
func (s *Service) HandleAnalyticsDemandHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	now := s.now().UTC()
	days := 14
	end := now.Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))

	predictedQtyByDay := map[string]int64{}
	actualQtyByDay := map[string]int64{}
	if s.portalSpanner != nil {
		baselines, err := planning.LoadBaselineDayTotals(r.Context(), s.portalSpanner, sid, start, end)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_baselines_failed"})
			return
		}
		predictedQtyByDay = baselines
		actuals, err := planning.LoadCompletedActuals(r.Context(), s.portalSpanner, sid, start, end)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_actuals_failed"})
			return
		}
		actualQtyByDay = planning.DayActualTotals(actuals, sid)
	}
	recs, _ := s.listAIRecommendations(r.Context(), sid, AIRecommendationQuery{Status: "PENDING", Limit: 100})
	writeJSON(w, http.StatusOK, buildDemandHistoryFromDayMaps(predictedQtyByDay, actualQtyByDay, recs, now, days))
}

// HandleAnalyticsDemandAccuracy serves GET /v1/supplier/analytics/demand/accuracy (§8.4).
func (s *Service) HandleAnalyticsDemandAccuracy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	wh := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
	pid := strings.TrimSpace(r.URL.Query().Get("product_id"))
	days := 28
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if n, err := parsePositiveInt(raw); err == nil && n <= 90 {
			days = n
		}
	}
	resp := demandAccuracyResponse{
		Enabled:     planning.ForecastAccuracyEnabled(),
		PeriodDays:  days,
		GeneratedAt: s.now().UTC().Format(time.RFC3339Nano),
		Series:      []demandAccuracySeriesRow{},
	}
	if !resp.Enabled || s.portalSpanner == nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	rows, err := planning.ListAccuracyRows(r.Context(), s.portalSpanner, sid, wh, pid, days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_accuracy_failed"})
		return
	}
	var sumAbs, sumSigned, sumActual, sumForecast int64
	var alertCount int
	var maxAbsTS float64
	var sample7, sample28 int64
	for _, row := range rows {
		sumAbs += row.AbsError
		sumSigned += row.SignedError
		sumActual += row.ActualQty
		sumForecast += row.ForecastQty
		if row.AlertTs {
			alertCount++
		}
		if abs := mathAbs(row.TrackingSignal); abs >= maxAbsTS {
			maxAbsTS = abs
			resp.TrackingSignal = row.TrackingSignal
		}
		if d := row.ForecastDate.String(); d > resp.AsOf {
			resp.AsOf = d
		}
		if row.SampleDays7 > sample7 {
			sample7 = row.SampleDays7
		}
		if row.SampleDays28 > sample28 {
			sample28 = row.SampleDays28
		}
		resp.Series = append(resp.Series, demandAccuracySeriesRow{
			Date:           row.ForecastDate.String(),
			WarehouseID:    row.WarehouseID,
			ProductID:      row.ProductID,
			ForecastQty:    row.ForecastQty,
			ActualQty:      row.ActualQty,
			Wape7:          row.Wape7,
			Wape28:         row.Wape28,
			Bias7:          row.Bias7,
			Bias28:         row.Bias28,
			TrackingSignal: row.TrackingSignal,
			AlertTs:        row.AlertTs,
		})
	}
	if sumActual > 0 {
		resp.Wape28 = float64(sumAbs) / float64(sumActual)
		resp.Bias28 = float64(sumSigned) / float64(sumActual)
		resp.Wape7 = resp.Wape28
		resp.Bias7 = resp.Bias28
	}
	resp.SampleDays7 = sample7
	resp.SampleDays28 = sample28
	resp.AlertCount = alertCount
	resp.ForecastUnits = sumForecast
	resp.ActualUnits = sumActual
	writeJSON(w, http.StatusOK, resp)
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func parsePositiveInt(raw string) (int, error) {
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, strconv.ErrSyntax
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}

func (s *Service) buildDemandToday(ctx context.Context, supplierID string, now time.Time, query demandAnalyticsQuery) (demandSummaryResponse, error) {
	recs, err := s.listAIRecommendations(ctx, supplierID, AIRecommendationQuery{Status: "PENDING", Limit: 100})
	if err != nil {
		if s.log != nil {
			s.log.Warn("demand today ai recommendations read failed", "supplier_id", supplierID, "err", err)
		}
		recs = nil
	}
	items := make([]demandSummaryItem, 0)
	retailers := map[string]struct{}{}
	var predictionCount int64
	var totalValue int64
	var totalQty int64
	source := ""

	for _, rec := range recs {
		if query.Granularity == "micro" && query.RetailerID != "" &&
			!strings.EqualFold(strings.TrimSpace(rec.AggregateID), query.RetailerID) {
			continue
		}
		if !isDemandRecommendation(rec) {
			continue
		}
		predictionCount++
		if source == "" {
			source = "inventory_hint"
		} else if source != "inventory_hint" && source != "demand_forecast_baseline" {
			source = "mixed"
		}
		qty := int64(rec.Score)
		if qty <= 0 {
			qty = 1
		}
		totalQty += qty
		totalValue += qty * 100_00
		sku := strings.TrimSpace(rec.AggregateID)
		if sku == "" {
			sku = rec.RecommendationID
		}
		retailers[rec.AggregateID] = struct{}{}
		items = append(items, demandSummaryItem{
			SkuID:         sku,
			ProductName:   rec.Explanation,
			TotalQty:      qty,
			RetailerCount: 1,
		})
	}

	if predictionCount == 0 && s.inventorySvc != nil {
		levels, listErr := s.inventorySvc.ListBySupplier(ctx, supplierID)
		if listErr == nil {
			for _, level := range levels {
				if level.QuantityOnHand > level.ReorderThreshold {
					continue
				}
				predictionCount++
				gap := level.ReorderThreshold - level.QuantityOnHand
				if gap < 1 {
					gap = 1
				}
				totalQty += gap
				items = append(items, demandSummaryItem{
					SkuID:       level.ProductID,
					ProductName: level.ProductID,
					TotalQty:    gap,
				})
			}
		}
	}

	merged := s.mergeDemandBaselineItems(ctx, supplierID, now, items, &source, query.WarehouseID)
	conf, confErr := planning.AggregateDemandConfidence(ctx, s.portalSpanner, supplierID, planning.DemandConfidenceQuery{
		Granularity:     query.Granularity,
		WarehouseID:     query.WarehouseID,
		RetailerID:      query.RetailerID,
		ForecastDate:    now,
		FallbackQty:     totalQty,
		SourceHint:      source,
		PredictionCount: predictionCount,
	})
	if confErr != nil {
		if s.log != nil {
			s.log.Warn("demand today confidence aggregation failed", "supplier_id", supplierID, "err", confErr)
		}
		conf = planning.FallbackDemandConfidence(totalQty, source, predictionCount)
	}
	normalizedSource := planning.NormalizeBaselineSource(source)
	conf.BaselineSource = planning.NormalizeBaselineSource(conf.BaselineSource, source)
	return demandSummaryResponse{
		TotalRetailers:  int64(len(retailers)),
		TotalPallets:    totalQty,
		TotalValue:      totalValue,
		PredictionCount: predictionCount,
		Items:           merged,
		BaselineSource:  normalizedSource,
		Granularity:     query.Granularity,
		Confidence:      &conf,
		GeneratedAt:     now.Format(time.RFC3339Nano),
	}, nil
}

func (s *Service) listAIRecommendations(ctx context.Context, supplierID string, query AIRecommendationQuery) ([]AIRecommendation, error) {
	repo, ok := s.repo.(aiRecommendationRepository)
	if !ok {
		return nil, nil
	}
	return repo.ListAIRecommendations(ctx, supplierID, query)
}

func isDemandRecommendation(rec AIRecommendation) bool {
	aggregate := strings.ToUpper(strings.TrimSpace(rec.AggregateType))
	action := strings.ToUpper(strings.TrimSpace(rec.Action))
	return strings.Contains(aggregate, "INVENTORY") ||
		strings.Contains(aggregate, "DEMAND") ||
		strings.Contains(action, "RESTOCK") ||
		strings.Contains(action, "REPLENISH")
}

func buildVelocityResponse(orders []SupplierOrder, now time.Time, days int) analyticsVelocityResponse {
	if days < 1 {
		days = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	created := map[string]int{}
	completed := map[string]int{}
	for day := 0; day < days; day++ {
		key := start.AddDate(0, 0, day).Format("2006-01-02")
		created[key] = 0
		completed[key] = 0
	}
	for _, order := range orders {
		createdAt, ok := parseOrderTimestamp(order.CreatedAt)
		if !ok || createdAt.Before(start) {
			continue
		}
		created[createdAt.Format("2006-01-02")]++
		if strings.EqualFold(order.Status, "COMPLETED") {
			updatedAt, ok := parseOrderTimestamp(order.UpdatedAt)
			if ok && !updatedAt.Before(start) {
				completed[updatedAt.Format("2006-01-02")]++
			}
		}
	}
	points := make([]analyticsVelocityPoint, 0, days)
	for day := 0; day < days; day++ {
		key := start.AddDate(0, 0, day).Format("2006-01-02")
		points = append(points, analyticsVelocityPoint{
			Date:            key,
			OrdersCreated:   created[key],
			OrdersCompleted: completed[key],
		})
	}
	return analyticsVelocityResponse{
		PeriodDays:  days,
		Points:      points,
		GeneratedAt: now.Format(time.RFC3339Nano),
	}
}

func buildRevenueResponse(orders []SupplierOrder, currency string, now time.Time, days int) analyticsRevenueResponse {
	if days < 1 {
		days = 30
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	seriesMap := map[string]int64{}
	for day := 0; day < days; day++ {
		seriesMap[start.AddDate(0, 0, day).Format("2006-01-02")] = 0
	}
	var total int64
	for _, order := range orders {
		if !strings.EqualFold(order.Status, "COMPLETED") {
			continue
		}
		updatedAt, ok := parseOrderTimestamp(order.UpdatedAt)
		if !ok || updatedAt.Before(start) {
			continue
		}
		key := updatedAt.Format("2006-01-02")
		seriesMap[key] += order.TotalMinor
		total += order.TotalMinor
	}
	series := make([]analyticsRevenuePoint, 0, days)
	for day := 0; day < days; day++ {
		key := start.AddDate(0, 0, day).Format("2006-01-02")
		series = append(series, analyticsRevenuePoint{Date: key, RevenueMinor: seriesMap[key]})
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	return analyticsRevenueResponse{
		Currency:    currency,
		TotalMinor:  total,
		Series:      series,
		GeneratedAt: now.Format(time.RFC3339Nano),
	}
}

func buildDemandHistoryFromDayMaps(
	predictedQtyByDay, actualQtyByDay map[string]int64,
	recs []AIRecommendation,
	now time.Time,
	days int,
) demandHistoryResponse {
	if days < 1 {
		days = 14
	}
	if predictedQtyByDay == nil {
		predictedQtyByDay = map[string]int64{}
	}
	if actualQtyByDay == nil {
		actualQtyByDay = map[string]int64{}
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	timeSeries := make([]demandHistoryPoint, 0, days)
	for day := 0; day < days; day++ {
		key := start.AddDate(0, 0, day).Format("2006-01-02")
		pred := predictedQtyByDay[key]
		act := actualQtyByDay[key]
		timeSeries = append(timeSeries, demandHistoryPoint{
			Date:         key,
			Predicted:    pred,
			Actual:       act,
			PredictedQty: pred,
			ActualQty:    act,
		})
	}
	upcoming := make([]demandUpcomingRow, 0, len(recs))
	for _, rec := range recs {
		if !isDemandRecommendation(rec) {
			continue
		}
		qty := int64(rec.Score)
		if qty <= 0 {
			qty = 1
		}
		upcoming = append(upcoming, demandUpcomingRow{
			Date:         rec.ExpiresAt,
			RetailerName: rec.AggregateID,
			SkuID:        rec.AggregateID,
			ProductName:  rec.Explanation,
			PredictedQty: qty,
		})
	}
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].Date < upcoming[j].Date
	})
	return demandHistoryResponse{TimeSeries: timeSeries, Upcoming: upcoming}
}

func parseOrderTimestamp(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts.UTC(), true
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts.UTC(), true
	}
	return time.Time{}, false
}
