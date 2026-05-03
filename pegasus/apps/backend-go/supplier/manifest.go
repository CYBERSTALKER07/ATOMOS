package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/routing"
	"backend-go/telemetry"
	"backend-go/workers"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// LEO — LOGISTICS EXECUTION ORCHESTRATOR
// Supplier Truck Manifest state machine: DRAFT → LOADING → SEALED → DISPATCHED → COMPLETED
//
// Phase I (DRAFT):   Planning only — routes are simulated. Not visible to drivers.
// Phase II (LOADING): Payloader selects truck. Manifest is MUTABLE (iPad overrides accepted).
// Phase III (SEALED): Locked. JIT route materialized. Driver receives final route.
// ═══════════════════════════════════════════════════════════════════════════════

// TetrisBuffer is re-exported from dispatch/ via dispatch_shim.go so the
// volumetric safety margin has a single authoritative value.

// DLQThreshold is the number of OVERFLOW exceptions before admin escalation.
const DLQThreshold = 3

// ── Manifest Service ────────────────────────────────────────────────────────

type ManifestService struct {
	Spanner *spanner.Client
	Cache   *cache.Cache

	MapsAPIKey    string // Google Maps Directions API key for JIT route optimization at seal
	DepotLocation string // fallback "lat,lng" when warehouse depot is unresolved
}

func emitPayloadSyncEvent(txn *spanner.ReadWriteTransaction, supplierID, warehouseID, manifestID, reason, traceID string, timestamp time.Time) error {
	if supplierID == "" || manifestID == "" {
		return nil
	}
	return outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventPayloadSync, kafkaEvents.TopicMain,
		kafkaEvents.PayloadSyncEvent{
			SupplierID:  supplierID,
			WarehouseID: warehouseID,
			ManifestID:  manifestID,
			Reason:      reason,
			Timestamp:   timestamp,
		}, traceID)
}

// ── CreateDraftManifest ─────────────────────────────────────────────────────
// Called internally by the auto-dispatcher after bin-packing.
// Creates a DRAFT manifest + ManifestOrders rows. Orders stay PENDING.
func (s *ManifestService) CreateDraftManifest(
	ctx context.Context,
	supplierID string,
	warehouseID string,
	routeID string,
	truckID string,
	driverID string,
	maxVolumeVU float64,
	regionCode string,
	orders []DispatchOrder,
	loadingManifest []LoadingManifestEntry,
) (string, error) {
	manifestID := uuid.New().String()
	now := time.Now().UTC()

	effectiveMax := maxVolumeVU * TetrisBuffer
	totalVol := 0.0
	for _, o := range orders {
		totalVol += o.VolumeVU
	}

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation

		// Insert manifest in DRAFT state
		mutations = append(mutations, spanner.Insert("SupplierTruckManifests",
			[]string{
				"ManifestId", "SupplierId", "WarehouseId", "RouteId",
				"TruckId", "DriverId", "State",
				"TotalVolumeVU", "MaxVolumeVU", "StopCount", "RegionCode",
				"CreatedAt", "UpdatedAt",
			},
			[]interface{}{
				manifestID, supplierID, nullStr(warehouseID), routeID,
				truckID, driverID, "DRAFT",
				totalVol, effectiveMax, int64(len(orders)), nullStr(regionCode),
				spanner.CommitTimestamp, spanner.CommitTimestamp,
			},
		))

		// Insert ManifestOrders with LIFO loading order
		for _, o := range orders {
			loadSeq := 0
			for _, le := range loadingManifest {
				if le.OrderID == o.OrderID {
					loadSeq = le.LoadSequence
					break
				}
			}
			mutations = append(mutations, spanner.Insert("ManifestOrders",
				[]string{"ManifestId", "OrderId", "SequenceIndex", "LoadingOrder", "VolumeVU", "State"},
				[]interface{}{manifestID, o.OrderID, int64(loadSeq), int64(loadSeq), o.VolumeVU, "ASSIGNED"},
			))

			// Link order to manifest (but keep PENDING state — no LOADED until payloader starts)
			mutations = append(mutations, spanner.Update("Orders",
				[]string{"OrderId", "ManifestId", "RouteId"},
				[]interface{}{o.OrderID, manifestID, routeID},
			))
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Emit ROUTE_CREATED via outbox (atomic with manifest row)
		if err := outbox.EmitJSON(txn, "Route", routeID, kafkaEvents.EventRouteCreated, kafkaEvents.TopicMain, kafkaEvents.RouteCreatedEvent{
			RouteID:     routeID,
			ManifestID:  manifestID,
			DriverID:    driverID,
			TruckID:     truckID,
			SupplierID:  supplierID,
			WarehouseID: warehouseID,
			StopCount:   len(orders),
			VolumeVU:    totalVol,
			Timestamp:   now,
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("emit route created: %w", err)
		}

		// Emit ORDER_ASSIGNED per order via outbox
		for _, o := range orders {
			if err := outbox.EmitJSON(txn, "Order", o.OrderID, kafkaEvents.EventOrderAssigned, kafkaEvents.TopicMain, kafkaEvents.OrderAssignedEvent{
				OrderID:     o.OrderID,
				RouteID:     routeID,
				DriverID:    driverID,
				SupplierID:  supplierID,
				WarehouseID: warehouseID,
				Timestamp:   now,
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
				return fmt.Errorf("emit order assigned %s: %w", o.OrderID, err)
			}
		}

		// Emit FLEET_DISPATCHED for downstream notification fan-out
		// (driver assignment, retailer ETA, supplier dashboard). Atomic with
		// the manifest mutation — if the txn aborts, no false dispatch signal.
		orderIDs := make([]string, 0, len(orders))
		for _, o := range orders {
			orderIDs = append(orderIDs, o.OrderID)
		}
		if err := outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventFleetDispatched, kafkaEvents.TopicMain, kafkaEvents.FleetDispatchedEvent{
			RouteID:     routeID,
			ManifestID:  manifestID,
			OrderIDs:    orderIDs,
			DriverID:    driverID,
			SupplierID:  supplierID,
			WarehouseId: warehouseID,
			GeoZone:     regionCode,
			Timestamp:   now,
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("emit fleet dispatched: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("create draft manifest: %w", err)
	}

	log.Printf("[LEO] DRAFT manifest %s created: %d orders, %.1f/%.1f VU, driver=%s",
		manifestID[:8], len(orders), totalVol, effectiveMax, driverID[:8])

	return manifestID, nil
}

// ── HandleStartLoading ──────────────────────────────────────────────────────
// POST /v1/supplier/manifests/{id}/start-loading
// Payloader selects truck on iPad → DRAFT → LOADING transition.
// Orders transition from PENDING → LOADED.
func (s *ManifestService) HandleStartLoading() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		manifestID := ExtractPathParam(r.URL.Path, "manifests")
		if manifestID == "" {
			http.Error(w, `{"error":"manifest_id required"}`, http.StatusBadRequest)
			return
		}
		_ = r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)

		now := time.Now().UTC()

		_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read manifest + verify state
			row, err := txn.ReadRow(ctx, "SupplierTruckManifests",
				spanner.Key{manifestID},
				[]string{"State", "SupplierId"})
			if err != nil {
				return fmt.Errorf("manifest not found: %w", err)
			}
			var state, supplierID string
			if err := row.Columns(&state, &supplierID); err != nil {
				return err
			}
			if state != "DRAFT" {
				return fmt.Errorf("INVALID_STATE: manifest is %s, expected DRAFT", state)
			}

			// Transition manifest DRAFT → LOADING
			mutations := []*spanner.Mutation{
				spanner.Update("SupplierTruckManifests",
					[]string{"ManifestId", "State", "UpdatedAt"},
					[]interface{}{manifestID, "LOADING", spanner.CommitTimestamp}),
			}

			// Transition all ASSIGNED orders from PENDING → LOADED
			orderIter := txn.Read(ctx, "ManifestOrders",
				spanner.Key{manifestID}.AsPrefix(),
				[]string{"OrderId", "State"})
			defer orderIter.Stop()
			for {
				oRow, err := orderIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				var orderID, moState string
				if err := oRow.Columns(&orderID, &moState); err != nil {
					return err
				}
				if moState == "ASSIGNED" {
					mutations = append(mutations, spanner.Update("Orders",
						[]string{"OrderId", "State"},
						[]interface{}{orderID, "LOADED"}))
				}
			}

			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}

			traceID := telemetry.TraceIDFromContext(ctx)

			// MANIFEST_LOADING_STARTED — atomic with the DRAFT→LOADING transition.
			if err := outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventManifestLoadingStarted, kafkaEvents.TopicMain,
				kafkaEvents.ManifestLifecycleEvent{
					ManifestID: manifestID,
					SupplierId: supplierID,
					State:      "LOADING",
					Timestamp:  now,
				}, traceID); err != nil {
				return err
			}

			return emitPayloadSyncEvent(txn, supplierID, "", manifestID, kafkaEvents.EventManifestLoadingStarted, traceID, now)
		})

		if err != nil {
			if isInvalidState(err) {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			log.Printf("[LEO] start-loading error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if s.Cache != nil {
			s.Cache.Invalidate(r.Context(),
				cache.PrefixManifestDetail+manifestID,
				cache.PrefixManifestOrders+manifestID,
			)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"manifest_id": manifestID,
			"state":       "LOADING",
			"message":     "Loading phase started. Manifest is mutable — accept iPad overrides.",
		})
	}
}

