package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/pkg/pin"
	"backend-go/spannerx"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ─── Driver DTOs ──────────────────────────────────────────────────────────────

type CreateDriverReq struct {
	Phone        string `json:"phone"`
	Name         string `json:"name"`
	DriverType   string `json:"driver_type"`
	VehicleType  string `json:"vehicle_type"`
	LicensePlate string `json:"license_plate"`
	VehicleId    string `json:"vehicle_id,omitempty"`
}

type DriverItem struct {
	DriverID     string  `json:"driver_id"`
	Name         string  `json:"name"`
	Phone        string  `json:"phone"`
	DriverType   string  `json:"driver_type"`
	VehicleType  string  `json:"vehicle_type"`
	LicensePlate string  `json:"license_plate"`
	IsActive     bool    `json:"is_active"`
	TruckStatus  string  `json:"truck_status"`
	CreatedAt    string  `json:"created_at"`
	VehicleId    string  `json:"vehicle_id,omitempty"`
	VehicleClass string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU  float64 `json:"max_volume_vu,omitempty"`
}

const (
	warehouseDriverStatusAvailable   = "AVAILABLE"
	warehouseDriverStatusIdle        = "IDLE"
	warehouseDriverStatusInTransit   = "IN_TRANSIT"
	warehouseDriverStatusReturning   = "RETURNING"
	warehouseDriverStatusMaintenance = "MAINTENANCE"
)

type warehouseAssignVehicleRequest struct {
	VehicleID string `json:"vehicle_id"`
}

type warehouseAssignVehicleResponse struct {
	Status                   string `json:"status"`
	DriverID                 string `json:"driver_id"`
	VehicleID                string `json:"vehicle_id,omitempty"`
	PreviouslyAssignedDriver string `json:"previously_assigned_driver,omitempty"`
}

type warehouseFleetMutationRuleError struct {
	StatusCode int
	Message    string
}

func (e *warehouseFleetMutationRuleError) Error() string {
	return e.Message
}

type warehouseDriverAssignmentState struct {
	DriverID     string
	SupplierID   string
	HomeNodeType string
	HomeNodeID   string
	WarehouseID  string
	VehicleID    string
	RouteID      string
	TruckStatus  string
}

type warehouseVehicleAssignmentState struct {
	VehicleID    string
	SupplierID   string
	HomeNodeType string
	HomeNodeID   string
	WarehouseID  string
	IsActive     bool
}

// PIN generation is handled by backend-go/pkg/pin.GenerateUnique.

// ─── Handlers ─────────────────────────────────────────────────────────────────

