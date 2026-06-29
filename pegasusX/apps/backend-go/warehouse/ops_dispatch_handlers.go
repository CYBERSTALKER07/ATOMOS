package warehouse

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/httppagination"
)

func (s *Service) handleOpsDispatchPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	limit, offset := httppagination.ParseLimitOffset(r, 300, 5000)

	var previewBody struct {
		OrderIDs []string `json:"order_ids"`
	}
	if r.Method == http.MethodPost && r.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		_ = json.Unmarshal(body, &previewBody)
	}

	undispatched := make([]map[string]any, 0)
	windowConstrained := 0
	dispatchRows := make([]dispatch.DispatchableOrder, 0)
	if s.spannerClient != nil && strings.TrimSpace(s.supplierID) != "" {
		repo := dispatch.NewRepository(s.spannerClient)
		rows, err := repo.FetchDispatchable(r.Context(), dispatch.FetchParams{
			SupplierID:  s.supplierID,
			WarehouseID: whID,
			Limit:       limit,
			Offset:      offset,
		})
		if err == nil {
			allDispatchRows := rows
			preview := dispatch.BuildPreview(allDispatchRows)
			undispatched = make([]map[string]any, 0, len(preview.UndispatchedOrders))
			for _, row := range preview.UndispatchedOrders {
				totalMinor, _ := row["total_minor"].(int64)
				undispatched = append(undispatched, map[string]any{
					"order_id":               row["order_id"],
					"retailer_id":            row["retailer_id"],
					"retailer_name":          row["retailer_name"],
					"total_uzs":              int(totalMinor / 100),
					"total_minor":            totalMinor,
					"currency":               row["currency"],
					"receiving_window_open":  row["receiving_window_open"],
					"receiving_window_close": row["receiving_window_close"],
					"has_receiving_window":   row["has_receiving_window"],
					"volume_vu":              row["volume_vu"],
				})
			}
			windowConstrained = preview.WindowConstrained
			dispatchRows = allDispatchRows
		}
	} else if s.opsOrders != nil {
		rows, err := s.opsOrders(r.Context(), whID, 200)
		if err == nil {
			for _, o := range rows {
				if strings.EqualFold(o.Status, "PENDING") || strings.EqualFold(o.Status, "LOADED") {
					undispatched = append(undispatched, map[string]any{
						"order_id":      o.OrderID,
						"retailer_name": "Retailer " + o.RetailerID,
						"total_uzs":     int(o.TotalMinor / 100),
						"item_count":    3,
					})
				}
			}
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		for _, o := range s.orders {
			if strings.EqualFold(o.Status, "PENDING") {
				undispatched = append(undispatched, map[string]any{
					"order_id":      o.OrderID,
					"retailer_name": "Retailer " + o.RetailerID,
					"total_uzs":     int(o.TotalMinor / 100),
					"item_count":    3,
				})
			}
		}
		s.mu.RUnlock()
	}

	available := make([]map[string]any, 0)
	unavailable := make([]map[string]any, 0)
	var solveDrivers []PortalDriver
	sid := s.resolveDispatchSupplierID(r.Context(), whID)
	if s.opsDrivers != nil {
		drivers, err := s.opsDrivers(r.Context(), whID)
		if err == nil {
			solveDrivers = drivers
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		solveDrivers = append([]PortalDriver(nil), s.drivers...)
		s.mu.RUnlock()
	}
	fleetCtx := fleetDispatchContext{InTransit: map[string]bool{}, TopOff: map[string]manifest.DriverManifestCapacity{}}
	if len(solveDrivers) > 0 {
		var err error
		fleetCtx, err = s.loadFleetDispatchContext(r.Context(), sid, whID, collectWarehouseDriverIDs(solveDrivers))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_preview_failed"})
			return
		}
	}
	selectedRows := filterDispatchRowsByOrderIDs(dispatchRows, previewBody.OrderIDs)
	for _, d := range solveDrivers {
		entry := driverPreviewEntry(d, fleetCtx)
		truckStatus, isUnavailable, reason := warehouseDriverAvailability(d, fleetCtx)
		entry["truck_status"] = truckStatus
		if isUnavailable {
			entry["unavailable_reason"] = reason
			unavailable = append(unavailable, entry)
		} else {
			available = append(available, entry)
		}
	}

	response := map[string]any{
		"preview_ready":               true,
		"undispatched_orders":         undispatched,
		"available_drivers":           available,
		"unavailable_drivers":         unavailable,
		"window_constrained_count":    windowConstrained,
		"fleet_effective_capacity_vu": fleetEffectiveCapacityVU(solveDrivers, fleetCtx),
	}
	if len(previewBody.OrderIDs) > 0 {
		response["selected_orders_volume_vu"] = sumOrderVolumeVU(selectedRows)
	}
	if len(dispatchRows) > 0 && len(solveDrivers) > 0 {
		planMeta, _ := s.solveDispatchPreview(r.Context(), whID, dispatchRows, fleetCtx, solveDrivers, previewBody.OrderIDs)
		for k, v := range planMeta {
			response[k] = v
		}
	}
	w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Page-Offset", strconv.Itoa(offset))
	w.Header().Set("X-Page-Has-More", strconv.FormatBool(len(undispatched) == limit))
	writeJSON(w, http.StatusOK, response)
}

