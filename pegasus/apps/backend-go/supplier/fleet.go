package supplier

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
	"backend-go/order"
	"backend-go/outbox"
	"backend-go/pkg/pin"
	"backend-go/spannerx"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

type assignVehicleRuleError struct {
	status           int
	message          string
	conflictDriverID string
}

func (e *assignVehicleRuleError) Error() string {
	return e.message
}

func effectiveFleetHomeNode(nodeType, nodeID, warehouseID string) (string, string) {
	if nodeType != "" && nodeID != "" {
		return nodeType, nodeID
	}
	if warehouseID != "" {
		return auth.HomeNodeTypeWarehouse, warehouseID
	}
	return "", ""
}

func isVehicleAssignmentLocked(status, routeID string) bool {
	if strings.TrimSpace(routeID) != "" {
		return true
	}
	switch strings.TrimSpace(status) {
	case "LOADING", "READY", "IN_TRANSIT", "RETURNING":
		return true
	default:
		return false
	}
}

func writeAssignVehicleError(w http.ResponseWriter, err *assignVehicleRuleError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.status)
	body := map[string]string{"error": err.message}
	if err.conflictDriverID != "" {
		body["conflict_driver_id"] = err.conflictDriverID
	}
	json.NewEncoder(w).Encode(body)
}

// ── Request / Response DTOs ───────────────────────────────────────────────

type CreateDriverRequest struct {
	Phone        string `json:"phone"`
	Name         string `json:"name"`
	DriverType   string `json:"driver_type"`  // IN_HOUSE | CONTRACTOR
	VehicleType  string `json:"vehicle_type"` // Legacy label (Box Truck, etc.)
	LicensePlate string `json:"license_plate"`
	VehicleId    string `json:"vehicle_id,omitempty"` // FK → Vehicles table
	// V.O.I.D. Phase VII — home-node override (GLOBAL_ADMIN only; scoped
	// callers derive the home node from JWT claims).
	HomeNodeType string `json:"home_node_type,omitempty"` // WAREHOUSE | FACTORY
	HomeNodeId   string `json:"home_node_id,omitempty"`
}

type CreateDriverResponse struct {
	DriverID     string  `json:"driver_id"`
	Name         string  `json:"name"`
	Phone        string  `json:"phone"`
	DriverType   string  `json:"driver_type"`
	VehicleType  string  `json:"vehicle_type"`
	LicensePlate string  `json:"license_plate"`
	Pin          string  `json:"pin"` // Plaintext — shown ONCE
	VehicleId    string  `json:"vehicle_id,omitempty"`
	VehicleClass string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU  float64 `json:"max_volume_vu,omitempty"`
}

type DriverListItem struct {
	DriverID          string  `json:"driver_id"`
	Name              string  `json:"name"`
	Phone             string  `json:"phone"`
	DriverType        string  `json:"driver_type"`
	VehicleType       string  `json:"vehicle_type"`
	LicensePlate      string  `json:"license_plate"`
	IsActive          bool    `json:"is_active"`
	TruckStatus       string  `json:"truck_status"`
	CreatedAt         string  `json:"created_at"`
	VehicleId         string  `json:"vehicle_id,omitempty"`
	VehicleClass      string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU       float64 `json:"max_volume_vu,omitempty"`
	EstimatedReturnAt *string `json:"estimated_return_at,omitempty"`
	ReturnDurationSec *int64  `json:"return_duration_sec,omitempty"`
	OfflineReason     string  `json:"offline_reason,omitempty"`
	OfflineReasonNote string  `json:"offline_reason_note,omitempty"`
	OfflineAt         *string `json:"offline_at,omitempty"`
	CurrentLocation   string  `json:"current_location,omitempty"`
}

type DriverDetail struct {
	DriverListItem
	TotalDeliveries int64  `json:"total_deliveries"`
	CurrentLocation string `json:"current_location,omitempty"`
}

// PIN generation is handled by backend-go/pkg/pin.GenerateUnique.

// ── Handlers ──────────────────────────────────────────────────────────────