// HandleOpsDrivers — GET (list) / POST (create) for /v1/warehouse/ops/drivers
func HandleOpsDrivers(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listOpsDrivers(w, r, spannerClient)
		case http.MethodPost:
			createOpsDriver(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleOpsDriverDetail — GET /v1/warehouse/ops/drivers/{id}
// Also handles POST /v1/warehouse/ops/drivers/{id}/rotate-pin
func HandleOpsDriverDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/drivers/")

		// Route: POST /v1/warehouse/ops/drivers/{id}/rotate-pin
		if strings.HasSuffix(path, "/rotate-pin") {
			driverID := strings.TrimSuffix(path, "/rotate-pin")
			rotateOpsDriverPIN(w, r, spannerClient, ops, driverID)
			return
		}

		if strings.HasSuffix(path, "/assign-vehicle") {
			driverID := strings.TrimSuffix(path, "/assign-vehicle")
			assignOpsDriverVehicle(w, r, spannerClient, ops, driverID)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		driverID := strings.TrimSuffix(path, "/")
		if driverID == "" || strings.Contains(driverID, "/") {
			http.Error(w, "driver_id required", http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), COALESCE(d.DriverType, ''),
			             COALESCE(d.VehicleType, ''), COALESCE(d.LicensePlate, ''),
			             COALESCE(d.IsActive, true), COALESCE(d.TruckStatus, 'IDLE'),
			             d.CreatedAt,
			             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, ''),
			             COALESCE(v.MaxVolumeVU, 0)
			      FROM Drivers d LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      WHERE d.DriverId = @driverId AND d.SupplierId = @sid AND (d.WarehouseId = @whId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @whId))`,
			Params: map[string]interface{}{
				"driverId": driverID,
				"sid":      ops.SupplierID,
				"whId":     ops.WarehouseID,
			},
		}
		iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("[WH FLEET] driver detail error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var d DriverItem
		var createdAt time.Time
		if err := row.Columns(&d.DriverID, &d.Name, &d.Phone, &d.DriverType,
			&d.VehicleType, &d.LicensePlate, &d.IsActive, &d.TruckStatus,
			&createdAt, &d.VehicleId, &d.VehicleClass, &d.MaxVolumeVU); err != nil {
			log.Printf("[WH FLEET] driver detail parse: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		d.CreatedAt = createdAt.Format(time.RFC3339)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	}
}

func listOpsDrivers(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), COALESCE(d.DriverType, ''),
		             COALESCE(d.VehicleType, ''), COALESCE(d.LicensePlate, ''),
		             COALESCE(d.IsActive, true), COALESCE(d.TruckStatus, 'IDLE'),
		             d.CreatedAt,
		             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, ''),
		             COALESCE(v.MaxVolumeVU, 0)
		      FROM Drivers d LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
		      WHERE d.SupplierId = @sid AND (d.WarehouseId = @whId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @whId))
		      ORDER BY d.CreatedAt DESC`,
		Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
	}

	// Filter by status
	if status := r.URL.Query().Get("status"); status != "" {
		stmt.SQL = strings.Replace(stmt.SQL, "ORDER BY", "AND d.TruckStatus = @filterStatus ORDER BY", 1)
		stmt.Params["filterStatus"] = status
	}
	// Active filter
	if active := r.URL.Query().Get("active"); active == "true" {
		stmt.SQL = strings.Replace(stmt.SQL, "ORDER BY", "AND d.IsActive = true ORDER BY", 1)
	} else if active == "false" {
		stmt.SQL = strings.Replace(stmt.SQL, "ORDER BY", "AND d.IsActive = false ORDER BY", 1)
	}

	iter := spannerx.StaleQuery(r.Context(), client, stmt)
	defer iter.Stop()

	var drivers []DriverItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[WH FLEET] list drivers error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var d DriverItem
		var createdAt time.Time
		if err := row.Columns(&d.DriverID, &d.Name, &d.Phone, &d.DriverType,
			&d.VehicleType, &d.LicensePlate, &d.IsActive, &d.TruckStatus,
			&createdAt, &d.VehicleId, &d.VehicleClass, &d.MaxVolumeVU); err != nil {
			log.Printf("[WH FLEET] parse error: %v", err)
			continue
		}
		d.CreatedAt = createdAt.Format(time.RFC3339)
		drivers = append(drivers, d)
	}
	if drivers == nil {
		drivers = []DriverItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"drivers": drivers, "total": len(drivers)})
}

