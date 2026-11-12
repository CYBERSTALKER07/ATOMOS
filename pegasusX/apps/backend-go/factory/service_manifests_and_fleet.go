package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (s *Service) loadFactoryTransfersFromSpanner(ctx context.Context) ([]TransferRow, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if fid == "" || sid == "" || s.spannerClient == nil {
		return nil, fmt.Errorf("factory transfer scope required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, OrderId, ManifestId, State, TotalVolumeVU,
		      DriverId, VehicleId, ReassignDepth, ExceptionCount, CreatedAt, UpdatedAt
		      FROM FactoryInternalTransfers@{FORCE_INDEX=Idx_FactoryTransfers_ByFactoryId}
		      WHERE FactoryId = @fid AND SupplierId = @sid
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"fid": fid, "sid": sid},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.spannerClient.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	var out []TransferRow
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("factory transfer query: %w", err)
		}
		var t TransferRow
		var orderID, manifestID, driverID, vehicleID spanner.NullString
		var totalVolume float64
		var reassignDepth, exceptionCount int64
		var createdAt, updatedAt time.Time
		if err := row.Columns(&t.TransferID, &orderID, &manifestID, &t.State, &totalVolume, &driverID, &vehicleID,
			&reassignDepth, &exceptionCount, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("factory transfer scan: %w", err)
		}
		t.OrderID = orderID.StringVal
		t.ManifestID = manifestID.StringVal
		t.TotalVU = int64(totalVolume)
		t.DriverID = driverID.StringVal
		t.VehicleID = vehicleID.StringVal
		t.ReassignDepth = int(reassignDepth)
		t.ExceptionCount = exceptionCount
		t.CreatedAt = createdAt.Format(time.RFC3339Nano)
		t.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) loadFactoryManifestsFromSpanner(ctx context.Context) ([]ManifestRow, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if fid == "" || sid == "" || s.spannerClient == nil {
		return nil, fmt.Errorf("factory manifest scope required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, State, TotalVolumeVU, MaxVolumeVU, StopCount, TransferCount,
		      DriverId, VehicleId, CreatedAt, UpdatedAt
		      FROM FactoryTruckManifests
		      WHERE FactoryId = @fid AND SupplierId = @sid
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"fid": fid, "sid": sid},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.spannerClient.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	var out []ManifestRow
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("factory manifest query: %w", err)
		}
		var m ManifestRow
		var totalVolume, maxVolume float64
		var stopCount, transferCount int64
		var driverID, vehicleID spanner.NullString
		var createdAt, updatedAt time.Time
		if err := row.Columns(&m.ManifestID, &m.State, &totalVolume, &maxVolume, &stopCount, &transferCount,
			&driverID, &vehicleID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("factory manifest scan: %w", err)
		}
		m.TotalVolumeVU = int64(totalVolume)
		m.MaxVolumeVU = int64(maxVolume)
		m.TransferCnt = int(transferCount)
		m.StopCount = int(stopCount)
		m.DriverID = driverID.StringVal
		m.VehicleID = vehicleID.StringVal
		m.TruckID = m.VehicleID
		m.CreatedAt = createdAt.Format(time.RFC3339Nano)
		m.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		out = append(out, m)
	}
	return out, nil
}

func (s *Service) hydrateFactoryManifestsFromSpanner(ctx context.Context) error {
	if s == nil || s.spannerClient == nil {
		return nil
	}
	manifests, err := s.loadFactoryManifestsFromSpanner(ctx)
	if err != nil {
		return err
	}
	transfers, err := s.loadFactoryTransfersFromSpanner(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.manifests = manifests
	s.transfers = transfers
	s.rebuildManifestTransfersLocked()
	s.mu.Unlock()
	return nil
}

// HandleManifests serves GET /v1/factory/manifests.
func (s *Service) HandleManifests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	state := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))

	if err := s.hydrateFactoryManifestsFromSpanner(r.Context()); err != nil {
		s.log.WarnContext(r.Context(), "factory manifest list hydrate failed", "err", err)
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestRow(nil), s.manifests...)
	s.mu.Unlock()

	if state != "" {
		filtered := make([]ManifestRow, 0, len(rows))
		for i := range rows {
			if rows[i].State == state {
				filtered = append(filtered, rows[i])
			}
		}
		rows = filtered
	}
	for i := range rows {
		if rows[i].TruckID == "" {
			rows[i].TruckID = rows[i].VehicleID
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"manifests": rows})
}

