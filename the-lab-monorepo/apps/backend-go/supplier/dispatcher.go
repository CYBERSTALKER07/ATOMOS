package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/dispatch"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/proximity"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// AUTO-DISPATCH ENGINE — CLUSTER-FIRST, ROUTE-SECOND
// Three-phase pipeline: K-Means spatial clustering → Atomic bin-packing →
// Outlier sweep with density penalty. ZERO order splitting.
//
// Dispatch constants (TetrisBuffer, MaxDetourRadius, MaxWaypointsPerManifest,
// VUDivisor) and driver status vocabulary are re-exported from dispatch/ via
// dispatch_shim.go so supplier and dispatch share a single source of truth.
// ═══════════════════════════════════════════════════════════════════════════════

// IsDispatchable returns true if a driver with the given TruckStatus can be
// assigned new routes. Only IDLE/AVAILABLE and RETURNING drivers qualify.
// A driver flagged IsOffline is never dispatchable regardless of TruckStatus.
func IsDispatchable(status string, isOffline bool) bool {
	return dispatch.IsDispatchable(status, isOffline)
}

// ═══════════════════════════════════════════════════════════════════════════════
// MDVRP DOMAIN STRUCTS — Multi-Depot Vehicle Routing
// Warehouse, Truck, Product model the physical topology.
// ═══════════════════════════════════════════════════════════════════════════════

// Warehouse represents a physical depot from which trucks depart.
// Each warehouse has H3-indexed coverage and an independent fleet.
type Warehouse struct {
	WarehouseID      string
	SupplierID       string
	Name             string
	Address          string
	Lat              float64
	Lng              float64
	H3Indexes        []string // H3 res7 cell IDs covering this depot's delivery zone
	CoverageRadiusKm float64
	IsActive         bool
	IsOnShift        bool
}

// Truck wraps a Spanner Vehicles row with the dual-entry VU/LWH system.
// MaxVolumeVU is the authoritative capacity. If LWH dimensions are provided
// and no VehicleClass override exists, VU is computed via CalculateVU.
type Truck struct {
	VehicleID    string
	WarehouseID  string
	SupplierID   string
	VehicleClass string  // CLASS_A | CLASS_B | CLASS_C
	MaxVolumeVU  float64 // Authoritative capacity in Volumetric Units
	LengthCM     *float64
	WidthCM      *float64
	HeightCM     *float64
	IsActive     bool
}

// EffectiveCapacity returns the usable VU after applying TetrisBuffer.
func (t Truck) EffectiveCapacity() float64 {
	return t.MaxVolumeVU * TetrisBuffer
}

// Product models a catalog item with the dual-entry volumetric system.
// VolumetricUnit takes precedence; LWH dimensions are the fallback.
type Product struct {
	SkuID          string
	SupplierID     string
	Name           string
	VolumetricUnit float64  // Abstract VU (0 means "use LWH")
	LengthCM       *float64 // Physical cargo dimensions (optional)
	WidthCM        *float64
	HeightCM       *float64
}

// ── Volumetric Normalizer ───────────────────────────────────────────────────

// CalculateVU converts physical dimensions (cm) to Volumetric Units.
// Formula: (L × W × H) / 5000. This is the industry-standard dimensional
// weight divisor used by DHL, FedEx, and UPS.
func CalculateVU(lengthCM, widthCM, heightCM float64) float64 {
	return (lengthCM * widthCM * heightCM) / VUDivisor
}

// NormalizeProductVU returns the effective VU for a product.
// Priority: VolumetricUnit (if > 0) → CalculateVU(LWH) → fallback 1.0.
func NormalizeProductVU(p Product) float64 {
	if p.VolumetricUnit > 0 {
		return p.VolumetricUnit
	}
	if p.LengthCM != nil && p.WidthCM != nil && p.HeightCM != nil {
		vu := CalculateVU(*p.LengthCM, *p.WidthCM, *p.HeightCM)
		if vu > 0 {
			return vu
		}
	}
	return 1.0
}

// ═══════════════════════════════════════════════════════════════════════════════
// ASSIGNMENT RESULT — Output of the Smart Fit dispatch algorithm
//
// AssignmentResult / SplitOrder / OrderChunk are aliased from dispatch/ via
// dispatch_shim.go. The data shape is shared across supplier, warehouse, and
// factory scopes; centralising it prevents field drift.
// ═══════════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════════
// FLEET RESOLUTION — Multi-Depot Fleet Snapshot
// ═══════════════════════════════════════════════════════════════════════════════

// GetAvailableFleet fetches all trucks bound to a specific warehouse that have
// a dispatchable driver assigned. Implements strict state-checking:
//   - Vehicle: IsActive=true, WarehouseId matches
//   - Driver: IsActive=true (not offline), TruckStatus IN (AVAILABLE, RETURNING)
//   - VehicleId IS NOT NULL (driver must be assigned to a truck)
func GetAvailableFleet(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) ([]Truck, error) {
	sql := `SELECT v.VehicleId, COALESCE(v.WarehouseId, ''), v.VehicleClass,
	               v.MaxVolumeVU, v.LengthCM, v.WidthCM, v.HeightCM
	        FROM Vehicles v
	        JOIN Drivers d ON d.VehicleId = v.VehicleId
	        WHERE v.SupplierId = @sid
	          AND v.IsActive = true
	          AND d.IsActive = true
	          AND COALESCE(d.TruckStatus, 'AVAILABLE') IN ('AVAILABLE', 'RETURNING')
	          AND d.VehicleId IS NOT NULL`

	params := map[string]interface{}{"sid": supplierID}
	if warehouseID != "" {
		sql += " AND (v.WarehouseId = @warehouseId OR (v.HomeNodeType = 'WAREHOUSE' AND v.HomeNodeId = @warehouseId))"
		params["warehouseId"] = warehouseID
	}

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	var trucks []Truck
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query available fleet: %w", err)
		}

		var t Truck
		var whID spanner.NullString
		if err := row.Columns(&t.VehicleID, &whID, &t.VehicleClass,
			&t.MaxVolumeVU, &t.LengthCM, &t.WidthCM, &t.HeightCM); err != nil {
			return nil, fmt.Errorf("parse fleet row: %w", err)
		}
		t.WarehouseID = whID.StringVal
		t.SupplierID = supplierID
		t.IsActive = true
		trucks = append(trucks, t)
	}
	return trucks, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// SMART FIT DISPATCH — AssignFleetToOrders
//
// Execution order is CRITICAL:
//   1. VOLUME — TotalOrderVU is pre-computed before this function is called.
//   2. FLEET CAPACITY — selectBestVehicle finds the smallest truck that fits.
//   3. SPATIAL CLUSTER — H3 cell grouping for multi-stop consolidation.
//
// Do NOT cluster before verifying volume. If you assign H3 clusters before
// checking volume, trucks will be routed to zones where the required inventory
// physically exceeds their maximum payload capacity.
// ═══════════════════════════════════════════════════════════════════════════════