func createOpsDriver(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"only warehouse admins can create drivers"}`, http.StatusForbidden)
		return
	}

	var req CreateDriverReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Phone == "" || req.Name == "" {
		http.Error(w, `{"error":"phone and name required"}`, http.StatusBadRequest)
		return
	}

	// Check duplicate phone
	dupStmt := spanner.Statement{
		SQL:    `SELECT DriverId FROM Drivers WHERE Phone = @phone LIMIT 1`,
		Params: map[string]interface{}{"phone": req.Phone},
	}
	dupIter := client.Single().Query(r.Context(), dupStmt)
	dupRow, dupErr := dupIter.Next()
	dupIter.Stop()
	if dupErr == nil && dupRow != nil {
		http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
		return
	}

	driverID := uuid.New().String()
	vehicleID := req.VehicleId
	if vehicleID == "" {
		vehicleID = ""
	}

	// V.O.I.D. Phase VII — warehouse-scoped creation is always home-based at
	// the caller's WarehouseId; dual-write HomeNodeType/HomeNodeId alongside
	// the legacy WarehouseId column.
	cols := []string{"DriverId", "SupplierId", "WarehouseId", "Name", "Phone", "PinHash",
		"DriverType", "VehicleType", "LicensePlate", "IsActive", "TruckStatus", "CreatedAt",
		"HomeNodeType", "HomeNodeId"}

	if vehicleID != "" {
		cols = append(cols, "VehicleId")
	}

	driverEvent := kafkaEvents.DriverCreatedEvent{
		DriverID:     driverID,
		SupplierId:   ops.SupplierID,
		Name:         req.Name,
		Phone:        req.Phone,
		DriverType:   req.DriverType,
		HomeNodeType: "WAREHOUSE",
		HomeNodeId:   ops.WarehouseID,
		CreatedBy:    ops.UserID,
		Timestamp:    time.Now().UTC(),
	}
	var pinResult *pin.Result
	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Generate globally unique 8-digit PIN inside the txn for atomicity.
		var pinErr error
		pinResult, pinErr = pin.GenerateUnique(ctx, txn, pin.EntityDriver, driverID)
		if pinErr != nil {
			return fmt.Errorf("generate PIN: %w", pinErr)
		}

		vals := []interface{}{driverID, ops.SupplierID, ops.WarehouseID, req.Name, req.Phone, pinResult.BcryptHash,
			req.DriverType, req.VehicleType, req.LicensePlate, true, "IDLE", spanner.CommitTimestamp,
			"WAREHOUSE", ops.WarehouseID}
		if vehicleID != "" {
			vals = append(vals, vehicleID)
		}

		m := spanner.Insert("Drivers", cols, vals)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		return outbox.EmitJSON(txn, "Driver", driverID, kafkaEvents.EventDriverCreated, kafkaEvents.TopicMain, driverEvent, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[WH FLEET] insert driver error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"driver_id": driverID,
		"name":      req.Name,
		"phone":     req.Phone,
		"pin":       pinResult.Plaintext,
	})
}

// rotateOpsDriverPIN generates a new globally-unique PIN for a warehouse-scoped driver.
// POST /v1/warehouse/ops/drivers/{id}/rotate-pin
func rotateOpsDriverPIN(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, ops *auth.WarehouseOps, driverID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if driverID == "" {
		http.Error(w, `{"error":"driver_id required"}`, http.StatusBadRequest)
		return
	}

	// Verify driver exists and belongs to this warehouse.
	row, err := spannerClient.Single().ReadRow(r.Context(), "Drivers",
		spanner.Key{driverID}, []string{"SupplierId", "WarehouseId", "HomeNodeType", "HomeNodeId"})
	if err != nil {
		http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
		return
	}
	var ownerSID, whID string
	var homeNodeType, homeNodeID spanner.NullString
	if err := row.Columns(&ownerSID, &whID, &homeNodeType, &homeNodeID); err != nil {
		http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
		return
	}
	if ownerSID != ops.SupplierID {
		http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
		return
	}
	scopeMatch := whID == ops.WarehouseID ||
		(homeNodeType.Valid && homeNodeType.StringVal == "WAREHOUSE" && homeNodeID.Valid && homeNodeID.StringVal == ops.WarehouseID)
	if !scopeMatch {
		http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
		return
	}

	var pinResult *pin.Result
	_, err = spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var rotErr error
		pinResult, rotErr = pin.Rotate(ctx, txn, pin.EntityDriver, driverID)
		if rotErr != nil {
			return rotErr
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Drivers", []string{"DriverId", "PinHash"}, []interface{}{driverID, pinResult.BcryptHash}),
		})
	})
	if err != nil {
		log.Printf("[WH FLEET] rotate driver PIN failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"driver_id": driverID,
		"pin":       pinResult.Plaintext,
	})
}

func assignOpsDriverVehicle(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps, driverID string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"warehouse admin required"}`, http.StatusForbidden)
		return
	}
	driverID = strings.TrimSuffix(strings.TrimSpace(driverID), "/")
	if driverID == "" || strings.Contains(driverID, "/") {
		http.Error(w, `{"error":"driver_id required"}`, http.StatusBadRequest)
		return
	}

	var req warehouseAssignVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	targetVehicleID := strings.TrimSpace(req.VehicleID)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := warehouseAssignVehicleResponse{
		Status:    warehouseAssignmentStatus(targetVehicleID),
		DriverID:  driverID,
		VehicleID: targetVehicleID,
	}
	cacheDriverIDs := []string{driverID}

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		driverState, err := readWarehouseDriverAssignmentState(ctx, txn, ops, driverID)
		if err != nil {
			return err
		}
		if ruleErr := warehouseVehicleAssignmentRuleError(driverState); ruleErr != nil {
			return ruleErr
		}
		if driverState.VehicleID == targetVehicleID {
			return nil
		}

		mutations := make([]*spanner.Mutation, 0, 2)
		if targetVehicleID != "" {
			vehicleState, err := readWarehouseVehicleAssignmentState(ctx, txn, ops, targetVehicleID)
			if err != nil {
				return err
			}
			if !vehicleState.IsActive {
				return &warehouseFleetMutationRuleError{StatusCode: http.StatusConflict, Message: "vehicle is inactive and cannot be assigned"}
			}
			if !warehouseFleetHomeNodeMatch(driverState.HomeNodeType, driverState.HomeNodeID, driverState.WarehouseID, vehicleState.HomeNodeType, vehicleState.HomeNodeID, vehicleState.WarehouseID) {
				return &warehouseFleetMutationRuleError{StatusCode: http.StatusForbidden, Message: "vehicle is outside this warehouse fleet scope"}
			}
			conflictingDriver, err := readWarehouseDriverByVehicle(ctx, txn, ops, targetVehicleID, driverID)
			if err != nil {
				return err
			}
			if conflictingDriver != nil {
				if ruleErr := warehouseVehicleAssignmentRuleError(*conflictingDriver); ruleErr != nil {
					return ruleErr
				}
				mutations = append(mutations, spanner.Update(
					"Drivers",
					[]string{"DriverId", "VehicleId"},
					[]interface{}{conflictingDriver.DriverID, warehouseNullableString("")},
				))
				cacheDriverIDs = append(cacheDriverIDs, conflictingDriver.DriverID)
				response.PreviouslyAssignedDriver = conflictingDriver.DriverID
			}
		}

		driverColumns := []string{"DriverId", "VehicleId"}
		driverValues := []interface{}{driverID, warehouseNullableString(targetVehicleID)}
		if targetVehicleID != "" && strings.EqualFold(driverState.TruckStatus, warehouseDriverStatusMaintenance) {
			driverColumns = append(driverColumns, "TruckStatus")
			driverValues = append(driverValues, warehouseDriverStatusAvailable)
		}
		mutations = append(mutations, spanner.Update("Drivers", driverColumns, driverValues))
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		var ruleErr *warehouseFleetMutationRuleError
		if errors.As(err, &ruleErr) {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, ruleErr.Message), ruleErr.StatusCode)
			return
		}
		log.Printf("[WH FLEET] assign vehicle error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	warehouseInvalidateDriverProfiles(ctx, cacheDriverIDs...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func warehouseAssignmentStatus(vehicleID string) string {
	if vehicleID == "" {
		return "UNASSIGNED"
	}
	return "ASSIGNED"
}

