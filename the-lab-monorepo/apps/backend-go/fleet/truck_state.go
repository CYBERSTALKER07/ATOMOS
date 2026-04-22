package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/routing"
	"backend-go/telemetry"
	"backend-go/workers"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
)

// Valid TruckStatus values — the two-key handshake state machine:
//
//	AVAILABLE → LOADING → READY (Payloader seals) → IN_TRANSIT (Driver departs)
//	→ (auto-release back to AVAILABLE when manifest empty)
//	Any state except IN_TRANSIT → MAINTENANCE (admin override)
const (
	StatusAvailable   = "AVAILABLE"
	StatusLoading     = "LOADING"
	StatusReady       = "READY"
	StatusInTransit   = "IN_TRANSIT"
	StatusReturning   = "RETURNING"
	StatusMaintenance = "MAINTENANCE"
)

// RetailerPusher is a minimal interface for pushing real-time events to retailer devices.
// Implemented by ws.RetailerHub — defined here to avoid circular imports.
type RetailerPusher interface {
	PushToRetailer(retailerID string, payload interface{}) bool
}

// ── KEY 1: Payloader Seal ───────────────────────────────────────────────────
// POST /v1/fleet/trucks/{truck_id}/seal
// The Payloader closes the doors and seals the truck.
// Transition: AVAILABLE|LOADING → READY. Does NOT dispatch.

func HandleTruckSeal(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse truck_id from path: /v1/fleet/trucks/{truck_id}/seal
		path := strings.TrimPrefix(r.URL.Path, "/v1/fleet/trucks/")
		truckID := strings.TrimSuffix(path, "/seal")
		if truckID == "" || strings.Contains(truckID, "/") {
			http.Error(w, `{"error":"truck_id required in path"}`, http.StatusBadRequest)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{truckID}, []string{"TruckStatus", "IsActive"})
			if err != nil {
				return fmt.Errorf("truck %s not found: %w", truckID, err)
			}

			var currentStatus spanner.NullString
			var isActive spanner.NullBool
			if err := row.Columns(&currentStatus, &isActive); err != nil {
				return err
			}

			status := StatusAvailable
			if currentStatus.Valid {
				status = currentStatus.StringVal
			}

			// Guard: only AVAILABLE or LOADING trucks can be sealed
			if status != StatusAvailable && status != StatusLoading {
				return fmt.Errorf("truck %s cannot be sealed from state %s (must be AVAILABLE or LOADING)", truckID, status)
			}

			mut := spanner.Update("Drivers",
				[]string{"DriverId", "TruckStatus"},
				[]interface{}{truckID, StatusReady},
			)
			return txn.BufferWrite([]*spanner.Mutation{mut})
		})

		if err != nil {
			log.Printf("[FLEET] seal error for truck %s: %v", truckID, err)
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			if strings.Contains(err.Error(), "cannot be sealed") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[FLEET] truck %s sealed → READY (payloader: %s)", truckID, claims.UserID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "READY",
			"truck_id": truckID,
			"message":  "Truck sealed. Waiting for driver to confirm departure.",
		})
	}
}

// ── KEY 2: Driver Depart ────────────────────────────────────────────────────
// POST /v1/fleet/driver/depart
// The Driver confirms departure. Accepts AVAILABLE, LOADING, or READY states,
// bypassing the strict two-key handshake for operational speed.
// Transition: AVAILABLE|LOADING|READY → IN_TRANSIT. Records departure timestamp.

