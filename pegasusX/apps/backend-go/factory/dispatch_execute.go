package factory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"google.golang.org/api/iterator"
)

// FactoryDispatchRequest is the solver-class input for factory → warehouse dispatch.
type FactoryDispatchRequest struct {
	FactoryID       string
	SupplierID      string
	Mode            string
	TransferIDs     []string
	Routes          []FactoryDispatchRoute
	ForceCapacity   bool
	AcceptPartial   bool
	PlanFingerprint string
	Reason          string
}

// FactoryDispatchRoute is one MANUAL truck assignment (transfer_ids analog of order_ids).
type FactoryDispatchRoute struct {
	DriverID    string   `json:"driver_id"`
	VehicleID   string   `json:"vehicle_id,omitempty"`
	TransferIDs []string `json:"transfer_ids"`
	OrderIDs    []string `json:"order_ids"` // accepted as transfer ids (warehouse-shaped clients)
}

// FactoryDispatchResult is the honesty JSON for POST /v1/factory/dispatch.
type FactoryDispatchResult struct {
	Status               string                             `json:"status"`
	SupplierID           string                             `json:"supplier_id,omitempty"`
	FactoryID            string                             `json:"factory_id,omitempty"`
	CreatedManifestCount int                                `json:"created_manifest_count"`
	ManifestsCreated     int                                `json:"manifests_created"`
	ManifestID           string                             `json:"manifest_id"`
	ManifestIDs          []string                           `json:"manifest_ids,omitempty"`
	TransferCount        int                                `json:"transfer_count"`
	TruckPlate           string                             `json:"truck_plate,omitempty"`
	StopCount            int                                `json:"stop_count,omitempty"`
	Unassigned           []string                           `json:"unassigned,omitempty"`
	OrphanTransferIDs    []string                           `json:"orphan_transfer_ids,omitempty"`
	OptimizerSource      string                             `json:"optimizer_source,omitempty"`
	OptimizerClass       string                             `json:"optimizer_class,omitempty"`
	DispatchAlgo         string                             `json:"dispatch_algo,omitempty"`
	PlanFingerprint      string                             `json:"plan_fingerprint,omitempty"`
	Warnings             []string                           `json:"warnings,omitempty"`
	CapacityWarnings     []factoryCapacityWarning           `json:"capacity_warnings,omitempty"`
	OverflowWarnings     []dispatch.RetailerOverflowWarning `json:"overflow_warnings,omitempty"`
	Manifests            []FactoryDispatchRouteOut          `json:"manifests,omitempty"`
	UpdatedAt            string                             `json:"updated_at,omitempty"`
}

// FactoryDispatchRouteOut is one committed factory manifest.
type FactoryDispatchRouteOut struct {
	ManifestID  string   `json:"manifest_id"`
	RouteID     string   `json:"route_id,omitempty"`
	DriverID    string   `json:"driver_id"`
	VehicleID   string   `json:"vehicle_id,omitempty"`
	TransferIDs []string `json:"transfer_ids"`
	VolumeVU    float64  `json:"volume_vu"`
	MaxVolume   float64  `json:"max_volume_vu"`
}

type factoryCapacityWarning struct {
	DriverID                  string   `json:"driver_id"`
	LoadedVU                  float64  `json:"loaded_vu"`
	MaxVolumeVU               float64  `json:"max_volume_vu"`
	EffectiveMaxVU            float64  `json:"effective_max_vu"`
	ExcessVU                  float64  `json:"excess_vu,omitempty"`
	SuggestedUnselectOrderIDs []string `json:"suggested_unselect_order_ids,omitempty"`
	SuggestedDeferOrderIDs    []string `json:"suggested_defer_order_ids,omitempty"`
	FleetEffectiveCapacityVU  float64  `json:"fleet_effective_capacity_vu,omitempty"`
	RequestedVolumeVU         float64  `json:"requested_volume_vu,omitempty"`
}

type factoryDispatchTransfer struct {
	TransferID     string
	FactoryID      string
	SupplierID     string
	WarehouseID    string
	WarehouseName  string
	VolumeVU       float64
	Lat            float64
	Lng            float64
	ReassignDepth  int64
	ExceptionCount int64
	CreatedAt      time.Time
}

