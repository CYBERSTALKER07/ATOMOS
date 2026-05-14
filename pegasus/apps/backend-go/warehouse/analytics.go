package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Analytics ────────────────────────────────────────────────────────────────
// Warehouse-scoped analytics: revenue, order volume, fleet utilization.

type AnalyticsResponse struct {
	WarehouseID     string          `json:"warehouse_id"`
	Period          string          `json:"period"` // "7d" or "30d"
	TotalRevenue    int64           `json:"total_revenue"`
	TotalOrders     int64           `json:"total_orders"`
	CompletedOrders int64           `json:"completed_orders"`
	CancelledOrders int64           `json:"cancelled_orders"`
	AvgOrderValue   float64         `json:"avg_order_value"`
	TopProducts     []TopProduct    `json:"top_products"`
	DailyBreakdown  []DailyMetric   `json:"daily_breakdown"`
	FleetUtil       FleetUtilMetric `json:"fleet_utilization"`
	ImportFreshness ImportFreshness `json:"import_freshness"`
	ImportAnomalyQ  ImportAnomalyQ  `json:"import_anomaly_queue"`
}

type ImportFreshness struct {
	AppliedRows30D   int64  `json:"applied_rows_30d"`
	AppliedSKUs30D   int64  `json:"applied_skus_30d"`
	QuantityDelta30D int64  `json:"quantity_delta_30d"`
	LastSessionID    string `json:"last_session_id,omitempty"`
	LastAppliedAt    string `json:"last_applied_at,omitempty"`
}

type ImportAnomalyQ struct {
	OpenRows30D        int64  `json:"open_rows_30d"`
	AffectedSessions30 int64  `json:"affected_sessions_30d"`
	LastSessionID      string `json:"last_session_id,omitempty"`
	LastDetectedAt     string `json:"last_detected_at,omitempty"`
	LastDetail         string `json:"last_detail,omitempty"`
}

type TopProduct struct {
	SkuID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
	TotalQty    int64  `json:"total_qty"`
	Revenue     int64  `json:"revenue"`
}

type DailyMetric struct {
	Date            string `json:"date"`
	Orders          int64  `json:"orders"`
	CompletedOrders int64  `json:"completed"`
	Revenue         int64  `json:"revenue"`
}

type FleetUtilMetric struct {
	TotalDrivers   int64   `json:"total_drivers"`
	ActiveDrivers  int64   `json:"active_drivers"`
	UtilizationPct float64 `json:"utilization_pct"`
	AvgStopsPerDay float64 `json:"avg_stops_per_day"`
}