// ── HandleInjectOrder ───────────────────────────────────────────────────────
// POST /v1/supplier/manifests/{id}/inject-order
// Mid-Load Addition: admin or payloader adds an order to a LOADING manifest.
// Guards: manifest must be LOADING, order must be PENDING with no ManifestId,
// and currentVolume + order.TotalVolumeVU ≤ manifest.MaxVolumeVU.
// Route recalculation is deferred to seal time — driver sees nothing mid-load.
func (s *ManifestService) HandleInjectOrder() http.HandlerFunc {
	type injectReq struct {
		OrderID string `json:"order_id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		manifestID := ExtractPathParam(r.URL.Path, "manifests")
		if manifestID == "" {
			http.Error(w, `{"error":"manifest_id required"}`, http.StatusBadRequest)
			return
		}
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)

		var req injectReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		var newTotalVol float64
		var newStopCount int64

		_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// 1. Verify manifest is LOADING
			mRow, err := txn.ReadRow(ctx, "SupplierTruckManifests",
				spanner.Key{manifestID},
				[]string{"State", "SupplierId", "TotalVolumeVU", "MaxVolumeVU", "StopCount"})
			if err != nil {
				return fmt.Errorf("manifest not found: %w", err)
			}
			var state, supplierID string
			var totalVol, maxVol float64
			var stopCount int64
			if err := mRow.Columns(&state, &supplierID, &totalVol, &maxVol, &stopCount); err != nil {
				return err
			}
			if state != "LOADING" {
				return fmt.Errorf("INVALID_STATE: manifest is %s, expected LOADING for injection", state)
			}

			// 2. Verify order is PENDING with no manifest assignment
			oRow, err := txn.ReadRow(ctx, "Orders",
				spanner.Key{req.OrderID},
				[]string{"State", "ManifestId", "TotalVolumeVU", "SupplierId"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var orderState string
			var orderManifest spanner.NullString
			var orderVol float64
			var orderSupplier string
			if err := oRow.Columns(&orderState, &orderManifest, &orderVol, &orderSupplier); err != nil {
				return err
			}
			if orderState != "PENDING" {
				return fmt.Errorf("INVALID_STATE: order is %s, expected PENDING for injection", orderState)
			}
			if orderManifest.Valid && orderManifest.StringVal != "" {
				return fmt.Errorf("INVALID_STATE: order already assigned to manifest %s", orderManifest.StringVal)
			}
			if orderSupplier != supplierID {
				return fmt.Errorf("INVALID_STATE: order belongs to different supplier")
			}

			// 3. Volumetric guard: ensure order fits
			if totalVol+orderVol > maxVol {
				return fmt.Errorf("VOLUME_CONFLICT: injection would exceed capacity (%.1f + %.1f > %.1f VU)",
					totalVol, orderVol, maxVol)
			}

			newTotalVol = totalVol + orderVol
			newStopCount = stopCount + 1

			// 4. Insert ManifestOrder + link order + update manifest totals
			mutations := []*spanner.Mutation{
				spanner.Insert("ManifestOrders",
					[]string{"ManifestId", "OrderId", "SequenceIndex", "LoadingOrder", "VolumeVU", "State"},
					[]interface{}{manifestID, req.OrderID, newStopCount, newStopCount, orderVol, "ASSIGNED"}),

				spanner.Update("Orders",
					[]string{"OrderId", "ManifestId", "State"},
					[]interface{}{req.OrderID, manifestID, "LOADED"}),

				spanner.Update("SupplierTruckManifests",
					[]string{"ManifestId", "TotalVolumeVU", "StopCount", "UpdatedAt"},
					[]interface{}{manifestID, newTotalVol, newStopCount, spanner.CommitTimestamp}),
			}

			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}

			traceID := telemetry.TraceIDFromContext(ctx)

			// MANIFEST_ORDER_INJECTED — atomic with the inject mutations.
			if err := outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventManifestOrderInjected, kafkaEvents.TopicMain,
				kafkaEvents.ManifestOrderInjectedEvent{
					ManifestID:       manifestID,
					OrderID:          req.OrderID,
					SupplierId:       supplierID,
					NewTotalVolumeVU: newTotalVol,
					InjectedBy:       claims.ResolveSupplierID(),
					Timestamp:        time.Now().UTC(),
				}, traceID); err != nil {
				return err
			}

			return emitPayloadSyncEvent(txn, supplierID, "", manifestID, kafkaEvents.EventManifestOrderInjected, traceID, time.Now().UTC())
		})

		if err != nil {
			if isInvalidState(err) || isVolumeConflict(err) {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			log.Printf("[LEO] inject-order error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if s.Cache != nil {
			s.Cache.Invalidate(r.Context(),
				cache.PrefixManifestDetail+manifestID,
				cache.PrefixManifestOrders+manifestID,
			)
		}

		log.Printf("[LEO] INJECTED order %s into manifest %s: %.1f VU, %d stops",
			req.OrderID[:min(8, len(req.OrderID))], manifestID[:min(8, len(manifestID))], newTotalVol, newStopCount)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"manifest_id":     manifestID,
			"order_id":        req.OrderID,
			"total_volume_vu": newTotalVol,
			"max_volume_vu":   0, // filled by caller if needed
			"stop_count":      newStopCount,
			"message":         "Order injected into active loading session. Route recalc deferred to seal.",
		})
	}
}

// ── HandleSealManifest ──────────────────────────────────────────────────────
// POST /v1/supplier/manifests/{id}/seal
// Payloader performs "Slide to Seal" → volumetric validation → LOADING → SEALED.
// Triggers JIT route optimization and pushes ROUTE_FINALIZED to driver.
func (s *ManifestService) HandleSealManifest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		manifestID := ExtractPathParam(r.URL.Path, "manifests")
		if manifestID == "" {
			http.Error(w, `{"error":"manifest_id required"}`, http.StatusBadRequest)
			return
		}
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		sealedBy := claims.ResolveSupplierID()

		// Phase C: Admin force-seal override — ?override=admin bypasses volumetric validation.
		// Only SUPPLIER (admin) role can invoke force-seal. Payloaders cannot bypass.
		forceOverride := r.URL.Query().Get("override") == "admin"
		overrideReason := r.URL.Query().Get("reason")
		var overrideCount int64
		if forceOverride {
			if claims.Role != "ADMIN" && claims.Role != "SUPPLIER" {
				http.Error(w, `{"error":"only admin/supplier can force-seal"}`, http.StatusForbidden)
				return
			}
			// SOVEREIGN ACTION: Force-seal override requires GLOBAL_ADMIN
			if err := auth.RequireGlobalAdmin(w, claims); err != nil {
				return
			}

			// ── Force-Seal Rate Limiter: 5 per 24h per supplier ──────────────
			// Count overrides in the last 24 hours from SupplierOverrides table.
			// At ≥3/5: emit Kafka AlertEvent (Critical Usage warning) via outbox below
			// At >5/5: hard stop — return 429 Too Many Requests
			overrideCountStmt := spanner.Statement{
				SQL: `SELECT COUNT(*) AS cnt FROM SupplierOverrides
				      WHERE SupplierId = @supplierID
				        AND OverrideType = 'FORCE_SEAL'
				        AND Timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)`,
				Params: map[string]interface{}{"supplierID": claims.ResolveSupplierID()},
			}
			countIter := s.Spanner.Single().Query(r.Context(), overrideCountStmt)
			countRow, countErr := countIter.Next()
			if countErr == nil {
				countRow.Columns(&overrideCount)
			}
			countIter.Stop()
		}

		// Force-seal alert flag — emitted via outbox INSIDE the seal txn below
		// so the alert is atomic with the actual seal commit. If the seal aborts,
		// the alert is rolled back automatically.
		var emitForceSealAlert bool
		var forceSealCount int64
		if forceOverride {
			if overrideCount > 5 {
				http.Error(w, `{"error":"force_seal_rate_limited","message":"Override quota exhausted (5/24h). Bay locked until window slides or Global Admin resets."}`, http.StatusTooManyRequests)
				return
			}
			if overrideCount >= 3 {
				emitForceSealAlert = true
				forceSealCount = overrideCount + 1
				log.Printf("[LEO] FORCE-SEAL WARNING: supplier %s at %d/5 overrides in 24h", claims.ResolveSupplierID()[:min(8, len(claims.ResolveSupplierID()))], overrideCount+1)
			}
		}

		type sealResult struct {
			supplierID  string
			driverID    string
			truckID     string
			routeID     string
			stopCount   int
			totalVolume float64
			maxVolume   float64
			warehouseID string
		}
		var result sealResult

		_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read manifest
			row, err := txn.ReadRow(ctx, "SupplierTruckManifests",
				spanner.Key{manifestID},
				[]string{"State", "SupplierId", "DriverId", "TruckId", "RouteId", "MaxVolumeVU", "WarehouseId"})
			if err != nil {
				return fmt.Errorf("manifest not found: %w", err)
			}
			var state, supplierID, driverID, truckID string
			var routeID, warehouseID spanner.NullString
			var maxVol float64
			if err := row.Columns(&state, &supplierID, &driverID, &truckID, &routeID, &maxVol, &warehouseID); err != nil {
				return err
			}
			if state != "LOADING" {
				return fmt.Errorf("INVALID_STATE: manifest is %s, expected LOADING", state)
			}

			// Calculate actual volume from ASSIGNED orders only
			orderIter := txn.Read(ctx, "ManifestOrders",
				spanner.Key{manifestID}.AsPrefix(),
				[]string{"OrderId", "VolumeVU", "State"})
			defer orderIter.Stop()

			var actualVolume float64
			var assignedCount int
			for {
				oRow, err := orderIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				var oid, moState string
				var vol float64
				if err := oRow.Columns(&oid, &vol, &moState); err != nil {
					return err
				}
				if moState == "ASSIGNED" {
					actualVolume += vol
					assignedCount++
				}
			}

			// ── THE ZERO MISCALCULATION RULE ────────────────────────────
			// Volumetric validation: reject seal if volume exceeds capacity.
			// Phase C: Admin force-seal bypasses this check — audit event emitted instead.
			if actualVolume > maxVol && !forceOverride {
				return fmt.Errorf("VOLUME_CONFLICT: actual %.1f VU exceeds max %.1f VU (Tetris Buffer applied)", actualVolume, maxVol)
			}

			// Transition LOADING → SEALED
			mutations := []*spanner.Mutation{
				spanner.Update("SupplierTruckManifests",
					[]string{"ManifestId", "State", "TotalVolumeVU", "StopCount",
						"SealedAt", "SealedBy", "UpdatedAt"},
					[]interface{}{manifestID, "SEALED", actualVolume, int64(assignedCount),
						spanner.CommitTimestamp, sealedBy, spanner.CommitTimestamp}),
			}

			// Mark all ASSIGNED ManifestOrders as SEALED
			// Transition orders from LOADED → DISPATCHED (sealed = ready for driver)
			orderIter2 := txn.Read(ctx, "ManifestOrders",
				spanner.Key{manifestID}.AsPrefix(),
				[]string{"OrderId", "State"})
			defer orderIter2.Stop()
			for {
				oRow, err := orderIter2.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				var oid, moState string
				if err := oRow.Columns(&oid, &moState); err != nil {
					return err
				}
				if moState == "ASSIGNED" {
					mutations = append(mutations,
						spanner.Update("ManifestOrders",
							[]string{"ManifestId", "OrderId", "State"},
							[]interface{}{manifestID, oid, "SEALED"}),
						spanner.Update("Orders",
							[]string{"OrderId", "State"},
							[]interface{}{oid, "DISPATCHED"}),
					)
				}
			}

			result = sealResult{
				supplierID:  supplierID,
				driverID:    driverID,
				truckID:     truckID,
				routeID:     routeID.StringVal,
				stopCount:   assignedCount,
				totalVolume: actualVolume,
				maxVolume:   maxVol,
				warehouseID: warehouseID.StringVal,
			}

			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}

			now := time.Now().UTC()

			traceID := telemetry.TraceIDFromContext(ctx)

			// MANIFEST_SEALED — always emitted via outbox, atomic with the seal commit.
			if err := outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventManifestSealed, kafkaEvents.TopicMain,
				kafkaEvents.ManifestLifecycleEvent{
					ManifestID:  manifestID,
					SupplierId:  supplierID,
					DriverID:    driverID,
					TruckID:     truckID,
					State:       "SEALED",
					StopCount:   int(assignedCount),
					VolumeVU:    actualVolume,
					MaxVolumeVU: maxVol,
					SealedBy:    sealedBy,
					Timestamp:   now,
				}, traceID); err != nil {
				return err
			}

			if err := emitPayloadSyncEvent(txn, supplierID, warehouseID.StringVal, manifestID, kafkaEvents.EventManifestSealed, traceID, now); err != nil {
				return err
			}

			// MANIFEST_FORCE_SEALED — emitted only when admin override was used.
			// Audit-trail event distinct from the threshold-cross FORCE_SEAL_ALERT below.
			if forceOverride {
				if err := outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventManifestForceSeal, kafkaEvents.TopicMain,
					kafkaEvents.ManifestForceSealEvent{
						ManifestID:   manifestID,
						SupplierId:   supplierID,
						SealedBy:     sealedBy,
						Override:     true,
						VolumeAtSeal: actualVolume,
						MaxVolumeVU:  maxVol,
						Reason:       overrideReason,
						Timestamp:    now,
					}, telemetry.TraceIDFromContext(ctx)); err != nil {
					return err
				}
			}

			// FORCE_SEAL_ALERT — emitted via outbox INSIDE the seal txn so the
			// alert is atomic with the actual seal commit. Only fires when the
			// supplier crosses the 3/5 force-seal threshold in 24h.
			if emitForceSealAlert {
				return outbox.EmitJSON(txn, "Manifest", manifestID, kafkaEvents.EventForceSealAlert, kafkaEvents.TopicMain,
					kafkaEvents.ForceSealAlertEvent{
						SupplierID:  supplierID,
						WarehouseID: warehouseID.StringVal,
						ManifestID:  manifestID,
						Count24h:    forceSealCount,
						Quota:       5,
						SealedBy:    sealedBy,
						Timestamp:   now,
					}, telemetry.TraceIDFromContext(ctx))
			}
			return nil
		})

		if err != nil {
			if isInvalidState(err) || isVolumeConflict(err) {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			log.Printf("[LEO] seal error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		now := time.Now().UTC()

		// Phase C: Audit-ledger row for force-seal (rate-limit backing store).
		// The MANIFEST_FORCE_SEALED event is now emitted via outbox INSIDE the
		// seal txn (atomic with the commit); this Apply only records the audit
		// ledger entry that backs the 3/5-in-24h quota check.
		if forceOverride {
			log.Printf("[LEO] FORCE-SEALED manifest %s by admin %s (%.1f/%.1f VU, reason: %s)",
				manifestID[:min(8, len(manifestID))], sealedBy[:min(8, len(sealedBy))],
				result.totalVolume, result.maxVolume, overrideReason)

			overrideID := uuid.New().String()
			_, _ = s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Insert("SupplierOverrides",
						[]string{"SupplierId", "OverrideId", "OverrideType", "Timestamp", "Reason"},
						[]interface{}{result.supplierID, overrideID, "FORCE_SEAL", spanner.CommitTimestamp,
							fmt.Sprintf("manifest=%s volume=%.1f/%.1f reason=%s", manifestID, result.totalVolume, result.maxVolume, overrideReason)},
					),
				})
			})
		}

		// MANIFEST_SEALED is now emitted via outbox INSIDE the seal txn above
		// (atomic with the state mutation). No post-commit emission needed.
		_ = now

		if s.Cache != nil {
			s.Cache.Invalidate(r.Context(),
				cache.PrefixManifestDetail+manifestID,
				cache.PrefixManifestOrders+manifestID,
			)
		}

		// ── Phase A: JIT Route Optimization at Seal Time ────────────────
		// Fetch DISPATCHED orders for this driver and trigger route recalculation.
		// Runs async so the seal response is not blocked by Google Maps API latency.
		if s.MapsAPIKey != "" && result.driverID != "" {
			capturedDriverID := result.driverID
			capturedManifestID := manifestID
			capturedWarehouseID := result.warehouseID
			workers.ETAPool.Submit(func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				orders, err := routing.GetLoadedOrdersForDriver(bgCtx, s.Spanner, capturedDriverID)
				if err != nil {
					log.Printf("[LEO] JIT route fetch failed for driver %s: %v", capturedDriverID[:min(8, len(capturedDriverID))], err)
					return
				}
				if len(orders) < 2 {
					log.Printf("[LEO] JIT route skipped for driver %s: only %d stops", capturedDriverID[:min(8, len(capturedDriverID))], len(orders))
					return
				}

				// Resolve depot: warehouse coords or fallback
				depot := s.DepotLocation
				if capturedWarehouseID != "" {
					if wDepot, err := resolveWarehouseDepot(bgCtx, s.Spanner, capturedWarehouseID); err == nil && wDepot != "" {
						depot = wDepot
					}
				}

				if err := routing.OptimizeDriverRoute(bgCtx, s.Spanner, s.MapsAPIKey, depot, orders); err != nil {
					log.Printf("[LEO] JIT route optimization failed for driver %s: %v", capturedDriverID[:min(8, len(capturedDriverID))], err)
					return
				}

				log.Printf("[LEO] JIT route optimized at seal: driver %s, %d stops, manifest %s",
					capturedDriverID[:min(8, len(capturedDriverID))], len(orders), capturedManifestID[:min(8, len(capturedManifestID))])

				// ROUTE_FINALIZED — emitted via outbox in a tiny RWTxn after the
				// optimizer's Apply commits. The outbox row is atomic with itself;
				// the route mutations are already durable from the optimizer above.
				_, emitErr := s.Spanner.ReadWriteTransaction(bgCtx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					return outbox.EmitJSON(txn, "Route", capturedManifestID, kafkaEvents.EventRouteFinalized, kafkaEvents.TopicMain,
						kafkaEvents.RouteFinalizedEvent{
							ManifestID: capturedManifestID,
							DriverID:   capturedDriverID,
							StopCount:  len(orders),
							Timestamp:  time.Now().UTC(),
						}, telemetry.TraceIDFromContext(ctx))
				})
				if emitErr != nil {
					log.Printf("[LEO] ROUTE_FINALIZED outbox emit failed for manifest %s: %v",
						capturedManifestID[:min(8, len(capturedManifestID))], emitErr)
				}
			})
		}

		log.Printf("[LEO] SEALED manifest %s: %d stops, %.1f/%.1f VU, sealed by %s%s",
			manifestID[:min(8, len(manifestID))], result.stopCount, result.totalVolume, result.maxVolume,
			sealedBy[:min(8, len(sealedBy))], func() string {
				if forceOverride {
					return " [FORCE OVERRIDE]"
				}
				return ""
			}())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"manifest_id":    manifestID,
			"state":          "SEALED",
			"stop_count":     result.stopCount,
			"volume_vu":      result.totalVolume,
			"max_vu":         result.maxVolume,
			"sealed_by":      sealedBy,
			"force_override": forceOverride,
			"message":        "Manifest sealed. Route finalized. Driver notified.",
		})
	}
}

// ── HandleListManifests ─────────────────────────────────────────────────────
// GET /v1/supplier/manifests?state=DRAFT|LOADING|SEALED|...&truck_id=...
func (s *ManifestService) HandleListManifests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		stateFilter := r.URL.Query().Get("state")
		truckIDFilter := r.URL.Query().Get("truck_id")

		sql := `SELECT ManifestId, SupplierId, COALESCE(WarehouseId, ''), COALESCE(RouteId, ''),
		               TruckId, DriverId, State, TotalVolumeVU, MaxVolumeVU, StopCount,
		               COALESCE(RegionCode, ''), SealedAt, DispatchedAt, CreatedAt
		        FROM SupplierTruckManifests
		        WHERE SupplierId = @sid`
		params := map[string]interface{}{"sid": claims.ResolveSupplierID()}

		if stateFilter != "" {
			sql += " AND State = @state"
			params["state"] = stateFilter
		}
		if truckIDFilter != "" {
			sql += " AND TruckId = @truck_id"
			params["truck_id"] = truckIDFilter
		}
		sql += " ORDER BY CreatedAt DESC LIMIT 100"

		iter := s.Spanner.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()

		type manifestRow struct {
			ManifestID  string  `json:"manifest_id"`
			SupplierID  string  `json:"supplier_id"`
			WarehouseID string  `json:"warehouse_id,omitempty"`
			RouteID     string  `json:"route_id,omitempty"`
			TruckID     string  `json:"truck_id"`
			DriverID    string  `json:"driver_id"`
			State       string  `json:"state"`
			TotalVolume float64 `json:"total_volume_vu"`
			MaxVolume   float64 `json:"max_volume_vu"`
			StopCount   int64   `json:"stop_count"`
			RegionCode  string  `json:"region_code,omitempty"`
			SealedAt    string  `json:"sealed_at,omitempty"`
			DispatchAt  string  `json:"dispatched_at,omitempty"`
			CreatedAt   string  `json:"created_at"`
		}

		var results []manifestRow
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[LEO] list manifests error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var m manifestRow
			var sealedAt, dispatchAt spanner.NullTime
			if err := row.Columns(&m.ManifestID, &m.SupplierID, &m.WarehouseID, &m.RouteID,
				&m.TruckID, &m.DriverID, &m.State, &m.TotalVolume, &m.MaxVolume, &m.StopCount,
				&m.RegionCode, &sealedAt, &dispatchAt, &m.CreatedAt); err != nil {
				continue
			}
			if sealedAt.Valid {
				m.SealedAt = sealedAt.Time.Format(time.RFC3339)
			}
			if dispatchAt.Valid {
				m.DispatchAt = dispatchAt.Time.Format(time.RFC3339)
			}
			results = append(results, m)
		}

		if results == nil {
			results = []manifestRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"manifests": results})
	}
}

// ── HandleManifestDetail ────────────────────────────────────────────────────
// GET /v1/supplier/manifests/{id}
func (s *ManifestService) HandleManifestDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		manifestID := ExtractPathParam(r.URL.Path, "manifests")
		if manifestID == "" {
			http.Error(w, `{"error":"manifest_id required"}`, http.StatusBadRequest)
			return
		}

		// Fetch manifest
		row, err := s.Spanner.Single().ReadRow(r.Context(), "SupplierTruckManifests",
			spanner.Key{manifestID},
			[]string{"ManifestId", "SupplierId", "TruckId", "DriverId", "State",
				"TotalVolumeVU", "MaxVolumeVU", "StopCount", "RegionCode",
				"SealedAt", "SealedBy", "DispatchedAt", "CreatedAt"})
		if err != nil {
			http.Error(w, `{"error":"manifest not found"}`, http.StatusNotFound)
			return
		}
		var mID, supID, truckID, driverID, state string
		var totalVol, maxVol float64
		var stops int64
		var region spanner.NullString
		var sealedAt, dispatchAt spanner.NullTime
		var sealedBy spanner.NullString
		var createdAt string
		if err := row.Columns(&mID, &supID, &truckID, &driverID, &state,
			&totalVol, &maxVol, &stops, &region,
			&sealedAt, &sealedBy, &dispatchAt, &createdAt); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Fetch manifest orders
		orderIter := s.Spanner.Single().Read(r.Context(), "ManifestOrders",
			spanner.Key{manifestID}.AsPrefix(),
			[]string{"OrderId", "SequenceIndex", "LoadingOrder", "VolumeVU", "State", "RemovedReason"})
		defer orderIter.Stop()

		type moRow struct {
			OrderID       string  `json:"order_id"`
			SequenceIndex int64   `json:"sequence_index"`
			LoadingOrder  int64   `json:"loading_order"`
			VolumeVU      float64 `json:"volume_vu"`
			State         string  `json:"state"`
			RemovedReason string  `json:"removed_reason,omitempty"`
		}
		var moRows []moRow
		for {
			oRow, err := orderIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var mo moRow
			var removedReason spanner.NullString
			if err := oRow.Columns(&mo.OrderID, &mo.SequenceIndex, &mo.LoadingOrder, &mo.VolumeVU, &mo.State, &removedReason); err != nil {
				continue
			}
			if removedReason.Valid {
				mo.RemovedReason = removedReason.StringVal
			}
			moRows = append(moRows, mo)
		}
		if moRows == nil {
			moRows = []moRow{}
		}

		resp := map[string]interface{}{
			"manifest_id":     mID,
			"supplier_id":     supID,
			"truck_id":        truckID,
			"driver_id":       driverID,
			"state":           state,
			"total_volume_vu": totalVol,
			"max_volume_vu":   maxVol,
			"stop_count":      stops,
			"region_code":     region.StringVal,
			"created_at":      createdAt,
			"orders":          moRows,
		}
		if sealedAt.Valid {
			resp["sealed_at"] = sealedAt.Time.Format(time.RFC3339)
		}
		if sealedBy.Valid {
			resp["sealed_by"] = sealedBy.StringVal
		}
		if dispatchAt.Valid {
			resp["dispatched_at"] = dispatchAt.Time.Format(time.RFC3339)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── HandleDriverManifestGate ────────────────────────────────────────────────
// GET /v1/driver/manifest-gate?manifest_id=X
// Ghost Stop Prevention: returns 403 if manifest is not SEALED.
// Driver apps call this before enabling "Start Route" button.
func (s *ManifestService) HandleDriverManifestGate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		manifestID := r.URL.Query().Get("manifest_id")
		if manifestID == "" {
			http.Error(w, `{"error":"manifest_id required"}`, http.StatusBadRequest)
			return
		}

		row, err := s.Spanner.Single().ReadRow(r.Context(), "SupplierTruckManifests",
			spanner.Key{manifestID},
			[]string{"State", "SealedAt", "StopCount", "TotalVolumeVU"})
		if err != nil {
			http.Error(w, `{"error":"manifest not found"}`, http.StatusNotFound)
			return
		}
		var state string
		var sealedAt spanner.NullTime
		var stopCount int64
		var totalVol float64
		if err := row.Columns(&state, &sealedAt, &stopCount, &totalVol); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch state {
		case "SEALED", "DISPATCHED", "COMPLETED":
			// ✅ Route is cleared for navigation
			json.NewEncoder(w).Encode(map[string]interface{}{
				"manifest_id": manifestID,
				"state":       state,
				"cleared":     true,
				"stop_count":  stopCount,
				"volume_vu":   totalVol,
			})
		default:
			// 🚫 Ghost Stop Prevention — driver cannot navigate
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"manifest_id": manifestID,
				"state":       state,
				"cleared":     false,
				"error":       "AWAITING_PAYLOAD_SEAL",
				"message":     "Manifest is in " + state + " state. Wait for Payloader to complete loading and seal.",
			})
		}
	}
}

// ── HandleManifestException ─────────────────────────────────────────────────
// POST /v1/payload/manifest-exception
// Payloader removes an order from a LOADING manifest (damaged, overflow, manual).
// Order is re-injected into the dispatch pool with DispatchPriority=10.
// After 3 OVERFLOW exceptions, order escalates to admin DLQ.
func (s *ManifestService) HandleManifestException() http.HandlerFunc {
	type exceptionReq struct {
		ManifestID string `json:"manifest_id"`
		OrderID    string `json:"order_id"`
		Reason     string `json:"reason"` // OVERFLOW | DAMAGED | MANUAL
		Metadata   string `json:"metadata,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)

		var req exceptionReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if req.ManifestID == "" || req.OrderID == "" || req.Reason == "" {
			http.Error(w, `{"error":"manifest_id, order_id, reason required"}`, http.StatusBadRequest)
			return
		}
		// Validate reason
		switch req.Reason {
		case "OVERFLOW", "DAMAGED", "MANUAL":
		default:
			http.Error(w, `{"error":"reason must be OVERFLOW, DAMAGED, or MANUAL"}`, http.StatusBadRequest)
			return
		}

		exceptionID := uuid.New().String()
		var escalated bool
		var overflowCount int64

		_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Verify manifest is in LOADING state
			mRow, err := txn.ReadRow(ctx, "SupplierTruckManifests",
				spanner.Key{req.ManifestID}, []string{"State", "SupplierId", "TotalVolumeVU"})
			if err != nil {
				return fmt.Errorf("manifest not found: %w", err)
			}
			var state, supplierID string
			var totalVol float64
			if err := mRow.Columns(&state, &supplierID, &totalVol); err != nil {
				return err
			}
			if state != "LOADING" {
				return fmt.Errorf("INVALID_STATE: manifest is %s, expected LOADING", state)
			}

			// Read order's current OverflowCount
			oRow, err := txn.ReadRow(ctx, "Orders",
				spanner.Key{req.OrderID}, []string{"OverflowCount"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var oc spanner.NullInt64
			if err := oRow.Columns(&oc); err != nil {
				return err
			}
			currentCount := oc.Int64
			if req.Reason == "OVERFLOW" {
				currentCount++
			}
			overflowCount = currentCount

			// DLQ escalation: 3+ OVERFLOW attempts → admin must intervene
			if req.Reason == "OVERFLOW" && currentCount >= DLQThreshold {
				escalated = true
			}

			// Read ManifestOrder to get volume for recalculation
			moRow, err := txn.ReadRow(ctx, "ManifestOrders",
				spanner.Key{req.ManifestID, req.OrderID}, []string{"VolumeVU"})
			if err != nil {
				return fmt.Errorf("manifest order not found: %w", err)
			}
			var orderVol float64
			if err := moRow.Columns(&orderVol); err != nil {
				return err
			}

			// Determine removal state
			removalState := "REMOVED_MANUAL"
			switch req.Reason {
			case "OVERFLOW":
				removalState = "REMOVED_OVERFLOW"
			case "DAMAGED":
				removalState = "REMOVED_DAMAGED"
			}

			mutations := []*spanner.Mutation{
				// Mark ManifestOrder as removed
				spanner.Update("ManifestOrders",
					[]string{"ManifestId", "OrderId", "State", "RemovedReason"},
					[]interface{}{req.ManifestID, req.OrderID, removalState, req.Reason}),

				// Update manifest volume and stop count
				spanner.Update("SupplierTruckManifests",
					[]string{"ManifestId", "TotalVolumeVU", "StopCount", "UpdatedAt"},
					[]interface{}{req.ManifestID, totalVol - orderVol, spanner.CommitTimestamp, spanner.CommitTimestamp}),

				// Re-inject order: clear manifest assignment, set priority, update overflow count
				spanner.Update("Orders",
					[]string{"OrderId", "ManifestId", "RouteId", "State", "DispatchPriority", "OverflowCount"},
					[]interface{}{req.OrderID, nil, nil, "PENDING", int64(10), currentCount}),

				// Record exception
				spanner.Insert("ManifestExceptions",
					[]string{"ExceptionId", "OrderId", "ManifestId", "SupplierId",
						"Reason", "Metadata", "AttemptCount", "CreatedAt", "UpdatedAt"},
					[]interface{}{exceptionID, req.OrderID, req.ManifestID, supplierID,
						req.Reason, nullStr(req.Metadata), currentCount,
						spanner.CommitTimestamp, spanner.CommitTimestamp}),
			}

			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}

			now := time.Now().UTC()

			traceID := telemetry.TraceIDFromContext(ctx)

			// MANIFEST_ORDER_EXCEPTION — atomic with the exception mutations.
			if err := outbox.EmitJSON(txn, "Manifest", req.ManifestID, kafkaEvents.EventManifestOrderException, kafkaEvents.TopicMain,
				kafkaEvents.ManifestOrderExceptionEvent{
					ExceptionID:  exceptionID,
					ManifestID:   req.ManifestID,
					OrderID:      req.OrderID,
					SupplierId:   supplierID,
					Reason:       req.Reason,
					AttemptCount: currentCount,
					Escalated:    escalated,
					Metadata:     req.Metadata,
					Timestamp:    now,
				}, traceID); err != nil {
				return err
			}

			if err := emitPayloadSyncEvent(txn, supplierID, "", req.ManifestID, kafkaEvents.EventManifestOrderException, traceID, now); err != nil {
				return err
			}

			// MANIFEST_DLQ_ESCALATION — distinct audit event when a 3x overflow
			// crosses the DLQ threshold. Reuses the exception payload shape;
			// the topic discriminator is the AggregateType + JSON shape.
			if escalated {
				if err := outbox.EmitJSON(txn, "Manifest", req.ManifestID, kafkaEvents.EventManifestDLQEscalation, kafkaEvents.TopicMain,
					kafkaEvents.ManifestOrderExceptionEvent{
						ExceptionID:  exceptionID,
						ManifestID:   req.ManifestID,
						OrderID:      req.OrderID,
						SupplierId:   supplierID,
						Reason:       "3x OVERFLOW — requires admin intervention",
						AttemptCount: currentCount,
						Escalated:    true,
						Timestamp:    now,
					}, telemetry.TraceIDFromContext(ctx)); err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			if isInvalidState(err) {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			log.Printf("[LEO] exception error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if escalated {
			log.Printf("[LEO-DLQ] Order %s escalated after %d OVERFLOW attempts", req.OrderID[:8], overflowCount)
		}

		log.Printf("[LEO] Exception: order %s removed from manifest %s (%s), priority=10, overflow=%d, escalated=%v",
			req.OrderID[:8], req.ManifestID[:8], req.Reason, overflowCount, escalated)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exception_id":   exceptionID,
			"order_id":       req.OrderID,
			"manifest_id":    req.ManifestID,
			"reason":         req.Reason,
			"overflow_count": overflowCount,
			"escalated":      escalated,
			"reinjected":     true,
			"new_priority":   10,
		})
	}
}

