package warehouse

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/dispatch"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/spannerx"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Dispatch ─────────────────────────────────────────────────────────────────
// Warehouse-scoped dispatch trigger — delegates to the existing auto-dispatch
// engine but restricts visibility to the warehouse's own fleet and orders.

type DispatchableOrder struct {
	OrderID       string  `json:"order_id"`
	RetailerName  string  `json:"retailer_name"`
	TotalAmount   int64   `json:"total_amount"`
	TotalVolumeVU float64 `json:"total_volume_vu,omitempty"`
	RetailerLat   float64 `json:"retailer_lat"`
	RetailerLng   float64 `json:"retailer_lng"`
	ItemCount     int64   `json:"item_count"`
}

type DispatchableDriver struct {
	DriverID     string  `json:"driver_id"`
	Name         string  `json:"name"`
	TruckStatus  string  `json:"truck_status"`
	VehicleID    string  `json:"vehicle_id,omitempty"`
	VehicleClass string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU  float64 `json:"max_volume_vu,omitempty"`
}

// HandleOpsDispatchPreview — GET for /v1/warehouse/ops/dispatch/preview
// Returns orders awaiting dispatch and available drivers. When the optimiser
// client is armed, it also fires plan.OptimizeAndValidate in shadow mode
// against the same hydrated input — the preview response is unchanged
// (UI Freeze) but the shadow path emits structured logs + counter
// increments, mirroring the supplier-side wire.
func HandleOpsDispatchPreview(spannerClient *spanner.Client, optimizer *optimizerclient.Client, counters *plan.SourceCounters) http.HandlerFunc {
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

		// Undispatched orders
		orderStmt := spanner.Statement{
			SQL: `SELECT o.OrderId, COALESCE(rt.StoreName, ''), COALESCE(o.TotalAmount, 0),
			             COALESCE(rt.Latitude, 0), COALESCE(rt.Longitude, 0),
			             (SELECT COUNT(*) FROM OrderLineItems li WHERE li.OrderId = o.OrderId)
			      FROM Orders o
			      LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
			      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			        AND o.State = 'PENDING' AND o.DriverId IS NULL
			      ORDER BY o.CreatedAt ASC
			      LIMIT 100`,
			Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
		}
		oIter := spannerx.StaleQuery(ctx, spannerClient, orderStmt)
		defer oIter.Stop()

		var orders []DispatchableOrder
		for {
			row, err := oIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH DISPATCH] orders query error: %v", err)
				break
			}
			var o DispatchableOrder
			if err := row.Columns(&o.OrderID, &o.RetailerName, &o.TotalAmount,
				&o.RetailerLat, &o.RetailerLng, &o.ItemCount); err == nil {
				orders = append(orders, o)
			}
		}
		if orders == nil {
			orders = []DispatchableOrder{}
		}

		// Available drivers
		driverStmt := spanner.Statement{
			SQL: `SELECT d.DriverId, d.Name, COALESCE(d.TruckStatus, 'IDLE'),
			             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, ''),
			             COALESCE(v.MaxVolumeVU, 0)
			      FROM Drivers d
			      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      WHERE d.SupplierId = @sid AND (d.WarehouseId = @whId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @whId))
			        AND d.IsActive = true AND d.IsOffline = false
			        AND d.TruckStatus IN ('IDLE','AVAILABLE','RETURNING')
			      ORDER BY d.Name`,
			Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
		}
		dIter := spannerx.StaleQuery(ctx, spannerClient, driverStmt)
		defer dIter.Stop()

		var drivers []DispatchableDriver
		for {
			row, err := dIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH DISPATCH] drivers query error: %v", err)
				break
			}
			var d DispatchableDriver
			if err := row.Columns(&d.DriverID, &d.Name, &d.TruckStatus,
				&d.VehicleID, &d.VehicleClass, &d.MaxVolumeVU); err == nil {
				drivers = append(drivers, d)
			}
		}
		if drivers == nil {
			drivers = []DispatchableDriver{}
		}

		// ── Phase 2 Optimizer (SHADOW MODE) ─────────────────────────────
		// Same shadow protocol as supplier.HandleAutoDispatch: fire-and-
		// forget goroutine, structured slog event, atomic counter
		// increment. The HTTP response shape is unchanged (UI Freeze).
		if optimizer != nil && len(orders) > 0 && len(drivers) > 0 {
			shadowOrders := make([]dispatch.DispatchableOrder, len(orders))
			for i, o := range orders {
				shadowOrders[i] = dispatch.DispatchableOrder{
					OrderID:      o.OrderID,
					RetailerName: o.RetailerName,
					Amount:       o.TotalAmount,
					Lat:          o.RetailerLat,
					Lng:          o.RetailerLng,
					VolumeVU:     o.TotalVolumeVU,
				}
			}
			shadowFleet := make([]dispatch.AvailableDriver, len(drivers))
			for i, d := range drivers {
				shadowFleet[i] = dispatch.AvailableDriver{
					DriverID:     d.DriverID,
					DriverName:   d.Name,
					VehicleID:    d.VehicleID,
					VehicleClass: d.VehicleClass,
					MaxVolumeVU:  d.MaxVolumeVU,
				}
			}
			traceID := telemetry.TraceIDFromContext(r.Context())
			if traceID == "" {
				traceID = telemetry.GenerateTraceID()
			}
			supplierID := ops.SupplierID
			warehouseID := ops.WarehouseID
			go func(orderCount, driverCount int) {
				shadowCtx, shadowCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shadowCancel()
				t0 := time.Now()
				result, source, err := plan.OptimizeAndValidate(shadowCtx, optimizer, plan.Job{
					TraceID:    traceID,
					SupplierID: supplierID,
					HomeNodeID: warehouseID,
					Orders:     shadowOrders,
					Fleet:      shadowFleet,
				})
				elapsed := time.Since(t0)
				if err != nil {
					counters.RecordError()
					slog.Warn("dispatch.optimize.shadow",
						"surface", "warehouse_preview",
						"supplier_id", supplierID,
						"warehouse_id", warehouseID,
						"trace_id", traceID,
						"orders", orderCount,
						"drivers", driverCount,
						"source", source,
						"elapsed_ms", elapsed.Milliseconds(),
						"err", err.Error(),
					)
					return
				}
				counters.Record(source)
				slog.Info("dispatch.optimize.shadow",
					"surface", "warehouse_preview",
					"supplier_id", supplierID,
					"warehouse_id", warehouseID,
					"trace_id", traceID,
					"orders", orderCount,
					"drivers", driverCount,
					"source", source,
					"elapsed_ms", elapsed.Milliseconds(),
					"routes", len(result.Routes),
					"orphans", len(result.Orphans),
				)
			}(len(orders), len(drivers))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"orders":            orders,
			"drivers":           drivers,
			"pending_count":     len(orders),
			"available_drivers": len(drivers),
		})
	}
}