// HandleFleetDrivers routes GET (list) and POST (create) for /v1/supplier/fleet/drivers
func HandleFleetDrivers(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listDrivers(w, r, spannerClient)
		case http.MethodPost:
			createDriver(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleFleetDriverDetail routes GET for /v1/supplier/fleet/drivers/{id}
// Also handles PATCH /v1/supplier/fleet/drivers/{id}/assign-vehicle
// Also handles POST /v1/supplier/fleet/drivers/{id}/rotate-pin
func HandleFleetDriverDetail(spannerClient *spanner.Client) http.HandlerFunc {
	assignHandler := HandleAssignVehicle(spannerClient)
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/fleet/drivers/")

		// Route: /v1/supplier/fleet/drivers/{id}/assign-vehicle
		if strings.HasSuffix(path, "/assign-vehicle") {
			assignHandler.ServeHTTP(w, r)
			return
		}

		// Route: POST /v1/supplier/fleet/drivers/{id}/rotate-pin
		if strings.HasSuffix(path, "/rotate-pin") {
			driverID := strings.TrimSuffix(path, "/rotate-pin")
			rotateDriverPIN(w, r, spannerClient, driverID)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if path == "" || strings.Contains(path, "/") {
			http.Error(w, "driver_id required in path", http.StatusBadRequest)
			return
		}
		driverID := path

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		// Fetch driver detail
		stmt := spanner.Statement{
			SQL: `SELECT DriverId, Name, COALESCE(Phone, ''), COALESCE(DriverType, ''),
			             COALESCE(VehicleType, ''), COALESCE(LicensePlate, ''),
			             COALESCE(IsActive, true), COALESCE(CurrentLocation, ''),
			             CreatedAt
			      FROM Drivers
			      WHERE DriverId = @driverId AND SupplierId = @supplierId`,
			Params: map[string]interface{}{
				"driverId":   driverID,
				"supplierId": supplierID,
			},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("[FLEET] driver detail query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var d DriverDetail
		var createdAt time.Time
		if err := row.Columns(&d.DriverID, &d.Name, &d.Phone, &d.DriverType,
			&d.VehicleType, &d.LicensePlate, &d.IsActive, &d.CurrentLocation, &createdAt); err != nil {
			log.Printf("[FLEET] driver detail parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		d.CreatedAt = createdAt.Format(time.RFC3339)

		// Count completed deliveries
		countStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Orders WHERE DriverId = @driverId AND State = 'COMPLETED'`,
			Params: map[string]interface{}{
				"driverId": driverID,
			},
		}
		countIter := spannerClient.Single().Query(r.Context(), countStmt)
		defer countIter.Stop()
		countRow, err := countIter.Next()
		if err == nil {
			var count int64
			if err := countRow.Columns(&count); err == nil {
				d.TotalDeliveries = count
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	}
}

// HandleDriverLogin authenticates a driver with phone + PIN
func HandleDriverLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phone string `json:"phone"`
			Pin   string `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Phone == "" || req.Pin == "" {
			http.Error(w, `{"error":"phone and pin required"}`, http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT d.DriverId, d.Name, COALESCE(d.PinHash, ''), COALESCE(d.IsActive, true),
			             COALESCE(d.VehicleType, ''), COALESCE(d.LicensePlate, ''),
			             COALESCE(d.SupplierId, ''), COALESCE(d.VehicleId, ''),
			             COALESCE(v.VehicleClass, ''), COALESCE(v.MaxVolumeVU, 0),
			             COALESCE(d.WarehouseId, ''),
			             COALESCE(w.Name, ''), COALESCE(w.Lat, 0), COALESCE(w.Lng, 0)
			      FROM Drivers d
			      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      LEFT JOIN Warehouses w ON d.WarehouseId = w.WarehouseId
			      WHERE d.Phone = @phone`,
			Params: map[string]interface{}{
				"phone": req.Phone,
			},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("[DRIVER AUTH] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var driverID, name, pinHash, vehicleType, licensePlate, supplierID, vehicleID, vehicleClass string
		var warehouseID, warehouseName string
		var warehouseLat, warehouseLng float64
		var maxVolumeVU float64
		var isActive bool
		if err := row.Columns(&driverID, &name, &pinHash, &isActive, &vehicleType, &licensePlate, &supplierID,
			&vehicleID, &vehicleClass, &maxVolumeVU, &warehouseID, &warehouseName, &warehouseLat, &warehouseLng); err != nil {
			log.Printf("[DRIVER AUTH] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !isActive {
			http.Error(w, `{"error":"account deactivated"}`, http.StatusForbidden)
			return
		}

		if pinHash == "" {
			http.Error(w, `{"error":"no credentials configured"}`, http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(req.Pin)); err != nil {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		token, err := auth.GenerateTestToken(driverID, "DRIVER")
		if err != nil {
			log.Printf("[DRIVER AUTH] token generation error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Mint Firebase custom token (graceful degradation)
		var firebaseToken string
		if auth.FirebaseAuthClient != nil {
			var fbUid string
			if err := spannerClient.Single().Query(r.Context(), spanner.Statement{
				SQL:    "SELECT COALESCE(FirebaseUid, '') FROM Drivers WHERE DriverId = @id",
				Params: map[string]interface{}{"id": driverID},
			}).Do(func(row *spanner.Row) error { return row.Columns(&fbUid) }); err != nil {
				log.Printf("[DRIVER AUTH] firebase UID lookup failed for driver %s: %v", driverID, err)
			}
			if fbUid != "" {
				if token, err := auth.MintCustomToken(r.Context(), fbUid, map[string]interface{}{"role": "DRIVER", "driver_id": driverID, "supplier_id": supplierID}); err != nil {
					log.Printf("[DRIVER AUTH] firebase token mint failed for driver %s: %v", driverID, err)
				} else {
					firebaseToken = token
				}
			}
		}

		resp := map[string]interface{}{
			"token":          token,
			"user_id":        driverID,
			"role":           "DRIVER",
			"name":           name,
			"vehicle_type":   vehicleType,
			"license_plate":  licensePlate,
			"supplier_id":    supplierID,
			"vehicle_id":     vehicleID,
			"vehicle_class":  vehicleClass,
			"max_volume_vu":  maxVolumeVU,
			"warehouse_id":   warehouseID,
			"warehouse_name": warehouseName,
			"warehouse_lat":  warehouseLat,
			"warehouse_lng":  warehouseLng,
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Private Handlers ──────────────────────────────────────────────────────

func createDriver(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req CreateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Phone == "" || req.Name == "" {
		http.Error(w, `{"error":"phone and name are required"}`, http.StatusBadRequest)
		return
	}

	// Validate driver type
	if req.DriverType != "IN_HOUSE" && req.DriverType != "CONTRACTOR" {
		req.DriverType = "IN_HOUSE"
	}

	driverID := fmt.Sprintf("DRV-%s", order.GenerateSecureToken())

	// Check for duplicate phone
	dupStmt := spanner.Statement{
		SQL:    `SELECT DriverId FROM Drivers WHERE Phone = @phone LIMIT 1`,
		Params: map[string]interface{}{"phone": req.Phone},
	}
	dupIter := spannerClient.Single().Query(r.Context(), dupStmt)
	dupRow, dupErr := dupIter.Next()
	dupIter.Stop()
	if dupErr == nil && dupRow != nil {
		http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
		return
	}

	warehouseID := auth.EffectiveWarehouseID(r.Context())

	// V.O.I.D. Phase VII — derive canonical (HomeNodeType, HomeNodeId) from the
	// caller's JWT scope; reject body overrides that try to escape scope.
	homeNodeType, homeNodeID, scopeOK := auth.ApplyHomeNodeOverride(claims, req.HomeNodeType, req.HomeNodeId)
	if !scopeOK {
		http.Error(w, `{"error":"home_node_id outside caller scope"}`, http.StatusForbidden)
		return
	}
	// Dual-write: keep WarehouseId populated whenever HomeNodeType=WAREHOUSE so
	// legacy read paths continue to work during the migration window.
	if homeNodeType == auth.HomeNodeTypeWarehouse && warehouseID == "" {
		warehouseID = homeNodeID
	}

	driverEvent := kafkaEvents.DriverCreatedEvent{
		DriverID:     driverID,
		SupplierId:   supplierID,
		Name:         req.Name,
		Phone:        req.Phone,
		DriverType:   req.DriverType,
		HomeNodeType: homeNodeType,
		HomeNodeId:   homeNodeID,
		CreatedBy:    claims.UserID,
		Timestamp:    time.Now().UTC(),
	}
	var pinResult *pin.Result
	_, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Generate globally unique 8-digit PIN inside the txn for atomicity.
		var pinErr error
		pinResult, pinErr = pin.GenerateUnique(ctx, txn, pin.EntityDriver, driverID)
		if pinErr != nil {
			return fmt.Errorf("generate PIN: %w", pinErr)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("Drivers",
				[]string{"DriverId", "Name", "Phone", "PinHash", "SupplierId", "DriverType",
					"VehicleType", "LicensePlate", "VehicleId", "IsActive", "CreatedAt",
					"WarehouseId", "HomeNodeType", "HomeNodeId"},
				[]interface{}{driverID, req.Name, req.Phone, pinResult.BcryptHash, supplierID,
					req.DriverType, req.VehicleType, req.LicensePlate, nullStr(req.VehicleId), true, spanner.CommitTimestamp,
					nullStr(warehouseID), nullStr(homeNodeType), nullStr(homeNodeID)},
			),
		}); err != nil {
			return err
		}
		return outbox.EmitJSON(txn, "Driver", driverID, kafkaEvents.EventDriverCreated, kafkaEvents.TopicMain, driverEvent, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[FLEET] Spanner insert failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Post-commit notification fan-out (EventType-keyed for dispatcher routing).

	// Create Firebase Auth user for driver (phone-based, graceful degradation)
	fbUid, fbErr := auth.CreateFirebaseUser(r.Context(), "", "", req.Name, req.Phone, "DRIVER", map[string]interface{}{
		"driver_id":   driverID,
		"supplier_id": supplierID,
	})
	if fbErr == nil && fbUid != "" {
		if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Drivers", []string{"DriverId", "FirebaseUid"}, []interface{}{driverID, fbUid}),
			})
		}); err != nil {
			log.Printf("[FLEET] firebase UID mirror failed for driver %s: %v", driverID, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateDriverResponse{
		DriverID:     driverID,
		Name:         req.Name,
		Phone:        req.Phone,
		DriverType:   req.DriverType,
		VehicleType:  req.VehicleType,
		LicensePlate: req.LicensePlate,
		Pin:          pinResult.Plaintext,
		VehicleId:    req.VehicleId,
	})
}

func listDrivers(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	stmt := spanner.Statement{
		SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), COALESCE(d.DriverType, ''),
		             COALESCE(d.VehicleType, ''), COALESCE(d.LicensePlate, ''),
		             COALESCE(d.IsActive, true), COALESCE(d.TruckStatus, 'AVAILABLE'), d.CreatedAt,
		             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, ''), COALESCE(v.MaxVolumeVU, 0),
		             d.EstimatedReturnAt, d.ReturnDurationSec,
		             COALESCE(d.OfflineReason, ''), COALESCE(d.OfflineReasonNote, ''), d.OfflineAt,
		             COALESCE(d.CurrentLocation, '')
		      FROM Drivers d
		      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
		      WHERE d.SupplierId = @supplierId
		      ORDER BY d.CreatedAt DESC`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}
	stmt = auth.AppendWarehouseFilterStmt(r.Context(), stmt, "d")

	iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
	defer iter.Stop()

	drivers := []DriverListItem{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[FLEET] list drivers query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var d DriverListItem
		var createdAt time.Time
		var estReturnAt spanner.NullTime
		var returnDurSec spanner.NullInt64
		var offlineAt spanner.NullTime
		if err := row.Columns(&d.DriverID, &d.Name, &d.Phone, &d.DriverType,
			&d.VehicleType, &d.LicensePlate, &d.IsActive, &d.TruckStatus, &createdAt,
			&d.VehicleId, &d.VehicleClass, &d.MaxVolumeVU, &estReturnAt, &returnDurSec,
			&d.OfflineReason, &d.OfflineReasonNote, &offlineAt, &d.CurrentLocation); err != nil {
			log.Printf("[FLEET] list drivers parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		d.CreatedAt = createdAt.Format(time.RFC3339)
		if estReturnAt.Valid {
			s := estReturnAt.Time.Format(time.RFC3339)
			d.EstimatedReturnAt = &s
		}
		if returnDurSec.Valid {
			d.ReturnDurationSec = &returnDurSec.Int64
		}
		if offlineAt.Valid {
			s := offlineAt.Time.Format(time.RFC3339)
			d.OfflineAt = &s
		}
		drivers = append(drivers, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drivers)
}

// ── Vehicle Assignment ────────────────────────────────────────────────────

// HandleAssignVehicle handles PATCH /v1/supplier/fleet/drivers/{id}/assign-vehicle
func HandleAssignVehicle(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse driver ID: /v1/supplier/fleet/drivers/{id}/assign-vehicle
		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/fleet/drivers/")
		path = strings.TrimSuffix(path, "/assign-vehicle")
		if path == "" {
			http.Error(w, "driver_id required in path", http.StatusBadRequest)
			return
		}
		driverID := path

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		var req struct {
			VehicleId string `json:"vehicle_id"` // empty string = unassign
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		cols := []string{"DriverId", "SupplierId", "VehicleId"}
		trimmedVehicleID := strings.TrimSpace(req.VehicleId)
		vehicleVal := spanner.NullString{StringVal: trimmedVehicleID, Valid: trimmedVehicleID != ""}
		var clearedDriverID string

		_, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			driverRow, err := txn.ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{"SupplierId", "VehicleId", "TruckStatus", "HomeNodeType", "HomeNodeId", "WarehouseId", "RouteId"})
			if err != nil {
				if spanner.ErrCode(err) == codes.NotFound {
					return &assignVehicleRuleError{status: http.StatusNotFound, message: "driver not found"}
				}
				return fmt.Errorf("read driver %s for vehicle assignment: %w", driverID, err)
			}

			var currentSupplierID string
			var currentVehicleID, truckStatus, driverHomeNodeType, driverHomeNodeID, driverWarehouseID, routeID spanner.NullString
			if err := driverRow.Columns(&currentSupplierID, &currentVehicleID, &truckStatus, &driverHomeNodeType, &driverHomeNodeID, &driverWarehouseID, &routeID); err != nil {
				return fmt.Errorf("parse driver %s for vehicle assignment: %w", driverID, err)
			}
			if currentSupplierID != supplierID {
				return &assignVehicleRuleError{status: http.StatusForbidden, message: "driver is outside supplier scope"}
			}

			currentAssignedVehicleID := strings.TrimSpace(currentVehicleID.StringVal)
			if currentAssignedVehicleID == trimmedVehicleID {
				return nil
			}

			driverStatus := strings.TrimSpace(truckStatus.StringVal)
			driverRouteID := strings.TrimSpace(routeID.StringVal)
			if isVehicleAssignmentLocked(driverStatus, driverRouteID) {
				return &assignVehicleRuleError{status: http.StatusConflict, message: fmt.Sprintf("cannot change vehicle while driver is on an active route or truck status is %s", driverStatus)}
			}

			if trimmedVehicleID != "" {
				vehicleRow, err := txn.ReadRow(ctx, "Vehicles", spanner.Key{trimmedVehicleID}, []string{"SupplierId", "IsActive", "HomeNodeType", "HomeNodeId", "WarehouseId"})
				if err != nil {
					if spanner.ErrCode(err) == codes.NotFound {
						return &assignVehicleRuleError{status: http.StatusNotFound, message: "vehicle not found"}
					}
					return fmt.Errorf("read vehicle %s for assignment: %w", trimmedVehicleID, err)
				}

				var vehicleSupplierID string
				var vehicleIsActive bool
				var vehicleHomeNodeType, vehicleHomeNodeID, vehicleWarehouseID spanner.NullString
				if err := vehicleRow.Columns(&vehicleSupplierID, &vehicleIsActive, &vehicleHomeNodeType, &vehicleHomeNodeID, &vehicleWarehouseID); err != nil {
					return fmt.Errorf("parse vehicle %s for assignment: %w", trimmedVehicleID, err)
				}
				if vehicleSupplierID != supplierID {
					return &assignVehicleRuleError{status: http.StatusForbidden, message: "vehicle is outside supplier scope"}
				}
				if !vehicleIsActive {
					return &assignVehicleRuleError{status: http.StatusConflict, message: "vehicle is inactive"}
				}

				driverNodeType, driverNodeID := effectiveFleetHomeNode(strings.TrimSpace(driverHomeNodeType.StringVal), strings.TrimSpace(driverHomeNodeID.StringVal), strings.TrimSpace(driverWarehouseID.StringVal))
				vehicleNodeType, vehicleNodeID := effectiveFleetHomeNode(strings.TrimSpace(vehicleHomeNodeType.StringVal), strings.TrimSpace(vehicleHomeNodeID.StringVal), strings.TrimSpace(vehicleWarehouseID.StringVal))
				if driverNodeType != "" && vehicleNodeType != "" && (driverNodeType != vehicleNodeType || driverNodeID != vehicleNodeID) {
					return &assignVehicleRuleError{status: http.StatusForbidden, message: "driver and vehicle home nodes do not match"}
				}

				conflictStmt := spanner.Statement{
					SQL:    `SELECT DriverId, COALESCE(TruckStatus, 'AVAILABLE'), COALESCE(RouteId, '') FROM Drivers WHERE VehicleId = @vehicleId AND DriverId != @driverId AND SupplierId = @supplierId LIMIT 1`,
					Params: map[string]interface{}{"vehicleId": trimmedVehicleID, "driverId": driverID, "supplierId": supplierID},
				}
				conflictIter := txn.Query(ctx, conflictStmt)
				defer conflictIter.Stop()
				conflictRow, conflictErr := conflictIter.Next()
				if conflictErr != nil && conflictErr != iterator.Done {
					return fmt.Errorf("check vehicle %s assignment conflict: %w", trimmedVehicleID, conflictErr)
				}
				if conflictErr == nil && conflictRow != nil {
					var conflictDriverID, conflictStatus, conflictRouteID string
					if err := conflictRow.Columns(&conflictDriverID, &conflictStatus, &conflictRouteID); err != nil {
						return fmt.Errorf("parse vehicle %s conflict row: %w", trimmedVehicleID, err)
					}
					if isVehicleAssignmentLocked(conflictStatus, conflictRouteID) {
						return &assignVehicleRuleError{status: http.StatusConflict, message: "vehicle is assigned to a driver with an active route", conflictDriverID: conflictDriverID}
					}
					if err := txn.BufferWrite([]*spanner.Mutation{
						spanner.Update("Drivers", cols, []interface{}{conflictDriverID, supplierID, spanner.NullString{}}),
					}); err != nil {
						return fmt.Errorf("clear existing vehicle assignment for driver %s: %w", conflictDriverID, err)
					}
					clearedDriverID = conflictDriverID
				}
			}

			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Drivers", cols, []interface{}{driverID, supplierID, vehicleVal}),
			})
		})
		if err != nil {
			var ruleErr *assignVehicleRuleError
			if errors.As(err, &ruleErr) {
				writeAssignVehicleError(w, ruleErr)
				return
			}
			log.Printf("[FLEET] assign vehicle failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		cache.Invalidate(r.Context(), cache.DriverProfile(driverID))
		if clearedDriverID != "" && clearedDriverID != driverID {
			cache.Invalidate(r.Context(), cache.DriverProfile(clearedDriverID))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":                     "VEHICLE_ASSIGNED",
			"driver_id":                  driverID,
			"vehicle_id":                 trimmedVehicleID,
			"previously_assigned_driver": clearedDriverID,
		})
	}
}

// HandleDriverProfile returns the authenticated driver's profile with current vehicle assignment.
// GET /v1/driver/profile — DRIVER role only (JWT contains DriverId as UserID).
func HandleDriverProfile(spannerClient *spanner.Client, rc *cache.Cache, flight *singleflight.Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		driverID := claims.UserID
		cacheKey := cache.DriverProfile(driverID)

		// Read-through: serve from Redis if warm
		if rc != nil && rc.Client() != nil {
			cacheCtx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			cached, err := rc.Client().Get(cacheCtx, cacheKey).Result()
			cancel()
			if err == nil && cached != "" {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				w.Write([]byte(cached))
				return
			}
		}

		// Singleflight: coalesce concurrent cache-miss reads
		val, err, _ := flight.Do(cacheKey, func() (interface{}, error) {
			return fetchDriverProfile(r.Context(), spannerClient, driverID)
		})
		if err != nil {
			log.Printf("[DRIVER PROFILE] fetch error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		body := val.([]byte)

		// Backfill cache
		if rc != nil && rc.Client() != nil {
			go func() {
				setCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				rc.Client().Set(setCtx, cacheKey, string(body), cache.TTLProfile)
				cancel()
			}()
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "MISS")
		w.Write(body)
	}
}

// fetchDriverProfile reads a driver's profile + vehicle/warehouse join from Spanner.
func fetchDriverProfile(ctx context.Context, spannerClient *spanner.Client, driverID string) ([]byte, error) {
	stmt := spanner.Statement{
		SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), COALESCE(d.DriverType, ''),
		             COALESCE(d.VehicleType, ''), COALESCE(d.LicensePlate, ''),
		             COALESCE(d.IsActive, true), COALESCE(d.SupplierId, ''),
		             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, ''), COALESCE(v.MaxVolumeVU, 0),
		             COALESCE(d.WarehouseId, ''),
		             COALESCE(w.Name, ''), COALESCE(w.Lat, 0), COALESCE(w.Lng, 0)
		      FROM Drivers d
		      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
		      LEFT JOIN Warehouses w ON d.WarehouseId = w.WarehouseId
		      WHERE d.DriverId = @driverId`,
		Params: map[string]interface{}{
			"driverId": driverID,
		},
	}

	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("driver %s not found", driverID)
	}
	if err != nil {
		return nil, fmt.Errorf("query driver %s: %w", driverID, err)
	}

	var name, phone, driverType, vehicleType, licensePlate, supplierID, vehicleID, vehicleClass string
	var warehouseID, warehouseName string
	var warehouseLat, warehouseLng float64
	var maxVolumeVU float64
	var isActive bool
	if err := row.Columns(&driverID, &name, &phone, &driverType, &vehicleType, &licensePlate,
		&isActive, &supplierID, &vehicleID, &vehicleClass, &maxVolumeVU,
		&warehouseID, &warehouseName, &warehouseLat, &warehouseLng); err != nil {
		return nil, fmt.Errorf("parse driver %s: %w", driverID, err)
	}

	return json.Marshal(map[string]interface{}{
		"driver_id":      driverID,
		"name":           name,
		"phone":          phone,
		"driver_type":    driverType,
		"vehicle_type":   vehicleType,
		"license_plate":  licensePlate,
		"is_active":      isActive,
		"supplier_id":    supplierID,
		"vehicle_id":     vehicleID,
		"vehicle_class":  vehicleClass,
		"max_volume_vu":  maxVolumeVU,
		"warehouse_id":   warehouseID,
		"warehouse_name": warehouseName,
		"warehouse_lat":  warehouseLat,
		"warehouse_lng":  warehouseLng,
	})
}

// rotateDriverPIN generates a new globally-unique PIN for a supplier-scoped driver.
// POST /v1/supplier/fleet/drivers/{id}/rotate-pin
func rotateDriverPIN(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, driverID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if driverID == "" {
		http.Error(w, `{"error":"driver_id required"}`, http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	// Verify driver exists and belongs to this supplier.
	row, err := spannerClient.Single().ReadRow(r.Context(), "Drivers",
		spanner.Key{driverID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"driver not found"}`, http.StatusNotFound)
		return
	}
	var ownerID string
	if err := row.Columns(&ownerID); err != nil || ownerID != supplierID {
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
		log.Printf("[FLEET] rotate driver PIN failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"driver_id": driverID,
		"pin":       pinResult.Plaintext,
	})
}

// nullStr returns a spanner.NullString — Valid only if non-empty (for nullable STRING columns)
func nullStr(s string) spanner.NullString {
	if s == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}