// HandleManifestDetail serves GET /v1/factory/manifests/{manifestID}.
func (s *Service) HandleManifestDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := strings.TrimSpace(chi.URLParam(r, "manifestID"))
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	if err := s.hydrateFactoryManifestsFromSpanner(r.Context()); err != nil {
		s.log.WarnContext(r.Context(), "factory manifest detail hydrate failed", "err", err)
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		s.mu.Unlock()
		platform.WriteErrorWithExplain(w, http.StatusNotFound, "manifest_not_found", nil)
		return
	}
	snapshot := s.manifestDetailSnapshotLocked(s.manifests[idx])
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"manifest":      snapshot.Manifest,
		"transfers":     snapshot.Transfers,
		"transitions":   snapshot.Transitions,
		"reassignments": snapshot.Reassignments,
		"exceptions":    snapshot.Exceptions,
		"route_id":      snapshot.RouteID,
		"stop_count":    snapshot.StopCount,
		"order_count":   snapshot.OrderCount,
	})
}

// HandleManifestStartLoading serves POST /v1/factory/manifests/{manifestID}/start-loading.
func (s *Service) HandleManifestStartLoading(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateDraft, manifestStateLoading, "START_LOADING")
}

// HandleManifestSeal serves POST /v1/factory/manifests/{manifestID}/seal.
func (s *Service) HandleManifestSeal(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateLoading, manifestStateSealed, "SEAL")
}

// HandleManifestDispatch serves POST /v1/factory/manifests/{manifestID}/dispatch.
func (s *Service) HandleManifestDispatch(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateSealed, manifestStateDispatched, "DISPATCH")
}

// HandleManifestComplete serves POST /v1/factory/manifests/{manifestID}/complete.
func (s *Service) HandleManifestComplete(w http.ResponseWriter, r *http.Request) {
	s.handleManifestTransition(w, r, manifestStateDispatched, manifestStateCompleted, "COMPLETE")
}