// AssignFleetToOrders implements the 4-rule Smart Fit protocol:
//
//	Rule 1 — Consolidation: Find the smallest single truck that fits TotalOrderVU.
//	Rule 2 — Multi-Stop:    Group nearby orders by H3 cell, fill truck to 95% cap.
//	Rule 3 — The Split:     If order exceeds max fleet capacity, split into chunks.
//	Rule 4 — Override:      Respect IgnoreCapacity boolean for manual payloads.
//
// All computation is pure — no Spanner calls. Designed for execution inside a
// ReadWriteTransaction where the caller handles persistence.
func AssignFleetToOrders(orders []dispatchableOrder, fleet []availableDriver) *AssignmentResult {
	result := &AssignmentResult{
		Routes:  []DispatchRoute{},
		Splits:  []SplitOrder{},
		Orphans: []GeoOrder{},
	}

	if len(orders) == 0 || len(fleet) == 0 {
		// All orders become orphans if no fleet
		for _, o := range orders {
			result.Orphans = append(result.Orphans, orderToGeo(o))
		}
		return result
	}

	// ── Step 1: Find max fleet capacity for split threshold ─────────────────
	maxFleetCap := 0.0
	for _, d := range fleet {
		eff := d.MaxVolumeVU * TetrisBuffer
		if eff > maxFleetCap {
			maxFleetCap = eff
		}
	}

	// ── Step 2: Separate normal orders vs. split-required vs. override ──────
	var normalOrders []dispatchableOrder
	for _, o := range orders {
		// Rule 4: IgnoreCapacity — skip volume check entirely
		if o.IgnoreCapacity {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("ORDER %s: IgnoreCapacity=true, bypassing volume check (%.1f VU)", o.OrderID, o.VolumeVU))
			normalOrders = append(normalOrders, o)
			continue
		}

		// Rule 3: The Split — order exceeds max fleet capacity
		if o.VolumeVU > maxFleetCap && maxFleetCap > 0 {
			numChunks := int(math.Ceil(o.VolumeVU / maxFleetCap))
			split := SplitOrder{
				OriginalOrderID: o.OrderID,
				TotalVolumeVU:   o.VolumeVU,
				Reason:          "EXCEEDS_MAX_FLEET_CAPACITY",
				Chunks:          make([]OrderChunk, numChunks),
			}
			remaining := o.VolumeVU
			for ci := 0; ci < numChunks; ci++ {
				chunkVol := math.Min(remaining, maxFleetCap)
				split.Chunks[ci] = OrderChunk{
					ChunkIndex: ci,
					VolumeVU:   chunkVol,
				}
				remaining -= chunkVol
			}
			result.Splits = append(result.Splits, split)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("ORDER %s: %.1f VU exceeds max fleet capacity %.1f VU — split into %d chunks",
					o.OrderID, o.VolumeVU, maxFleetCap, numChunks))

			// Create synthetic sub-orders for each chunk so they enter the normal pipeline
			for ci, chunk := range split.Chunks {
				sub := o
				sub.OrderID = fmt.Sprintf("%s-CHUNK-%d", o.OrderID, ci)
				sub.VolumeVU = chunk.VolumeVU
				normalOrders = append(normalOrders, sub)
			}
			continue
		}

		normalOrders = append(normalOrders, o)
	}

	// ── Step 3: H3 Cell Grouping (spatial clustering) ───────────────────────
	// Group orders by H3 hex cell for multi-stop consolidation.
	// Orders in the same cell are geographically proximate (~1.2km).
	cellGroups := make(map[string][]dispatchableOrder)
	cellOrder := []string{} // preserve insertion order for deterministic output
	for _, o := range normalOrders {
		cell := proximity.LookupCell(o.Lat, o.Lng)
		if _, exists := cellGroups[cell]; !exists {
			cellOrder = append(cellOrder, cell)
		}
		cellGroups[cell] = append(cellGroups[cell], o)
	}

	// ── Step 4: Per-cell greedy bin-packing ─────────────────────────────────
	// For each H3 cell group, compute aggregate volume and find the best truck.
	// Rule 2: Multi-stop — fill truck to TetrisBuffer (95%) with same-cell orders.

	// Track which drivers are already assigned to routes
	driverRouteMap := make(map[string]int) // driverID → index in result.Routes

	for _, cell := range cellOrder {
		group := cellGroups[cell]

		// Sort by volume DESC — pack largest orders first (decreasing first-fit)
		sort.Slice(group, func(i, j int) bool {
			return group[i].VolumeVU > group[j].VolumeVU
		})

		for _, o := range group {
			geo := orderToGeo(o)
			placed := false

			// Rule 4: IgnoreCapacity → assign to largest available truck
			if o.IgnoreCapacity {
				// Find the driver with the most remaining capacity
				bestRoute := -1
				bestRemaining := -1.0
				for ri := range result.Routes {
					rem := result.Routes[ri].MaxVolume - result.Routes[ri].LoadedVolume
					if rem > bestRemaining {
						bestRoute = ri
						bestRemaining = rem
					}
				}
				if bestRoute >= 0 {
					geo.Assigned = true
					result.Routes[bestRoute].Orders = append(result.Routes[bestRoute].Orders, geo)
					result.Routes[bestRoute].LoadedVolume += o.VolumeVU
					placed = true
				} else {
					// No route yet — create one with the largest truck
					match, ok := selectBestVehicle(0, fleet) // 0 VU ensures largest by overflow
					if ok {
						route := DispatchRoute{
							DriverID:     match.Driver.DriverID,
							MaxVolume:    match.Driver.MaxVolumeVU * TetrisBuffer,
							LoadedVolume: o.VolumeVU,
							Orders:       []GeoOrder{geo},
						}
						geo.Assigned = true
						driverRouteMap[match.Driver.DriverID] = len(result.Routes)
						result.Routes = append(result.Routes, route)
						placed = true
					}
				}
				if !placed {
					result.Orphans = append(result.Orphans, geo)
				}
				continue
			}

			// Rule 1: Consolidation — try to fit into an existing same-cell route
			for ri := range result.Routes {
				remaining := result.Routes[ri].MaxVolume - result.Routes[ri].LoadedVolume
				if remaining >= o.VolumeVU {
					geo.Assigned = true
					result.Routes[ri].Orders = append(result.Routes[ri].Orders, geo)
					result.Routes[ri].LoadedVolume += o.VolumeVU
					placed = true
					break
				}
			}
			if placed {
				continue
			}

			// Need a new truck — find the smallest that fits (Rule 1)
			match, ok := selectBestVehicle(o.VolumeVU, fleet)
			if !ok {
				result.Orphans = append(result.Orphans, geo)
				continue
			}

			if match.Overflow {
				geo.CapacityOverflow = true
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("ORDER %s: %.1f VU overflows best truck %s (%.1f effective VU)",
						o.OrderID, o.VolumeVU, match.Driver.VehicleClass, match.Driver.MaxVolumeVU*TetrisBuffer))
			}

			// Check if this driver already has a route
			if ri, exists := driverRouteMap[match.Driver.DriverID]; exists {
				remaining := result.Routes[ri].MaxVolume - result.Routes[ri].LoadedVolume
				if remaining >= o.VolumeVU || o.IgnoreCapacity {
					geo.Assigned = true
					result.Routes[ri].Orders = append(result.Routes[ri].Orders, geo)
					result.Routes[ri].LoadedVolume += o.VolumeVU
					continue
				}
			}

			// Create a new route for this driver
			route := DispatchRoute{
				DriverID:     match.Driver.DriverID,
				MaxVolume:    match.Driver.MaxVolumeVU * TetrisBuffer,
				LoadedVolume: o.VolumeVU,
				Orders:       []GeoOrder{geo},
			}
			geo.Assigned = true
			driverRouteMap[match.Driver.DriverID] = len(result.Routes)
			result.Routes = append(result.Routes, route)
		}
	}

	return result
}

// orderToGeo converts a dispatchableOrder to a GeoOrder for the algorithm pipeline.
func orderToGeo(o dispatchableOrder) GeoOrder {
	return GeoOrder{
		OrderID:              o.OrderID,
		RetailerID:           o.RetailerID,
		RetailerName:         o.RetailerName,
		Amount:               o.Amount,
		Lat:                  o.Lat,
		Lng:                  o.Lng,
		Volume:               o.VolumeVU,
		ReceivingWindowOpen:  o.ReceivingWindowOpen,
		ReceivingWindowClose: o.ReceivingWindowClose,
		IsRecovery:           o.IsRecovery,
	}
}

