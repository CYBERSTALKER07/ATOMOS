package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

// dispatchExecuteManifestState is written on execute; payloader seals before depart.
const dispatchExecuteManifestState = "DRAFT"

// dispatchExecuteCommittedStatus is returned when one or more manifests are persisted.
const dispatchExecuteCommittedStatus = "dispatched"

// DispatchExecuteResult is the supplier-portal response for a committed dispatch.
// Status is "no_op" when nothing was committed; "dispatched" when one or more manifests were written.
type DispatchExecuteResult struct {
	Status           string                 `json:"status"`
	SupplierID       string                 `json:"supplier_id"`
	WarehouseID      string                 `json:"warehouse_id,omitempty"`
	ManifestsCreated int                    `json:"manifests_created"`
	OrdersAssigned   int                    `json:"orders_assigned"`
	OptimizerSource  string                 `json:"optimizer_source,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	Manifests        []DispatchExecuteRoute `json:"manifests"`
	Orphans          []string               `json:"orphan_order_ids,omitempty"`
}

// DispatchExecuteRoute is one committed truck manifest summary.
type DispatchExecuteRoute struct {
	ManifestID string   `json:"manifest_id"`
	RouteID    string   `json:"route_id"`
	DriverID   string   `json:"driver_id"`
	VehicleID  string   `json:"vehicle_id,omitempty"`
	OrderIDs   []string `json:"order_ids"`
	VolumeVU   float64  `json:"volume_vu"`
	MaxVolume  float64  `json:"max_volume_vu"`
}

// HandleDispatchExecute serves POST /v1/supplier/dispatch/execute. It runs the
// optimiser over the supplier's dispatchable orders, then atomically persists
// one DRAFT manifest per route, flips each assigned order to LOADED with its
// driver/vehicle/route/manifest binding, and emits ROUTE_CREATED +
// MANIFEST_DRAFT_CREATED + ORDER_ASSIGNED so every downstream client converges.
func (s *Service) HandleDispatchExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "dispatch_unavailable"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	// Idempotency guard (API flavor): same key + same body → replay; different body → 409.
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" && s.idem != nil {
		hash := sha256Hex(body)
		rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
		switch {
		case errors.Is(err, idempotency.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_payload_mismatch"})
			return
		case err != nil:
			s.log.Warn("idempotency guard failed", "err", err)
		case hit:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.StatusCode)
			_, _ = w.Write(rec.Response)
			return
		}
	}

	sid := s.scopedSupplierID(r)
	warehouseFilter := resolveSupplierDispatchWarehouseID(r)

	result, err := s.executeDispatch(r.Context(), sid, warehouseFilter)
	if err != nil {
		s.log.ErrorContext(r.Context(), "dispatch execute failed", "supplier_id", sid, "warehouse_id", warehouseFilter, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_execute_failed"})
		return
	}

	resultBytes, _ := json.Marshal(result)

	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" && s.idem != nil {
		_ = s.idem.Save(r.Context(), key, idempotency.Record{
			BodyHash:   sha256Hex(body),
			StatusCode: http.StatusOK,
			Response:   resultBytes,
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour)
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(sid))
	}
	s.broadcastDispatchCommitted(r.Context(), sid, result)
	s.log.InfoContext(r.Context(), "dispatch executed",
		"supplier_id", sid,
		"warehouse_id", warehouseFilter,
		"manifests", result.ManifestsCreated,
		"orders_assigned", result.OrdersAssigned,
		"optimizer_source", result.OptimizerSource,
	)
	writeJSON(w, http.StatusOK, result)
}

func (s *Service) executeDispatch(ctx context.Context, supplierID, warehouseID string) (DispatchExecuteResult, error) {
	out := DispatchExecuteResult{
		Status:      "no_op",
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		Manifests:   []DispatchExecuteRoute{},
	}

	lockWarehouseID := strings.TrimSpace(warehouseID)
	var freezeLocks map[string]dispatch.FreezeLock
	if lockWarehouseID != "" {
		var err error
		freezeLocks, err = dispatch.LoadFreezeLocks(ctx, s.portalSpanner, lockWarehouseID)
		if err != nil {
			return DispatchExecuteResult{}, err
		}
		if frozen, reason := dispatch.IsWarehouseDispatchFrozen(freezeLocks, lockWarehouseID); frozen {
			out.Warnings = append(out.Warnings, reason)
			return out, nil
		}
	}

	repo := dispatch.NewRepository(s.portalSpanner)
	rows, err := dispatch.FetchAllDispatchable(ctx, repo, dispatch.FetchParams{
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		StrongRead:  true,
	})
	if err != nil {
		return DispatchExecuteResult{}, err
	}
	if lockWarehouseID == "" {
		lockWarehouseID = resolveDispatchLockWarehouseID(rows)
		if lockWarehouseID != "" {
			freezeLocks, err = dispatch.LoadFreezeLocks(ctx, s.portalSpanner, lockWarehouseID)
			if err != nil {
				return DispatchExecuteResult{}, err
			}
			if frozen, reason := dispatch.IsWarehouseDispatchFrozen(freezeLocks, lockWarehouseID); frozen {
				out.Warnings = append(out.Warnings, reason)
				return out, nil
			}
		}
	}
	rows = dispatch.FilterFreezeLockedOrders(freezeLocks, rows)
	if len(rows) == 0 {
		return out, nil
	}

	fleet, vehicleByDriver, err := s.buildDispatchFleet(ctx, supplierID, warehouseID, freezeLocks)
	if err != nil {
		return DispatchExecuteResult{}, err
	}
	if len(fleet) == 0 {
		out.Warnings = append(out.Warnings, "no_available_drivers")
		return out, nil
	}

	homeNodeID := strings.TrimSpace(warehouseID)
	if homeNodeID == "" {
		homeNodeID = strings.TrimSpace(fleet[0].DriverID)
	}
	depot := dispatch.ResolveDepot(ctx, s.portalSpanner, warehouseID, dispatch.DepotCoords{
		Lat: s.fallbackDepotLat,
		Lng: s.fallbackDepotLng,
	})
	job := plan.BuildSolveJob(ctx, supplierID, homeNodeID, depot, rows, fleet)
	assignment, source, err := plan.OptimizeAndValidate(ctx, s.optimizerClient, job)
	if err != nil {
		return DispatchExecuteResult{}, err
	}
	out.OptimizerSource = source
	if assignment != nil {
		out.Warnings = append(out.Warnings, assignment.Warnings...)
		for _, orphan := range assignment.Orphans {
			out.Orphans = append(out.Orphans, orphan.OrderID)
		}
	}
	if assignment == nil || len(assignment.Routes) == 0 {
		return out, nil
	}

	now := s.now().UTC()
	lockID := ""
	if lockWarehouseID != "" {
		lockID, err = dispatch.AcquireManualDispatchLock(ctx, s.portalSpanner, lockWarehouseID, supplierID, supplierID, now)
		if errors.Is(err, dispatch.ErrDispatchLocked) {
			out.Warnings = append(out.Warnings, "warehouse_dispatch_locked")
			return out, nil
		}
		if err != nil {
			return DispatchExecuteResult{}, err
		}
		defer func() {
			if releaseErr := dispatch.ReleaseManualDispatchLock(ctx, s.portalSpanner, lockWarehouseID, lockID, supplierID, supplierID, s.now().UTC()); releaseErr != nil {
				s.log.WarnContext(ctx, "dispatch execute lock release failed", "warehouse_id", lockWarehouseID, "lock_id", lockID, "err", releaseErr)
			}
		}()
	}

	committed := make([]DispatchExecuteRoute, 0, len(assignment.Routes))
	type pendingEvent struct {
		aggregateType string
		aggregateID   string
		payload       any
	}

	chunkSize := 50 // 50 routes per chunk

	err = spannerutils.RunChunkedTransaction(ctx, s.portalSpanner, assignment.Routes, chunkSize, func(ctx context.Context, txn *spanner.ReadWriteTransaction, routes []dispatch.DispatchRoute) error {
		batch := &manifest.SupplierWriteBatch{}
		var chunkQueued []pendingEvent

		for _, route := range routes {
			driverID := strings.TrimSpace(route.DriverID)
			if driverID == "" || len(route.Orders) == 0 {
				continue
			}
			manifestID := uuid.NewString()
			routeID := uuid.NewString()
			vehicleID := strings.TrimSpace(vehicleByDriver[driverID])
			batch.Manifests = append(batch.Manifests, manifest.SupplierTruckRow{
				ManifestID:    manifestID,
				SupplierID:    supplierID,
				WarehouseID:   warehouseID,
				RouteID:       routeID,
				TruckID:       vehicleID,
				DriverID:      driverID,
				State:         dispatchExecuteManifestState,
				TotalVolumeVU: route.LoadedVolume,
				MaxVolumeVU:   route.MaxVolume,
				StopCount:     int64(len(route.Orders)),
				CreatedAt:     now,
			})

			orderIDs := make([]string, 0, len(route.Orders))
			for idx, stop := range route.Orders {
				orderID := strings.TrimSpace(stop.OrderID)
				if orderID == "" {
					continue
				}
				batch.Orders = append(batch.Orders, manifest.SupplierManifestOrderRow{
					ManifestID:    manifestID,
					OrderID:       orderID,
					SequenceIndex: int64(idx),
					LoadingOrder:  int64(idx),
					VolumeVU:      stop.Volume,
					State:         "LOADED",
					UpdatedAt:     now,
				})
				batch.OrderPatches = append(batch.OrderPatches, manifest.OrderPatch{
					OrderID:    orderID,
					Status:     "LOADED",
					ManifestID: manifestID,
					DriverID:   driverID,
					VehicleID:  vehicleID,
					RouteID:    routeID,
					UpdatedAt:  now,
				})
				chunkQueued = append(chunkQueued, pendingEvent{
					aggregateType: events.AggregateOrder,
					aggregateID:   orderID,
					payload: events.OrderEvent{
						BaseEvent:   events.BaseEvent{Type: events.EventOrderAssigned},
						OrderID:     orderID,
						SupplierID:  supplierID,
						RetailerID:  stop.RetailerID,
						WarehouseID: warehouseID,
						DriverID:    driverID,
						VehicleID:   vehicleID,
						RouteID:     routeID,
						ManifestID:  manifestID,
						Status:      "LOADED",
					},
				})
				orderIDs = append(orderIDs, orderID)
			}

			chunkQueued = append(chunkQueued,
				pendingEvent{
					aggregateType: events.AggregateRoute,
					aggregateID:   routeID,
					payload: events.RouteEvent{
						BaseEvent:   events.BaseEvent{Type: events.EventRouteCreated},
						RouteID:     routeID,
						ManifestID:  manifestID,
						SupplierID:  supplierID,
						WarehouseID: warehouseID,
						DriverID:    driverID,
						VehicleID:   vehicleID,
						OrderIDs:    orderIDs,
						OrderCount:  len(orderIDs),
					},
				},
				pendingEvent{
					aggregateType: events.AggregateManifest,
					aggregateID:   manifestID,
					payload: events.ManifestEvent{
						BaseEvent:   events.BaseEvent{Type: events.EventManifestDraftCreated},
						ManifestID:  manifestID,
						RouteID:     routeID,
						SupplierID:  supplierID,
						WarehouseID: warehouseID,
						DriverID:    driverID,
						StopCount:   int64(len(orderIDs)),
					},
				},
			)

			committed = append(committed, DispatchExecuteRoute{
				ManifestID: manifestID,
				RouteID:    routeID,
				DriverID:   driverID,
				VehicleID:  vehicleID,
				OrderIDs:   orderIDs,
				VolumeVU:   route.LoadedVolume,
				MaxVolume:  route.MaxVolume,
			})
			out.OrdersAssigned += len(orderIDs)
		}

		if len(batch.Manifests) == 0 {
			return nil
		}

		store := manifest.NewStore(s.portalSpanner)
		if err := store.CommitSupplierTxn(ctx, txn, batch, func(buf outbox.TxnBuffer) error {
			for _, evt := range chunkQueued {
				if err := outbox.EmitJSON(ctx, buf, evt.aggregateType, evt.aggregateID, events.TopicMain, evt.payload); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return DispatchExecuteResult{}, err
	}
	out.ManifestsCreated = len(committed)
	out.Manifests = committed
	if len(committed) > 0 {
		out.Status = dispatchExecuteCommittedStatus
	}
	if s.manifestStore != nil && len(committed) > 0 {
		manifestIDs := make([]string, 0, len(committed))
		for _, route := range committed {
			manifestIDs = append(manifestIDs, route.ManifestID)
		}
		s.manifestStore.PersistDispatchPreviewGeometries(ctx, manifestIDs)
	}
	return out, nil
}

// buildDispatchFleet hydrates the active driver+vehicle fleet for the optimiser
// and a driver→vehicle lookup for manifest binding.
func (s *Service) buildDispatchFleet(ctx context.Context, supplierID, warehouseID string, freezeLocks map[string]dispatch.FreezeLock) ([]dispatch.AvailableDriver, map[string]string, error) {
	vehiclesByID := make(map[string]dispatch.VehicleSpec)
	if vehicles, err := s.repo.ListFleetVehicles(ctx, supplierID); err == nil {
		for _, vehicle := range vehicles {
			id, spec := dispatch.VehicleSpecIndex(vehicle.VehicleID, vehicle.VehicleClass, vehicle.MaxVolumeVU)
			vehiclesByID[id] = spec
		}
	}
	drivers, err := s.repo.ListFleetDrivers(ctx, supplierID)
	if err != nil {
		return nil, nil, err
	}
	busy, err := s.driversOnActiveManifests(ctx, supplierID, warehouseID, collectSupplierDriverIDs(drivers, warehouseID))
	if err != nil {
		return nil, nil, err
	}
	driverInputs := make([]dispatch.FleetDriverInput, 0, len(drivers))
	vehicleByDriver := make(map[string]string, len(drivers))
	for _, driver := range drivers {
		if !driver.IsActive {
			continue
		}
		if warehouseID != "" && !strings.EqualFold(strings.TrimSpace(driver.HomeNodeID), warehouseID) {
			continue
		}
		driverID := strings.TrimSpace(driver.DriverID)
		if busy[driverID] {
			continue
		}
		if len(dispatch.FilterFreezeLockedDriverIDs(freezeLocks, []string{driverID})) == 0 {
			continue
		}
		driverInputs = append(driverInputs, dispatch.FleetDriverInput{
			DriverID:    driver.DriverID,
			DriverName:  driver.Name,
			VehicleID:   driver.VehicleID,
			IsActive:    driver.IsActive,
			TruckStatus: "AVAILABLE",
			HomeNodeID:  driver.HomeNodeID,
		})
		vehicleByDriver[strings.TrimSpace(driver.DriverID)] = strings.TrimSpace(driver.VehicleID)
	}
	return dispatch.BuildAvailableFleet(driverInputs, vehiclesByID), vehicleByDriver, nil
}

func (s *Service) broadcastDispatchCommitted(ctx context.Context, supplierID string, result DispatchExecuteResult) {
	if s.portalSupplierHub == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type":              "DISPATCH_COMMITTED",
		"supplier_id":       supplierID,
		"manifests_created": result.ManifestsCreated,
		"orders_assigned":   result.OrdersAssigned,
		"timestamp":         s.now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return
	}
	s.portalSupplierHub.Broadcast(ctx, "supplier:"+supplierID, payload)
}

func resolveDispatchLockWarehouseID(rows []dispatch.DispatchableOrder) string {
	for _, row := range rows {
		if warehouseID := strings.TrimSpace(row.WarehouseID); warehouseID != "" {
			return warehouseID
		}
	}
	return ""
}