// ── HandleListExceptions ────────────────────────────────────────────────────
// GET /v1/supplier/manifest-exceptions?escalated=true
func (s *ManifestService) HandleListExceptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		escalatedOnly := r.URL.Query().Get("escalated") == "true"

		sql := `SELECT e.ExceptionId, e.OrderId, e.ManifestId, e.Reason,
		               e.AttemptCount, e.CreatedAt, e.Metadata
		        FROM ManifestExceptions e
		        WHERE e.SupplierId = @sid`
		params := map[string]interface{}{"sid": claims.ResolveSupplierID()}

		if escalatedOnly {
			sql += " AND e.AttemptCount >= @dlq"
			params["dlq"] = DLQThreshold
		}
		sql += " ORDER BY e.CreatedAt DESC LIMIT 200"

		iter := s.Spanner.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()

		type exRow struct {
			ExceptionID string `json:"exception_id"`
			OrderID     string `json:"order_id"`
			ManifestID  string `json:"manifest_id"`
			Reason      string `json:"reason"`
			Attempts    int64  `json:"attempt_count"`
			CreatedAt   string `json:"created_at"`
			Metadata    string `json:"metadata,omitempty"`
		}
		var rows []exRow
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[LEO] list exceptions error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var ex exRow
			var meta spanner.NullString
			if err := row.Columns(&ex.ExceptionID, &ex.OrderID, &ex.ManifestID,
				&ex.Reason, &ex.Attempts, &ex.CreatedAt, &meta); err != nil {
				continue
			}
			if meta.Valid {
				ex.Metadata = meta.StringVal
			}
			rows = append(rows, ex)
		}
		if rows == nil {
			rows = []exRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"exceptions": rows})
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// nullStr is defined in fleet.go — reused here via package scope.

