package warehouse

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

func (s *Service) HandleOpsDrivers(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/drivers")
	path = strings.Trim(path, "/")
	if parts := strings.Split(path, "/"); len(parts) == 2 && parts[1] == "assign-vehicle" {
		s.handleAssignVehicle(w, r, parts[0])
		return
	}
	if path != "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		whID := warehouseIDFromRequest(r)
		if s.opsDrivers != nil {
			drivers, err := s.opsDrivers(r.Context(), whID)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]any{"drivers": drivers})
				return
			}
			s.log.WarnContext(r.Context(), "ops drivers query failed, falling back", "err", err)
		}
		s.ensurePortalSeed()
		s.mu.RLock()
		drivers := append([]PortalDriver(nil), s.drivers...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"drivers": drivers})
	case http.MethodPost:
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

		var req struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
			Pin   string `json:"pin"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		driverID := "drv-" + uuid.NewString()[:8]
		pin := strings.TrimSpace(req.Pin)
		if pin == "" {
			var err error
			pin, err = generateOpsDriverPIN(4)
			if err != nil {
				s.log.ErrorContext(r.Context(), "failed to generate driver pin", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_generate_driver_pin"})
				return
			}
		}
		pinHash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
		if err != nil {
			s.log.ErrorContext(r.Context(), "failed to hash driver pin", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_hash_driver_pin"})
			return
		}

		if s.spannerClient != nil {
			now := s.now().UTC()
			whID := warehouseIDFromRequest(r)
			if err := s.createOpsDriverSpanner(r.Context(), opsDriverCreateParams{
				DriverID:    driverID,
				Name:        strings.TrimSpace(req.Name),
				Phone:       strings.TrimSpace(req.Phone),
				PinHash:     string(pinHash),
				SupplierID:  strings.TrimSpace(s.resolveSupplierScope(r.Context())),
				WarehouseID: whID,
				CreatedAt:   now,
			}); err != nil {
				s.log.ErrorContext(r.Context(), "failed to create driver", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_driver"})
				return
			}
		} else {
			driver := PortalDriver{
				DriverID:    driverID,
				Name:        strings.TrimSpace(req.Name),
				Phone:       strings.TrimSpace(req.Phone),
				TruckStatus: "AVAILABLE",
				IsActive:    true,
			}
			s.mu.Lock()
			s.drivers = append(s.drivers, driver)
			s.mu.Unlock()
		}

		resp := map[string]any{
			"driver_id": driverID,
			"pin":       pin,
		}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusCreated, resp)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleAssignVehicle(w http.ResponseWriter, r *http.Request, driverID string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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

	var req struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	vehicleID := strings.TrimSpace(req.VehicleID)
	whID := warehouseIDFromRequest(r)
	if s.spannerClient != nil {
		if err := s.persistDriverVehicleAssignment(r.Context(), whID, driverID, vehicleID); err != nil {
			var fleetErr *FleetMutationError
			if errors.As(err, &fleetErr) {
				writeJSON(w, fleetErr.StatusCode, map[string]string{"error": fleetErr.Code, "message": fleetErr.Message})
				return
			}
			if strings.Contains(err.Error(), "driver_not_found") {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "driver_not_found"})
				return
			}
			s.log.ErrorContext(r.Context(), "assign vehicle failed", "driver_id", driverID, "vehicle_id", vehicleID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "assign_vehicle_failed"})
			return
		}
		resp := map[string]string{"status": "assigned", "driver_id": driverID, "vehicle_id": vehicleID}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusOK, resp)
		return
	}

	s.ensurePortalSeed()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.drivers {
		if s.drivers[i].DriverID != driverID {
			continue
		}
		s.drivers[i].VehicleID = vehicleID
		resp := map[string]string{"status": "assigned", "driver_id": driverID, "vehicle_id": vehicleID}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "driver_not_found"})
}

func (s *Service) persistDriverVehicleAssignment(ctx context.Context, warehouseID, driverID, vehicleID string) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_not_configured")
	}
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	now := s.now().UTC()
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		state, err := readDriverAssignmentState(ctx, txn, sid, warehouseID, driverID)
		if err != nil {
			if errors.Is(err, errDriverNotFound) {
				s.log.WarnContext(ctx, "driver not found for vehicle assignment", "driver_id", driverID, "sid", sid, "warehouse_id", warehouseID)
				return fmt.Errorf("driver_not_found: %w", err)
			}
			s.log.ErrorContext(ctx, "failed to read driver assignment state", "driver_id", driverID, "sid", sid, "warehouse_id", warehouseID, "err", err)
			return err
		}
		activeDriverOrders, err := countActiveOrdersForDriver(ctx, txn, driverID)
		if err != nil {
			return err
		}
		if err := driverAssignmentGuard(state, activeDriverOrders); err != nil {
			return err
		}
		if vehicleID != "" {
			if activeVehicleOrders, err := countActiveOrdersForVehicle(ctx, txn, vehicleID); err != nil {
				return err
			} else if activeVehicleOrders > 0 {
				return &FleetMutationError{
					StatusCode: http.StatusConflict,
					Code:       "vehicle_active_orders",
					Message:    fmt.Sprintf("vehicle %s has active orders and cannot be reassigned", vehicleID),
				}
			}
			conflict, err := readDriverByVehicle(ctx, txn, sid, warehouseID, vehicleID, driverID)
			if err != nil {
				return err
			}
			if conflict != nil {
				conflictOrders, err := countActiveOrdersForDriver(ctx, txn, conflict.DriverID)
				if err != nil {
					return err
				}
				if guardErr := driverAssignmentGuard(*conflict, conflictOrders); guardErr != nil {
					return &FleetMutationError{
						StatusCode: http.StatusConflict,
						Code:       "vehicle_driver_active",
						Message:    fmt.Sprintf("vehicle %s is assigned to active driver %s", vehicleID, conflict.DriverID),
					}
				}
			}
		}

		homeNodeID := state.HomeNodeID
		if homeNodeID == "" {
			homeNodeID = warehouseID
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Drivers", map[string]any{
				"DriverId":  driverID,
				"VehicleId": nullableWarehouseString(vehicleID),
				"UpdatedAt": now,
			}),
		}
		if vehicleID != "" {
			clearStmt := spanner.Statement{
				SQL: `SELECT DriverId FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode}
				      WHERE HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @wid AND VehicleId = @vid AND DriverId != @driverId`,
				Params: map[string]any{"wid": homeNodeID, "vid": vehicleID, "driverId": driverID},
			}
			iter := txn.Query(ctx, clearStmt)
			defer iter.Stop()
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				var otherDriverID string
				if err := row.Columns(&otherDriverID); err != nil {
					return err
				}
				mutations = append(mutations, spanner.UpdateMap("Drivers", map[string]any{
					"DriverId":  otherDriverID,
					"VehicleId": spanner.NullString{},
					"UpdatedAt": now,
				}))
			}
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "Driver", driverID, events.TopicDispatch, map[string]any{
			"type":         "DRIVER_VEHICLE_ASSIGNED",
			"driver_id":    driverID,
			"vehicle_id":   vehicleID,
			"warehouse_id": warehouseID,
			"supplier_id":  sid,
		}); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}
		return txn.BufferWrite(mutations)
	})
	return err
}

func (s *Service) HandleOpsVehicles(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/vehicles")
	path = strings.Trim(path, "/")
	if path != "" {
		s.handleOpsVehicleByID(w, r, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		whID := warehouseIDFromRequest(r)
		if s.opsVehicles != nil {
			vehicles, err := s.opsVehicles(r.Context(), whID)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]any{
					"vehicles": wirePortalVehicles(vehicles),
					"total":    len(vehicles),
				})
				return
			}
			s.log.WarnContext(r.Context(), "ops vehicles query failed, falling back", "err", err)
		}
		s.ensurePortalSeed()
		s.mu.RLock()
		vehicles := append([]PortalVehicle(nil), s.vehicles...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"vehicles": wirePortalVehicles(vehicles),
			"total":    len(vehicles),
		})
	case http.MethodPost:
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

		var req struct {
			Label        string  `json:"label"`
			LicensePlate string  `json:"license_plate"`
			VehicleClass string  `json:"vehicle_class"`
			MaxVolumeVU  float64 `json:"max_volume_vu"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		vehicleID := "veh-" + uuid.NewString()[:8]

		if s.spannerClient != nil {
			now := s.now().UTC()
			whID := warehouseIDFromRequest(r)
			maxVU := resolveVehicleMaxVU(req.VehicleClass, req.MaxVolumeVU)
			if err := s.createOpsVehicleSpanner(r.Context(), opsVehicleCreateParams{
				VehicleID:    vehicleID,
				Label:        strings.TrimSpace(req.Label),
				LicensePlate: strings.TrimSpace(req.LicensePlate),
				VehicleClass: strings.TrimSpace(req.VehicleClass),
				MaxVolumeVU:  maxVU,
				SupplierID:   strings.TrimSpace(s.resolveSupplierScope(r.Context())),
				WarehouseID:  whID,
				CreatedAt:    now,
			}); err != nil {
				s.log.ErrorContext(r.Context(), "failed to create vehicle", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_vehicle"})
				return
			}
		} else {
			vehicle := PortalVehicle{
				VehicleID:    vehicleID,
				Label:        req.Label,
				LicensePlate: req.LicensePlate,
				VehicleClass: req.VehicleClass,
				IsActive:     true,
			}
			s.mu.Lock()
			s.vehicles = append(s.vehicles, vehicle)
			s.mu.Unlock()
		}

		resp := map[string]any{
			"vehicle_id":    vehicleID,
			"label":         req.Label,
			"license_plate": req.LicensePlate,
			"vehicle_class": req.VehicleClass,
			"max_volume_vu": resolveVehicleMaxVU(req.VehicleClass, req.MaxVolumeVU),
			"is_active":     true,
		}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusCreated, resp)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOpsVehicleByID(w http.ResponseWriter, r *http.Request, vehicleID string) {
	switch r.Method {
	case http.MethodGet:
		s.handleOpsVehicleGet(w, r, vehicleID)
	case http.MethodPatch:
		s.handleOpsVehiclePatch(w, r, vehicleID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOpsVehicleGet(w http.ResponseWriter, r *http.Request, vehicleID string) {
	vehicleID = strings.TrimSpace(vehicleID)
	if vehicleID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "vehicle_id_required"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	vehicle, ok, err := s.getOpsVehicle(r.Context(), whID, vehicleID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "vehicle_lookup_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vehicle_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vehicle": wirePortalVehicle(vehicle)})
}

func (s *Service) getOpsVehicle(ctx context.Context, warehouseID, vehicleID string) (PortalVehicle, bool, error) {
	if s.opsVehicles != nil {
		vehicles, err := s.opsVehicles(ctx, warehouseID)
		if err != nil {
			return PortalVehicle{}, false, err
		}
		for _, v := range vehicles {
			if v.VehicleID == vehicleID {
				return v, true, nil
			}
		}
		return PortalVehicle{}, false, nil
	}
	s.ensurePortalSeed()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.vehicles {
		if v.VehicleID == vehicleID {
			return v, true, nil
		}
	}
	return PortalVehicle{}, false, nil
}

func (s *Service) handleOpsVehiclePatch(w http.ResponseWriter, r *http.Request, vehicleID string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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

	var req struct {
		IsActive          bool   `json:"is_active"`
		UnavailableReason string `json:"unavailable_reason"`
		UnavailableNote   string `json:"unavailable_note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	now := time.Now().UTC()
	reason := ""
	note := ""
	if !req.IsActive {
		reason = normalizeWarehouseVehicleReason(req.UnavailableReason)
		note = strings.TrimSpace(req.UnavailableNote)
		if reason == VehicleReasonOther && note == "" {
			note = strings.TrimSpace(req.UnavailableReason)
		}
	}
	if s.spannerClient != nil {
		if err := s.patchOpsVehicleSpanner(r.Context(), opsVehiclePatchParams{
			VehicleID:         vehicleID,
			WarehouseID:       whID,
			SupplierID:        s.resolveSupplierScope(r.Context()),
			IsActive:          req.IsActive,
			UnavailableReason: req.UnavailableReason,
			UnavailableNote:   req.UnavailableNote,
			UpdatedAt:         now,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "vehicle_update_failed"})
			return
		}
		idemCommitted = true
		writeJSON(w, http.StatusOK, map[string]any{
			"status":             "updated",
			"vehicle_id":         vehicleID,
			"is_active":          req.IsActive,
			"unavailable_reason": reason,
			"unavailable_note":   note,
		})
		return
	}

	s.ensurePortalSeed()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.vehicles {
		if s.vehicles[i].VehicleID != vehicleID {
			continue
		}
		s.vehicles[i].IsActive = req.IsActive
		s.vehicles[i].UnavailableReason = reason
		s.vehicles[i].UnavailableNote = note
		idemCommitted = true
		writeJSON(w, http.StatusOK, map[string]any{
			"status":             "updated",
			"vehicle_id":         vehicleID,
			"is_active":          req.IsActive,
			"unavailable_reason": reason,
			"unavailable_note":   note,
		})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "vehicle_not_found"})
}

// generateOpsDriverPIN returns a cryptographically random numeric PIN of length n.
func generateOpsDriverPIN(n int) (string, error) {
	if n <= 0 {
		n = 4
	}
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + v.Int64()))
	}
	return b.String(), nil
}