// HandleDriverDepart transitions a truck to IN_TRANSIT and kicks off live ETA
// computation via Google Maps Directions API with traffic awareness.
func HandleDriverDepart(client *spanner.Client, mapsAPIKey string, retailerHub RetailerPusher) http.HandlerFunc {
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

		var req struct {
			TruckID string `json:"truck_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TruckID == "" {
			http.Error(w, `{"error":"truck_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		departedAt := time.Now().UTC()
		var supplierID string
		var capturedRouteID string

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{req.TruckID}, []string{"TruckStatus", "DriverId", "RouteId", "SupplierId"})
			if err != nil {
				return fmt.Errorf("truck %s not found: %w", req.TruckID, err)
			}

			var currentStatus spanner.NullString
			var driverID string
			var routeID spanner.NullString
			var sid spanner.NullString
			if err := row.Columns(&currentStatus, &driverID, &routeID, &sid); err != nil {
				return err
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}
			if routeID.Valid {
				capturedRouteID = routeID.StringVal
			}

			status := StatusAvailable
			if currentStatus.Valid {
				status = currentStatus.StringVal
			}

			// Allow driver to instantly dispatch, bypassing the warehouse seal
			if status != StatusAvailable && status != StatusLoading && status != StatusReady {
				return fmt.Errorf("truck %s is %s — must be AVAILABLE, LOADING, or READY to depart", req.TruckID, status)
			}

			// Guard: JWT user must be the assigned driver
			if claims.UserID != driverID {
				return fmt.Errorf("driver mismatch: JWT user %s is not assigned to truck %s (assigned: %s)", claims.UserID, req.TruckID, driverID)
			}

			mut := spanner.Update("Drivers",
				[]string{"DriverId", "TruckStatus", "DepartedAt"},
				[]interface{}{req.TruckID, StatusInTransit, departedAt},
			)

			if routeID.Valid && routeID.StringVal != "" {
				stmt := spanner.Statement{
					SQL: `UPDATE Orders
					      SET State = 'IN_TRANSIT'
					      WHERE RouteId = @routeId AND State IN ('LOADED', 'DISPATCHED')`,
					Params: map[string]interface{}{"routeId": routeID.StringVal},
				}
				if _, err := txn.Update(ctx, stmt); err != nil {
					return fmt.Errorf("failed to advance route %s orders to IN_TRANSIT: %w", routeID.StringVal, err)
				}
			}

			if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
				return err
			}

			// LEO Phase V — manifest SEALED → DISPATCHED rollup.
			// Driver depart is the canonical "manifest leaves the gate" trigger.
			// Look up the driver's currently SEALED manifest, advance it, and
			// emit MANIFEST_DISPATCHED via outbox atomically with the truck mutation.
			mStmt := spanner.Statement{
				SQL: `SELECT ManifestId, SupplierId, TruckId, StopCount, TotalVolumeVU, MaxVolumeVU
				      FROM SupplierTruckManifests
				      WHERE DriverId = @driverId AND State = 'SEALED'
				      ORDER BY SealedAt DESC LIMIT 1`,
				Params: map[string]interface{}{"driverId": driverID},
			}
			mIter := txn.Query(ctx, mStmt)
			defer mIter.Stop()
			mRow, mErr := mIter.Next()
			if mErr == nil {
				var manifestID, manifestSupplierID, manifestTruckID string
				var stopCount int64
				var totalVU, maxVU float64
				if err := mRow.Columns(&manifestID, &manifestSupplierID, &manifestTruckID, &stopCount, &totalVU, &maxVU); err != nil {
					return err
				}
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("SupplierTruckManifests",
						[]string{"ManifestId", "State", "DispatchedAt", "UpdatedAt"},
						[]interface{}{manifestID, "DISPATCHED", spanner.CommitTimestamp, spanner.CommitTimestamp},
					),
				}); err != nil {
					return err
				}
				if err := outbox.EmitJSON(txn, "Manifest", manifestID,
					kafkaEvents.EventManifestDispatched, kafkaEvents.TopicMain,
					kafkaEvents.ManifestLifecycleEvent{
						ManifestID:  manifestID,
						SupplierId:  manifestSupplierID,
						DriverID:    driverID,
						TruckID:     manifestTruckID,
						State:       "DISPATCHED",
						StopCount:   int(stopCount),
						VolumeVU:    totalVU,
						MaxVolumeVU: maxVU,
						Timestamp:   departedAt,
					}, telemetry.TraceIDFromContext(ctx)); err != nil {
					return err
				}
			}
			// No SEALED manifest is non-fatal — driver may be using legacy
			// non-LEO dispatch (direct truck assignment without a manifest).

			return nil
		})

		if err != nil {
			log.Printf("[FLEET] driver depart error for truck %s: %v", req.TruckID, err)
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			if strings.Contains(err.Error(), "must be AVAILABLE") || strings.Contains(err.Error(), "driver mismatch") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[FLEET] truck %s departed → IN_TRANSIT (driver: %s, departed_at: %s)",
			req.TruckID, claims.UserID, departedAt.Format(time.RFC3339))

		// Push ORDER_STATE_CHANGED (IN_TRANSIT) to supplier admin portal via WebSocket
		if supplierID != "" {
			capturedSupplier := supplierID
			capturedTruck := req.TruckID
			capturedDriver := claims.UserID
			workers.EventPool.Submit(func() {
				telemetry.FleetHub.BroadcastOrderStateChange(capturedSupplier, capturedTruck, "IN_TRANSIT", capturedDriver)
			})
		}

		// Compute live traffic-aware ETAs asynchronously via dedicated ETA pool
		if mapsAPIKey != "" && supplierID != "" {
			capturedSupplier := supplierID
			capturedDriver := claims.UserID
			capturedKey := mapsAPIKey
			capturedDeparted := departedAt
			workers.ETAPool.Submit(func() {
				etaCtx, etaCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer etaCancel()

				// Resolve supplier's warehouse coordinates for depot/return
				depotLat, depotLng := resolveWarehouse(etaCtx, client, capturedSupplier)
				if depotLat == 0 && depotLng == 0 {
					log.Printf("[ETA] No warehouse coordinates for supplier %s — skipping ETA", capturedSupplier)
					return
				}

				if err := routing.ComputeLiveETAs(etaCtx, client, capturedKey, depotLat, depotLng, depotLat, depotLng, capturedDriver, capturedDeparted); err != nil {
					log.Printf("[ETA] Failed to compute live ETAs for driver %s: %v", capturedDriver, err)
					return
				}

				// Push ETA_UPDATED to supplier via WebSocket
				telemetry.FleetHub.BroadcastETAUpdate(capturedSupplier, capturedDriver)
			})
		}

		// Push ORDER_STATUS_CHANGED (IN_TRANSIT) to ALL retailers on this route via WebSocket
		if retailerHub != nil && capturedRouteID != "" {
			capturedRoute := capturedRouteID
			workers.EventPool.Submit(func() {
				pushCtx, pushCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer pushCancel()
				stmt := spanner.Statement{
					SQL:    `SELECT OrderId, RetailerId FROM Orders WHERE RouteId = @routeId AND State = 'IN_TRANSIT'`,
					Params: map[string]interface{}{"routeId": capturedRoute},
				}
				iter := client.Single().Query(pushCtx, stmt)
				defer iter.Stop()
				ts := time.Now().UTC().Format(time.RFC3339)
				for {
					row, err := iter.Next()
					if err != nil {
						break
					}
					var orderID, retailerID string
					if err := row.Columns(&orderID, &retailerID); err == nil && retailerID != "" {
						retailerHub.PushToRetailer(retailerID, map[string]interface{}{
							"type":      ws.EventOrderStatusChanged,
							"order_id":  orderID,
							"state":     "IN_TRANSIT",
							"timestamp": ts,
						})
					}
				}
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "IN_TRANSIT",
			"truck_id":    req.TruckID,
			"departed_at": departedAt.Format(time.RFC3339),
			"message":     "Truck departed. Route active.",
		})
	}
}

