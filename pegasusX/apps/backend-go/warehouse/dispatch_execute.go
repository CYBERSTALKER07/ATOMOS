package warehouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// DispatchExecuteRequest is the service-layer input for warehouse dispatch commit.
type DispatchExecuteRequest struct {
	WarehouseID      string
	SupplierID       string
	Mode             string
	Routes           []DispatchExecuteRoute
	OrderIDs         []string
	ForceCapacity    bool
	AcceptPartial    bool
	PlanFingerprint  string
	// AllowRetailerSplit permits the system to split a retailer's consolidated
	// order across multiple trucks when it exceeds a single truck's capacity.
	// When false and overflow is detected, the response returns status
	// "capacity_overflow" with OverflowWarnings detailing which orders + by how
	// much they exceed the largest truck, requiring admin action before commit.
	AllowRetailerSplit bool
}

// ExecuteDispatch runs the warehouse smart-dispatch optimizer and commits manifests.
func (s *Service) ExecuteDispatch(ctx context.Context, req DispatchExecuteRequest) (DispatchExecuteResult, error) {
	whID := strings.TrimSpace(req.WarehouseID)
	sid := strings.TrimSpace(req.SupplierID)
	if sid == "" {
		sid = strings.TrimSpace(s.supplierID)
	}

	out := DispatchExecuteResult{
		Status:      "no_op",
		SupplierID:  sid,
		WarehouseID: whID,
		Manifests:   []DispatchExecuteRoute{},
	}
	if s.spannerClient == nil {
		return out, fmt.Errorf("dispatch_unavailable")
	}
	if whID == "" || sid == "" {
		return out, fmt.Errorf("warehouse_scope_required")
	}

	if locked, reason := s.isWarehouseDispatchLocked(ctx, whID); locked {
		out.Warnings = append(out.Warnings, reason)
		return out, nil
	}

	repo := dispatch.NewRepository(s.spannerClient)
	rows, err := dispatch.FetchAllDispatchable(ctx, repo, dispatch.FetchParams{
		SupplierID:  sid,
		WarehouseID: whID,
		StrongRead:  true,
	})
	if err != nil {
		return out, fmt.Errorf("fetch dispatchable: %w", err)
	}
	rows = filterLockedOrders(ctx, s, whID, rows)
	rows = filterDispatchRowsByOrderIDs(rows, req.OrderIDs)
	rows, _ = s.applyZoneOverridesToDispatchRows(ctx, sid, rows)
	if len(rows) == 0 {
		return out, nil
	}

	solveDrivers, err := s.loadDispatchDrivers(ctx, whID)
	if err != nil {
		return out, err
	}
	solveDrivers = filterLockedDrivers(whID, solveDrivers, s.loadDispatchLocks(ctx, whID))
	fleetCtx, err := s.loadFleetDispatchContext(ctx, sid, whID, collectWarehouseDriverIDs(solveDrivers))
	if err != nil {
		return out, fmt.Errorf("fleet dispatch context: %w", err)
	}

	if fp := strings.TrimSpace(req.PlanFingerprint); fp != "" && strings.ToUpper(req.Mode) != "MANUAL" {
		currentFP := computeDispatchPlanFingerprint(rows, fleetCtx, req.OrderIDs)
		if currentFP != fp {
			out.Status = "plan_stale"
			out.Warnings = append(out.Warnings, "plan_fingerprint_mismatch")
			return out, nil
		}
	}

	var driverInputs []dispatch.FleetDriverInput
	vehicleByDriver := make(map[string]string)
	for _, input := range buildFleetDriverInputs(solveDrivers, fleetCtx, whID) {
		driverInputs = append(driverInputs, input)
		vehicleByDriver[strings.TrimSpace(input.DriverID)] = strings.TrimSpace(input.VehicleID)
	}

	var assignment *dispatch.AssignmentResult
	var source string

	if strings.ToUpper(req.Mode) == "MANUAL" {
		source = "manual"
		assignment = buildManualAssignment(rows, solveDrivers, fleetCtx, req.Routes)
	} else {
		fleet := dispatch.BuildAvailableFleet(driverInputs, nil)
		if len(fleet) == 0 {
			out.Warnings = append(out.Warnings, "no_available_drivers")
			return out, nil
		}
		depot := dispatch.ResolveDepot(ctx, s.spannerClient, whID, dispatch.DepotCoords{
			Lat: s.fallbackDepotLat,
			Lng: s.fallbackDepotLng,
		})
		job := plan.BuildSolveJob(ctx, sid, whID, depot, rows, fleet)
		assignment, source, err = plan.OptimizeAndValidate(ctx, s.optimizerClient, job)
		if err != nil {
			return out, fmt.Errorf("optimize dispatch: %w", err)
		}
	}

	out.OptimizerSource = source
	if assignment != nil {
		out.Warnings = append(out.Warnings, assignment.Warnings...)
		for _, orphan := range assignment.Orphans {
			out.Orphans = append(out.Orphans, orphan.OrderID)
		}
		// Surface retailer overflow warnings so the warehouse admin can act.
		if len(assignment.OverflowWarnings) > 0 {
			out.OverflowWarnings = assignment.OverflowWarnings
			if !req.AllowRetailerSplit {
				// Block commit: admin must decide to split or cancel orders.
				out.Status = "capacity_overflow"
				out.Warnings = append(out.Warnings, "retailer_order_exceeds_max_truck")
				return out, nil
			}
		}
		// Carry split shipment groups to the result (populated after commit below).
		out.SplitShipmentGroups = assignment.SplitShipmentGroups
	}
	if assignment == nil || len(assignment.Routes) == 0 {
		return out, nil
	}

	var capacityWarnings []DispatchCapacityWarning
	mode := strings.ToUpper(req.Mode)
	if mode == "MANUAL" {
		capacityWarnings = manualCapacityWarnings(assignment.Routes, solveDrivers, fleetCtx)
		if len(capacityWarnings) > 0 {
			out.CapacityWarnings = capacityWarnings
			if !req.ForceCapacity {
				out.Status = "capacity_exceeded"
				out.Warnings = append(out.Warnings, "capacity_exceeded")
				return out, nil
			}
			out.Warnings = append(out.Warnings, "capacity_override")
		}
	} else if mode == "AUTO" {
		requestedVU := sumOrderVolumeVU(rows)
		fleetVU := fleetEffectiveCapacityVU(solveDrivers, fleetCtx)
		capacityWarnings = autoCapacityWarnings(assignment, solveDrivers, fleetCtx, requestedVU, fleetVU)
		if len(capacityWarnings) > 0 || len(out.Orphans) > 0 {
			out.CapacityWarnings = capacityWarnings
			if !req.ForceCapacity && !req.AcceptPartial {
				out.Status = "capacity_exceeded"
				out.Warnings = append(out.Warnings, "capacity_exceeded")
				return out, nil
			}
			if req.ForceCapacity && len(capacityWarnings) > 0 {
				out.Warnings = append(out.Warnings, "capacity_override")
			}
		}
	}

	now := s.now().UTC()
	batch := &manifest.SupplierWriteBatch{}
	committed := make([]DispatchExecuteRoute, 0, len(assignment.Routes))
	type pendingEvent struct {
		aggregateType string
		aggregateID   string
		payload       any
	}
	queued := make([]pendingEvent, 0, len(rows)+len(assignment.Routes))


	for _, route := range assignment.Routes {
		driverID := strings.TrimSpace(route.DriverID)
		if driverID == "" || len(route.Orders) == 0 {
			continue
		}
		vehicleID := strings.TrimSpace(vehicleByDriver[driverID])
		manifestID := uuid.NewString()
		routeID := uuid.NewString()
		seqBase := int64(0)
		existingTotalVU := 0.0
		truckMaxVU := route.MaxVolume
		isTopOff := false
		if top := fleetCtx.topOffFor(driverID); top != nil && top.ManifestID != "" {
			isTopOff = true
			manifestID = top.ManifestID
			routeID = top.RouteID
			seqBase = top.StopCount
			existingTotalVU = top.TotalVolumeVU
			if top.MaxVolumeVU > 0 {
				truckMaxVU = top.MaxVolumeVU
			}
		}
		newVolumeVU := route.LoadedVolume
		totalVolumeVU := existingTotalVU + newVolumeVU
		stopCount := seqBase + int64(len(route.Orders))

		batch.Manifests = append(batch.Manifests, manifest.SupplierTruckRow{
			ManifestID:    manifestID,
			SupplierID:    sid,
			WarehouseID:   whID,
			RouteID:       routeID,
			TruckID:       vehicleID,
			DriverID:      driverID,
			State:         warehouseDispatchExecuteManifestState,
			TotalVolumeVU: totalVolumeVU,
			MaxVolumeVU:   truckMaxVU,
			StopCount:     stopCount,
			CreatedAt:     now,
		})

		orderIDs := make([]string, 0, len(route.Orders))
		for idx, stop := range route.Orders {
			orderID := strings.TrimSpace(stop.OrderID)
			if orderID == "" {
				continue
			}
			seq := seqBase + int64(idx)
			batch.Orders = append(batch.Orders, manifest.SupplierManifestOrderRow{
				ManifestID:    manifestID,
				OrderID:       orderID,
				SequenceIndex: seq,
				LoadingOrder:  seq,
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
			queued = append(queued, pendingEvent{
				aggregateType: events.AggregateOrder,
				aggregateID:   orderID,
				payload: events.OrderEvent{
					BaseEvent:   events.BaseEvent{Type: events.EventOrderAssigned},
					OrderID:     orderID,
					SupplierID:  sid,
					RetailerID:  stop.RetailerID,
					WarehouseID: whID,
					DriverID:    driverID,
					VehicleID:   vehicleID,
					RouteID:     routeID,
					ManifestID:  manifestID,
					Status:      "LOADED",
				},
			})
			orderIDs = append(orderIDs, orderID)
		}

		if !isTopOff {
			queued = append(queued,
				pendingEvent{
					aggregateType: events.AggregateRoute,
					aggregateID:   routeID,
					payload: events.RouteEvent{
						BaseEvent:   events.BaseEvent{Type: events.EventRouteCreated},
						RouteID:     routeID,
						ManifestID:  manifestID,
						SupplierID:  sid,
						WarehouseID: whID,
						DriverID:    driverID,
						VehicleID:   vehicleID,
						OrderIDs:    orderIDs,
					},
				},
				pendingEvent{
					aggregateType: events.AggregateManifest,
					aggregateID:   manifestID,
					payload: events.ManifestEvent{
						BaseEvent:   events.BaseEvent{Type: events.EventManifestDraftCreated},
						ManifestID:  manifestID,
						RouteID:     routeID,
						SupplierID:  sid,
						WarehouseID: whID,
						DriverID:    driverID,
						StopCount:   stopCount,
					},
				},
			)
		}

		committed = append(committed, DispatchExecuteRoute{
			ManifestID: manifestID,
			RouteID:    routeID,
			DriverID:   driverID,
			VehicleID:  vehicleID,
			OrderIDs:   orderIDs,
			VolumeVU:   newVolumeVU,
			MaxVolume:  truckMaxVU,
		})
		out.OrdersAssigned += len(orderIDs)
	}

	if len(committed) == 0 {
		return out, nil
	}

	store := manifest.NewStore(s.spannerClient)
	if err := store.CommitSupplier(ctx, batch, func(buf outbox.TxnBuffer) error {
		versions := manifest.OrderPatchVersionByID(batch.OrderPatches)
		for _, evt := range queued {
			payload := evt.payload
			if oe, ok := payload.(events.OrderEvent); ok {
				if version, ok := versions[oe.OrderID]; ok {
					oe.Version = version
					oe.BaseEvent.Version = version
					payload = oe
				}
			}
			if err := outbox.EmitJSON(ctx, buf, evt.aggregateType, evt.aggregateID, events.TopicMain, payload); err != nil {
				return err
			}
		}
		// Emit SplitShipmentCreated for each split group so payment service
		// and driver/retailer apps receive the deduplication contract.
		for _, grp := range out.SplitShipmentGroups {
			splitEvt := events.SplitShipmentEvent{
				BaseEvent:     events.BaseEvent{Type: events.EventSplitShipmentCreated},
				SplitGroupID:  grp.SplitGroupID,
				SupplierID:    sid,
				WarehouseID:   whID,
				RetailerID:    grp.RetailerID,
				SharedRouteID: grp.SharedRouteID,
				OrderIDs:      grp.OrderIDs,
				ManifestIDs:   grp.ManifestIDs,
				DriverIDs:     grp.DriverIDs,
				TruckCount:    len(grp.ManifestIDs),
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateRoute, grp.SharedRouteID, events.TopicMain, splitEvt); err != nil {
				return err
			}
		}
		if req.ForceCapacity && len(capacityWarnings) > 0 {
			if auditBuf, ok := buf.(outbox.TxnAuditBuffer); ok {
				actorID := warehouseActorID(ctx)
				for _, warning := range capacityWarnings {
					if err := outbox.WriteAudit(ctx, auditBuf, sid, actorID, "WAREHOUSE_ADMIN", "DISPATCH_CAPACITY_OVERRIDE", "DispatchRoute", warning.DriverID, map[string]any{
						"warehouse_id":                 whID,
						"loaded_vu":                    warning.LoadedVU,
						"max_volume_vu":                warning.MaxVolumeVU,
						"effective_max_vu":             warning.EffectiveMaxVU,
						"excess_vu":                    warning.ExcessVU,
						"suggested_unselect_order_ids": warning.SuggestedUnselectOrderIDs,
					}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}); err != nil {
		return out, fmt.Errorf("commit dispatch: %w", err)
	}

	out.Status = "dispatched"
	out.ManifestsCreated = len(committed)
	out.Manifests = committed
	if s.manifestStore != nil && len(committed) > 0 {
		manifestIDs := make([]string, 0, len(committed))
		for _, route := range committed {
			manifestIDs = append(manifestIDs, route.ManifestID)
		}
		s.manifestStore.PersistDispatchPreviewGeometries(ctx, manifestIDs)
	}
	return out, nil
}


func (s *Service) resolveDispatchSupplierID(ctx context.Context, warehouseID string) string {
	if ops := auth.GetWarehouseOps(ctx); ops != nil && strings.TrimSpace(ops.SupplierID) != "" {
		return strings.TrimSpace(ops.SupplierID)
	}
	if sid, ok := auth.ResolveSupplierID(ctx); ok && strings.TrimSpace(sid) != "" {
		return strings.TrimSpace(sid)
	}
	return strings.TrimSpace(s.supplierID)
}

func (s *Service) loadDispatchDrivers(ctx context.Context, warehouseID string) ([]PortalDriver, error) {
	if s.opsDrivers != nil {
		return s.opsDrivers(ctx, warehouseID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]PortalDriver(nil), s.drivers...), nil
}

func (s *Service) loadDispatchLocks(ctx context.Context, warehouseID string) map[string]DispatchLock {
	locks, err := s.repo.GetLocks(ctx, warehouseID)
	if err != nil {
		s.log.WarnContext(ctx, "dispatch locks read failed", "warehouse_id", warehouseID, "err", err)
		return map[string]DispatchLock{}
	}
	return locks
}

func (s *Service) isWarehouseDispatchLocked(ctx context.Context, warehouseID string) (bool, string) {
	locks := s.loadDispatchLocks(ctx, warehouseID)
	for _, lock := range locks {
		entityType := strings.ToUpper(strings.TrimSpace(lock.EntityType))
		entityID := strings.TrimSpace(lock.EntityID)
		if entityType == "WAREHOUSE" && (entityID == warehouseID || entityID == "warehouse-scope") {
			return true, "warehouse_dispatch_locked"
		}
	}
	return false, ""
}

func filterLockedOrders(ctx context.Context, s *Service, warehouseID string, rows []dispatch.DispatchableOrder) []dispatch.DispatchableOrder {
	locks := s.loadDispatchLocks(ctx, warehouseID)
	if len(locks) == 0 {
		return rows
	}
	lockedOrders := make(map[string]struct{})
	for _, lock := range locks {
		if strings.EqualFold(lock.EntityType, "ORDER") {
			lockedOrders[strings.TrimSpace(lock.EntityID)] = struct{}{}
		}
	}
	if len(lockedOrders) == 0 {
		return rows
	}
	filtered := make([]dispatch.DispatchableOrder, 0, len(rows))
	for _, row := range rows {
		if _, blocked := lockedOrders[row.OrderID]; blocked {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func filterLockedDrivers(warehouseID string, drivers []PortalDriver, locks map[string]DispatchLock) []PortalDriver {
	if len(locks) == 0 {
		return drivers
	}
	lockedDrivers := make(map[string]struct{})
	for _, lock := range locks {
		if strings.EqualFold(lock.EntityType, "DRIVER") {
			lockedDrivers[strings.TrimSpace(lock.EntityID)] = struct{}{}
		}
	}
	if len(lockedDrivers) == 0 {
		return drivers
	}
	filtered := make([]PortalDriver, 0, len(drivers))
	for _, driver := range drivers {
		if _, blocked := lockedDrivers[driver.DriverID]; blocked {
			continue
		}
		filtered = append(filtered, driver)
	}
	return filtered
}

func buildManualAssignment(rows []dispatch.DispatchableOrder, drivers []PortalDriver, fleetCtx fleetDispatchContext, manualRoutes []DispatchExecuteRoute) *dispatch.AssignmentResult {
	assignment := &dispatch.AssignmentResult{}
	orderMap := make(map[string]dispatch.DispatchableOrder, len(rows))
	for _, row := range rows {
		orderMap[row.OrderID] = row
	}
	driverMap := make(map[string]PortalDriver, len(drivers))
	for _, d := range drivers {
		driverMap[d.DriverID] = d
	}
	for _, mr := range manualRoutes {
		route := dispatch.DispatchRoute{DriverID: mr.DriverID}
		if d, ok := driverMap[mr.DriverID]; ok {
			route.MaxVolume = d.MaxVolumeVU
		}
		for _, oid := range mr.OrderIDs {
			if o, ok := orderMap[oid]; ok {
				route.Orders = append(route.Orders, o.ToGeo())
				route.LoadedVolume += o.VolumeVU
			}
		}
		if top := fleetCtx.topOffFor(mr.DriverID); top != nil {
			route.LoadedVolume += top.TotalVolumeVU
		}
		if len(route.Orders) > 0 {
			assignment.Routes = append(assignment.Routes, route)
		}
	}
	return assignment
}

func manualCapacityWarnings(routes []dispatch.DispatchRoute, drivers []PortalDriver, fleetCtx fleetDispatchContext) []DispatchCapacityWarning {
	driverMap := make(map[string]PortalDriver, len(drivers))
	for _, d := range drivers {
		driverMap[d.DriverID] = d
	}
	warnings := make([]DispatchCapacityWarning, 0)
	for _, route := range routes {
		maxVU := route.MaxVolume
		if d, ok := driverMap[route.DriverID]; ok && d.MaxVolumeVU > 0 {
			maxVU = d.MaxVolumeVU
		}
		if maxVU <= 0 {
			continue
		}
		effective := maxVU * dispatch.TetrisBuffer
		if route.LoadedVolume <= effective {
			continue
		}
		checkRoute := route
		if top := fleetCtx.topOffFor(route.DriverID); top != nil {
			checkRoute.LoadedVolume = route.LoadedVolume - top.TotalVolumeVU
		}
		suggested, excess := dispatch.SuggestOrdersToUnselect(checkRoute)
		warnings = append(warnings, DispatchCapacityWarning{
			DriverID:                  route.DriverID,
			LoadedVU:                  route.LoadedVolume,
			MaxVolumeVU:               maxVU,
			EffectiveMaxVU:            effective,
			ExcessVU:                  excess,
			SuggestedUnselectOrderIDs: suggested,
		})
	}
	return warnings
}

func autoCapacityWarnings(assignment *dispatch.AssignmentResult, drivers []PortalDriver, fleetCtx fleetDispatchContext, requestedVU, fleetVU float64) []DispatchCapacityWarning {
	if assignment == nil {
		return nil
	}
	warnings := manualCapacityWarnings(assignment.Routes, drivers, fleetCtx)
	if len(warnings) > 0 || requestedVU <= fleetVU {
		return enrichAutoCapacityWarnings(warnings, requestedVU, fleetVU, assignment.Orphans)
	}
	if requestedVU > fleetVU {
		deferrals := orphanOrderIDs(assignment.Orphans)
		warnings = append(warnings, DispatchCapacityWarning{
			DriverID:                  "fleet",
			LoadedVU:                  requestedVU,
			MaxVolumeVU:               fleetVU / dispatch.TetrisBuffer,
			EffectiveMaxVU:            fleetVU,
			ExcessVU:                  requestedVU - fleetVU,
			SuggestedDeferOrderIDs:    deferrals,
			FleetEffectiveCapacityVU:  fleetVU,
			RequestedVolumeVU:         requestedVU,
		})
	}
	return warnings
}

func enrichAutoCapacityWarnings(warnings []DispatchCapacityWarning, requestedVU, fleetVU float64, orphans []dispatch.GeoOrder) []DispatchCapacityWarning {
	deferrals := orphanOrderIDs(orphans)
	for i := range warnings {
		warnings[i].FleetEffectiveCapacityVU = fleetVU
		warnings[i].RequestedVolumeVU = requestedVU
		if len(deferrals) > 0 {
			warnings[i].SuggestedDeferOrderIDs = deferrals
		}
	}
	if len(deferrals) > 0 && len(warnings) == 0 {
		warnings = append(warnings, DispatchCapacityWarning{
			DriverID:                 "fleet",
			LoadedVU:                 requestedVU,
			EffectiveMaxVU:           fleetVU,
			SuggestedDeferOrderIDs:   deferrals,
			FleetEffectiveCapacityVU: fleetVU,
			RequestedVolumeVU:        requestedVU,
		})
	}
	return warnings
}

func orphanOrderIDs(orphans []dispatch.GeoOrder) []string {
	ids := make([]string, 0, len(orphans))
	for _, o := range orphans {
		if id := strings.TrimSpace(o.OrderID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