func emptyFactoryDispatchResult(factoryID, supplierID, algo string) FactoryDispatchResult {
	return FactoryDispatchResult{
		Status:               "no_op",
		FactoryID:            factoryID,
		SupplierID:           supplierID,
		CreatedManifestCount: 0,
		ManifestsCreated:     0,
		ManifestID:           "",
		ManifestIDs:          []string{},
		Unassigned:           []string{},
		OptimizerClass:       OptimizerHeuristic,
		DispatchAlgo:         algo,
	}
}

func (s *Service) handleSolverDispatch(w http.ResponseWriter, r *http.Request, body []byte) {
	if s.idem != nil && idempotencyKeyFromRequest(r) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "idempotency_key_required"})
		return
	}
	var req struct {
		Mode            string                 `json:"mode"`
		TransferIDs     []string               `json:"transfer_ids"`
		Routes          []FactoryDispatchRoute `json:"routes"`
		DriverID        string                 `json:"driver_id"`
		VehicleID       string                 `json:"vehicle_id"`
		ForceCapacity   bool                   `json:"force_capacity"`
		AcceptPartial   bool                   `json:"accept_partial"`
		PlanFingerprint string                 `json:"plan_fingerprint"`
		Reason          string                 `json:"reason"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	if strings.EqualFold(strings.TrimSpace(req.Mode), "MANUAL") && len(req.Routes) == 0 && strings.TrimSpace(req.DriverID) != "" {
		req.Routes = []FactoryDispatchRoute{{
			DriverID:    req.DriverID,
			VehicleID:   req.VehicleID,
			TransferIDs: req.TransferIDs,
		}}
	}

	out, err := s.ExecuteFactoryDispatch(r.Context(), FactoryDispatchRequest{
		FactoryID:       s.resolveFactoryNode(r.Context()),
		SupplierID:      s.resolveSupplierScope(r.Context()),
		Mode:            req.Mode,
		TransferIDs:     req.TransferIDs,
		Routes:          req.Routes,
		ForceCapacity:   req.ForceCapacity,
		AcceptPartial:   req.AcceptPartial,
		PlanFingerprint: req.PlanFingerprint,
		Reason:          req.Reason,
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "factory dispatch failed", "err", err)
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "dispatch_failed", err)
		return
	}

	status := http.StatusOK
	switch out.Status {
	case "plan_stale", "capacity_exceeded", "capacity_overflow":
		status = http.StatusConflict
	}
	if out.Status == "dispatched" {
		for _, m := range out.Manifests {
			s.invalidateFactoryKeys(r.Context(), factoryManifestKey(m.ManifestID), factoryManifestListKey(out.SupplierID), factoryTransferListKey(out.SupplierID))
		}
		s.broadcastFactoryEvent(r.Context(), events.EventManifestDraftCreated, map[string]any{
			"manifest_ids":           out.ManifestIDs,
			"created_manifest_count": out.CreatedManifestCount,
			"transfer_count":         out.TransferCount,
			"optimizer_source":       out.OptimizerSource,
			"optimizer_class":        out.OptimizerClass,
			"dispatch_algo":          out.DispatchAlgo,
			"updated_at":             out.UpdatedAt,
		})
		s.log.InfoContext(r.Context(), "factory.dispatch.committed",
			"trace_id", outbox.TraceIDFromContext(r.Context()),
			"factory_id", out.FactoryID,
			"manifests", out.CreatedManifestCount,
			"optimizer_source", out.OptimizerSource,
		)
	}
	s.writeIdempotentJSON(w, r, body, status, out)
}

// ExecuteFactoryDispatch runs the warehouse solver class on factory→warehouse transfers
// and commits FactoryTruckManifests only. Empty queue is a no-op (no invent, no outbox).
func (s *Service) ExecuteFactoryDispatch(ctx context.Context, req FactoryDispatchRequest) (FactoryDispatchResult, error) {
	fid := strings.TrimSpace(req.FactoryID)
	sid := strings.TrimSpace(req.SupplierID)
	if sid == "" {
		sid = s.resolveSupplierScope(ctx)
	}
	out := emptyFactoryDispatchResult(fid, sid, plan.SourceFallbackPhase1)
	if s.spannerClient == nil {
		return out, fmt.Errorf("dispatch_unavailable")
	}
	if fid == "" || sid == "" {
		out.Status = "no_op"
		out.Warnings = append(out.Warnings, "factory_scope_required")
		return out, nil
	}

	transfers, err := s.fetchFactoryDispatchTransfers(ctx, fid, sid, req.TransferIDs)
	if err != nil {
		return out, err
	}
	if len(transfers) == 0 {
		out.DispatchAlgo = plan.SourcePureSmallBatch
		out.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		return out, nil
	}

	driverInputs, vehicleByDriver, driverMaxVU, err := s.loadFactoryDispatchFleet(ctx, fid, sid)
	if err != nil {
		return out, err
	}

	rows := factoryTransfersToOrders(transfers)
	fp := factoryDispatchFingerprint(rows, driverInputs, req.TransferIDs)
	out.PlanFingerprint = fp
	mode := strings.ToUpper(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "AUTO"
	}

	if fpCheck := strings.TrimSpace(req.PlanFingerprint); fpCheck != "" && mode != "MANUAL" {
		if fpCheck != fp {
			out.Status = "plan_stale"
			out.Warnings = append(out.Warnings, "plan_fingerprint_mismatch")
			return out, nil
		}
	}

	var assignment *dispatch.AssignmentResult
	var source string
	if mode == "MANUAL" {
		source = "manual"
		assignment = factoryBuildManualAssignment(rows, req.Routes, driverMaxVU)
	} else {
		fleet := dispatch.BuildAvailableFleet(driverInputs, nil)
		if len(fleet) == 0 {
			out.Warnings = append(out.Warnings, "no_available_drivers")
			out.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
			return out, nil
		}
		depot := s.resolveFactoryDepot(ctx, fid)
		job := plan.BuildSolveJob(ctx, sid, fid, depot, rows, fleet)
		var opt *optimizerclient.Client
		if s != nil {
			opt = s.optimizerClient
		}
		assignment, source, err = plan.OptimizeAndValidate(ctx, opt, job)
		if err != nil {
			return out, fmt.Errorf("optimize factory dispatch: %w", err)
		}
	}

	out.OptimizerSource = source
	out.OptimizerClass = plan.OptimizerClass(source)
	out.DispatchAlgo = source
	if assignment != nil {
		out.Warnings = append(out.Warnings, assignment.Warnings...)
		orphans := make([]string, 0, len(assignment.Orphans))
		for _, o := range assignment.Orphans {
			if id := strings.TrimSpace(o.OrderID); id != "" {
				orphans = append(orphans, id)
			}
		}
		out.OrphanTransferIDs = orphans
		out.Unassigned = orphans
		if len(assignment.OverflowWarnings) > 0 {
			out.OverflowWarnings = assignment.OverflowWarnings
			if !req.AcceptPartial && !req.ForceCapacity {
				out.Status = "capacity_overflow"
				out.Warnings = append(out.Warnings, "warehouse_drop_exceeds_max_truck")
				return out, nil
			}
		}
	}
	if assignment == nil || len(assignment.Routes) == 0 {
		out.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		return out, nil
	}

	capWarnings := factoryCapacityFromDispatch(assignment, driverMaxVU, rows)
	if mode == "MANUAL" {
		if len(capWarnings) > 0 && !req.ForceCapacity {
			out.CapacityWarnings = capWarnings
			out.Status = "capacity_exceeded"
			out.Warnings = append(out.Warnings, "capacity_exceeded")
			return out, nil
		}
		if req.ForceCapacity && len(capWarnings) > 0 {
			out.CapacityWarnings = capWarnings
			out.Warnings = append(out.Warnings, "capacity_override")
		}
	} else if len(capWarnings) > 0 || len(out.OrphanTransferIDs) > 0 {
		out.CapacityWarnings = capWarnings
		if !req.ForceCapacity && !req.AcceptPartial {
			out.Status = "capacity_exceeded"
			out.Warnings = append(out.Warnings, "capacity_exceeded")
			return out, nil
		}
		if req.ForceCapacity && len(capWarnings) > 0 {
			out.Warnings = append(out.Warnings, "capacity_override")
		}
	}

	now := s.now().UTC()
	assignment.Routes = dispatch.ExpandOversizeRoutes(assignment.Routes, now.UnixMilli())

	byID := make(map[string]factoryDispatchTransfer, len(transfers))
	for _, t := range transfers {
		byID[t.TransferID] = t
	}

	batch := &manifest.FactoryWriteBatch{}
	committed := make([]FactoryDispatchRouteOut, 0, len(assignment.Routes))
	type pendingEvent struct {
		aggregateType string
		aggregateID   string
		payload       any
	}
	queued := make([]pendingEvent, 0, len(assignment.Routes)+len(transfers))
	assignedCount := 0

	for _, route := range assignment.Routes {
		driverID := strings.TrimSpace(route.DriverID)
		if driverID == "" || len(route.Orders) == 0 {
			continue
		}
		vehicleID := strings.TrimSpace(vehicleByDriver[driverID])
		manifestID := uuid.NewString()
		routeID := strings.TrimSpace(route.RouteID)
		if routeID == "" {
			routeID = "AUTO-" + driverID + "-" + strconv.FormatInt(now.UnixMilli(), 10)
		}
		stopCount := int64(len(route.Orders))
		maxVU := route.MaxVolume
		if maxVU <= 0 {
			maxVU = driverMaxVU[driverID]
		}
		batch.Manifests = append(batch.Manifests, manifest.FactoryTruckRow{
			ManifestID:    manifestID,
			FactoryID:     fid,
			SupplierID:    sid,
			DriverID:      driverID,
			VehicleID:     vehicleID,
			State:         manifestStateDraft,
			TotalVolumeVU: route.LoadedVolume,
			MaxVolumeVU:   maxVU,
			StopCount:     stopCount,
			TransferCount: stopCount,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
		transferIDs := make([]string, 0, len(route.Orders))
		for _, stop := range route.Orders {
			tid := strings.TrimSpace(stop.OrderID)
			src, ok := byID[tid]
			if !ok || tid == "" {
				continue
			}
			batch.Transfers = append(batch.Transfers, manifest.FactoryTransferRow{
				TransferID:     src.TransferID,
				FactoryID:      src.FactoryID,
				SupplierID:     src.SupplierID,
				ManifestID:     manifestID,
				State:          "ASSIGNED",
				TotalVolumeVU:  src.VolumeVU,
				DriverID:       driverID,
				VehicleID:      vehicleID,
				ReassignDepth:  src.ReassignDepth,
				ExceptionCount: src.ExceptionCount,
				CreatedAt:      src.CreatedAt,
				UpdatedAt:      now,
			})
			transferIDs = append(transferIDs, tid)
		}
		if len(transferIDs) == 0 {
			continue
		}
		queued = append(queued, pendingEvent{
			aggregateType: events.AggregateManifest,
			aggregateID:   manifestID,
			payload: events.ManifestEvent{
				BaseEvent:      events.BaseEvent{Type: events.EventManifestDraftCreated},
				ManifestID:     manifestID,
				ManifestDomain: events.ManifestDomainFactory,
				SupplierID:     sid,
				FactoryID:      fid,
				RouteID:        routeID,
				TransferCount:  len(transferIDs),
				TotalVolumeVU:  int64(route.LoadedVolume),
				DriverID:       driverID,
				VehicleID:      vehicleID,
			},
		})
		committed = append(committed, FactoryDispatchRouteOut{
			ManifestID:  manifestID,
			RouteID:     routeID,
			DriverID:    driverID,
			VehicleID:   vehicleID,
			TransferIDs: transferIDs,
			VolumeVU:    route.LoadedVolume,
			MaxVolume:   maxVU,
		})
		assignedCount += len(transferIDs)
	}

	if len(committed) == 0 {
		out.UpdatedAt = now.Format(time.RFC3339Nano)
		return out, nil
	}

	store := manifest.NewStore(s.spannerClient)
	if err := store.CommitFactory(ctx, batch, func(buf outbox.TxnBuffer) error {
		for _, evt := range queued {
			if err := outbox.EmitJSON(ctx, buf, evt.aggregateType, evt.aggregateID, events.TopicMain, evt.payload); err != nil {
				return err
			}
		}
		if req.ForceCapacity && len(capWarnings) > 0 {
			if auditBuf, ok := buf.(outbox.TxnAuditBuffer); ok {
				actorID := factoryActorID(ctx)
				for _, warning := range capWarnings {
					if err := outbox.WriteAudit(ctx, auditBuf, sid, actorID, "FACTORY_ADMIN", "DISPATCH_CAPACITY_OVERRIDE", "FactoryTruckManifest", warning.DriverID, map[string]any{
						"factory_id":                   fid,
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
		return out, fmt.Errorf("commit factory dispatch: %w", err)
	}

	ids := make([]string, 0, len(committed))
	for _, c := range committed {
		ids = append(ids, c.ManifestID)
	}
	out.Status = "dispatched"
	out.CreatedManifestCount = len(committed)
	out.ManifestsCreated = len(committed)
	out.ManifestID = ids[0]
	out.ManifestIDs = ids
	out.TransferCount = assignedCount
	out.StopCount = assignedCount
	out.Manifests = committed
	out.UpdatedAt = now.Format(time.RFC3339Nano)
	if committed[0].VehicleID != "" {
		out.TruckPlate = committed[0].VehicleID
	}
	return out, nil
}

func factoryTransfersToOrders(transfers []factoryDispatchTransfer) []dispatch.DispatchableOrder {
	rows := make([]dispatch.DispatchableOrder, 0, len(transfers))
	for _, t := range transfers {
		rows = append(rows, dispatch.DispatchableOrder{
			OrderID:      t.TransferID,
			RetailerID:   t.WarehouseID,
			RetailerName: t.WarehouseName,
			WarehouseID:  t.WarehouseID,
			Status:       "CREATED",
			Lat:          t.Lat,
			Lng:          t.Lng,
			VolumeVU:     t.VolumeVU,
			H3Cell:       dispatch.H3CellLookup(t.Lat, t.Lng),
		})
	}
	return rows
}

func factoryBuildManualAssignment(rows []dispatch.DispatchableOrder, routes []FactoryDispatchRoute, driverMaxVU map[string]float64) *dispatch.AssignmentResult {
	inputs := make([]dispatch.ManualRouteInput, 0, len(routes))
	for _, r := range routes {
		ids := append([]string{}, r.TransferIDs...)
		ids = append(ids, r.OrderIDs...)
		inputs = append(inputs, dispatch.ManualRouteInput{
			DriverID: r.DriverID,
			OrderIDs: ids,
		})
	}
	return dispatch.BuildManualAssignment(rows, inputs, driverMaxVU)
}

func factoryCapacityFromDispatch(assignment *dispatch.AssignmentResult, driverMaxVU map[string]float64, rows []dispatch.DispatchableOrder) []factoryCapacityWarning {
	if assignment == nil {
		return nil
	}
	raw := dispatch.ManualCapacityWarnings(assignment.Routes, driverMaxVU)
	out := make([]factoryCapacityWarning, 0, len(raw)+1)
	for _, w := range raw {
		out = append(out, factoryCapacityWarning{
			DriverID:                  w.DriverID,
			LoadedVU:                  w.LoadedVU,
			MaxVolumeVU:               w.MaxVolumeVU,
			EffectiveMaxVU:            w.EffectiveMaxVU,
			ExcessVU:                  w.ExcessVU,
			SuggestedUnselectOrderIDs: w.SuggestedUnselectOrderIDs,
		})
	}
	requested := 0.0
	for _, r := range rows {
		requested += r.VolumeVU
	}
	fleetVU := 0.0
	for _, maxVU := range driverMaxVU {
		if maxVU > 0 {
			fleetVU += maxVU * dispatch.TetrisBuffer
		}
	}
	if requested > fleetVU && fleetVU > 0 {
		deferrals := make([]string, 0, len(assignment.Orphans))
		for _, o := range assignment.Orphans {
			if id := strings.TrimSpace(o.OrderID); id != "" {
				deferrals = append(deferrals, id)
			}
		}
		out = append(out, factoryCapacityWarning{
			DriverID:                 "fleet",
			LoadedVU:                 requested,
			MaxVolumeVU:              fleetVU / dispatch.TetrisBuffer,
			EffectiveMaxVU:           fleetVU,
			ExcessVU:                 requested - fleetVU,
			SuggestedDeferOrderIDs:   deferrals,
			FleetEffectiveCapacityVU: fleetVU,
			RequestedVolumeVU:        requested,
		})
	}
	return out
}

func factoryDispatchFingerprint(rows []dispatch.DispatchableOrder, fleet []dispatch.FleetDriverInput, filter []string) string {
	parts := make([]string, 0, len(rows)+len(fleet)+1)
	if len(filter) > 0 {
		filtered := append([]string(nil), filter...)
		sort.Strings(filtered)
		parts = append(parts, "transfers:"+strings.Join(filtered, ","))
	} else {
		ids := make([]string, 0, len(rows))
		for _, o := range rows {
			ids = append(ids, o.OrderID)
		}
		sort.Strings(ids)
		parts = append(parts, "transfers:"+strings.Join(ids, ","))
	}
	vols := make([]string, 0, len(rows))
	for _, o := range rows {
		vols = append(vols, o.OrderID+":"+strconv.FormatFloat(o.VolumeVU, 'f', 2, 64))
	}
	sort.Strings(vols)
	parts = append(parts, "vol:"+strings.Join(vols, ","))
	drivers := make([]string, 0, len(fleet))
	for _, d := range fleet {
		drivers = append(drivers, d.DriverID+":"+d.VehicleID+":"+strconv.FormatFloat(d.MaxVolumeVU, 'f', 2, 64))
	}
	sort.Strings(drivers)
	parts = append(parts, "fleet:"+strings.Join(drivers, ","))
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func (s *Service) fetchFactoryDispatchTransfers(ctx context.Context, factoryID, supplierID string, filterIDs []string) ([]factoryDispatchTransfer, error) {
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT t.TransferId, t.FactoryId, t.SupplierId, COALESCE(t.WarehouseId, ''),
		             t.TotalVolumeVU, COALESCE(t.ReassignDepth, 0), COALESCE(t.ExceptionCount, 0), t.CreatedAt,
		             IFNULL(w.Lat, 0), IFNULL(w.Lng, 0), COALESCE(w.Name, '')
		      FROM FactoryInternalTransfers t
		      LEFT JOIN Warehouses w ON t.WarehouseId = w.WarehouseId
		      WHERE t.FactoryId = @fid AND t.SupplierId = @sid
		        AND t.State IN UNNEST(@states)
		        AND (t.ManifestId IS NULL OR t.ManifestId = '')`,
		Params: map[string]any{
			"fid":    factoryID,
			"sid":    supplierID,
			"states": []string{TransferStateCreated, TransferStateApproved},
		},
	})
	defer iter.Stop()
	allow := map[string]struct{}{}
	for _, id := range filterIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			allow[id] = struct{}{}
		}
	}
	var out []factoryDispatchTransfer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fetch factory transfers: %w", err)
		}
		var t factoryDispatchTransfer
		if err := row.Columns(&t.TransferID, &t.FactoryID, &t.SupplierID, &t.WarehouseID, &t.VolumeVU, &t.ReassignDepth, &t.ExceptionCount, &t.CreatedAt, &t.Lat, &t.Lng, &t.WarehouseName); err != nil {
			continue
		}
		if len(allow) > 0 {
			if _, ok := allow[t.TransferID]; !ok {
				continue
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) loadFactoryDispatchFleet(ctx context.Context, factoryID, supplierID string) ([]dispatch.FleetDriverInput, map[string]string, map[string]float64, error) {
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT d.DriverId, d.Name, COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, ''), COALESCE(v.MaxVolumeVU, 0)
		      FROM Drivers d
		      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
		      WHERE d.SupplierId = @sid AND d.IsActive = TRUE AND d.OnShift = TRUE
		        AND d.HomeNodeType = 'FACTORY' AND d.HomeNodeId = @fid`,
		Params: map[string]any{"sid": supplierID, "fid": factoryID},
	})
	defer iter.Stop()
	var inputs []dispatch.FleetDriverInput
	vehicleByDriver := map[string]string{}
	driverMaxVU := map[string]float64{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch factory fleet: %w", err)
		}
		var in dispatch.FleetDriverInput
		if err := row.Columns(&in.DriverID, &in.DriverName, &in.VehicleID, &in.VehicleClass, &in.MaxVolumeVU); err != nil {
			continue
		}
		in.IsActive = true
		in.HomeNodeID = factoryID
		in.MaxVolumeVU = dispatch.ResolveMaxVolumeVU(in.VehicleClass, in.MaxVolumeVU)
		inputs = append(inputs, in)
		if vid := strings.TrimSpace(in.VehicleID); vid != "" {
			vehicleByDriver[in.DriverID] = vid
		}
		if in.MaxVolumeVU > 0 {
			driverMaxVU[in.DriverID] = in.MaxVolumeVU
		}
	}
	return inputs, vehicleByDriver, driverMaxVU, nil
}

func (s *Service) resolveFactoryDepot(ctx context.Context, factoryID string) dispatch.DepotCoords {
	row, err := s.spannerClient.Single().ReadRow(ctx, "Factories", spanner.Key{factoryID}, []string{"Lat", "Lng"})
	if err != nil {
		return dispatch.DepotCoords{}
	}
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&lat, &lng); err != nil {
		return dispatch.DepotCoords{}
	}
	return dispatch.DepotCoords{Lat: lat.Float64, Lng: lng.Float64}
}

func factoryActorID(ctx context.Context) string {
	if claims, ok := auth.FromContext(ctx); ok && strings.TrimSpace(claims.Subject) != "" {
		return strings.TrimSpace(claims.Subject)
	}
	return "factory_ops"
}
