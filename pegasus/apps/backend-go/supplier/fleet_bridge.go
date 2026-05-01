package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// FLEET VOLUMETRICS — The Bridge Dashboard
// Returns fleet capacity vs. warehouse backlog per warehouse.
// ═══════════════════════════════════════════════════════════════════════════════

// FleetVolumetricsSummary is the per-warehouse fleet/backlog snapshot.
type FleetVolumetricsSummary struct {
	WarehouseID   string  `json:"warehouse_id"`
	WarehouseName string  `json:"warehouse_name"`
	FleetCapacity float64 `json:"fleet_capacity_vu"` // Total VU of all dispatchable trucks
	BacklogVU     float64 `json:"backlog_vu"`        // Total VU of unassigned active orders
	Utilization   float64 `json:"utilization_pct"`   // (BacklogVU / FleetCapacity) * 100
	TrucksReady   int     `json:"trucks_ready"`      // Dispatchable trucks
	TrucksTotal   int     `json:"trucks_total"`      // All active trucks
	OrdersPending int     `json:"orders_pending"`    // Orders awaiting dispatch
}

// HandleFleetVolumetrics returns fleet capacity vs. backlog for each warehouse.
// GET /v1/supplier/fleet-volumetrics
func HandleFleetVolumetrics(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// 1) Fleet capacity per warehouse
		fleetSQL := `SELECT COALESCE(v.WarehouseId, 'UNASSIGNED') AS wid,
		                    COUNT(*) AS total_trucks,
		                    SUM(IF(COALESCE(d.TruckStatus, 'AVAILABLE') IN ('AVAILABLE', 'RETURNING')
		                           AND (d.IsOffline IS NULL OR d.IsOffline = false), 1, 0)) AS ready_trucks,
		                    SUM(IF(COALESCE(d.TruckStatus, 'AVAILABLE') IN ('AVAILABLE', 'RETURNING')
		                           AND (d.IsOffline IS NULL OR d.IsOffline = false), v.MaxVolumeVU * 0.95, 0)) AS fleet_cap
		             FROM Vehicles v
		             JOIN Drivers d ON d.VehicleId = v.VehicleId
		             WHERE v.SupplierId = @sid AND v.IsActive = true AND d.IsActive = true
		             GROUP BY wid`
		fleetParams := map[string]interface{}{"sid": supplierID}

		// Apply warehouse scope if present
		warehouseScope := auth.EffectiveWarehouseID(r.Context())
		if warehouseScope != "" {
			fleetSQL = `SELECT COALESCE(v.WarehouseId, 'UNASSIGNED') AS wid,
			                   COUNT(*) AS total_trucks,
			                   SUM(IF(COALESCE(d.TruckStatus, 'AVAILABLE') IN ('AVAILABLE', 'RETURNING')
			                          AND (d.IsOffline IS NULL OR d.IsOffline = false), 1, 0)) AS ready_trucks,
			                   SUM(IF(COALESCE(d.TruckStatus, 'AVAILABLE') IN ('AVAILABLE', 'RETURNING')
			                          AND (d.IsOffline IS NULL OR d.IsOffline = false), v.MaxVolumeVU * 0.95, 0)) AS fleet_cap
			            FROM Vehicles v
			            JOIN Drivers d ON d.VehicleId = v.VehicleId
			            WHERE v.SupplierId = @sid AND v.IsActive = true AND d.IsActive = true
			                  AND v.WarehouseId = @warehouseId
			            GROUP BY wid`
			fleetParams["warehouseId"] = warehouseScope
		}

		fleetMap := make(map[string]*FleetVolumetricsSummary)
		iter := client.Single().Query(ctx, spanner.Statement{SQL: fleetSQL, Params: fleetParams})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[FLEET-VOL] fleet query: %v", err)
				http.Error(w, "Fleet query failed", http.StatusInternalServerError)
				return
			}
			var wid string
			var totalTrucks, readyTrucks int64
			var fleetCap float64
			if err := row.Columns(&wid, &totalTrucks, &readyTrucks, &fleetCap); err != nil {
				log.Printf("[FLEET-VOL] fleet row: %v", err)
				continue
			}
			fleetMap[wid] = &FleetVolumetricsSummary{
				WarehouseID:   wid,
				FleetCapacity: fleetCap,
				TrucksReady:   int(readyTrucks),
				TrucksTotal:   int(totalTrucks),
			}
		}

		// 2) Backlog per warehouse — orders not yet dispatched
		backlogSQL := `SELECT COALESCE(o.WarehouseId, 'UNASSIGNED') AS wid,
		                      COUNT(*) AS pending_count,
		                      IFNULL(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, 1.0)), 0) AS backlog_vu
		               FROM Orders o
		               JOIN OrderLineItems li ON o.OrderId = li.OrderId
		               LEFT JOIN SupplierProducts sp ON li.ProductId = sp.ProductId AND o.SupplierId = sp.SupplierId
		               WHERE o.SupplierId = @sid
		                 AND o.State IN ('PENDING', 'PENDING_REVIEW', 'READY_FOR_DISPATCH')
		                 AND (o.RouteId IS NULL OR o.RouteId = '')
		               GROUP BY wid`
		backlogParams := map[string]interface{}{"sid": supplierID}
		if warehouseScope != "" {
			backlogSQL = `SELECT COALESCE(o.WarehouseId, 'UNASSIGNED') AS wid,
			                     COUNT(*) AS pending_count,
			                     IFNULL(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, 1.0)), 0) AS backlog_vu
			              FROM Orders o
			              JOIN OrderLineItems li ON o.OrderId = li.OrderId
			              LEFT JOIN SupplierProducts sp ON li.ProductId = sp.ProductId AND o.SupplierId = sp.SupplierId
			              WHERE o.SupplierId = @sid
			                AND o.State IN ('PENDING', 'PENDING_REVIEW', 'READY_FOR_DISPATCH')
			                AND (o.RouteId IS NULL OR o.RouteId = '')
			                AND o.WarehouseId = @warehouseId
			              GROUP BY wid`
			backlogParams["warehouseId"] = warehouseScope
		}

		iter2 := client.Single().Query(ctx, spanner.Statement{SQL: backlogSQL, Params: backlogParams})
		defer iter2.Stop()
		for {
			row, err := iter2.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[FLEET-VOL] backlog query: %v", err)
				http.Error(w, "Backlog query failed", http.StatusInternalServerError)
				return
			}
			var wid string
			var pendingCount int64
			var backlogVU float64
			if err := row.Columns(&wid, &pendingCount, &backlogVU); err != nil {
				log.Printf("[FLEET-VOL] backlog row: %v", err)
				continue
			}
			if _, ok := fleetMap[wid]; !ok {
				fleetMap[wid] = &FleetVolumetricsSummary{WarehouseID: wid}
			}
			fleetMap[wid].BacklogVU = backlogVU
			fleetMap[wid].OrdersPending = int(pendingCount)
		}

		// 3) Resolve warehouse names
		for wid, summary := range fleetMap {
			if wid == "UNASSIGNED" {
				summary.WarehouseName = "Unassigned"
				continue
			}
			row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{wid}, []string{"Name"})
			if err == nil {
				var name string
				if row.Columns(&name) == nil {
					summary.WarehouseName = name
				}
			}
		}

		// 4) Compute utilization
		summaries := make([]FleetVolumetricsSummary, 0, len(fleetMap))
		for _, s := range fleetMap {
			if s.FleetCapacity > 0 {
				s.Utilization = (s.BacklogVU / s.FleetCapacity) * 100
			}
			summaries = append(summaries, *s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"summaries": summaries,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// DISPATCH QUEUE — Move orders from READY_FOR_DISPATCH to a DriverManifest
// This is the explicit "dispatch" action that an admin triggers after reviewing
// the auto-dispatch preview or manually selecting orders.
// ═══════════════════════════════════════════════════════════════════════════════

type DispatchQueueRequest struct {
	WarehouseID string   `json:"warehouse_id"`
	OrderIDs    []string `json:"order_ids"` // Specific orders, or empty for all READY_FOR_DISPATCH
	DriverID    string   `json:"driver_id"` // Optional: pin to a specific driver
}

type DispatchQueueResult struct {
	ManifestsCreated int             `json:"manifests_created"`
	OrdersDispatched int             `json:"orders_dispatched"`
	Orphans          []OrphanOrder   `json:"orphans"`
	Manifests        []TruckManifest `json:"manifests"`
}

// HandleDispatchQueue takes READY_FOR_DISPATCH orders and runs them through
// the auto-dispatch algorithm, persisting the resulting manifests.
// POST /v1/supplier/dispatch-queue
func HandleDispatchQueue(client *spanner.Client, readRouter proximity.ReadRouter, manifestSvc *ManifestService, optimizer *optimizerclient.Client, counters *plan.SourceCounters) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		var req DispatchQueueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Fetch READY_FOR_DISPATCH orders for this warehouse
		orderIDs := req.OrderIDs
		if len(orderIDs) == 0 {
			// No specific IDs — fetch all READY_FOR_DISPATCH for the warehouse
			fetched, err := fetchReadyOrders(ctx, client, supplierID, req.WarehouseID)
			if err != nil {
				log.Printf("[DISPATCH-Q] fetch ready orders: %v", err)
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			orderIDs = fetched
		}

		if len(orderIDs) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DispatchQueueResult{
				Manifests: []TruckManifest{},
				Orphans:   []OrphanOrder{},
			})
			return
		}

		// Delegate to the existing auto-dispatch engine with the specific order IDs
		var excludedTrucks []string
		result, err := runAutoDispatch(ctx, client, readRouter, supplierID, orderIDs, excludedTrucks, manifestSvc, optimizer, counters, false)
		if err != nil {
			log.Printf("[DISPATCH-Q] auto-dispatch: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		qResult := DispatchQueueResult{
			ManifestsCreated: len(result.Manifests),
			Manifests:        result.Manifests,
			Orphans:          result.Orphans,
		}
		for _, m := range result.Manifests {
			qResult.OrdersDispatched += len(m.Orders)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(qResult)
	}
}

// fetchReadyOrders returns OrderIDs in READY_FOR_DISPATCH state for a warehouse.
func fetchReadyOrders(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) ([]string, error) {
	sql := `SELECT OrderId FROM Orders
	        WHERE SupplierId = @sid
	          AND State = 'READY_FOR_DISPATCH'
	          AND (RouteId IS NULL OR RouteId = '')`
	params := map[string]interface{}{"sid": supplierID}
	if warehouseID != "" {
		sql += " AND WarehouseId = @warehouseId"
		params["warehouseId"] = warehouseID
	}
	sql += " ORDER BY CreatedAt ASC"

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ready orders: %w", err)
		}
		var id string
		if err := row.Columns(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// H3 ROUTE CLUSTER — Group orders by H3 cell for geographic dispatch batching
// Returns cluster analysis without executing dispatch, for the admin preview UI.
// ═══════════════════════════════════════════════════════════════════════════════

type H3Cluster struct {
	CellID        string   `json:"cell_id"` // H3 hex cell
	OrderCount    int      `json:"order_count"`
	TotalVU       float64  `json:"total_vu"`
	CenterLat     float64  `json:"center_lat"`
	CenterLng     float64  `json:"center_lng"`
	OrderIDs      []string `json:"order_ids"`
	RetailerNames []string `json:"retailer_names"`
}

type H3RoutePreview struct {
	Clusters     []H3Cluster `json:"clusters"`
	TotalOrders  int         `json:"total_orders"`
	TotalVU      float64     `json:"total_vu"`
	FleetCapVU   float64     `json:"fleet_capacity_vu"`
	TrucksNeeded int         `json:"trucks_needed_est"`
}

// HandleH3RoutePreview returns H3-clustered order groups for dispatch planning.
// GET /v1/supplier/dispatch-preview?warehouse_id=X
func HandleH3RoutePreview(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()
		warehouseID := r.URL.Query().Get("warehouse_id")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Fetch dispatchable orders
		orders, err := fetchDispatchableOrders(ctx, client, readRouter, supplierID, nil)
		if err != nil {
			log.Printf("[H3-PREVIEW] fetch orders: %v", err)
			http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
			return
		}

		// Filter by warehouse if specified
		if warehouseID != "" {
			filtered := orders[:0]
			for _, o := range orders {
				filtered = append(filtered, o)
			}
			orders = filtered
		}

		// H3 cell clustering
		type cellAcc struct {
			orders []dispatchableOrder
			sumLat float64
			sumLng float64
		}
		cells := make(map[string]*cellAcc)
		cellOrder := []string{}

		for _, o := range orders {
			cell := proximity.LookupCell(o.Lat, o.Lng)
			if _, exists := cells[cell]; !exists {
				cells[cell] = &cellAcc{}
				cellOrder = append(cellOrder, cell)
			}
			acc := cells[cell]
			acc.orders = append(acc.orders, o)
			acc.sumLat += o.Lat
			acc.sumLng += o.Lng
		}

		totalVU := 0.0
		clusters := make([]H3Cluster, 0, len(cells))
		for _, cell := range cellOrder {
			acc := cells[cell]
			clusterVU := 0.0
			orderIDs := make([]string, len(acc.orders))
			retailers := make([]string, len(acc.orders))
			for i, o := range acc.orders {
				orderIDs[i] = o.OrderID
				retailers[i] = o.RetailerName
				clusterVU += o.VolumeVU
			}
			totalVU += clusterVU
			n := float64(len(acc.orders))
			clusters = append(clusters, H3Cluster{
				CellID:        cell,
				OrderCount:    len(acc.orders),
				TotalVU:       clusterVU,
				CenterLat:     acc.sumLat / n,
				CenterLng:     acc.sumLng / n,
				OrderIDs:      orderIDs,
				RetailerNames: retailers,
			})
		}

		// Fleet capacity snapshot
		fleetReadClient := client
		if warehouseID != "" {
			if whLat, whLng, ok := fetchWarehouseOrigin(ctx, client, warehouseID); ok {
				fleetReadClient = proximity.ReadClientForRetailer(client, readRouter, whLat, whLng)
			}
		}
		fleet, _ := GetAvailableFleet(ctx, fleetReadClient, supplierID, warehouseID)
		fleetCap := 0.0
		for _, t := range fleet {
			fleetCap += t.EffectiveCapacity()
		}

		trucksNeeded := 0
		if fleetCap > 0 {
			avg := fleetCap / float64(len(fleet))
			if avg > 0 {
				trucksNeeded = int(totalVU/avg) + 1
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(H3RoutePreview{
			Clusters:     clusters,
			TotalOrders:  len(orders),
			TotalVU:      totalVU,
			FleetCapVU:   fleetCap,
			TrucksNeeded: trucksNeeded,
		})
	}
}
