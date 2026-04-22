package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"

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

		baseParams := map[string]interface{}{
			"sid":   ops.SupplierID,
			"whId":  ops.WarehouseID,
			"start": startDate,
		}

		resp := AnalyticsResponse{
			WarehouseID: ops.WarehouseID,
			Period:      periodLabel,
		}

		ctx := r.Context()

		// Total + completed + cancelled orders
		countQuery := func(states string) int64 {
			sql := `SELECT COUNT(*) FROM Orders WHERE SupplierId = @sid AND WarehouseId = @whId AND CreatedAt >= @start`
			if states != "" {
				sql += " AND State IN UNNEST(@states)"
			}
			p := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID, "start": startDate}
			if states != "" {
				p["states"] = splitStates(states)
			}
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
		revStmt := spanner.Statement{
			SQL: `SELECT COALESCE(SUM(TotalAmount), 0) FROM Orders
			      WHERE SupplierId = @sid AND WarehouseId = @whId
			      AND State = 'COMPLETED' AND CreatedAt >= @start`,
			Params: baseParams,
		}
		revIter := spannerx.StaleQuery(ctx, spannerClient, revStmt)
		defer revIter.Stop()
		if row, err := revIter.Next(); err == nil {
			row.Columns(&resp.TotalRevenue)
		}

		if resp.CompletedOrders > 0 {
			resp.AvgOrderValue = float64(resp.TotalRevenue) / float64(resp.CompletedOrders)
		}

		// Top products
		topStmt := spanner.Statement{
			SQL: `SELECT li.SkuId, COALESCE(sp.Name, ''), SUM(li.Quantity), SUM(li.Quantity * li.UnitPrice)
			      FROM OrderLineItems li
			      JOIN Orders o ON li.OrderId = o.OrderId
			      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
			      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			        AND o.State = 'COMPLETED' AND o.CreatedAt >= @start
			      GROUP BY li.SkuId, sp.Name
			      ORDER BY SUM(li.Quantity) DESC
			      LIMIT 10`,
			Params: baseParams,
		}
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
		dailyStmt := spanner.Statement{
			SQL: `SELECT CAST(DATE(o.CreatedAt) AS STRING) as day,
			             COUNT(*) as total,
			             COUNTIF(o.State = 'COMPLETED') as completed,
			             COALESCE(SUM(CASE WHEN o.State = 'COMPLETED' THEN o.TotalAmount ELSE 0 END), 0) as rev
			      FROM Orders o
			      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId AND o.CreatedAt >= @start
			      GROUP BY day ORDER BY day`,
			Params: baseParams,
		}
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

func splitStates(s string) []string {
	return []string{s}
}