type DispatchExecuteResult struct {
	Status           string                    `json:"status"`
	SupplierID       string                    `json:"supplier_id"`
	WarehouseID      string                    `json:"warehouse_id,omitempty"`
	ManifestsCreated int                       `json:"manifests_created"`
	OrdersAssigned   int                       `json:"orders_assigned"`
	OptimizerSource  string                    `json:"optimizer_source,omitempty"`
	Warnings         []string                  `json:"warnings,omitempty"`
	CapacityWarnings []DispatchCapacityWarning `json:"capacity_warnings,omitempty"`
	Manifests        []DispatchExecuteRoute    `json:"manifests"`
	Orphans          []string                  `json:"orphan_order_ids,omitempty"`
}

type DispatchCapacityWarning struct {
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

type DispatchExecuteRoute struct {
	ManifestID string   `json:"manifest_id"`
	RouteID    string   `json:"route_id"`
	DriverID   string   `json:"driver_id"`
	VehicleID  string   `json:"vehicle_id,omitempty"`
	OrderIDs   []string `json:"order_ids"`
	VolumeVU   float64  `json:"volume_vu"`
	MaxVolume  float64  `json:"max_volume_vu"`
}

func (s *Service) handleOpsDispatchExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !s.requireMutationIdempotencyKey(w, r) {
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "dispatch_unavailable"})
		return
	}
	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	whID := warehouseIDFromRequest(r)
	sid := s.resolveDispatchSupplierID(r.Context(), whID)

	var req struct {
		Mode            string                 `json:"mode"`
		Routes          []DispatchExecuteRoute `json:"routes"`
		OrderIDs        []string               `json:"order_ids"`
		ForceCapacity   bool                   `json:"force_capacity"`
		AcceptPartial   bool                   `json:"accept_partial"`
		PlanFingerprint string                 `json:"plan_fingerprint"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	out, err := s.ExecuteDispatch(r.Context(), DispatchExecuteRequest{
		WarehouseID:     whID,
		SupplierID:      sid,
		Mode:            req.Mode,
		Routes:          req.Routes,
		OrderIDs:        req.OrderIDs,
		ForceCapacity:   req.ForceCapacity,
		AcceptPartial:   req.AcceptPartial,
		PlanFingerprint: req.PlanFingerprint,
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "dispatch execute failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_execute_failed"})
		return
	}

	if out.Status == "dispatched" {
		s.persistDispatchRun(r.Context(), out, req.Mode, warehouseActorID(r.Context()))
		s.log.InfoContext(r.Context(), "dispatch executed",
			"warehouse_id", whID,
			"manifests", out.ManifestsCreated,
			"orders_assigned", out.OrdersAssigned,
			"optimizer_source", out.OptimizerSource,
		)
		s.broadcastWarehouseEvent(r.Context(), whID, map[string]any{
			"type":              "DISPATCH_COMMITTED",
			"trace_id":          outbox.TraceIDFromContext(r.Context()),
			"warehouse_id":      whID,
			"manifests_created": out.ManifestsCreated,
			"orders_assigned":   out.OrdersAssigned,
			"optimizer_source":  out.OptimizerSource,
			"timestamp":         s.now().UTC().Format(time.RFC3339Nano),
		})
	}
	if encoded, err := json.Marshal(out); err == nil {
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, encoded)
		idemCommitted = true
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) handleOpsDispatchSettings(w http.ResponseWriter, r *http.Request) {
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		enabled, err := s.repo.GetAutoDispatch(r.Context(), whID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "failed to get auto_dispatch_enabled", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"warehouse_id":          whID,
			"auto_dispatch_enabled": enabled,
		})

	case http.MethodPatch:
		var payload struct {
			AutoDispatchEnabled *bool `json:"auto_dispatch_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()

		if payload.AutoDispatchEnabled != nil {
			err := s.repo.UpdateAutoDispatch(r.Context(), whID, *payload.AutoDispatchEnabled, func(buf outbox.TxnBuffer) error {
				eventPayload := events.WarehouseEvent{
					BaseEvent:   events.BaseEvent{Type: "WAREHOUSE_DISPATCH_SETTINGS_UPDATED"},
					WarehouseID: whID,
					SupplierID:  s.supplierID,
				}
				return outbox.EmitJSON(r.Context(), buf, events.AggregateWarehouse, whID, events.TopicMain, eventPayload)
			})
			if err != nil {
				s.log.ErrorContext(r.Context(), "failed to update auto_dispatch_enabled", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
				return
			}
			s.broadcastWarehouseEvent(r.Context(), whID, map[string]any{
				"type":                  "WAREHOUSE_DISPATCH_SETTINGS_UPDATED",
				"warehouse_id":          whID,
				"auto_dispatch_enabled": *payload.AutoDispatchEnabled,
			})
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}