// ── Request / Response Types ────────────────────────────────────────────────

type AutoDispatchRequest struct {
	// If empty, auto-dispatch considers ALL unassigned PENDING/LOADED orders for this supplier.
	OrderIDs []string `json:"order_ids,omitempty"`
	// Soft exclusions — trucks the Admin wants to hold back for this specific run.
	ExcludedTruckIds []string `json:"excluded_truck_ids,omitempty"`
}

type AutoDispatchResult struct {
	SnapshotTimestamp  string              `json:"snapshot_timestamp"`
	Manifests          []TruckManifest     `json:"manifests"`
	Orphans            []OrphanOrder       `json:"orphans"`
	TimeWindowWarnings []TimeWindowWarning `json:"time_window_warnings,omitempty"`
}

type TruckManifest struct {
	ManifestID         string                 `json:"manifest_id,omitempty"`
	RouteID            string                 `json:"route_id"`
	DriverID           string                 `json:"driver_id"`
	DriverName         string                 `json:"driver_name"`
	VehicleType        string                 `json:"vehicle_type"`
	VehicleClass       string                 `json:"vehicle_class"`
	MaxVolumeVU        float64                `json:"max_volume_vu"`
	UsedVolumeVU       float64                `json:"used_volume_vu"`
	Orders             []DispatchOrder        `json:"orders"`
	LoadingManifest    []LoadingManifestEntry `json:"loading_manifest"`
	GeoZone            string                 `json:"geo_zone"`
	ForceAssignedCount int                    `json:"force_assigned_count,omitempty"`
	NavigationURL      string                 `json:"navigation_url,omitempty"`
	NavigationSegments []string               `json:"navigation_segments,omitempty"`
	SegmentCount       int                    `json:"segment_count"`
	GPSWaypoints       []GPSWaypoint          `json:"gps_waypoints,omitempty"`
}

// GPSWaypoint represents a single ordered stop in the route's GPS sequence.
type GPSWaypoint struct {
	Sequence             int     `json:"sequence"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	RetailerName         string  `json:"retailer_name"`
	OrderID              string  `json:"order_id"`
	EstimatedArrivalTime string  `json:"estimated_arrival_time,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
}