func isInvalidState(err error) bool {
	return err != nil && len(err.Error()) > 14 && err.Error()[:14] == "INVALID_STATE:"
}

func isVolumeConflict(err error) bool {
	return err != nil && len(err.Error()) > 16 && err.Error()[:16] == "VOLUME_CONFLICT:"
}

// ExtractPathParam extracts value after /segment/ from URL path.
// e.g. /v1/supplier/manifests/abc-123/seal → "abc-123"
func ExtractPathParam(path, segment string) string {
	idx := 0
	for i := 0; i < len(path)-len(segment); i++ {
		if path[i:i+len(segment)] == segment {
			idx = i + len(segment)
			break
		}
	}
	if idx == 0 || idx >= len(path) {
		return ""
	}
	if path[idx] == '/' {
		idx++
	}
	end := idx
	for end < len(path) && path[end] != '/' {
		end++
	}
	return path[idx:end]
}

// resolveWarehouseDepot fetches "lat,lng" depot string from Warehouses table for JIT route optimization.
func resolveWarehouseDepot(ctx context.Context, client *spanner.Client, warehouseID string) (string, error) {
	row, err := client.Single().ReadRow(ctx, "Warehouses",
		spanner.Key{warehouseID},
		[]string{"Latitude", "Longitude"})
	if err != nil {
		return "", err
	}
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&lat, &lng); err != nil {
		return "", err
	}
	if !lat.Valid || !lng.Valid {
		return "", fmt.Errorf("warehouse %s has no coordinates", warehouseID)
	}
	return fmt.Sprintf("%f,%f", lat.Float64, lng.Float64), nil
}