func (s *Service) handleManifestTransition(w http.ResponseWriter, r *http.Request, expectedState, toState, action string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := strings.TrimSpace(chi.URLParam(r, "manifestID"))
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}

	var req transitionRequest
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	eventType := ""
	switch toState {
	case manifestStateLoading:
		eventType = events.EventManifestLoadingStarted
	case manifestStateSealed:
		eventType = events.EventManifestSealed
	case manifestStateDispatched:
		eventType = events.EventManifestDispatched
	case manifestStateCompleted:
		eventType = events.EventManifestCompleted
	}

	var manifest ManifestRow
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		updated, transitionErr := s.transitionManifestLocked(manifestID, expectedState, toState, action, req.Reason)
		if transitionErr != nil {
			return transitionErr
		}
		manifest = updated
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if eventType == "" {
			return nil
		}
		payload := s.manifestOutboxFields(r.Context(), manifest, eventType)
		payload.Reason = strings.TrimSpace(req.Reason)
		payload.Action = action
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifest.ManifestID, events.TopicMain, payload)
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid_state:") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_state", "message": err.Error()})
			return
		}
		if err.Error() == "manifest_not_found" {
			platform.WriteErrorWithExplain(w, http.StatusNotFound, "manifest_not_found", err)
			return
		}
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "transition_failed", err)
		return
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(manifest.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	if eventType != "" {
		s.broadcastFactoryEvent(r.Context(), eventType, map[string]any{
			"manifest_id": manifest.ManifestID,
			"state":       manifest.State,
			"action":      action,
			"reason":      strings.TrimSpace(req.Reason),
			"route_id":    routeIDForManifest(manifest),
			"driver_id":   manifest.DriverID,
			"vehicle_id":  manifest.VehicleID,
			"order_count": manifest.TransferCnt,
			"updated_at":  manifest.UpdatedAt,
		})
	}

	resp := map[string]any{
		"status":      strings.ToLower(action) + "_applied",
		"manifest_id": manifest.ManifestID,
		"state":       manifest.State,
		"updated_at":  manifest.UpdatedAt,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

type factoryFleetSpannerResult struct {
	IOSVehicles   []map[string]any
	FleetVehicles []FleetVehicle
}

func (s *Service) loadFactoryFleetFromSpanner(ctx context.Context, factoryID string) (*factoryFleetSpannerResult, error) {
	if s.spannerClient == nil {
		return nil, nil
	}
	factoryID = strings.TrimSpace(factoryID)
	if factoryID == "" {
		factoryID = s.factoryNodeID
	}
	if factoryID == "" {
		return nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT 
				v.VehicleId,
				v.LicensePlate,
				v.MaxVolumeVU,
				v.IsActive,
				IFNULL(v.UnavailableReason, ''),
				IFNULL(m.ManifestId, ''),
				IFNULL(m.State, ''),
				IFNULL(m.DriverId, ''),
				IFNULL(d.Name, '')
		      FROM Vehicles v
		      LEFT JOIN FactoryTruckManifests m 
		        ON m.VehicleId = v.VehicleId 
		       AND m.FactoryId = @fid 
		       AND m.State IN ('LOADING', 'SEALED', 'DISPATCHED')
		      LEFT JOIN Drivers d 
		        ON d.DriverId = m.DriverId
		      WHERE v.HomeNodeType = 'FACTORY' 
		        AND v.HomeNodeId = @fid
		      ORDER BY v.VehicleId ASC`,
		Params: map[string]any{"fid": factoryID},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.spannerClient.Single().Query(ctx, stmt)
	}
	defer iter.Stop()

	iosList := make([]map[string]any, 0, 8)
	fleetList := make([]FleetVehicle, 0, 8)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query factory fleet spanner: %w", err)
		}
		var (
			vehicleID, licensePlate, unavailReason          string
			manifestID, manifestState, driverID, driverName string
			maxVolumeVU                                     float64
			isActive                                        bool
		)
		if err := row.Columns(&vehicleID, &licensePlate, &maxVolumeVU, &isActive, &unavailReason, &manifestID, &manifestState, &driverID, &driverName); err != nil {
			return nil, fmt.Errorf("scan factory fleet spanner: %w", err)
		}

		status := "READY"
		if !isActive {
			if strings.TrimSpace(unavailReason) != "" {
				status = strings.ToUpper(strings.TrimSpace(unavailReason))
			} else {
				status = "UNAVAILABLE"
			}
		} else if strings.TrimSpace(manifestState) != "" {
			status = strings.ToUpper(strings.TrimSpace(manifestState))
		}

		fleetList = append(fleetList, FleetVehicle{
			VehicleID: vehicleID,
			PlateNo:   licensePlate,
			State:     status,
		})

		capM3 := maxVolumeVU * 0.1
		if capM3 <= 0 {
			capM3 = 12.0
		}
		capKg := maxVolumeVU * 25.0
		if capKg <= 0 {
			capKg = 3200.0
		}
		capL := maxVolumeVU * 100.0
		if capL <= 0 {
			capL = 12000.0
		}

		routeID := ""
		currentRoute := ""
		if strings.TrimSpace(manifestID) != "" {
			routeID = "route_" + strings.TrimSpace(manifestID)
			currentRoute = "Manifest " + strings.TrimSpace(manifestID)
		}

		iosList = append(iosList, map[string]any{
			"id":               vehicleID,
			"plate_number":     licensePlate,
			"capacity_m3":      capM3,
			"capacity_kg":      capKg,
			"capacity_l":       capL,
			"status":           status,
			"driver_name":      strings.TrimSpace(driverName),
			"current_route_id": routeID,
			"current_route":    currentRoute,
			"manifest_id":      strings.TrimSpace(manifestID),
			"driver_id":        strings.TrimSpace(driverID),
		})
	}

	return &factoryFleetSpannerResult{
		IOSVehicles:   iosList,
		FleetVehicles: fleetList,
	}, nil
}

// HandleFleetDrivers serves GET /v1/factory/fleet/drivers.
func (s *Service) HandleFleetDrivers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]FleetDriver(nil), s.fleetDrivers...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"drivers": rows})
}

// HandleFleetVehicles serves GET /v1/factory/fleet/vehicles.
func (s *Service) HandleFleetVehicles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	factoryID, _ := s.scopedFactoryID(r)
	if factoryID == "" {
		factoryID = s.factoryNodeID
	}
	if s.spannerClient != nil {
		res, err := s.loadFactoryFleetFromSpanner(r.Context(), factoryID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "factory fleet vehicles spanner failed", "err", err, "factory_id", factoryID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_vehicles_failed"})
			return
		}
		if res != nil {
			writeJSON(w, http.StatusOK, map[string]any{"vehicles": res.FleetVehicles})
			return
		}
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]FleetVehicle(nil), s.fleetVehicles...)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"vehicles": rows})
}

// HandleStaff serves GET /v1/factory/staff and POST /v1/factory/staff (create).
func (s *Service) HandleStaff(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListStaff(w, r)
		return
	case http.MethodPost:
		s.handleCreateStaff(w, r)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleListStaff(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient != nil {
		rows, err := s.loadFactoryStaffFromSpanner(r.Context())
		if err != nil {
			s.log.WarnContext(r.Context(), "factory staff list failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_list_failed"})
			return
		}
		if len(rows) > 0 || !s.portalSeedEnabled() {
			mapped := make([]map[string]any, len(rows))
			for i := range rows {
				mapped[i] = iosStaffMemberPayload(rows[i])
			}
			writeJSON(w, http.StatusOK, map[string]any{"staff": mapped})
			return
		}
	}
	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]StaffRow(nil), s.staff...)
	s.mu.Unlock()
	if !s.portalSeedEnabled() && s.spannerClient == nil {
		rows = nil
	}
	mapped := make([]map[string]any, len(rows))
	for i := range rows {
		mapped[i] = iosStaffMemberPayload(rows[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"staff": mapped})
}

func (s *Service) loadFactoryStaffFromSpanner(ctx context.Context) ([]StaffRow, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if fid == "" || sid == "" || s.spannerClient == nil {
		return nil, fmt.Errorf("factory staff scope required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT UserId, Name, COALESCE(Phone, ''), SupplierRole
		      FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_BySupplierUpdated}
		      WHERE SupplierId = @sid AND AssignedFactoryId = @fid AND IsActive = true
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"sid": sid, "fid": fid},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.spannerClient.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	var out []StaffRow
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("factory staff query: %w", err)
		}
		var rec StaffRow
		if err := row.Columns(&rec.StaffID, &rec.Name, &rec.Phone, &rec.Role); err != nil {
			return nil, fmt.Errorf("factory staff scan: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *Service) handleCreateStaff(w http.ResponseWriter, r *http.Request) {
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	var req struct {
		Name     string `json:"name"`
		Role     string `json:"role"`
		Phone    string `json:"phone"`
		PIN      string `json:"pin"`
		Password string `json:"password"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Role = strings.TrimSpace(req.Role)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name_required"})
		return
	}
	if req.Role == "" {
		req.Role = "FACTORY_OPERATOR"
	}
	if !factoryStaffRoleAllowed(req.Role) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role_invalid"})
		return
	}
	secret, invite, err := resolveStaffSecret(req.PIN, req.Password)
	if err != nil {
		if errors.Is(err, errStaffSecretTooShort) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pin_or_password_too_short"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_secret_failed"})
		return
	}
	hash, err := hashFactoryStaffSecret(secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_password_hash_failed"})
		return
	}

	supplierID := s.resolveSupplierScope(r.Context())
	factoryID := s.resolveFactoryNode(r.Context())
	var row StaffRow
	err = s.repo.RunTx(r.Context(), func(ctx context.Context, tx FactoryTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		row = StaffRow{
			StaffID:      s.nextIDLocked("stf"),
			Name:         req.Name,
			Role:         req.Role,
			Phone:        req.Phone,
			PasswordHash: hash,
		}
		if err := tx.SaveStaff(ctx, row); err != nil {
			return err
		}
		s.staff = append(s.staff, row)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateFactory, row.StaffID, events.TopicMain, events.FactoryEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventFactoryStaffCreated},
			FactoryID:    factoryID,
			SupplierID:   supplierID,
			UserID:       row.StaffID,
			SupplierRole: row.Role,
		})
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "factory staff create failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "staff_create_failed"})
		return
	}
	s.invalidateFactoryKeys(r.Context(), factoryStaffListKey(supplierID))
	s.broadcastFactoryEvent(r.Context(), events.EventFactoryStaffCreated, map[string]any{
		"staff_id":    row.StaffID,
		"factory_id":  factoryID,
		"supplier_id": supplierID,
		"role":        row.Role,
	})
	s.log.InfoContext(r.Context(), "factory.staff_created", "staff_id", row.StaffID, "factory_id", factoryID)
	payload := iosStaffMemberPayload(row)
	payload["must_set_password"] = invite != ""
	if invite != "" {
		payload["invite_token"] = invite
	}
	s.writeIdempotentJSON(w, r, body, http.StatusCreated, payload)
}

