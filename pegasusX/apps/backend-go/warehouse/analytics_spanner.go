package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

const analyticsTopProductOrderLimit = 500

type analyticsTopProductAgg struct {
	ProductName string
	TotalQty    int64
	Revenue     int64
}

func parseAnalyticsPeriod(period string) (days int, label string) {
	switch strings.TrimSpace(period) {
	case "30d":
		return 30, "30d"
	default:
		return 7, "7d"
	}
}

func (s *Service) loadOpsAnalytics(ctx context.Context, warehouseID, period, supplierID string) (map[string]any, error) {
	if s == nil || s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return nil, fmt.Errorf("warehouse_id required")
	}
	days, periodLabel := parseAnalyticsPeriod(period)
	startAt := s.now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	readCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	txn := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15 * time.Second))
	defer txn.Close()

	params := map[string]any{
		"warehouseId": warehouseID,
		"startAt":     startAt,
	}
	if strings.TrimSpace(supplierID) == "" {
		supplierID = s.analyticsSupplierID(ctx)
	}
	if strings.TrimSpace(supplierID) != "" {
		params["supplierId"] = supplierID
	}

	var totalOrders, completedOrders, cancelledOrders, totalRevenue int64
	countSQL := `SELECT
		COUNT(*) AS total,
		COUNTIF(Status = 'COMPLETED') AS completed,
		COUNTIF(Status = 'CANCELLED') AS cancelled,
		COALESCE(SUM(CASE WHEN Status = 'COMPLETED' THEN TotalMinor ELSE 0 END), 0) AS revenue
	FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated}
	WHERE WarehouseId = @warehouseId AND CreatedAt >= @startAt`
	if _, ok := params["supplierId"]; ok {
		countSQL += ` AND SupplierId = @supplierId`
	}
	if err := txn.Query(readCtx, spanner.Statement{SQL: countSQL, Params: params}).Do(func(row *spanner.Row) error {
		return row.Columns(&totalOrders, &completedOrders, &cancelledOrders, &totalRevenue)
	}); err != nil {
		return nil, fmt.Errorf("analytics order counts: %w", err)
	}

	var avgOrderValue float64
	if completedOrders > 0 {
		avgOrderValue = float64(totalRevenue) / float64(completedOrders)
	}

	dailyBreakdown, err := s.loadAnalyticsDailyBreakdown(readCtx, txn, params)
	if err != nil {
		return nil, err
	}
	topProducts, err := s.loadAnalyticsTopProducts(readCtx, txn, params)
	if err != nil {
		return nil, err
	}
	fleetUtil, err := s.loadAnalyticsFleetUtilization(readCtx, txn, warehouseID, params, startAt)
	if err != nil {
		return nil, err
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		supplierID = s.analyticsSupplierID(ctx)
	}
	importReadTxn := s.spannerClient.Single()
	defer importReadTxn.Close()
	importFreshness, err := s.loadAnalyticsImportFreshness(readCtx, importReadTxn, warehouseID, startAt, supplierID)
	if err != nil {
		return nil, err
	}
	importAnomaly, err := s.loadAnalyticsImportAnomalyQueue(readCtx, importReadTxn, warehouseID, startAt, supplierID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"warehouse_id":         warehouseID,
		"period":               periodLabel,
		"total_orders":         totalOrders,
		"total_revenue":        totalRevenue,
		"completed_orders":     completedOrders,
		"cancelled_orders":     cancelledOrders,
		"avg_order_value":      avgOrderValue,
		"top_products":         topProducts,
		"daily_breakdown":      dailyBreakdown,
		"daily":                dailyBreakdown,
		"fleet_utilization":    fleetUtil,
		"fleet_utilization_pct": fleetUtil["utilization_pct"],
		"import_freshness":     importFreshness,
		"import_anomaly_queue": importAnomaly,
	}, nil
}

func (s *Service) loadAnalyticsDailyBreakdown(ctx context.Context, txn *spanner.ReadOnlyTransaction, params map[string]any) ([]map[string]any, error) {
	sql := `SELECT CAST(DATE(CreatedAt) AS STRING) AS day,
		COUNT(*) AS orders,
		COUNTIF(Status = 'COMPLETED') AS completed,
		COALESCE(SUM(CASE WHEN Status = 'COMPLETED' THEN TotalMinor ELSE 0 END), 0) AS revenue
	FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated}
	WHERE WarehouseId = @warehouseId AND CreatedAt >= @startAt`
	if _, ok := params["supplierId"]; ok {
		sql += ` AND SupplierId = @supplierId`
	}
	sql += ` GROUP BY day ORDER BY day`

	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	rows := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("analytics daily breakdown: %w", err)
		}
		var day string
		var orders, completed, revenue int64
		if err := row.Columns(&day, &orders, &completed, &revenue); err != nil {
			return nil, fmt.Errorf("scan daily breakdown: %w", err)
		}
		rows = append(rows, map[string]any{
			"date":      day,
			"orders":    orders,
			"completed": completed,
			"revenue":   revenue,
		})
	}
	return rows, nil
}