func warehouseVehicleAssignmentRuleError(state warehouseDriverAssignmentState) *warehouseFleetMutationRuleError {
	switch strings.ToUpper(strings.TrimSpace(state.TruckStatus)) {
	case warehouseDriverStatusInTransit, warehouseDriverStatusReturning:
		return &warehouseFleetMutationRuleError{
			StatusCode: http.StatusConflict,
			Message:    fmt.Sprintf("driver %s is %s and cannot change vehicle assignment until route completes", state.DriverID, strings.ToUpper(strings.TrimSpace(state.TruckStatus))),
		}
	}
	if strings.TrimSpace(state.RouteID) != "" {
		return &warehouseFleetMutationRuleError{
			StatusCode: http.StatusConflict,
			Message:    fmt.Sprintf("driver %s has an active route and cannot change vehicle assignment", state.DriverID),
		}
	}
	return nil
}

func readWarehouseDriverAssignmentState(ctx context.Context, txn *spanner.ReadWriteTransaction, ops *auth.WarehouseOps, driverID string) (warehouseDriverAssignmentState, error) {
	row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{"SupplierId", "HomeNodeType", "HomeNodeId", "WarehouseId", "VehicleId", "RouteId", "TruckStatus"})
	if err != nil {
		return warehouseDriverAssignmentState{}, &warehouseFleetMutationRuleError{StatusCode: http.StatusNotFound, Message: "driver not found"}
	}

	var state warehouseDriverAssignmentState
	var homeNodeType, homeNodeID, warehouseID, vehicleID, routeID, truckStatus spanner.NullString
	if err := row.Columns(&state.SupplierID, &homeNodeType, &homeNodeID, &warehouseID, &vehicleID, &routeID, &truckStatus); err != nil {
		return warehouseDriverAssignmentState{}, err
	}
	state.DriverID = driverID
	state.HomeNodeType = nullStringValue(homeNodeType)
	state.HomeNodeID = nullStringValue(homeNodeID)
	state.WarehouseID = nullStringValue(warehouseID)
	state.VehicleID = nullStringValue(vehicleID)
	state.RouteID = nullStringValue(routeID)
	state.TruckStatus = strings.TrimSpace(nullStringValue(truckStatus))
	if state.TruckStatus == "" {
		state.TruckStatus = warehouseDriverStatusIdle
	}
	if state.SupplierID != ops.SupplierID || !warehouseHomeNodeMatches(state.HomeNodeType, state.HomeNodeID, state.WarehouseID, ops.WarehouseID) {
		return warehouseDriverAssignmentState{}, &warehouseFleetMutationRuleError{StatusCode: http.StatusNotFound, Message: "driver not found"}
	}
	return state, nil
}

