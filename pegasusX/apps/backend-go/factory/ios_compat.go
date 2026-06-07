package factory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Service) iosDashboardLocked() map[string]any {
	pending := int64(0)
	loading := int64(0)
	activeManifests := int64(len(s.manifests))
	dispatchedToday := int64(0)
	vehiclesAvailable := int64(0)

	for i := range s.transfers {
		switch strings.ToUpper(s.transfers[i].State) {
		case "CREATED", "APPROVED", "PENDING":
			pending++
		case "LOADING", "ASSIGNED":
			loading++
		case "DISPATCHED":
			dispatchedToday++
		}
	}
	for i := range s.manifests {
		state := strings.ToUpper(s.manifests[i].State)
		if state == manifestStateLoading || state == manifestStateDraft {
			activeManifests++
		}
		if state == manifestStateDispatched {
			dispatchedToday++
		}
	}
	for i := range s.fleetVehicles {
		if strings.EqualFold(s.fleetVehicles[i].State, "READY") || strings.EqualFold(s.fleetVehicles[i].State, "AVAILABLE") {
			vehiclesAvailable++
		}
	}
	onShift := int64(0)
	for i := range s.fleetDrivers {
		if s.fleetDrivers[i].OnShift {
			onShift++
		}
	}
	if onShift == 0 {
		onShift = int64(len(s.fleetDrivers))
	}

	return map[string]any{
		"pending_transfers":  pending,
		"loading_transfers":  loading,
		"active_manifests":   activeManifests,
		"dispatched_today":   dispatchedToday,
		"vehicles_total":     int64(len(s.fleetVehicles)),
		"vehicles_available": vehiclesAvailable,
		"staff_on_shift":     onShift,
		"critical_insights":  int64(len(s.manifestExceptions)),
		"supplier_id":        s.supplierID,
		"factory_id":         s.factoryNodeID,
		"updated_at":         s.now().Format(time.RFC3339Nano),
	}
}

func (s *Service) iosTransferPayload(row TransferRow) map[string]any {
	volumeVU := float64(row.TotalVU)
	return map[string]any{
		"id":                       row.TransferID,
		"transfer_id":              row.TransferID,
		"factory_id":               s.factoryNodeID,
		"source_factory_id":        s.factoryNodeID,
		"warehouse_id":             "wh_demo_1",
		"destination_warehouse_id": "wh_demo_1",
		"warehouse_name":           "Demo Warehouse",
		"state":                    row.State,
		"priority":                 "NORMAL",
		"total_items":              1,
		"total_volume_l":           volumeVU * 10,
		"total_volume_m3":          volumeVU,
		"total_volume_vu":          volumeVU,
		"notes":                    "",
		"created_at":               row.CreatedAt,
		"updated_at":               row.UpdatedAt,
		"items":                    []any{},
	}
}

func (s *Service) iosFleetVehiclesLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.fleetVehicles))
	for i := range s.fleetVehicles {
		v := s.fleetVehicles[i]
		out = append(out, map[string]any{
			"id":               v.VehicleID,
			"plate_number":     v.PlateNo,
			"capacity_m3":      12.0,
			"capacity_kg":      3200.0,
			"capacity_l":       12000.0,
			"status":           v.State,
			"driver_name":      "",
			"current_route_id": "",
			"current_route":    "",
		})
	}
	return out
}

// HandleFleet serves GET /v1/factory/fleet (iOS combined fleet snapshot).
func (s *Service) HandleFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	vehicles := s.iosFleetVehiclesLocked()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"vehicles": vehicles})
}

// HandleTransferByID serves GET /v1/factory/transfers/{transferID}.
func (s *Service) HandleTransferByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	transferID := strings.TrimSpace(chi.URLParam(r, "transferID"))
	if transferID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "transfer_id_required"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	var found *TransferRow
	for i := range s.transfers {
		if s.transfers[i].TransferID == transferID {
			row := s.transfers[i]
			found = &row
			break
		}
	}
	s.mu.Unlock()
	if found == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, s.iosTransferPayload(*found))
}

