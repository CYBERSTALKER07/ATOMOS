package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
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