// HandleOpsAnalytics — GET for /v1/warehouse/ops/analytics
func HandleOpsAnalytics(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		period := r.URL.Query().Get("period")
		days := 7
		if period == "30d" {
			days = 30
		}
		periodLabel := "7d"
		if days == 30 {
			periodLabel = "30d"
		}

		now := time.Now().UTC()
		startDate := now.AddDate(0, 0, -days).Truncate(24 * time.Hour)

		resp := AnalyticsResponse{
			WarehouseID: ops.WarehouseID,
			Period:      periodLabel,
		}

		ctx := r.Context()

		// Total + completed + cancelled orders
		countQuery := func(states string) int64 {
			sql := `SELECT COUNT(*)
				FROM Orders o
				LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
				WHERE o.SupplierId = @sid AND o.WarehouseId = @whId AND o.CreatedAt >= @start`
			if states != "" {
				sql += " AND o.State IN UNNEST(@states)"
			}
			p := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID, "start": startDate}
			if states != "" {
				p["states"] = splitStates(states)
			}
			sql, p = auth.AppendRegionFilter(ctx, sql, p, "rt")
			iter := spannerx.StaleQuery(ctx, spannerClient, spanner.Statement{SQL: sql, Params: p})
			defer iter.Stop()
			row, err := iter.Next()
			if err != nil {
				return 0
			}
			var c int64
			row.Columns(&c)
			return c
		}

		resp.TotalOrders = countQuery("")
		resp.CompletedOrders = countQuery("COMPLETED")
		resp.CancelledOrders = countQuery("CANCELLED")

		// Revenue
		revSQL := `SELECT COALESCE(SUM(o.TotalAmount), 0)
			FROM Orders o
			LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
			WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			  AND o.State = 'COMPLETED' AND o.CreatedAt >= @start`
		revParams := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID, "start": startDate}
		revSQL, revParams = auth.AppendRegionFilter(ctx, revSQL, revParams, "rt")
		revStmt := spanner.Statement{SQL: revSQL, Params: revParams}
		revIter := spannerx.StaleQuery(ctx, spannerClient, revStmt)
		defer revIter.Stop()
		if row, err := revIter.Next(); err == nil {
			row.Columns(&resp.TotalRevenue)
		}

		if resp.CompletedOrders > 0 {
			resp.AvgOrderValue = float64(resp.TotalRevenue) / float64(resp.CompletedOrders)
		}

		// Top products
		topSQL := `SELECT li.SkuId, COALESCE(sp.Name, ''), SUM(li.Quantity), SUM(li.Quantity * li.UnitPrice)
			FROM OrderLineItems li
			JOIN Orders o ON li.OrderId = o.OrderId
			LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
			LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
			WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			  AND o.State = 'COMPLETED' AND o.CreatedAt >= @start`
		topParams := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID, "start": startDate}
		topSQL, topParams = auth.AppendRegionFilter(ctx, topSQL, topParams, "rt")
		topSQL += ` GROUP BY li.SkuId, sp.Name
			ORDER BY SUM(li.Quantity) DESC
			LIMIT 10`
		topStmt := spanner.Statement{SQL: topSQL, Params: topParams}
		topIter := spannerx.StaleQuery(ctx, spannerClient, topStmt)
		defer topIter.Stop()
		for {
			row, err := topIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var tp TopProduct
			if err := row.Columns(&tp.SkuID, &tp.ProductName, &tp.TotalQty, &tp.Revenue); err == nil {
				resp.TopProducts = append(resp.TopProducts, tp)
			}
		}
		if resp.TopProducts == nil {
			resp.TopProducts = []TopProduct{}
		}

		// Daily breakdown
		dailySQL := `SELECT CAST(DATE(o.CreatedAt) AS STRING) as day,
		             COUNT(*) as total,
		             COUNTIF(o.State = 'COMPLETED') as completed,
		             COALESCE(SUM(CASE WHEN o.State = 'COMPLETED' THEN o.TotalAmount ELSE 0 END), 0) as rev
		      FROM Orders o
		      LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
		      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId AND o.CreatedAt >= @start`
		dailyParams := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID, "start": startDate}
		dailySQL, dailyParams = auth.AppendRegionFilter(ctx, dailySQL, dailyParams, "rt")
		dailySQL += " GROUP BY day ORDER BY day"
		dailyStmt := spanner.Statement{SQL: dailySQL, Params: dailyParams}
		dailyIter := spannerx.StaleQuery(ctx, spannerClient, dailyStmt)
		defer dailyIter.Stop()
		for {
			row, err := dailyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var dm DailyMetric
			if err := row.Columns(&dm.Date, &dm.Orders, &dm.CompletedOrders, &dm.Revenue); err == nil {
				resp.DailyBreakdown = append(resp.DailyBreakdown, dm)
			}
		}
		if resp.DailyBreakdown == nil {
			resp.DailyBreakdown = []DailyMetric{}
		}

		// Fleet utilization
		resp.FleetUtil.TotalDrivers = countDrivers(ctx, spannerClient, ops.SupplierID, ops.WarehouseID, false)
		resp.FleetUtil.ActiveDrivers = countDrivers(ctx, spannerClient, ops.SupplierID, ops.WarehouseID, true)
		if resp.FleetUtil.TotalDrivers > 0 {
			resp.FleetUtil.UtilizationPct = float64(resp.FleetUtil.ActiveDrivers) / float64(resp.FleetUtil.TotalDrivers) * 100
		}

		resp.ImportFreshness = loadImportFreshness(ctx, spannerClient, ops.SupplierID, ops.WarehouseID, startDate)
		resp.ImportAnomalyQ = loadImportAnomalyQueue(ctx, spannerClient, ops.SupplierID, ops.WarehouseID, startDate)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func countDrivers(ctx context.Context, client *spanner.Client, sid, whId string, activeOnly bool) int64 {
	sql := `SELECT COUNT(*) FROM Drivers WHERE SupplierId = @sid AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId)) AND IsActive = true`
	if activeOnly {
		sql += " AND TruckStatus IN ('IN_TRANSIT','LOADING','READY')"
	}
	iter := spannerx.StaleQuery(ctx, client, spanner.Statement{SQL: sql, Params: map[string]interface{}{"sid": sid, "whId": whId}})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0
	}
	var c int64
	row.Columns(&c)
	return c
}

