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
	WarehouseID   string
	SupplierID    string
	Mode          string
	Routes        []DispatchExecuteRoute
	ForceCapacity bool
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
	if len(rows) == 0 {
		return out, nil
	}

	solveDrivers, err := s.loadDispatchDrivers(ctx, whID)
	if err != nil {
		return out, err
	}
	solveDrivers = filterLockedDrivers(whID, solveDrivers, s.loadDispatchLocks(ctx, whID))

	var driverInputs []dispatch.FleetDriverInput
	vehicleByDriver := make(map[string]string)
	for _, driver := range solveDrivers {
		if !strings.EqualFold(driver.TruckStatus, "AVAILABLE") {
			continue
		}
		driverInputs = append(driverInputs, dispatch.FleetDriverInput{
			DriverID:     driver.DriverID,
			DriverName:   driver.Name,
			VehicleID:    driver.VehicleID,
			VehicleClass: driver.VehicleClass,
			MaxVolumeVU:  driver.MaxVolumeVU,
			IsActive:     driver.IsActive,
			TruckStatus:  driver.TruckStatus,
			HomeNodeID:   whID,
		})
		vehicleByDriver[strings.TrimSpace(driver.DriverID)] = strings.TrimSpace(driver.VehicleID)
	}

	var assignment *dispatch.AssignmentResult
	var source string

	if strings.ToUpper(req.Mode) == "MANUAL" {
		source = "manual"
		assignment = buildManualAssignment(rows, solveDrivers, req.Routes)
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
	}
	if assignment == nil || len(assignment.Routes) == 0 {
		return out, nil
	}

	if strings.ToUpper(req.Mode) == "MANUAL" {
		capacityWarnings := manualCapacityWarnings(assignment.Routes)
		if len(capacityWarnings) > 0 {
			out.CapacityWarnings = capacityWarnings
			if !req.ForceCapacity {
				out.Status = "capacity_exceeded"
				out.Warnings = append(out.Warnings, "capacity_exceeded")
				return out, nil
			}
			out.Warnings = append(out.Warnings, "capacity_override")
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
		manifestID := uuid.NewString()
		routeID := uuid.NewString()
		vehicleID := strings.TrimSpace(vehicleByDriver[driverID])
		batch.Manifests = append(batch.Manifests, manifest.SupplierTruckRow{
			ManifestID:    manifestID,
			SupplierID:    sid,
			WarehouseID:   whID,
			RouteID:       routeID,
			TruckID:       vehicleID,
			DriverID:      driverID,
			State:         warehouseDispatchExecuteManifestState,
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

	if len(committed) == 0 {
		return out, nil
	}

	store := manifest.NewStore(s.spannerClient)
	if err := store.CommitSupplier(ctx, batch, func(buf outbox.TxnBuffer) error {
		for _, evt := range queued {
			if err := outbox.EmitJSON(ctx, buf, evt.aggregateType, evt.aggregateID, events.TopicMain, evt.payload); err != nil {
				return err
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

func buildManualAssignment(rows []dispatch.DispatchableOrder, drivers []PortalDriver, manualRoutes []DispatchExecuteRoute) *dispatch.AssignmentResult {
	assignment := &dispatch.AssignmentResult{}
	orderMap := make(map[string]dispatch.DispatchableOrder, len(rows))
	for _, row := range rows {
		orderMap[row.OrderID] = row
	}
	for _, mr := range manualRoutes {
		route := dispatch.DispatchRoute{DriverID: mr.DriverID}
		for _, d := range drivers {
			if d.DriverID == mr.DriverID {
				route.MaxVolume = d.MaxVolumeVU
				break
			}
		}
		for _, oid := range mr.OrderIDs {
			if o, ok := orderMap[oid]; ok {
				route.Orders = append(route.Orders, o.ToGeo())
				route.LoadedVolume += o.VolumeVU
			}
		}
		if len(route.Orders) > 0 {
			assignment.Routes = append(assignment.Routes, route)
		}
	}
	return assignment
}

func manualCapacityWarnings(routes []dispatch.DispatchRoute) []DispatchCapacityWarning {
	warnings := make([]DispatchCapacityWarning, 0)
	for _, route := range routes {
		maxVU := route.MaxVolume
		if maxVU <= 0 {
			continue
		}
		effective := maxVU * dispatch.TetrisBuffer
		if route.LoadedVolume <= effective {
			continue
		}
		warnings = append(warnings, DispatchCapacityWarning{
			DriverID:      route.DriverID,
			LoadedVU:      route.LoadedVolume,
			MaxVolumeVU:   maxVU,
			EffectiveMaxVU: effective,
		})
	}
	return warnings
}