func readWarehouseVehicleAssignmentState(ctx context.Context, txn *spanner.ReadWriteTransaction, ops *auth.WarehouseOps, vehicleID string) (warehouseVehicleAssignmentState, error) {
	row, err := txn.ReadRow(ctx, "Vehicles", spanner.Key{vehicleID}, []string{"SupplierId", "HomeNodeType", "HomeNodeId", "WarehouseId", "IsActive"})
	if err != nil {
		return warehouseVehicleAssignmentState{}, &warehouseFleetMutationRuleError{StatusCode: http.StatusNotFound, Message: "vehicle not found"}
	}

	var state warehouseVehicleAssignmentState
	var homeNodeType, homeNodeID, warehouseID spanner.NullString
	var isActive spanner.NullBool
	if err := row.Columns(&state.SupplierID, &homeNodeType, &homeNodeID, &warehouseID, &isActive); err != nil {
		return warehouseVehicleAssignmentState{}, err
	}
	state.VehicleID = vehicleID
	state.HomeNodeType = nullStringValue(homeNodeType)
	state.HomeNodeID = nullStringValue(homeNodeID)
	state.WarehouseID = nullStringValue(warehouseID)
	state.IsActive = !isActive.Valid || isActive.Bool
	if state.SupplierID != ops.SupplierID || !warehouseHomeNodeMatches(state.HomeNodeType, state.HomeNodeID, state.WarehouseID, ops.WarehouseID) {
		return warehouseVehicleAssignmentState{}, &warehouseFleetMutationRuleError{StatusCode: http.StatusNotFound, Message: "vehicle not found"}
	}
	return state, nil
}