func (s *Service) loadAnalyticsTopProducts(ctx context.Context, txn *spanner.ReadOnlyTransaction, params map[string]any) ([]map[string]any, error) {
	sql := `SELECT LineItemsJson
	FROM Orders@{FORCE_INDEX=Idx_Orders_ByStatusWarehouse}
	WHERE WarehouseId = @warehouseId AND Status = 'COMPLETED' AND CreatedAt >= @startAt`
	if _, ok := params["supplierId"]; ok {
		sql += ` AND SupplierId = @supplierId`
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`
	queryParams := make(map[string]any, len(params)+1)
	for k, v := range params {
		queryParams[k] = v
	}
	queryParams["lim"] = int64(analyticsTopProductOrderLimit)

	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: queryParams})
	defer iter.Stop()

	agg := make(map[string]*analyticsTopProductAgg)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("analytics top products: %w", err)
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			return nil, fmt.Errorf("scan line items json: %w", err)
		}
		mergeLineItemsIntoTopProducts(agg, raw)
	}

	type ranked struct {
		key string
		row analyticsTopProductAgg
	}
	rankedRows := make([]ranked, 0, len(agg))
	for key, row := range agg {
		if row == nil {
			continue
		}
		rankedRows = append(rankedRows, ranked{key: key, row: *row})
	}
	sort.Slice(rankedRows, func(i, j int) bool {
		if rankedRows[i].row.TotalQty == rankedRows[j].row.TotalQty {
			return rankedRows[i].row.Revenue > rankedRows[j].row.Revenue
		}
		return rankedRows[i].row.TotalQty > rankedRows[j].row.TotalQty
	})
	if len(rankedRows) > 10 {
		rankedRows = rankedRows[:10]
	}

	out := make([]map[string]any, 0, len(rankedRows))
	for _, item := range rankedRows {
		out = append(out, map[string]any{
			"product_name": item.row.ProductName,
			"total_qty":    item.row.TotalQty,
			"total_sold":   item.row.TotalQty,
			"revenue":      item.row.Revenue,
		})
	}
	return out, nil
}

func mergeLineItemsIntoTopProducts(agg map[string]*analyticsTopProductAgg, raw []byte) {
	if len(raw) == 0 {
		return
	}
	var items []order.LineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return
	}
	for _, item := range items {
		key := strings.TrimSpace(item.SKU)
		if key == "" {
			key = strings.TrimSpace(item.Name)
		}
		if key == "" {
			continue
		}
		entry := agg[key]
		if entry == nil {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				name = key
			}
			entry = &analyticsTopProductAgg{ProductName: name}
			agg[key] = entry
		}
		qty := item.Quantity
		if qty < 0 {
			qty = 0
		}
		entry.TotalQty += qty
		entry.Revenue += qty * item.UnitPrice
	}
}

func (s *Service) loadAnalyticsFleetUtilization(ctx context.Context, txn *spanner.ReadOnlyTransaction, warehouseID string, params map[string]any, startAt time.Time) (map[string]any, error) {
	driverParams := map[string]any{
		"warehouseId": warehouseID,
		"homeType":    "WAREHOUSE",
	}
	driverSQL := `SELECT COUNT(*)
	FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode}
	WHERE HomeNodeType = @homeType AND HomeNodeId = @warehouseId AND IsActive = TRUE`
	if strings.TrimSpace(s.supplierID) != "" {
		driverSQL += ` AND SupplierId = @supplierId`
		driverParams["supplierId"] = s.supplierID
	}
	var totalDrivers int64
	if err := txn.Query(ctx, spanner.Statement{SQL: driverSQL, Params: driverParams}).Do(func(row *spanner.Row) error {
		return row.Columns(&totalDrivers)
	}); err != nil {
		return nil, fmt.Errorf("analytics fleet total: %w", err)
	}

	activeSQL := `SELECT COUNT(DISTINCT DriverId)
	FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated}
	WHERE WarehouseId = @warehouseId
	  AND CreatedAt >= @startAt
	  AND DriverId IS NOT NULL
	  AND Status IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'COMPLETED')`
	if _, ok := params["supplierId"]; ok {
		activeSQL += ` AND SupplierId = @supplierId`
	}
	var activeDrivers int64
	if err := txn.Query(ctx, spanner.Statement{SQL: activeSQL, Params: params}).Do(func(row *spanner.Row) error {
		return row.Columns(&activeDrivers)
	}); err != nil {
		return nil, fmt.Errorf("analytics fleet active: %w", err)
	}

	utilPct := 0.0
	if totalDrivers > 0 {
		utilPct = float64(activeDrivers) / float64(totalDrivers) * 100
	}
	return map[string]any{
		"total_drivers":    totalDrivers,
		"active_drivers":   activeDrivers,
		"utilization_pct":  utilPct,
		"avg_stops_per_day": 0.0,
	}, nil
}

func (s *Service) loadAnalyticsImportFreshness(ctx context.Context, txn *spanner.ReadOnlyTransaction, warehouseID string, startAt time.Time, supplierID string) (map[string]any, error) {
	out := map[string]any{
		"applied_rows_30d":   int64(0),
		"applied_skus_30d":   int64(0),
		"quantity_delta_30d": int64(0),
		"last_session_id":    "",
		"last_applied_at":    "",
	}
	params := map[string]any{
		"warehouseId": warehouseID,
		"startAt":     startAt,
	}
	sql := `SELECT
		COUNT(*) AS row_count,
		COUNT(DISTINCT ProductId) AS sku_count,
		COALESCE(SUM(QuantityOnHand), 0) AS quantity_sum,
		MAX(UpdatedAt) AS last_updated
	FROM SupplierInventoryV2
	WHERE WarehouseId = @warehouseId AND UpdatedAt >= @startAt`
	if strings.TrimSpace(supplierID) != "" {
		sql += ` AND SupplierId = @supplierId`
		params["supplierId"] = supplierID
	}
	var rowCount, skuCount, quantitySum int64
	var lastUpdated spanner.NullTime
	if err := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params}).Do(func(row *spanner.Row) error {
		return row.Columns(&rowCount, &skuCount, &quantitySum, &lastUpdated)
	}); err != nil {
		return out, nil
	}
	out["applied_rows_30d"] = rowCount
	out["applied_skus_30d"] = skuCount
	out["quantity_delta_30d"] = quantitySum
	if lastUpdated.Valid {
		out["last_applied_at"] = lastUpdated.Time.UTC().Format(time.RFC3339Nano)
	}
	out["freshness_source"] = "inventory_v2_proxy"
	if strings.TrimSpace(supplierID) != "" {
		sessionStmt := spanner.Statement{
			SQL: `SELECT session_id, updated_at
			      FROM SupplierImportSessions
			      WHERE supplier_id = @supplierId
			        AND status IN ('applied', 'APPLIED')
			      ORDER BY updated_at DESC
			      LIMIT 1`,
			Params: map[string]any{"supplierId": supplierID},
		}
		var sessionID string
		var sessionUpdated spanner.NullTime
		if err := txn.Query(ctx, sessionStmt).Do(func(row *spanner.Row) error {
			return row.Columns(&sessionID, &sessionUpdated)
		}); err == nil && strings.TrimSpace(sessionID) != "" {
			out["last_session_id"] = sessionID
			if sessionUpdated.Valid {
				out["last_import_session_at"] = sessionUpdated.Time.UTC().Format(time.RFC3339Nano)
			}
			out["freshness_source"] = "inventory_v2_proxy_with_session_anchor"
		}
	}
	return out, nil
}

func (s *Service) loadAnalyticsImportAnomalyQueue(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	warehouseID string,
	startAt time.Time,
	supplierID string,
) (map[string]any, error) {
	out := map[string]any{
		"open_rows_30d":         int64(0),
		"affected_sessions_30d": int64(0),
		"last_session_id":       "",
		"last_detected_at":      "",
		"last_detail":           "",
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return out, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT session_id, row_index, raw_data, cleaned_data, validation_errors, updated_at
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId
		        AND created_at >= @startAt
		        AND validation_errors IS NOT NULL
		        AND ARRAY_LENGTH(validation_errors) > 0
		      ORDER BY updated_at DESC
		      LIMIT 5000`,
		Params: map[string]any{
			"supplierId": supplierID,
			"startAt":    startAt,
		},
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	affectedSessions := make(map[string]struct{})
	var openRows int64
	var lastSessionID, lastDetectedAt, lastDetail string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out, nil
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
		if rowWarehouseID != "" && rowWarehouseID != warehouseID {
			continue
		}

		openRows++
		affectedSessions[sessionID] = struct{}{}
		if lastSessionID == "" {
			lastSessionID = sessionID
			if updatedAt.Valid {
				lastDetectedAt = updatedAt.Time.UTC().Format(time.RFC3339Nano)
			}
			if len(validationErrors) > 0 {
				lastDetail = validationErrors[0]
			} else {
				lastDetail = "Validation error detected"
			}
		}
	}

	out["open_rows_30d"] = openRows
	out["affected_sessions_30d"] = int64(len(affectedSessions))
	out["last_session_id"] = lastSessionID
	out["last_detected_at"] = lastDetectedAt
	out["last_detail"] = lastDetail
	return out, nil
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
	if value, ok := raw.(string); ok {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}