// LoadingManifestEntry describes the order in which items should be loaded onto the truck
// so that the first delivery stop is last-in (nearest to the doors).
// Sequence 1 = deepest in the truck (Back of Truck), sequence N = nearest the doors.
type LoadingManifestEntry struct {
	LoadSequence int     `json:"load_sequence"`
	OrderID      string  `json:"order_id"`
	RetailerName string  `json:"retailer_name"`
	VolumeVU     float64 `json:"volume_vu"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Instruction  string  `json:"instruction"`
}

type DispatchOrder struct {
	OrderID              string  `json:"order_id"`
	RetailerID           string  `json:"retailer_id"`
	RetailerName         string  `json:"retailer_name"`
	Amount               int64   `json:"amount"`
	VolumeVU             float64 `json:"volume_vu"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	ForceAssigned        bool    `json:"force_assigned,omitempty"`
	CapacityOverflow     bool    `json:"capacity_overflow,omitempty"`
	LogisticsIsolated    bool    `json:"logistics_isolated,omitempty"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
}

type OrphanOrder struct {
	OrderID      string `json:"order_id"`
	RetailerName string `json:"retailer_name"`
	Reason       string `json:"reason"`
}

// TimeWindowWarning flags an order whose estimated arrival exceeds the
// retailer's receiving window close time. Soft constraint — warns, not blocks.
type TimeWindowWarning struct {
	OrderID          string `json:"order_id"`
	RetailerName     string `json:"retailer_name"`
	WindowClose      string `json:"window_close"`
	EstimatedArrival string `json:"estimated_arrival"`
	DeltaMinutes     int    `json:"delta_minutes"`
	StopSequence     int    `json:"stop_sequence"`
	RouteID          string `json:"route_id"`
}

// ── Internal working types ──────────────────────────────────────────────────

type dispatchableOrder struct {
	OrderID              string
	RetailerID           string
	RetailerName         string
	Amount               int64
	Lat                  float64
	Lng                  float64
	VolumeVU             float64 // total VU for this order
	ReceivingWindowOpen  string  // "HH:MM" or ""
	ReceivingWindowClose string  // "HH:MM" or ""
	IgnoreCapacity       bool    // Manual payload override — skip volume check
	IsRecovery           bool    // Overflow-bounced; Clarke-Wright adds +10k savings boost
}

type availableDriver struct {
	DriverID     string
	Name         string
	VehicleType  string
	VehicleClass string
	MaxVolumeVU  float64
}

// ── Handler ─────────────────────────────────────────────────────────────────

func HandleAutoDispatch(client *spanner.Client, readRouter proximity.ReadRouter, manifestSvc *ManifestService, optimizer *optimizerclient.Client, counters *plan.SourceCounters) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		var req AutoDispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		result, err := runAutoDispatch(ctx, client, readRouter, supplierID, req.OrderIDs, req.ExcludedTruckIds, manifestSvc, optimizer, counters, false)
		if err != nil {
			log.Printf("[AUTO-DISPATCH] error for supplier %s: %v", supplierID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleDispatchRecommend runs the dispatch algorithm in dry-run mode.
// Returns recommended truck→order groupings without any Spanner mutations.
// POST /v1/supplier/manifests/dispatch-recommend
func HandleDispatchRecommend(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		var req AutoDispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		result, err := runAutoDispatch(ctx, client, readRouter, supplierID, req.OrderIDs, req.ExcludedTruckIds, nil, nil, nil, true)
		if err != nil {
			log.Printf("[DISPATCH-RECOMMEND] error for supplier %s: %v", supplierID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// ManualDispatchRequest is the body for POST /v1/supplier/manifests/manual-dispatch.
type ManualDispatchRequest struct {
	DriverID string   `json:"driver_id"`
	OrderIDs []string `json:"order_ids"`
}

// HandleManualDispatch creates a LEO DRAFT manifest for the specified driver
// and orders. Used by the Manual Dispatch mode where the admin has chosen
// exactly which orders go to which truck.
// POST /v1/supplier/manifests/manual-dispatch
func HandleManualDispatch(client *spanner.Client, readRouter proximity.ReadRouter, manifestSvc *ManifestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		var req ManualDispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.DriverID == "" || len(req.OrderIDs) == 0 {
			http.Error(w, `{"error":"driver_id and order_ids are required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Fetch driver info
		driver, err := fetchDriverByID(ctx, client, supplierID, req.DriverID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"driver lookup: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Fetch order data
		orders, err := fetchDispatchableOrders(ctx, client, readRouter, supplierID, req.OrderIDs)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"fetch orders: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			http.Error(w, `{"error":"no dispatchable orders found for the given IDs"}`, http.StatusBadRequest)
			return
		}

		// Convert to GeoOrders for ordering
		geoOrders := make([]GeoOrder, len(orders))
		for i, o := range orders {
			geoOrders[i] = GeoOrder{
				OrderID:      o.OrderID,
				RetailerID:   o.RetailerID,
				RetailerName: o.RetailerName,
				Amount:       o.Amount,
				Lat:          o.Lat,
				Lng:          o.Lng,
				Volume:       o.VolumeVU,
			}
		}

		// Nearest-neighbour ordering from depot
		var originLat, originLng float64
		if whID := auth.EffectiveWarehouseID(ctx); whID != "" {
			if lat, lng, ok := fetchWarehouseOrigin(ctx, client, whID); ok {
				originLat, originLng = lat, lng
			}
		}
		if originLat == 0 && originLng == 0 {
			c := clusterCentroid(geoOrders)
			originLat, originLng = c[0], c[1]
		}
		nearestNeighborSort(geoOrders, originLat, originLng)

		routeID := uuid.New().String()
		totalVol := 0.0
		dispatchOrders := make([]DispatchOrder, len(geoOrders))
		for i, o := range geoOrders {
			totalVol += o.Volume
			dispatchOrders[i] = DispatchOrder{
				OrderID:      o.OrderID,
				RetailerID:   o.RetailerID,
				RetailerName: o.RetailerName,
				Amount:       o.Amount,
				VolumeVU:     o.Volume,
				Lat:          o.Lat,
				Lng:          o.Lng,
			}
		}

		// Build LIFO loading manifest
		n := len(geoOrders)
		loadingManifest := make([]LoadingManifestEntry, n)
		for k, o := range geoOrders {
			seq := n - k
			instruction := fmt.Sprintf("Load position %d of %d", seq, n)
			if seq == 1 {
				instruction = "Load first — Back of Truck"
			} else if seq == n {
				instruction = "Load last — By the Doors"
			}
			loadingManifest[seq-1] = LoadingManifestEntry{
				LoadSequence: seq,
				OrderID:      o.OrderID,
				RetailerName: o.RetailerName,
				VolumeVU:     o.Volume,
				Lat:          o.Lat,
				Lng:          o.Lng,
				Instruction:  instruction,
			}
		}

		manifest := TruckManifest{
			RouteID:         routeID,
			DriverID:        driver.DriverID,
			DriverName:      driver.Name,
			VehicleType:     driver.VehicleType,
			VehicleClass:    driver.VehicleClass,
			MaxVolumeVU:     driver.MaxVolumeVU,
			UsedVolumeVU:    totalVol,
			Orders:          dispatchOrders,
			LoadingManifest: loadingManifest,
			GeoZone:         routeGeoZone(geoOrders),
		}
		manifest.NavigationSegments = buildNavigationSegments(geoOrders)
		manifest.SegmentCount = len(manifest.NavigationSegments)
		if len(manifest.NavigationSegments) > 0 {
			manifest.NavigationURL = manifest.NavigationSegments[0]
		}

		// Create LEO DRAFT manifest
		warehouseID := auth.EffectiveWarehouseID(ctx)
		if manifestSvc != nil {
			mID, err := manifestSvc.CreateDraftManifest(
				ctx, supplierID, warehouseID,
				routeID, driver.DriverID, driver.DriverID,
				driver.MaxVolumeVU, manifest.GeoZone,
				manifest.Orders, manifest.LoadingManifest,
			)
			if err != nil {
				log.Printf("[MANUAL-DISPATCH] LEO DRAFT creation failed for route %s: %v", routeID, err)
				http.Error(w, fmt.Sprintf(`{"error":"manifest creation: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			manifest.ManifestID = mID
		}

		log.Printf("[MANUAL-DISPATCH] supplier=%s driver=%s | %d orders, %.1f VU, route=%s",
			supplierID, driver.DriverID, len(orders), totalVol, routeID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}
}

// fetchDriverByID loads a single driver by ID, scoped to the supplier.
func fetchDriverByID(ctx context.Context, client *spanner.Client, supplierID, driverID string) (*availableDriver, error) {
	stmt := spanner.Statement{
		SQL: `SELECT d.DriverId, d.Name, d.VehicleType, d.VehicleClass,
		             COALESCE(v.MaxVolumeVU, 100) AS MaxVolumeVU
		      FROM Drivers d
		      LEFT JOIN Vehicles v ON d.DriverId = v.AssignedDriverId
		      WHERE d.DriverId = @did AND d.SupplierId = @sid
		      LIMIT 1`,
		Params: map[string]interface{}{
			"did": driverID,
			"sid": supplierID,
		},
	}
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("driver %s not found: %w", driverID, err)
	}
	var d availableDriver
	if err := row.Columns(&d.DriverID, &d.Name, &d.VehicleType, &d.VehicleClass, &d.MaxVolumeVU); err != nil {
		return nil, fmt.Errorf("scan driver: %w", err)
	}
	return &d, nil
}

// ── Core Algorithm ──────────────────────────────────────────────────────────

func runAutoDispatch(ctx context.Context, client *spanner.Client, readRouter proximity.ReadRouter, supplierID string, filterOrderIDs []string, excludedTruckIDs []string, manifestSvc *ManifestService, optimizer *optimizerclient.Client, counters *plan.SourceCounters, previewOnly bool) (*AutoDispatchResult, error) {
	snapshotTS := time.Now().UTC().Format(time.RFC3339)

	// ── Data Ingestion ──────────────────────────────────────────────────────
	orders, err := fetchDispatchableOrders(ctx, client, readRouter, supplierID, filterOrderIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch orders: %w", err)
	}
	if len(orders) == 0 {
		return &AutoDispatchResult{SnapshotTimestamp: snapshotTS, Manifests: []TruckManifest{}, Orphans: []OrphanOrder{}}, nil
	}

	// ── Fleet Snapshot (immutable for the duration of this calculation) ────
	drivers, err := fetchAvailableDrivers(ctx, client, supplierID)
	if err != nil {
		return nil, fmt.Errorf("fetch drivers: %w", err)
	}
	// ── Soft Exclusions: remove admin-deselected trucks from the snapshot ──
	if len(excludedTruckIDs) > 0 {
		excludeSet := make(map[string]bool, len(excludedTruckIDs))
		for _, id := range excludedTruckIDs {
			excludeSet[id] = true
		}
		filtered := drivers[:0]
		for _, d := range drivers {
			if !excludeSet[d.DriverID] {
				filtered = append(filtered, d)
			}
		}
		drivers = filtered
	}

	if len(drivers) == 0 {
		orphans := make([]OrphanOrder, len(orders))
		orderIDs := make([]string, len(orders))
		for i, o := range orders {
			orphans[i] = OrphanOrder{OrderID: o.OrderID, RetailerName: o.RetailerName, Reason: "NO_DRIVERS_AVAILABLE"}
			orderIDs[i] = o.OrderID
		}
		if !previewOnly {
			// Edge 10: Transition orphaned orders to NO_CAPACITY
			now := time.Now().UTC()
			_, txnErr := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				var mutations []*spanner.Mutation
				for _, oid := range orderIDs {
					mutations = append(mutations, spanner.Update("Orders",
						[]string{"OrderId", "State", "UpdatedAt"},
						[]interface{}{oid, "NO_CAPACITY", now}))
				}
				return txn.BufferWrite(mutations)
			})
			if txnErr != nil {
				log.Printf("[AUTO-DISPATCH] Failed to transition %d orders to NO_CAPACITY: %v", len(orderIDs), txnErr)
			} else {
				log.Printf("[AUTO-DISPATCH] Transitioned %d orders to NO_CAPACITY (zero trucks)", len(orderIDs))
			}
		}
		return &AutoDispatchResult{SnapshotTimestamp: snapshotTS, Manifests: []TruckManifest{}, Orphans: orphans}, nil
	}

	K := len(drivers)

	// Convert to GeoOrders for the algorithm pipeline
	geoOrders := make([]GeoOrder, len(orders))
	for i, o := range orders {
		geoOrders[i] = GeoOrder{
			OrderID:              o.OrderID,
			RetailerID:           o.RetailerID,
			RetailerName:         o.RetailerName,
			Amount:               o.Amount,
			Lat:                  o.Lat,
			Lng:                  o.Lng,
			Volume:               o.VolumeVU,
			ReceivingWindowOpen:  o.ReceivingWindowOpen,
			ReceivingWindowClose: o.ReceivingWindowClose,
		}
	}

	// Sort drivers by capacity DESC — biggest trucks absorb densest clusters
	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].MaxVolumeVU > drivers[j].MaxVolumeVU
	})

	// ── Phase 2 Optimizer (SHADOW MODE) ─────────────────────────────────────
	// When the optimiser client is armed, run plan.OptimizeAndValidate against
	// the same hydrated input and log a comparison line. The Phase 1 inline
	// pipeline below remains the canonical execution path until parity is
	// validated under load (Phase G). The optimiser result is intentionally
	// discarded here — flipping it to primary is a follow-up wave.
	if optimizer != nil && !previewOnly {
		shadowOrders := make([]dispatch.DispatchableOrder, len(orders))
		for i, o := range orders {
			shadowOrders[i] = dispatch.DispatchableOrder{
				OrderID:              o.OrderID,
				RetailerID:           o.RetailerID,
				RetailerName:         o.RetailerName,
				Amount:               o.Amount,
				Lat:                  o.Lat,
				Lng:                  o.Lng,
				VolumeVU:             o.VolumeVU,
				ReceivingWindowOpen:  o.ReceivingWindowOpen,
				ReceivingWindowClose: o.ReceivingWindowClose,
			}
		}
		shadowFleet := make([]dispatch.AvailableDriver, len(drivers))
		for i, d := range drivers {
			shadowFleet[i] = dispatch.AvailableDriver{
				DriverID:     d.DriverID,
				DriverName:   d.Name,
				VehicleID:    d.VehicleType,
				VehicleClass: d.VehicleClass,
				MaxVolumeVU:  d.MaxVolumeVU,
			}
		}
		go func(orderCount, driverCount int) {
			shadowCtx, shadowCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shadowCancel()
			t0 := time.Now()
			result, source, err := plan.OptimizeAndValidate(shadowCtx, optimizer, plan.Job{
				TraceID:    snapshotTS,
				SupplierID: supplierID,
				Orders:     shadowOrders,
				Fleet:      shadowFleet,
			})
			elapsed := time.Since(t0)
			if err != nil {
				counters.RecordError()
				slog.Warn("dispatch.optimize.shadow",
					"supplier_id", supplierID,
					"trace_id", snapshotTS,
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
				"supplier_id", supplierID,
				"trace_id", snapshotTS,
				"orders", orderCount,
				"drivers", driverCount,
				"source", source,
				"elapsed_ms", elapsed.Milliseconds(),
				"routes", len(result.Routes),
				"orphans", len(result.Orphans),
			)
		}(len(orders), len(drivers))
	}

	// ── PHASE 0: Retailer-Atomic Grouping ───────────────────────────────────
	// Collapse multiple orders for the same retailer into one super-order so
	// clustering + bin-packing keep them on a single truck. Expansion back to
	// real OrderIDs happens right before manifest assembly.
	maxTruckEff := 0.0
	for _, d := range drivers {
		if eff := d.MaxVolumeVU * TetrisBuffer; eff > maxTruckEff {
			maxTruckEff = eff
		}
	}
	packInput, retailerExpansion := groupOrdersByRetailer(geoOrders, maxTruckEff)

	// ── PHASE 1: K-Means Spatial Clustering ─────────────────────────────────
	clusters := kMeansCluster(packInput, K)

	// Sort clusters by total volume DESC so they pair with biggest trucks
	sort.Slice(clusters, func(i, j int) bool {
		totalI, totalJ := 0.0, 0.0
		for _, o := range clusters[i] {
			totalI += o.Volume
		}
		for _, o := range clusters[j] {
			totalJ += o.Volume
		}
		return totalI > totalJ
	})

	// ── Build truck routes ──────────────────────────────────────────────────
	// Tetris Buffer: reserve 5% capacity headroom for volumetric safety margin.
	routes := make([]DispatchRoute, K)
	for i := 0; i < K; i++ {
		routes[i] = DispatchRoute{
			DriverID:  drivers[i].DriverID,
			MaxVolume: drivers[i].MaxVolumeVU * TetrisBuffer,
			Orders:    []GeoOrder{},
		}
	}

	// ── PHASE 2: Atomic Bin-Packing ─────────────────────────────────────────
	var misfitPool []GeoOrder

	for ci, cluster := range clusters {
		if ci >= K {
			// More clusters than trucks — entire cluster goes to misfit
			misfitPool = append(misfitPool, cluster...)
			continue
		}

		// Sort orders within cluster: earliest receiving window close first,
		// with haversine distance to centroid as tiebreaker.
		// Orders without a window are treated as flexible ("23:59").
		centroid := clusterCentroid(cluster)
		sort.Slice(cluster, func(i, j int) bool {
			wcI := effectiveWindowClose(cluster[i].ReceivingWindowClose)
			wcJ := effectiveWindowClose(cluster[j].ReceivingWindowClose)
			if wcI != wcJ {
				return wcI < wcJ
			}
			distI := haversineKm(cluster[i].Lat, cluster[i].Lng, centroid[0], centroid[1])
			distJ := haversineKm(cluster[j].Lat, cluster[j].Lng, centroid[0], centroid[1])
			return distI < distJ
		})

		for _, order := range cluster {
			if routes[ci].LoadedVolume+order.Volume <= routes[ci].MaxVolume {
				order.Assigned = true
				routes[ci].Orders = append(routes[ci].Orders, order)
				routes[ci].LoadedVolume += order.Volume
			} else {
				// ATOMIC CONSTRAINT: entire order goes to misfit pool
				misfitPool = append(misfitPool, order)
			}
		}
	}

	// ── PHASE 3: Outlier Sweep (Density Penalty) ────────────────────────────
	// Candidate order: (orderCount ASC, detour ASC). Empty trucks bypass the
	// detour gate — there is no existing route to deviate from — which turns
	// a would-be orphan into a fresh route instead.
	var finalOrphans []GeoOrder
	for _, misfit := range misfitPool {
		placed := false
		bestTruck := -1
		bestCount := math.MaxInt
		bestDetour := math.MaxFloat64

		for ti := range routes {
			remaining := routes[ti].MaxVolume - routes[ti].LoadedVolume
			if remaining < misfit.Volume {
				continue
			}
			count := len(routes[ti].Orders)
			detour := 0.0
			if count > 0 {
				routeCentroid := clusterCentroid(routes[ti].Orders)
				detour = haversineKm(misfit.Lat, misfit.Lng, routeCentroid[0], routeCentroid[1])
				if detour > MaxDetourRadius {
					continue
				}
			}
			if count < bestCount || (count == bestCount && detour < bestDetour) {
				bestTruck = ti
				bestCount = count
				bestDetour = detour
			}
		}

		if bestTruck >= 0 {
			misfit.Assigned = true
			routes[bestTruck].Orders = append(routes[bestTruck].Orders, misfit)
			routes[bestTruck].LoadedVolume += misfit.Volume
			placed = true
		}

		if !placed {
			misfit.LogisticsIsolated = true
			finalOrphans = append(finalOrphans, misfit)
		}
	}

	// ── PHASE 4: Force-Assign (Zero-Orphan Guarantee) ──────────────────────
	// ALL orders MUST be assigned. Orders beyond MaxDetourRadius are force-assigned
	// to the least-loaded truck that has capacity, ignoring distance constraints.
	// If no truck has volume capacity, the order goes to the truck with the most
	// remaining capacity anyway (overload-tagged). Zero abandonment.
	if len(finalOrphans) > 0 && len(routes) > 0 {
		var stillOrphaned []GeoOrder
		for _, orphan := range finalOrphans {
			// Pass 1: trucks with capacity — (orderCount ASC, remaining DESC).
			// Empty/lightly-loaded trucks absorb orphans before consolidated ones.
			bestTruck := -1
			bestCount := math.MaxInt
			bestRemaining := -1.0
			for ti := range routes {
				remaining := routes[ti].MaxVolume - routes[ti].LoadedVolume
				if remaining < orphan.Volume {
					continue
				}
				count := len(routes[ti].Orders)
				if count < bestCount || (count == bestCount && remaining > bestRemaining) {
					bestTruck = ti
					bestCount = count
					bestRemaining = remaining
				}
			}
			// Pass 2: no truck has volume capacity — accept overflow on the
			// fewest-loaded truck, breaking ties by most remaining capacity.
			if bestTruck < 0 {
				bestCount = math.MaxInt
				bestRemaining = -math.MaxFloat64
				for ti := range routes {
					remaining := routes[ti].MaxVolume - routes[ti].LoadedVolume
					count := len(routes[ti].Orders)
					if count < bestCount || (count == bestCount && remaining > bestRemaining) {
						bestTruck = ti
						bestCount = count
						bestRemaining = remaining
					}
				}
			}
			if bestTruck >= 0 {
				orphan.Assigned = true
				orphan.ForceAssigned = true
				// Tag capacity overflow when loaded volume exceeds max
				if routes[bestTruck].LoadedVolume+orphan.Volume > routes[bestTruck].MaxVolume {
					orphan.CapacityOverflow = true
				}
				routes[bestTruck].Orders = append(routes[bestTruck].Orders, orphan)
				routes[bestTruck].LoadedVolume += orphan.Volume
			} else {
				stillOrphaned = append(stillOrphaned, orphan)
			}
		}
		finalOrphans = stillOrphaned
		if len(finalOrphans) > 0 {
			log.Printf("[AUTO-DISPATCH] WARNING: %d orders could not be force-assigned (zero trucks)", len(finalOrphans))
			if !previewOnly {
				// Edge 10: Transition truly unplaceable orders to NO_CAPACITY
				now := time.Now().UTC()
				noCapIDs := make([]string, len(finalOrphans))
				for i, o := range finalOrphans {
					noCapIDs[i] = o.OrderID
				}
				_, txnErr := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					var mutations []*spanner.Mutation
					for _, oid := range noCapIDs {
						mutations = append(mutations, spanner.Update("Orders",
							[]string{"OrderId", "State", "UpdatedAt"},
							[]interface{}{oid, "NO_CAPACITY", now}))
					}
					return txn.BufferWrite(mutations)
				})
				if txnErr != nil {
					log.Printf("[AUTO-DISPATCH] Failed to mark %d orders NO_CAPACITY: %v", len(noCapIDs), txnErr)
				}
			}
		}
	}

	// ── Expand retailer-atomic super-orders back to real OrderIDs ───────────
	// Every child order inherits Assigned/ForceAssigned/CapacityOverflow/
	// LogisticsIsolated from its parent super-order so Spanner writes and
	// orphan reporting use concrete OrderIDs.
	if len(retailerExpansion) > 0 {
		for ri := range routes {
			routes[ri].Orders = expandRetailerGroups(routes[ri].Orders, retailerExpansion)
		}
		finalOrphans = expandRetailerGroups(finalOrphans, retailerExpansion)
	}

	// ── Serialize to API Response ───────────────────────────────────────────
	// RULE OF 25: delegate splitting to the ManifestGroup module which
	// enforces the Google Maps Directions API 25-waypoint ceiling.
	var allWarnings []TimeWindowWarning
	manifests := make([]TruckManifest, 0, K)
	// driverManifestIndex maps driver index → list of manifest indices for LEO creation
	type driverManifestRef struct {
		driverIdx    int
		manifestIdxs []int
	}
	var driverManifestRefs []driverManifestRef

	// Resolve the depot origin once per run. For warehouse-scoped dispatch
	// (WAREHOUSE_ADMIN / PAYLOADER) every order ships from the same depot, so
	// nearest-neighbour sequencing and ETA computation are anchored there. For
	// supplier-wide dispatch spanning multiple warehouses, depotKnown is false
	// and each chunk falls back to its own cluster centroid.
	var depotLat, depotLng float64
	depotKnown := false
	if whID := auth.EffectiveWarehouseID(ctx); whID != "" {
		if lat, lng, ok := fetchWarehouseOrigin(ctx, client, whID); ok {
			depotLat, depotLng, depotKnown = lat, lng, true
		}
	}

	for i, route := range routes {
		if len(route.Orders) == 0 {
			continue
		}

		// Rule of 25: split orders into ≤25-stop chunks via ManifestGroup
		group := SplitManifest(drivers[i].DriverID, drivers[i].DriverID, route.Orders, MaxWaypointsPerManifest)

		ref := driverManifestRef{driverIdx: i}
		for _, chunk := range group.Chunks {
			// Geographic ordering: reorder chunk.Orders into delivery sequence
			// via greedy nearest-neighbour from the depot (or cluster centroid
			// when the depot is unknown). This drives the manifest stop list,
			// the LIFO loading manifest, and the GPS waypoint sequence.
			originLat, originLng := depotLat, depotLng
			if !depotKnown {
				c := clusterCentroid(chunk.Orders)
				originLat, originLng = c[0], c[1]
			}
			nearestNeighborSort(chunk.Orders, originLat, originLng)

			manifest := TruckManifest{
				RouteID:      chunk.RouteID,
				DriverID:     drivers[i].DriverID,
				DriverName:   drivers[i].Name,
				VehicleType:  drivers[i].VehicleType,
				VehicleClass: drivers[i].VehicleClass,
				MaxVolumeVU:  drivers[i].MaxVolumeVU,
				UsedVolumeVU: chunk.VolumeVU,
				GeoZone:      routeGeoZone(chunk.Orders),
				Orders:       make([]DispatchOrder, len(chunk.Orders)),
			}
			for j, o := range chunk.Orders {
				manifest.Orders[j] = DispatchOrder{
					OrderID:              o.OrderID,
					RetailerID:           o.RetailerID,
					RetailerName:         o.RetailerName,
					Amount:               o.Amount,
					VolumeVU:             o.Volume,
					Lat:                  o.Lat,
					Lng:                  o.Lng,
					ForceAssigned:        o.ForceAssigned,
					CapacityOverflow:     o.CapacityOverflow,
					LogisticsIsolated:    o.LogisticsIsolated,
					ReceivingWindowOpen:  o.ReceivingWindowOpen,
					ReceivingWindowClose: o.ReceivingWindowClose,
				}
				if o.ForceAssigned {
					manifest.ForceAssignedCount++
				}
			}

			// Build LIFO loading manifest
			n := len(chunk.Orders)
			manifest.LoadingManifest = make([]LoadingManifestEntry, n)
			for k, o := range chunk.Orders {
				seq := n - k
				instruction := fmt.Sprintf("Load position %d of %d", seq, n)
				if seq == 1 {
					instruction = "Load first — Back of Truck"
				} else if seq == n {
					instruction = "Load last — By the Doors"
				}
				manifest.LoadingManifest[seq-1] = LoadingManifestEntry{
					LoadSequence: seq,
					OrderID:      o.OrderID,
					RetailerName: o.RetailerName,
					VolumeVU:     o.Volume,
					Lat:          o.Lat,
					Lng:          o.Lng,
					Instruction:  instruction,
				}
			}

			// Build GPS waypoint sequence. ETAs are anchored at the same
			// origin used for nearest-neighbour sequencing to keep the
			// loading→departure→stop chain consistent.
			etas := computeStopETAs(chunk.Orders, originLat, originLng)
			manifest.GPSWaypoints = make([]GPSWaypoint, n)
			for k, o := range chunk.Orders {
				wp := GPSWaypoint{
					Sequence:             k + 1,
					Lat:                  o.Lat,
					Lng:                  o.Lng,
					RetailerName:         o.RetailerName,
					OrderID:              o.OrderID,
					ReceivingWindowClose: o.ReceivingWindowClose,
				}
				if k < len(etas) {
					wp.EstimatedArrivalTime = etas[k].ArrivalStr
				}
				manifest.GPSWaypoints[k] = wp
			}
			manifest.NavigationSegments = buildNavigationSegments(chunk.Orders)
			manifest.SegmentCount = len(manifest.NavigationSegments)
			if len(manifest.NavigationSegments) > 0 {
				manifest.NavigationURL = manifest.NavigationSegments[0]
			}

			// Time window risk detection
			for k, o := range chunk.Orders {
				if o.ReceivingWindowClose == "" || k >= len(etas) {
					continue
				}
				closeMin := parseTimeHHMM(o.ReceivingWindowClose)
				if closeMin > 0 && int(etas[k].ArrivalMinutes) > closeMin {
					allWarnings = append(allWarnings, TimeWindowWarning{
						OrderID:          o.OrderID,
						RetailerName:     o.RetailerName,
						WindowClose:      o.ReceivingWindowClose,
						EstimatedArrival: etas[k].ArrivalStr,
						DeltaMinutes:     int(etas[k].ArrivalMinutes) - closeMin,
						StopSequence:     k + 1,
						RouteID:          manifest.RouteID,
					})
				}
			}

			ref.manifestIdxs = append(ref.manifestIdxs, len(manifests))
			manifests = append(manifests, manifest)
		}
		driverManifestRefs = append(driverManifestRefs, ref)
	}

	// ── LEO: Create DRAFT manifest entities in Spanner ──────────────────────
	// Each dispatch route becomes a SupplierTruckManifest in DRAFT state.
	// Payloader must start-loading → seal before driver can navigate.
	if !previewOnly {
		warehouseID := auth.EffectiveWarehouseID(ctx)
		if manifestSvc != nil {
			for _, ref := range driverManifestRefs {
				drv := drivers[ref.driverIdx]
				for _, mi := range ref.manifestIdxs {
					m := &manifests[mi]
					mID, err := manifestSvc.CreateDraftManifest(
						ctx, supplierID, warehouseID,
						m.RouteID, drv.DriverID, drv.DriverID,
						drv.MaxVolumeVU, m.GeoZone,
						m.Orders, m.LoadingManifest,
					)
					if err != nil {
						log.Printf("[LEO] DRAFT creation failed for route %s: %v", m.RouteID, err)
					} else {
						m.ManifestID = mID
					}
				}
			}
		}
	}

	// ── FLEET_DISPATCHED is emitted INSIDE CreateDraftManifest's outbox txn
	// (atomic with the manifest commit). No post-loop emission needed.

	orphans := make([]OrphanOrder, len(finalOrphans))
	for i, o := range finalOrphans {
		reason := "INSUFFICIENT_FLEET_CAPACITY"
		if o.Volume > 0 {
			reason = fmt.Sprintf("NO_VIABLE_ROUTE (volume=%.2f)", o.Volume)
		}
		orphans[i] = OrphanOrder{OrderID: o.OrderID, RetailerName: o.RetailerName, Reason: reason}
	}

	if len(manifests) == 0 {
		manifests = []TruckManifest{}
	}
	if len(orphans) == 0 {
		orphans = []OrphanOrder{}
	}

	log.Printf("[AUTO-DISPATCH] supplier=%s | %d orders → %d trucks, %d misfits, %d orphans, %d time-window warnings",
		supplierID, len(orders), len(manifests), len(misfitPool), len(orphans), len(allWarnings))

	if !previewOnly {
		// ── Warehouse Load Tracking: increment queue depth per warehouse ────────
		if whID := auth.EffectiveWarehouseID(ctx); whID != "" && len(manifests) > 0 {
			// Single-warehouse dispatch — increment by manifest count
			cache.BulkIncrementQueueDepth(ctx, map[string]int{whID: len(manifests)})
		}
	}

	if len(allWarnings) == 0 {
		allWarnings = []TimeWindowWarning{}
	}

	if !previewOnly {
		// ── Persist dispatch metadata flags to Spanner ──────────────────────────
		// Write CapacityOverflow, LogisticsIsolated, and DispatchWarnings for
		// flagged orders so admin portal can surface them.
		var metaMutations []*spanner.Mutation
		for _, manifest := range manifests {
			for _, o := range manifest.Orders {
				if !o.CapacityOverflow && !o.LogisticsIsolated {
					continue
				}
				cols := []string{"OrderId"}
				vals := []interface{}{o.OrderID}

				if o.CapacityOverflow {
					cols = append(cols, "CapacityOverflow")
					vals = append(vals, true)
				}
				if o.LogisticsIsolated {
					cols = append(cols, "LogisticsIsolated")
					vals = append(vals, true)
				}

				// Build warnings JSON
				var warnings []string
				if o.CapacityOverflow {
					warnings = append(warnings, "CAPACITY_OVERFLOW")
				}
				if o.LogisticsIsolated {
					warnings = append(warnings, "LOGISTICS_ISOLATED")
				}
				if len(warnings) > 0 {
					warnJSON, _ := json.Marshal(warnings)
					cols = append(cols, "DispatchWarnings")
					vals = append(vals, string(warnJSON))
				}

				metaMutations = append(metaMutations, spanner.Update("Orders", cols, vals))
			}
		}
		if len(metaMutations) > 0 {
			if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite(metaMutations)
			}); err != nil {
				log.Printf("[AUTO-DISPATCH] WARNING: failed to persist dispatch metadata: %v", err)
				// Non-fatal — dispatch result is still valid
			} else {
				log.Printf("[AUTO-DISPATCH] Persisted dispatch metadata for %d flagged orders", len(metaMutations))
			}
		}
	}

	return &AutoDispatchResult{SnapshotTimestamp: snapshotTS, Manifests: manifests, Orphans: orphans, TimeWindowWarnings: allWarnings}, nil
}

// ── Step 1: Fetch Dispatchable Orders ───────────────────────────────────────

func fetchDispatchableOrders(ctx context.Context, client *spanner.Client, readRouter proximity.ReadRouter, supplierID string, filterOrderIDs []string) ([]dispatchableOrder, error) {
	// Query: get PENDING/LOADED orders (no RouteId) for this supplier's retailers,
	// joined with retailer location and order line items with pallet footprints.
	sql := `SELECT o.OrderId, o.RetailerId, r.Name AS RetailerName, o.Amount,
	               r.ShopLocation,
	               COALESCE(r.ReceivingWindowOpen, '') AS ReceivingWindowOpen,
	               COALESCE(r.ReceivingWindowClose, '') AS ReceivingWindowClose,
	               COALESCE(o.IsRecovery, FALSE) AS IsRecovery,
	               COALESCE(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, (sp.LengthCM * sp.WidthCM * sp.HeightCM / 5000.0), sp.PalletFootprint, 1.0)), 0) AS TotalVolumeVU
	        FROM Orders o
	        JOIN Retailers r ON o.RetailerId = r.RetailerId
	        LEFT JOIN OrderLineItems li ON o.OrderId = li.OrderId
	        LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
	        WHERE o.State IN ('PENDING', 'LOADED', 'READY_FOR_DISPATCH')
	          AND (o.RouteId IS NULL OR o.RouteId = '')
	          AND (o.ManifestId IS NULL OR o.ManifestId = '')
	          AND o.SupplierId = @sid`

	params := map[string]interface{}{"sid": supplierID}

	// Apply warehouse scope if present
	warehouseID := auth.EffectiveWarehouseID(ctx)
	if warehouseID != "" {
		sql += " AND o.WarehouseId = @warehouseId"
		params["warehouseId"] = warehouseID
	}

	if len(filterOrderIDs) > 0 {
		sql += " AND o.OrderId IN UNNEST(@orderIds)"
		params["orderIds"] = filterOrderIDs
	}

	sql += ` GROUP BY o.OrderId, o.RetailerId, r.Name, o.Amount, r.ShopLocation, r.ReceivingWindowOpen, r.ReceivingWindowClose, o.IsRecovery
	         ORDER BY COALESCE(o.DispatchPriority, 0) DESC, TotalVolumeVU DESC`

	readClient := client
	if warehouseID != "" {
		if whLat, whLng, ok := fetchWarehouseOrigin(ctx, client, warehouseID); ok {
			readClient = proximity.ReadClientForRetailer(client, readRouter, whLat, whLng)
		}
	}

	stmt := spanner.Statement{SQL: sql, Params: params}
	// Use explicit staleness snapshot to reduce Spanner read contention during
	// high-volume batch dispatch. 10-second staleness is acceptable for dispatch
	// since orders don't change state faster than the dispatch cycle.
	iter := readClient.Single().WithTimestampBound(spanner.ExactStaleness(10*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var results []dispatchableOrder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query dispatchable orders: %w", err)
		}

		var orderID, retailerID, retailerName string
		var amount spanner.NullInt64
		var shopLocation spanner.NullString
		var rwOpen, rwClose string
		var isRecovery spanner.NullBool
		var totalVU spanner.NullFloat64

		if err := row.Columns(&orderID, &retailerID, &retailerName, &amount, &shopLocation, &rwOpen, &rwClose, &isRecovery, &totalVU); err != nil {
			return nil, fmt.Errorf("parse order row: %w", err)
		}

		lat, lng := parseWKTPoint(shopLocation.StringVal)

		results = append(results, dispatchableOrder{
			OrderID:              orderID,
			RetailerID:           retailerID,
			RetailerName:         retailerName,
			Amount:               amount.Int64,
			Lat:                  lat,
			Lng:                  lng,
			VolumeVU:             totalVU.Float64,
			ReceivingWindowOpen:  rwOpen,
			ReceivingWindowClose: rwClose,
			IsRecovery:           isRecovery.Bool,
		})
	}

	return results, nil
}

// fetchWarehouseOrigin resolves a warehouse's depot coordinates for route
// sequencing. Tries the Redis warehouse-detail hash first (sub-ms), falls back
// to a Spanner point lookup. Returns ok=false when the warehouse has no
// persisted lat/lng — callers should default to a cluster-centroid origin.
func fetchWarehouseOrigin(ctx context.Context, client *spanner.Client, warehouseID string) (lat, lng float64, ok bool) {
	if warehouseID == "" {
		return 0, 0, false
	}
	// Redis-first path: warehouse-detail hash is refreshed on every warehouse
	// write and by the nightly geo-cache cron.
	if detail, err := cache.GetWarehouseDetail(ctx, warehouseID); err == nil && detail != nil {
		if detail.Lat != 0 || detail.Lng != 0 {
			return detail.Lat, detail.Lng, true
		}
	}
	// Spanner fallback — single-row point lookup on the PK.
	row, err := client.Single().ReadRow(ctx, "Warehouses",
		spanner.Key{warehouseID}, []string{"Lat", "Lng"})
	if err != nil {
		return 0, 0, false
	}
	var nLat, nLng spanner.NullFloat64
	if err := row.Columns(&nLat, &nLng); err != nil {
		return 0, 0, false
	}
	if !nLat.Valid || !nLng.Valid {
		return 0, 0, false
	}
	return nLat.Float64, nLng.Float64, true
}

// ── Step 2: Fetch Available Drivers ─────────────────────────────────────────

func fetchAvailableDrivers(ctx context.Context, client *spanner.Client, supplierID string) ([]availableDriver, error) {
	sql := `SELECT d.DriverId, d.Name, COALESCE(d.VehicleType, ''), COALESCE(v.VehicleClass, ''), v.MaxVolumeVU
	        FROM Drivers d
	        JOIN Vehicles v ON d.VehicleId = v.VehicleId
	        WHERE d.SupplierId = @sid
	          AND d.IsActive = true
	          AND (d.IsOffline IS NULL OR d.IsOffline = false)
	          AND COALESCE(d.TruckStatus, 'AVAILABLE') IN ('AVAILABLE', 'RETURNING')
	          AND d.VehicleId IS NOT NULL
	          AND v.IsActive = true`

	params := map[string]interface{}{"sid": supplierID}

	// Apply warehouse scope if present
	if whID := auth.EffectiveWarehouseID(ctx); whID != "" {
		sql += " AND (d.WarehouseId = @warehouseId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @warehouseId))"
		params["warehouseId"] = whID
	}

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	var drivers []availableDriver
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query drivers: %w", err)
		}

		var d availableDriver
		if err := row.Columns(&d.DriverID, &d.Name, &d.VehicleType, &d.VehicleClass, &d.MaxVolumeVU); err != nil {
			return nil, fmt.Errorf("parse driver row: %w", err)
		}
		drivers = append(drivers, d)
	}

	return drivers, nil
}

// ── Waiting Room: orders arriving after the dispatch snapshot ────────────────

// HandleWaitingRoom returns pending orders created after the given snapshot_timestamp.
// The supplier UI polls this to show "New Orders (Waiting Room)" badge.
func HandleWaitingRoom(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		after := r.URL.Query().Get("after")
		if after == "" {
			http.Error(w, `{"error":"after query param required (RFC3339 timestamp)"}`, http.StatusBadRequest)
			return
		}

		ts, err := time.Parse(time.RFC3339, after)
		if err != nil {
			http.Error(w, `{"error":"invalid RFC3339 timestamp"}`, http.StatusBadRequest)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sql := `SELECT o.OrderId, r.Name AS RetailerName, o.Amount, o.CreatedAt
		        FROM Orders o
		        JOIN Retailers r ON o.RetailerId = r.RetailerId
		        WHERE o.State = 'PENDING'
		          AND (o.RouteId IS NULL OR o.RouteId = '')
		          AND o.SupplierId = @sid
		          AND o.CreatedAt > @ts
		        ORDER BY o.CreatedAt ASC`

		params := map[string]interface{}{"ts": ts, "sid": supplierID}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			sql += " AND o.WarehouseId = @warehouseId"
			params["warehouseId"] = whID
		}

		stmt := spanner.Statement{SQL: sql, Params: params}
		iter := spannerx.StaleQuery(ctx, client, stmt)
		defer iter.Stop()

		type WaitingOrder struct {
			OrderID      string `json:"order_id"`
			RetailerName string `json:"retailer_name"`
			Amount       int64  `json:"amount"`
			CreatedAt    string `json:"created_at"`
		}

		var orders []WaitingOrder
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WAITING_ROOM] query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var orderID, retailerName string
			var amount spanner.NullInt64
			var createdAt spanner.NullTime
			if err := row.Columns(&orderID, &retailerName, &amount, &createdAt); err != nil {
				log.Printf("[WAITING_ROOM] row parse error: %v", err)
				continue
			}

			ca := ""
			if createdAt.Valid {
				ca = createdAt.Time.Format(time.RFC3339)
			}
			orders = append(orders, WaitingOrder{
				OrderID:      orderID,
				RetailerName: retailerName,
				Amount:       amount.Int64,
				CreatedAt:    ca,
			})
		}

		if orders == nil {
			orders = []WaitingOrder{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":  len(orders),
			"orders": orders,
		})
	}
}