func readWarehouseDriverByVehicle(ctx context.Context, txn *spanner.ReadWriteTransaction, ops *auth.WarehouseOps, vehicleID, excludeDriverID string) (*warehouseDriverAssignmentState, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DriverId, SupplierId, COALESCE(HomeNodeType, ''), COALESCE(HomeNodeId, ''),
		             COALESCE(WarehouseId, ''), COALESCE(VehicleId, ''), COALESCE(RouteId, ''),
		             COALESCE(TruckStatus, 'IDLE')
		      FROM Drivers
		      WHERE SupplierId = @sid AND VehicleId = @vehicleId
		        AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId))
		        AND DriverId != @excludeDriverId
		      LIMIT 1`,
		Params: map[string]interface{}{
			"sid":             ops.SupplierID,
			"vehicleId":       vehicleID,
			"whId":            ops.WarehouseID,
			"excludeDriverId": excludeDriverID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state warehouseDriverAssignmentState
	var homeNodeType, homeNodeID, warehouseID, assignedVehicleID, routeID, truckStatus spanner.NullString
	if err := row.Columns(&state.DriverID, &state.SupplierID, &homeNodeType, &homeNodeID, &warehouseID, &assignedVehicleID, &routeID, &truckStatus); err != nil {
		return nil, err
	}
	state.HomeNodeType = nullStringValue(homeNodeType)
	state.HomeNodeID = nullStringValue(homeNodeID)
	state.WarehouseID = nullStringValue(warehouseID)
	state.VehicleID = nullStringValue(assignedVehicleID)
	state.RouteID = nullStringValue(routeID)
	state.TruckStatus = strings.TrimSpace(nullStringValue(truckStatus))
	if state.TruckStatus == "" {
		state.TruckStatus = warehouseDriverStatusIdle
	}
	return &state, nil
}

func warehouseHomeNodeMatches(homeNodeType, homeNodeID, warehouseID, targetWarehouseID string) bool {
	if strings.TrimSpace(warehouseID) == targetWarehouseID {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(homeNodeType), "WAREHOUSE") && strings.TrimSpace(homeNodeID) == targetWarehouseID
}

func warehouseFleetHomeNodeMatch(driverHomeNodeType, driverHomeNodeID, driverWarehouseID, vehicleHomeNodeType, vehicleHomeNodeID, vehicleWarehouseID string) bool {
	driverNodeType, driverNodeID := effectiveWarehouseFleetHomeNode(driverHomeNodeType, driverHomeNodeID, driverWarehouseID)
	vehicleNodeType, vehicleNodeID := effectiveWarehouseFleetHomeNode(vehicleHomeNodeType, vehicleHomeNodeID, vehicleWarehouseID)
	return driverNodeType != "" && driverNodeType == vehicleNodeType && driverNodeID == vehicleNodeID
}

func effectiveWarehouseFleetHomeNode(homeNodeType, homeNodeID, warehouseID string) (string, string) {
	if strings.TrimSpace(homeNodeType) != "" && strings.TrimSpace(homeNodeID) != "" {
		return strings.TrimSpace(homeNodeType), strings.TrimSpace(homeNodeID)
	}
	if strings.TrimSpace(warehouseID) != "" {
		return "WAREHOUSE", strings.TrimSpace(warehouseID)
	}
	return "", ""
}

func warehouseNullableString(value string) spanner.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: trimmed, Valid: true}
}

func nullStringValue(value spanner.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.StringVal
}

func warehouseInvalidateDriverProfiles(ctx context.Context, driverIDs ...string) {
	if len(driverIDs) == 0 {
		return
	}
	keys := make([]string, 0, len(driverIDs))
	seen := make(map[string]struct{}, len(driverIDs))
	for _, driverID := range driverIDs {
		if strings.TrimSpace(driverID) == "" {
			continue
		}
		if _, exists := seen[driverID]; exists {
			continue
		}
		seen[driverID] = struct{}{}
		keys = append(keys, cache.DriverProfile(driverID))
	}
	if len(keys) > 0 {
		cache.Invalidate(ctx, keys...)
	}
}