func loadImportFreshness(ctx context.Context, client *spanner.Client, supplierID string, warehouseID string, startDate time.Time) ImportFreshness {
	freshness := ImportFreshness{}
	factStartDate := civil.DateOf(startDate)

	summaryStmt := spanner.Statement{
		SQL: `SELECT
				COALESCE(SUM(applied_rows), 0),
				COUNT(DISTINCT sku_id),
				COALESCE(SUM(quantity_delta), 0)
			  FROM SupplierImportAnalyticsFacts
			  WHERE supplier_id = @sid
			    AND warehouse_id = @whId
			    AND fact_date >= @factStart`,
		Params: map[string]interface{}{
			"sid":       supplierID,
			"whId":      warehouseID,
			"factStart": factStartDate,
		},
	}

	summaryIter := spannerx.StaleQuery(ctx, client, summaryStmt)
	defer summaryIter.Stop()
	if row, err := summaryIter.Next(); err == nil {
		_ = row.Columns(&freshness.AppliedRows30D, &freshness.AppliedSKUs30D, &freshness.QuantityDelta30D)
	}

	latestStmt := spanner.Statement{
		SQL: `SELECT last_session_id, last_applied_at
		      FROM SupplierImportAnalyticsFacts
		      WHERE supplier_id = @sid
		        AND warehouse_id = @whId
		      ORDER BY last_applied_at DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"sid":  supplierID,
			"whId": warehouseID,
		},
	}

	latestIter := spannerx.StaleQuery(ctx, client, latestStmt)
	defer latestIter.Stop()
	if row, err := latestIter.Next(); err == nil {
		var lastSessionID spanner.NullString
		var lastAppliedAt spanner.NullTime
		if parseErr := row.Columns(&lastSessionID, &lastAppliedAt); parseErr == nil {
			if lastSessionID.Valid {
				freshness.LastSessionID = lastSessionID.StringVal
			}
			if lastAppliedAt.Valid {
				freshness.LastAppliedAt = lastAppliedAt.Time.UTC().Format(time.RFC3339)
			}
		}
	}

	return freshness
}

func loadImportAnomalyQueue(ctx context.Context, client *spanner.Client, supplierID string, warehouseID string, startDate time.Time) ImportAnomalyQ {
	queue := ImportAnomalyQ{}
	affectedSessions := make(map[string]struct{})

	stmt := spanner.Statement{
		SQL: `SELECT session_id, row_index, raw_data, cleaned_data, validation_errors, updated_at
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @sid
		        AND created_at >= @start
		        AND validation_errors IS NOT NULL
		        AND ARRAY_LENGTH(validation_errors) > 0
		      ORDER BY updated_at DESC
		      LIMIT 5000`,
		Params: map[string]interface{}{
			"sid":   supplierID,
			"start": startDate,
		},
	}

	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}

		var sessionID string
		var rowIndex int64
		var rawDataJSON spanner.NullJSON
		var cleanedDataJSON spanner.NullJSON
		var validationErrors []string
		var updatedAt spanner.NullTime
		if err := row.Columns(&sessionID, &rowIndex, &rawDataJSON, &cleanedDataJSON, &validationErrors, &updatedAt); err != nil {
			continue
		}

		rawData := importAnomalyJSONMap(rawDataJSON)
		cleanedData := importAnomalyJSONMap(cleanedDataJSON)
		rowWarehouseID := strings.TrimSpace(importAnomalyStringValue(cleanedData, rawData, "warehouse_id", "warehouse"))
		if rowWarehouseID == "" || rowWarehouseID != warehouseID {
			continue
		}

		queue.OpenRows30D++
		affectedSessions[sessionID] = struct{}{}
		if queue.LastSessionID == "" {
			queue.LastSessionID = sessionID
			if updatedAt.Valid {
				queue.LastDetectedAt = updatedAt.Time.UTC().Format(time.RFC3339)
			}
			if len(validationErrors) > 0 {
				queue.LastDetail = validationErrors[0]
			} else {
				queue.LastDetail = "Validation error detected"
			}
		}
	}

	queue.AffectedSessions30 = int64(len(affectedSessions))
	return queue
}

func importAnomalyJSONMap(value spanner.NullJSON) map[string]any {
	if !value.Valid {
		return nil
	}
	mapped, ok := value.Value.(map[string]any)
	if ok {
		return mapped
	}
	if value.Value == nil {
		return nil
	}
	encoded, err := json.Marshal(value.Value)
	if err != nil {
		return nil
	}
	decoded := make(map[string]any)
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil
	}
	return decoded
}

func importAnomalyStringValue(primary map[string]any, fallback map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := importAnomalyAnyString(primary, key); value != "" {
			return value
		}
		if value := importAnomalyAnyString(fallback, key); value != "" {
			return value
		}
	}
	return ""
}

func importAnomalyAnyString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return value
	default:
		return ""
	}
}

func splitStates(s string) []string {
	return []string{s}
}