// resolveWarehouse looks up a supplier's warehouse coordinates from Spanner.
func resolveWarehouse(ctx context.Context, client *spanner.Client, supplierID string) (lat, lng float64) {
	row, err := client.Single().ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"WarehouseLat", "WarehouseLng"})
	if err != nil {
		return 0, 0
	}
	var wLat, wLng spanner.NullFloat64
	if err := row.Columns(&wLat, &wLng); err != nil {
		return 0, 0
	}
	if wLat.Valid && wLng.Valid {
		return wLat.Float64, wLng.Float64
	}
	return 0, 0
}

// ── Driver Auto-Release ─────────────────────────────────────────────────────
// CheckAndAutoReleaseTruck queries whether a truck's route is fully delivered.
// If all orders on the route are COMPLETED or CANCELLED, the truck
// transitions to RETURNING (not directly to AVAILABLE) so the supplier can
// track the driver's return to the warehouse with an ETA.
// is automatically released back to AVAILABLE.
//
// Called after every delivery completion (SubmitDelivery, CompleteDeliveryWithToken).

func CheckAndAutoReleaseTruck(ctx context.Context, client *spanner.Client, orderID, mapsAPIKey string) {
	// Step 1: Find the RouteId, DriverId, and SupplierId for this order
	row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"RouteId", "DriverId", "SupplierId"})
	if err != nil {
		log.Printf("[FLEET-AUTORELEASE] cannot read order %s: %v", orderID, err)
		return
	}

	var routeID, driverID, supplierID spanner.NullString
	if err := row.Columns(&routeID, &driverID, &supplierID); err != nil {
		log.Printf("[FLEET-AUTORELEASE] parse error for order %s: %v", orderID, err)
		return
	}

	// No route assigned — nothing to release
	if !routeID.Valid || routeID.StringVal == "" {
		return
	}
	// No driver assigned — skip
	if !driverID.Valid || driverID.StringVal == "" {
		return
	}

	// Step 2: Count remaining undelivered orders on this route
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) AS remaining
		      FROM Orders
		      WHERE RouteId = @routeId
		        AND State NOT IN ('COMPLETED', 'CANCELLED')`,
		Params: map[string]interface{}{"routeId": routeID.StringVal},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	countRow, err := iter.Next()
	if err != nil {
		log.Printf("[FLEET-AUTORELEASE] count query failed for route %s: %v", routeID.StringVal, err)
		return
	}

	var remaining int64
	if err := countRow.Columns(&remaining); err != nil {
		log.Printf("[FLEET-AUTORELEASE] count parse failed for route %s: %v", routeID.StringVal, err)
		return
	}

	if remaining > 0 {
		log.Printf("[FLEET-AUTORELEASE] route %s still has %d undelivered orders", routeID.StringVal, remaining)
		return
	}

	// Step 3: Manifest is empty — transition to RETURNING (not AVAILABLE)
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.Update("Drivers",
			[]string{"DriverId", "TruckStatus"},
			[]interface{}{driverID.StringVal, StatusReturning},
		)
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})

	if err != nil {
		log.Printf("[FLEET-AUTORELEASE] failed to transition truck %s to RETURNING: %v", driverID.StringVal, err)
		return
	}

	log.Printf("[FLEET-AUTORELEASE] truck %s → RETURNING (route %s fully delivered)", driverID.StringVal, routeID.StringVal)

	// Broadcast status change to supplier
	if supplierID.Valid && supplierID.StringVal != "" {
		capturedSID := supplierID.StringVal
		capturedDID := driverID.StringVal
		workers.EventPool.Submit(func() {
			telemetry.FleetHub.BroadcastOrderStateChange(capturedSID, capturedDID, "RETURNING", capturedDID)
		})
	}

	// Compute return-to-warehouse ETA asynchronously
	if mapsAPIKey != "" && supplierID.Valid && supplierID.StringVal != "" {
		capturedSID := supplierID.StringVal
		capturedDID := driverID.StringVal
		workers.ETAPool.Submit(func() {
			etaCtx, etaCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer etaCancel()

			depotLat, depotLng := resolveWarehouse(etaCtx, client, capturedSID)
			if depotLat == 0 && depotLng == 0 {
				return
			}

			// Use driver's last known position (from CurrentLocation column)
			driverLat, driverLng := resolveDriverPosition(etaCtx, client, capturedDID)
			if driverLat == 0 && driverLng == 0 {
				return
			}

			if _, _, err := routing.ComputeReturnETA(etaCtx, client, mapsAPIKey, driverLat, driverLng, depotLat, depotLng, capturedDID); err != nil {
				log.Printf("[ETA] Failed to compute return ETA for driver %s: %v", capturedDID, err)
				return
			}

			telemetry.FleetHub.BroadcastETAUpdate(capturedSID, capturedDID)
		})
	}
}

// resolveDriverPosition reads the driver's last known GPS coordinates from CurrentLocation.
func resolveDriverPosition(ctx context.Context, client *spanner.Client, driverID string) (lat, lng float64) {
	row, err := client.Single().ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{"CurrentLocation"})
	if err != nil {
		return 0, 0
	}
	var loc spanner.NullString
	if err := row.Columns(&loc); err != nil || !loc.Valid {
		return 0, 0
	}
	// CurrentLocation stored as "lat,lng" string
	var la, lo float64
	if _, err := fmt.Sscanf(loc.StringVal, "%f,%f", &la, &lo); err != nil {
		return 0, 0
	}
	return la, lo
}

// ── Return Complete ─────────────────────────────────────────────────────────
// POST /v1/fleet/driver/return-complete
// Driver confirms arrival at the warehouse. Transitions RETURNING → AVAILABLE.

func HandleReturnComplete(client *spanner.Client) http.HandlerFunc {
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

		var req struct {
			TruckID string `json:"truck_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TruckID == "" {
			http.Error(w, `{"error":"truck_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var supplierID string

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{req.TruckID}, []string{"TruckStatus", "DriverId", "SupplierId"})
			if err != nil {
				return fmt.Errorf("truck %s not found: %w", req.TruckID, err)
			}

			var currentStatus spanner.NullString
			var driverID string
			var sid spanner.NullString
			if err := row.Columns(&currentStatus, &driverID, &sid); err != nil {
				return err
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}

			status := StatusAvailable
			if currentStatus.Valid {
				status = currentStatus.StringVal
			}

			if status != StatusReturning {
				return fmt.Errorf("truck %s is %s — must be RETURNING to complete return", req.TruckID, status)
			}

			if claims.UserID != driverID {
				return fmt.Errorf("driver mismatch: JWT user %s is not assigned to truck %s", claims.UserID, req.TruckID)
			}

			// Clear ETA fields and transition to AVAILABLE
			mut := spanner.Update("Drivers",
				[]string{"DriverId", "TruckStatus", "EstimatedReturnAt", "ReturnDurationSec", "DepartedAt"},
				[]interface{}{req.TruckID, StatusAvailable, nil, nil, nil},
			)
			return txn.BufferWrite([]*spanner.Mutation{mut})
		})

		if err != nil {
			log.Printf("[FLEET] return-complete error for truck %s: %v", req.TruckID, err)
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			if strings.Contains(err.Error(), "must be RETURNING") || strings.Contains(err.Error(), "driver mismatch") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[FLEET] truck %s return complete → AVAILABLE (driver: %s)", req.TruckID, claims.UserID)

		if supplierID != "" {
			go telemetry.FleetHub.BroadcastOrderStateChange(supplierID, req.TruckID, "AVAILABLE", claims.UserID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "AVAILABLE",
			"truck_id": req.TruckID,
			"message":  "Driver returned to warehouse. Truck available.",
		})
	}
}

// ── Truck Status Query ──────────────────────────────────────────────────────
// GET /v1/fleet/trucks/{truck_id}/status

func HandleTruckStatus(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/v1/fleet/trucks/")
		truckID := strings.TrimSuffix(path, "/status")
		if truckID == "" || strings.Contains(truckID, "/") {
			http.Error(w, `{"error":"truck_id required in path"}`, http.StatusBadRequest)
			return
		}

		row, err := client.Single().ReadRow(r.Context(), "Drivers", spanner.Key{truckID},
			[]string{"DriverId", "Name", "VehicleType", "TruckStatus", "IsActive"})
		if err != nil {
			http.Error(w, `{"error":"truck not found"}`, http.StatusNotFound)
			return
		}

		var driverID, name string
		var vehicleType, truckStatus spanner.NullString
		var isActive spanner.NullBool
		if err := row.Columns(&driverID, &name, &vehicleType, &truckStatus, &isActive); err != nil {
			http.Error(w, `{"error":"parse error"}`, http.StatusInternalServerError)
			return
		}

		status := StatusAvailable
		if truckStatus.Valid {
			status = truckStatus.StringVal
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"truck_id":     driverID,
			"driver_name":  name,
			"vehicle_type": vehicleType.StringVal,
			"status":       status,
			"is_active":    isActive.Valid && isActive.Bool,
		})
	}
}

// ── Hard Exclusion: Admin Status Override ────────────────────────────────────
// PATCH /v1/fleet/trucks/{truck_id}/status
// Allows Admin to toggle a truck between AVAILABLE and MAINTENANCE.
// A truck in MAINTENANCE is permanently invisible to the Auto-Dispatcher
// until an Admin switches it back.

func HandleTruckStatusUpdate(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/v1/fleet/trucks/")
		truckID := strings.TrimSuffix(path, "/status")
		if truckID == "" || strings.Contains(truckID, "/") {
			http.Error(w, `{"error":"truck_id required in path"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		// Only allow explicit state transitions the Admin should control
		allowed := map[string]bool{
			StatusAvailable:   true,
			StatusMaintenance: true,
			StatusLoading:     true,
			StatusReady:       true,
			StatusReturning:   true,
		}
		if !allowed[req.Status] {
			http.Error(w, `{"error":"status must be AVAILABLE, LOADING, READY, RETURNING, or MAINTENANCE"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Verify truck exists
			row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{truckID}, []string{"TruckStatus"})
			if err != nil {
				return fmt.Errorf("truck %s not found: %w", truckID, err)
			}

			var currentStatus spanner.NullString
			if err := row.Columns(&currentStatus); err != nil {
				return err
			}

			current := StatusAvailable
			if currentStatus.Valid {
				current = currentStatus.StringVal
			}

			// Guard: cannot override a truck that's physically on the road
			if current == StatusInTransit || current == StatusReturning {
				return fmt.Errorf("truck %s is %s — cannot override until route completes", truckID, current)
			}

			mut := spanner.Update("Drivers",
				[]string{"DriverId", "TruckStatus"},
				[]interface{}{truckID, req.Status},
			)
			return txn.BufferWrite([]*spanner.Mutation{mut})
		})

		if err != nil {
			log.Printf("[FLEET] status update error for truck %s: %v", truckID, err)
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			if strings.Contains(err.Error(), "IN_TRANSIT") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[FLEET] truck %s status → %s (admin override)", truckID, req.Status)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"truck_id": truckID,
			"status":   req.Status,
			"message":  "Truck status updated.",
		})
	}
}
