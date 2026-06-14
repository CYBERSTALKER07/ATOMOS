package supplier

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
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
	TotalRetailers  int64               `json:"total_retailers"`
	TotalPallets    int64               `json:"total_pallets"`
	TotalValue      int64               `json:"total_value"`
	PredictionCount int64               `json:"prediction_count"`
	Items           []demandSummaryItem `json:"items"`
	GeneratedAt     string              `json:"generated_at"`
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
	writeJSON(w, http.StatusOK, buildRevenueResponse(orders, s.currency, now, 30))
}

// HandleAnalyticsDemandToday serves GET /v1/supplier/analytics/demand/today.
func (s *Service) HandleAnalyticsDemandToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	now := s.now().UTC()
	resp, err := s.buildDemandToday(r.Context(), sid, now)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_demand_summary_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleAnalyticsDemandHistory serves GET /v1/supplier/analytics/demand/history.
func (s *Service) HandleAnalyticsDemandHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	now := s.now().UTC()
	orders, err := s.listSupplierOrders(r.Context(), sid, "", "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_orders_failed"})
		return
	}
	recs, _ := s.listAIRecommendations(r.Context(), sid, AIRecommendationQuery{Status: "PENDING", Limit: 100})
	writeJSON(w, http.StatusOK, buildDemandHistory(orders, recs, now, 14))
}

func (s *Service) buildDemandToday(ctx context.Context, supplierID string, now time.Time) (demandSummaryResponse, error) {
	recs, err := s.listAIRecommendations(ctx, supplierID, AIRecommendationQuery{Status: "PENDING", Limit: 100})
	if err != nil {
		return demandSummaryResponse{}, err
	}
	items := make([]demandSummaryItem, 0)
	retailers := map[string]struct{}{}
	var predictionCount int64
	var totalValue int64
	var totalQty int64

	for _, rec := range recs {
		if !isDemandRecommendation(rec) {
			continue
		}
		predictionCount++
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

	return demandSummaryResponse{
		TotalRetailers:  int64(len(retailers)),
		TotalPallets:    totalQty,
		TotalValue:      totalValue,
		PredictionCount: predictionCount,
		Items:           items,
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
	if currency == "" {
		currency = "UZS"
	}
	return analyticsRevenueResponse{
		Currency:    currency,
		TotalMinor:  total,
		Series:      series,
		GeneratedAt: now.Format(time.RFC3339Nano),
	}
}

func buildDemandHistory(orders []SupplierOrder, recs []AIRecommendation, now time.Time, days int) demandHistoryResponse {
	if days < 1 {
		days = 14
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	actualByDay := map[string]int64{}
	actualQtyByDay := map[string]int64{}
	predictedByDay := map[string]int64{}
	predictedQtyByDay := map[string]int64{}
	for day := 0; day < days; day++ {
		key := start.AddDate(0, 0, day).Format("2006-01-02")
		actualByDay[key] = 0
		actualQtyByDay[key] = 0
		predictedByDay[key] = 0
		predictedQtyByDay[key] = 0
	}
	for _, order := range orders {
		if !strings.EqualFold(order.Status, "COMPLETED") {
			continue
		}
		updatedAt, ok := parseOrderTimestamp(order.UpdatedAt)
		if !ok || updatedAt.Before(start) {
			continue
		}
		key := updatedAt.Format("2006-01-02")
		actualByDay[key] += order.TotalMinor
		actualQtyByDay[key]++
	}
	for _, rec := range recs {
		if !isDemandRecommendation(rec) {
			continue
		}
		generatedAt, ok := parseOrderTimestamp(rec.GeneratedAt)
		if !ok {
			generatedAt = now
		}
		key := generatedAt.Format("2006-01-02")
		qty := int64(rec.Score)
		if qty <= 0 {
			qty = 1
		}
		predictedByDay[key] += qty * 100_00
		predictedQtyByDay[key] += qty
	}
	timeSeries := make([]demandHistoryPoint, 0, days)
	for day := 0; day < days; day++ {
		key := start.AddDate(0, 0, day).Format("2006-01-02")
		timeSeries = append(timeSeries, demandHistoryPoint{
			Date:         key,
			Predicted:    predictedByDay[key],
			Actual:       actualByDay[key],
			PredictedQty: predictedQtyByDay[key],
			ActualQty:    actualQtyByDay[key],
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