func factoryStaffRoleAllowed(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "FACTORY", "FACTORY_ADMIN", "FACTORY_STAFF", "FACTORY_OPERATOR", "FACTORY_DRIVER":
		return true
	default:
		return false
	}
}

var errNoDispatchableTransfers = errors.New("no_dispatchable_transfers")

func pickDispatchTransfers(transfers []TransferRow, req dispatchRequest) []TransferRow {
	available := make([]TransferRow, 0)
	for i := range transfers {
		if transfers[i].State == "CREATED" {
			available = append(available, transfers[i])
		}
	}
	selected := make([]TransferRow, 0)
	if len(req.TransferIDs) > 0 {
		for _, id := range req.TransferIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			for i := range transfers {
				if transfers[i].TransferID != id {
					continue
				}
				if dispatchableTransferState(transfers[i].State) && strings.TrimSpace(transfers[i].ManifestID) == "" {
					selected = append(selected, transfers[i])
				}
				break
			}
		}
	}
	if len(selected) == 0 {
		limit := len(available)
		if limit > 2 {
			limit = 2
		}
		if limit > 0 {
			selected = append(selected, available[:limit]...)
		}
	}
	return selected
}

// HandleDispatch serves POST /v1/factory/dispatch.
func (s *Service) HandleDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.spannerClient != nil {
		s.handleSolverDispatch(w, r, body)
		return
	}
	var req dispatchRequest
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	preview := pickDispatchTransfers(s.transfers, req)
	s.mu.Unlock()
	if len(preview) == 0 {
		nowTS := s.now().Format(time.RFC3339Nano)
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
			"status":                 "dispatch_planned",
			"created_manifest_count": 0,
			"manifests_created":      0,
			"manifest_id":            "",
			"transfer_count":         0,
			"updated_at":             nowTS,
			"optimizer_class":        OptimizerHeuristic,
			"dispatch_algo":          DispatchAlgoPickN,
		})
		return
	}

	var manifest ManifestRow
	selectedCount := 0
	now := ""
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		now = s.now().Format(time.RFC3339Nano)

		selected := pickDispatchTransfers(s.transfers, req)
		if len(selected) == 0 {
			return errNoDispatchableTransfers
		}

		if req.MaxVolumeVU <= 0 {
			req.MaxVolumeVU = 180
		}
		if strings.TrimSpace(req.DriverID) == "" {
			if len(s.fleetDrivers) == 0 {
				return errNoDispatchableTransfers
			}
			req.DriverID = s.fleetDrivers[0].DriverID
		}
		if strings.TrimSpace(req.VehicleID) == "" {
			if len(s.fleetVehicles) == 0 {
				return errNoDispatchableTransfers
			}
			req.VehicleID = s.fleetVehicles[0].VehicleID
		}

		manifestID := s.nextIDLocked("mf")
		var total int64
		for i := range selected {
			total += selected[i].TotalVU
		}

		manifest = ManifestRow{
			ManifestID:    manifestID,
			State:         manifestStateDraft,
			TransferCnt:   len(selected),
			TotalVolumeVU: total,
			MaxVolumeVU:   req.MaxVolumeVU,
			DriverID:      req.DriverID,
			VehicleID:     req.VehicleID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.manifests = append(s.manifests, manifest)

		transfers := make([]TransferRow, 0, len(selected))
		for _, transfer := range selected {
			transfer.ManifestID = manifestID
			transfer.DriverID = req.DriverID
			transfer.VehicleID = req.VehicleID
			transfer.State = "ASSIGNED"
			transfer.UpdatedAt = now
			transfers = append(transfers, transfer)
			for i := range s.transfers {
				if s.transfers[i].TransferID == transfer.TransferID {
					s.transfers[i] = transfer
					break
				}
			}
		}
		s.manifestTransfers[manifestID] = transfers
		s.appendTransitionLocked(manifestID, "CREATE_DRAFT", "", manifestStateDraft, req.Reason)
		selectedCount = len(selected)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifest.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:     events.BaseEvent{Type: events.EventManifestDraftCreated},
			ManifestID:    manifest.ManifestID,
			SupplierID:    s.resolveSupplierScope(r.Context()),
			FactoryID:     s.factoryNodeID,
			RouteID:       routeIDForManifest(manifest),
			TransferCount: manifest.TransferCnt,
			TotalVolumeVU: manifest.TotalVolumeVU,
			DriverID:      manifest.DriverID,
			VehicleID:     manifest.VehicleID,
		})
	})
	if errors.Is(err, errNoDispatchableTransfers) {
		nowTS := s.now().Format(time.RFC3339Nano)
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
			"status":                 "dispatch_planned",
			"created_manifest_count": 0,
			"manifests_created":      0,
			"manifest_id":            "",
			"transfer_count":         0,
			"updated_at":             nowTS,
			"optimizer_class":        OptimizerHeuristic,
			"dispatch_algo":          DispatchAlgoPickN,
		})
		return
	}
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "dispatch_failed", err)
		return
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(manifest.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestDraftCreated, map[string]any{
		"manifest_id":       manifest.ManifestID,
		"transfer_count":    manifest.TransferCnt,
		"total_volume_vu":   manifest.TotalVolumeVU,
		"route_id":          routeIDForManifest(manifest),
		"driver_id":         manifest.DriverID,
		"vehicle_id":        manifest.VehicleID,
		"created_manifests": 1,
		"updated_at":        now,
	})

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":                 "dispatch_planned",
		"created_manifest_count": 1,
		"manifests_created":      1,
		"manifest_id":            manifest.ManifestID,
		"transfer_count":         selectedCount,
		"updated_at":             now,
		"optimizer_class":        OptimizerHeuristic,
		"dispatch_algo":          DispatchAlgoPickN,
	})
}

// SetPlanning attaches the P5 planning engine (batcher / SLA / pull matrix).
