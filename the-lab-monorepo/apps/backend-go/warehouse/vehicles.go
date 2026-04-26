package warehouse

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ─── Vehicle DTOs ─────────────────────────────────────────────────────────────

var vehicleClassCapacity = map[string]float64{
	"CLASS_A": 50.0,
	"CLASS_B": 150.0,
	"CLASS_C": 400.0,
}

type CreateVehicleReq struct {
	VehicleClass string `json:"vehicle_class"` // CLASS_A | CLASS_B | CLASS_C
	Label        string `json:"label"`
	LicensePlate string `json:"license_plate"`
}

type VehicleItem struct {
	VehicleID    string  `json:"vehicle_id"`
	VehicleClass string  `json:"vehicle_class"`
	ClassLabel   string  `json:"class_label"`
	Label        string  `json:"label"`
	LicensePlate string  `json:"license_plate"`
	MaxVolumeVU  float64 `json:"max_volume_vu"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// HandleOpsVehicles — GET/POST for /v1/warehouse/ops/vehicles
func HandleOpsVehicles(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listOpsVehicles(w, r, spannerClient)
		case http.MethodPost:
			createOpsVehicle(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleOpsVehicleDetail — GET/PATCH for /v1/warehouse/ops/vehicles/{id}
func HandleOpsVehicleDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		vehicleID := parts[len(parts)-1]
		if vehicleID == "" || vehicleID == "vehicles" {
			http.Error(w, "vehicle_id required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			getOpsVehicle(w, r, spannerClient, ops, vehicleID)
		case http.MethodPatch:
			patchOpsVehicle(w, r, spannerClient, ops, vehicleID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listOpsVehicles(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, VehicleClass, COALESCE(Label, ''), COALESCE(LicensePlate, ''),
		             MaxVolumeVU, COALESCE(IsActive, true), CreatedAt
		      FROM Vehicles
		      WHERE SupplierId = @sid AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId))
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
	}

	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var vehicles []VehicleItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[WH VEHICLES] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var v VehicleItem
		var createdAt time.Time
		if err := row.Columns(&v.VehicleID, &v.VehicleClass, &v.Label,
			&v.LicensePlate, &v.MaxVolumeVU, &v.IsActive, &createdAt); err != nil {
			log.Printf("[WH VEHICLES] parse: %v", err)
			continue
		}
		v.CreatedAt = createdAt.Format(time.RFC3339)
		v.ClassLabel = vehicleClassLabel(v.VehicleClass)
		vehicles = append(vehicles, v)
	}
	if vehicles == nil {
		vehicles = []VehicleItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"vehicles": vehicles, "total": len(vehicles)})
}

func createOpsVehicle(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"warehouse admin required"}`, http.StatusForbidden)
		return
	}

	var req CreateVehicleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	cap, ok := vehicleClassCapacity[req.VehicleClass]
	if !ok {
		http.Error(w, `{"error":"invalid vehicle_class, use CLASS_A|CLASS_B|CLASS_C"}`, http.StatusBadRequest)
		return
	}

	vehicleID := uuid.New().String()
	// V.O.I.D. Phase VII — warehouse-scoped creation dual-writes HomeNode columns.
	m := spanner.Insert("Vehicles",
		[]string{"VehicleId", "SupplierId", "WarehouseId", "VehicleClass", "Label",
			"LicensePlate", "MaxVolumeVU", "IsActive", "CreatedAt",
			"HomeNodeType", "HomeNodeId"},
		[]interface{}{vehicleID, ops.SupplierID, ops.WarehouseID, req.VehicleClass, req.Label,
			req.LicensePlate, cap, true, spanner.CommitTimestamp,
			"WAREHOUSE", ops.WarehouseID},
	)
	vehicleEvent := kafkaEvents.VehicleCreatedEvent{
		VehicleID:    vehicleID,
		SupplierId:   ops.SupplierID,
		VehicleClass: req.VehicleClass,
		Label:        req.Label,
		LicensePlate: req.LicensePlate,
		MaxVolumeVU:  cap,
		HomeNodeType: "WAREHOUSE",
		HomeNodeId:   ops.WarehouseID,
		CreatedBy:    ops.UserID,
		Timestamp:    time.Now().UTC(),
	}
	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		return outbox.EmitJSON(txn, "Vehicle", vehicleID, kafkaEvents.EventVehicleCreated, kafkaEvents.TopicMain, vehicleEvent, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[WH VEHICLES] insert error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	kafkaEvents.EmitNotification(kafkaEvents.EventVehicleCreated, vehicleEvent)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vehicle_id":    vehicleID,
		"vehicle_class": req.VehicleClass,
		"max_volume_vu": cap,
		"label":         req.Label,
		"license_plate": req.LicensePlate,
	})
}

func getOpsVehicle(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps, vehicleID string) {
	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, VehicleClass, COALESCE(Label, ''), COALESCE(LicensePlate, ''),
		             MaxVolumeVU, COALESCE(IsActive, true), CreatedAt
		      FROM Vehicles
		      WHERE VehicleId = @vid AND SupplierId = @sid AND (WarehouseId = @whId OR (HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @whId))`,
		Params: map[string]interface{}{"vid": vehicleID, "sid": ops.SupplierID, "whId": ops.WarehouseID},
	}
	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		http.Error(w, `{"error":"vehicle not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[WH VEHICLES] detail error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	var v VehicleItem
	var createdAt time.Time
	if err := row.Columns(&v.VehicleID, &v.VehicleClass, &v.Label,
		&v.LicensePlate, &v.MaxVolumeVU, &v.IsActive, &createdAt); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	v.CreatedAt = createdAt.Format(time.RFC3339)
	v.ClassLabel = vehicleClassLabel(v.VehicleClass)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func patchOpsVehicle(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps, vehicleID string) {
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"warehouse admin required"}`, http.StatusForbidden)
		return
	}

	var req struct {
		Label        *string `json:"label,omitempty"`
		LicensePlate *string `json:"license_plate,omitempty"`
		IsActive     *bool   `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"VehicleId"}
	vals := []interface{}{vehicleID}
	if req.Label != nil {
		cols = append(cols, "Label")
		vals = append(vals, *req.Label)
	}
	if req.LicensePlate != nil {
		cols = append(cols, "LicensePlate")
		vals = append(vals, *req.LicensePlate)
	}
	if req.IsActive != nil {
		cols = append(cols, "IsActive")
		vals = append(vals, *req.IsActive)
	}
	if len(cols) == 1 {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	m := spanner.Update("Vehicles", cols, vals)
	if _, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[WH VEHICLES] patch error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "vehicle_id": vehicleID})
}

func vehicleClassLabel(class string) string {
	labels := map[string]string{
		"CLASS_A": "Damass / Minivan",
		"CLASS_B": "Transit Van",
		"CLASS_C": "Box Truck / Isuzu",
	}
	if l, ok := labels[class]; ok {
		return l
	}
	return class
}
