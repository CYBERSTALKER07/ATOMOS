package warehouse

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Dashboard ────────────────────────────────────────────────────────────────
// GET /v1/warehouse/ops/dashboard
// Returns KPIs: active orders, drivers on route, pending dispatches,
// today's revenue, inventory alerts, and fleet status.

type DashboardResponse struct {
	WarehouseID     string           `json:"warehouse_id"`
	WarehouseName   string           `json:"warehouse_name"`
	ActiveOrders    int64            `json:"active_orders"`
	CompletedToday  int64            `json:"completed_today"`
	PendingDispatch int64            `json:"pending_dispatch"`
	DriversOnRoute  int64            `json:"drivers_on_route"`
	DriversIdle     int64            `json:"drivers_idle"`
	TotalDrivers    int64            `json:"total_drivers"`
	TotalVehicles   int64            `json:"total_vehicles"`
	TodayRevenue    int64            `json:"today_revenue"`
	LowStockCount   int64            `json:"low_stock_count"`
	TotalStaff      int64            `json:"total_staff"`
	FleetStatus     []FleetStatusRow `json:"fleet_status"`
}

type FleetStatusRow struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

func HandleDashboard(spannerClient *spanner.Client) http.HandlerFunc {
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

		ctx := r.Context()
		resp := DashboardResponse{
			WarehouseID: ops.WarehouseID,
		}

		// Warehouse name
		whRow, err := spannerx.StaleReadRow(ctx, spannerClient, "Warehouses",
			spanner.Key{ops.WarehouseID}, []string{"Name"})
		if err == nil {
			whRow.Columns(&resp.WarehouseName)
		}

		dayStart := time.Now().UTC().Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		// Active orders (non-terminal states)
		countQuery := func(sql string, params map[string]interface{}) int64 {
			iter := spannerx.StaleQuery(ctx, spannerClient, spanner.Statement{SQL: sql, Params: params})
			defer iter.Stop()
			row, err := iter.Next()
			if err != nil {
				return 0
			}
			var c int64
			row.Columns(&c)
			return c
		}

		baseParams := map[string]interface{}{
			"whId": ops.WarehouseID,
			"sid":  ops.SupplierID,
		}

		resp.ActiveOrders = countQuery(
			`SELECT COUNT(*) FROM Orders WHERE SupplierId = @sid AND WarehouseId = @whId
			 AND State IN ('PENDING','LOADED','IN_TRANSIT','ARRIVING','ARRIVED','EN_ROUTE')`,
			baseParams)

		resp.CompletedToday = countQuery(
			`SELECT COUNT(*) FROM Orders WHERE SupplierId = @sid AND WarehouseId = @whId
			 AND State = 'COMPLETED' AND UpdatedAt >= @dayStart AND UpdatedAt < @dayEnd`,
			map[string]interface{}{"whId": ops.WarehouseID, "sid": ops.SupplierID, "dayStart": dayStart, "dayEnd": dayEnd})

		resp.PendingDispatch = countQuery(
			`SELECT COUNT(*) FROM Orders WHERE SupplierId = @sid AND WarehouseId = @whId
			 AND State = 'PENDING' AND DriverId IS NULL`,
			baseParams)

		// Driver stats
		resp.TotalDrivers = countQuery(
			`SELECT COUNT(*) FROM Drivers WHERE SupplierId = @sid AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId)) AND IsActive = true`,
			baseParams)
		resp.DriversOnRoute = countQuery(
			`SELECT COUNT(*) FROM Drivers WHERE SupplierId = @sid AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId))
			 AND IsActive = true AND TruckStatus IN ('IN_TRANSIT','LOADING','READY')`,
			baseParams)
		resp.DriversIdle = resp.TotalDrivers - resp.DriversOnRoute

		// Vehicles
		resp.TotalVehicles = countQuery(
			`SELECT COUNT(*) FROM Vehicles WHERE SupplierId = @sid AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId)) AND IsActive = true`,
			baseParams)

		// Revenue today
		revStmt := spanner.Statement{
			SQL: `SELECT COALESCE(SUM(o.TotalAmount), 0) FROM Orders o
			      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			      AND o.State = 'COMPLETED' AND o.UpdatedAt >= @dayStart AND o.UpdatedAt < @dayEnd`,
			Params: map[string]interface{}{"whId": ops.WarehouseID, "sid": ops.SupplierID, "dayStart": dayStart, "dayEnd": dayEnd},
		}
		revIter := spannerx.StaleQuery(ctx, spannerClient, revStmt)
		defer revIter.Stop()
		if row, err := revIter.Next(); err == nil {
			row.Columns(&resp.TodayRevenue)
		}

		// Low stock alerts
		resp.LowStockCount = countQuery(
			`SELECT COUNT(*) FROM SupplierInventory
			 WHERE SupplierId = @sid AND WarehouseId = @whId AND Quantity <= ReorderThreshold`,
			baseParams)

		// Staff count
		resp.TotalStaff = countQuery(
			`SELECT COUNT(*) FROM WarehouseStaff WHERE SupplierId = @sid AND WarehouseId = @whId AND IsActive = true`,
			baseParams)

		// Fleet status breakdown
		fleetStmt := spanner.Statement{
			SQL: `SELECT COALESCE(TruckStatus, 'IDLE') as status, COUNT(*) as cnt
			      FROM Drivers WHERE SupplierId = @sid AND WarehouseId = @whId AND IsActive = true
			      GROUP BY TruckStatus`,
			Params: baseParams,
		}
		fleetIter := spannerx.StaleQuery(ctx, spannerClient, fleetStmt)
		defer fleetIter.Stop()
		for {
			row, err := fleetIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH DASHBOARD] fleet status query error: %v", err)
				break
			}
			var status string
			var cnt int64
			if err := row.Columns(&status, &cnt); err == nil {
				resp.FleetStatus = append(resp.FleetStatus, FleetStatusRow{Status: status, Count: cnt})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