// HandleTransferTransition serves POST /v1/factory/transfers/{transferID}/transition.
func (s *Service) HandleTransferTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	transferID := strings.TrimSpace(chi.URLParam(r, "transferID"))
	if transferID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "transfer_id_required"})
		return
	}
	var req struct {
		TargetState string `json:"target_state"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	target := strings.ToUpper(strings.TrimSpace(req.TargetState))
	if target == "" {
		target = "APPROVED"
	}

	var row TransferRow
	err := s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		idx := -1
		for i := range s.transfers {
			if s.transfers[i].TransferID == transferID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errTransferNotFound
		}
		current := strings.ToUpper(strings.TrimSpace(s.transfers[idx].State))
		if !isValidTransferTransition(current, target) {
			return fmt.Errorf("invalid_transfer_transition:%s->%s", current, target)
		}
		s.transfers[idx].State = target
		s.transfers[idx].UpdatedAt = s.now().Format(time.RFC3339Nano)
		row = s.transfers[idx]
		if manifestID := strings.TrimSpace(row.ManifestID); manifestID != "" {
			for j := range s.manifestTransfers[manifestID] {
				if s.manifestTransfers[manifestID][j].TransferID == transferID {
					s.manifestTransfers[manifestID][j] = row
					break
				}
			}
		}
		return nil
	}, nil)
	if err != nil {
		if err == errTransferNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
			return
		}
		if strings.HasPrefix(err.Error(), "invalid_transfer_transition:") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_transfer_transition"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transfer_transition_failed"})
		return
	}
	s.invalidateFactoryKeys(r.Context(), factoryTransferListKey(s.supplierID))
	writeJSON(w, http.StatusOK, s.iosTransferPayload(row))
}

// HandleStaffDetail serves GET /v1/factory/staff/{staffID}.
func (s *Service) HandleStaffDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	staffID := strings.TrimSpace(chi.URLParam(r, "staffID"))
	s.mu.Lock()
	s.ensureDemoDataLocked()
	var row *StaffRow
	for i := range s.staff {
		if s.staff[i].StaffID == staffID {
			copy := s.staff[i]
			row = &copy
			break
		}
	}
	s.mu.Unlock()
	if row == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "staff_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, iosStaffMemberPayload(*row))
}

func iosStaffMemberPayload(row StaffRow) map[string]any {
	return map[string]any{
		"id":        row.StaffID,
		"staff_id":  row.StaffID,
		"name":      row.Name,
		"role":      row.Role,
		"phone":     "",
		"status":    "ACTIVE",
		"joined_at": "",
	}
}

// HandleReplenishmentInsights serves GET /v1/warehouse/replenishment/insights for factory Android/iOS.
func (s *Service) HandleReplenishmentInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	now := s.now().Format(time.RFC3339Nano)
	insights := []map[string]any{
		{
			"id":                  "ins_factory_1",
			"warehouse_id":        "wh_demo_1",
			"warehouse_name":      "Demo Warehouse",
			"product_id":          "prod_demo_1",
			"product_name":        "Demo SKU",
			"urgency":             "HIGH",
			"current_stock":       12,
			"avg_daily_velocity":  4.5,
			"days_until_stockout": 3,
			"reorder_quantity":    48,
			"status":              "OPEN",
			"created_at":          now,
		},
	}
	writeJSON(w, http.StatusOK, map[string]any{"insights": insights})
}

// HandleSupplyRequestTransition serves PATCH /v1/factory/supply-requests/{id}.
func (s *Service) HandleSupplyRequestTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "id"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}
	var req struct {
		Action          string `json:"action"`
		TransferOrderID string `json:"transfer_order_id"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	nextState := map[string]string{
		"ACKNOWLEDGE":      "ACKNOWLEDGED",
		"START_PRODUCTION": "IN_PRODUCTION",
		"MARK_READY":       "READY",
		"FULFILL":          "FULFILLED",
		"CANCEL":           "CANCELLED",
	}[action]
	if nextState == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_action"})
		return
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	idx := -1
	for i := range s.supplyRequests {
		if s.supplyRequests[i].RequestID == requestID {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
		return
	}
	s.supplyRequests[idx].Status = nextState
	s.supplyRequests[idx].UpdatedAt = s.now().Format(time.RFC3339Nano)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"request_id": requestID,
		"state":      nextState,
	})
}
