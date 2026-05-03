package warehouse

import (
	"context"
	"encoding/json"
	"errors"
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
	VehicleID          string  `json:"vehicle_id"`
	VehicleClass       string  `json:"vehicle_class"`
	ClassLabel         string  `json:"class_label"`
	Label              string  `json:"label"`
	LicensePlate       string  `json:"license_plate"`
	MaxVolumeVU        float64 `json:"max_volume_vu"`
	CapacityVU         float64 `json:"capacity_vu"`
	IsActive           bool    `json:"is_active"`
	Status             string  `json:"status"`
	UnavailableReason  string  `json:"unavailable_reason,omitempty"`
	CreatedAt          string  `json:"created_at"`
	AssignedDriverID   string  `json:"assigned_driver_id,omitempty"`
	AssignedDriverName string  `json:"assigned_driver_name,omitempty"`
	DriverTruckStatus  string  `json:"driver_truck_status,omitempty"`
}

const (
	warehouseVehicleUnavailableReasonMaintenance    = "MAINTENANCE"
	warehouseVehicleUnavailableReasonTruckDamaged   = "TRUCK_DAMAGED"
	warehouseVehicleUnavailableReasonRegulatoryHold = "REGULATORY_HOLD"
	warehouseVehicleUnavailableReasonManualHold     = "MANUAL_HOLD"
)

var warehouseVehicleUnavailableReasons = map[string]struct{}{
	warehouseVehicleUnavailableReasonMaintenance:    {},
	warehouseVehicleUnavailableReasonTruckDamaged:   {},
	warehouseVehicleUnavailableReasonRegulatoryHold: {},
	warehouseVehicleUnavailableReasonManualHold:     {},
}

func vehicleAvailabilityStatus(isActive bool) string {
	if isActive {
		return "AVAILABLE"
	}
	return "INACTIVE"
}

func normalizeVehicleItemAliases(item *VehicleItem) {
	if item == nil {
		return
	}
	item.CapacityVU = item.MaxVolumeVU
	item.Status = vehicleAvailabilityStatus(item.IsActive)
	if item.IsActive && strings.TrimSpace(item.DriverTruckStatus) != "" {
		item.Status = strings.TrimSpace(item.DriverTruckStatus)
	}
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
		SQL: `SELECT v.VehicleId, v.VehicleClass, COALESCE(v.Label, ''), COALESCE(v.LicensePlate, ''),
		             v.MaxVolumeVU, COALESCE(v.IsActive, true), COALESCE(v.UnavailableReason, ''), v.CreatedAt,
		             COALESCE(d.DriverId, ''), COALESCE(d.Name, ''), COALESCE(d.TruckStatus, '')
		      FROM Vehicles v
		      LEFT JOIN Drivers d ON d.VehicleId = v.VehicleId AND d.SupplierId = @sid
		        AND (d.WarehouseId = @whId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @whId))
		      WHERE v.SupplierId = @sid AND (v.WarehouseId = @whId OR (v.HomeNodeType = 'WAREHOUSE' AND v.HomeNodeId = @whId))
		      ORDER BY v.CreatedAt DESC`,
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
			&v.LicensePlate, &v.MaxVolumeVU, &v.IsActive, &v.UnavailableReason, &createdAt, &v.AssignedDriverID, &v.AssignedDriverName, &v.DriverTruckStatus); err != nil {
			log.Printf("[WH VEHICLES] parse: %v", err)
			continue
		}
		v.CreatedAt = createdAt.Format(time.RFC3339)
		v.ClassLabel = vehicleClassLabel(v.VehicleClass)
		normalizeVehicleItemAliases(&v)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vehicle_id":    vehicleID,
		"vehicle_class": req.VehicleClass,
		"max_volume_vu": cap,
		"capacity_vu":   cap,
		"label":         req.Label,
		"license_plate": req.LicensePlate,
		"status":        vehicleAvailabilityStatus(true),
		"is_active":     true,
	})
}

func getOpsVehicle(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps, vehicleID string) {
	stmt := spanner.Statement{
		SQL: `SELECT v.VehicleId, v.VehicleClass, COALESCE(v.Label, ''), COALESCE(v.LicensePlate, ''),
		             v.MaxVolumeVU, COALESCE(v.IsActive, true), COALESCE(v.UnavailableReason, ''), v.CreatedAt,
		             COALESCE(d.DriverId, ''), COALESCE(d.Name, ''), COALESCE(d.TruckStatus, '')
		      FROM Vehicles v
		      LEFT JOIN Drivers d ON d.VehicleId = v.VehicleId AND d.SupplierId = @sid
		        AND (d.WarehouseId = @whId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @whId))
		      WHERE v.VehicleId = @vid AND v.SupplierId = @sid AND (v.WarehouseId = @whId OR (v.HomeNodeType = 'WAREHOUSE' AND v.HomeNodeId = @whId))`,
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
		&v.LicensePlate, &v.MaxVolumeVU, &v.IsActive, &v.UnavailableReason, &createdAt, &v.AssignedDriverID, &v.AssignedDriverName, &v.DriverTruckStatus); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	v.CreatedAt = createdAt.Format(time.RFC3339)
	v.ClassLabel = vehicleClassLabel(v.VehicleClass)
	normalizeVehicleItemAliases(&v)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func patchOpsVehicle(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps, vehicleID string) {
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"warehouse admin required"}`, http.StatusForbidden)
		return
	}

	var req struct {
		Label             *string `json:"label,omitempty"`
		LicensePlate      *string `json:"license_plate,omitempty"`
		IsActive          *bool   `json:"is_active,omitempty"`
		UnavailableReason *string `json:"unavailable_reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	appliedUnavailableReason := ""
	cacheDriverIDs := []string{}
	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		vehicleState, err := readWarehouseVehicleAssignmentState(ctx, txn, ops, vehicleID)
		if err != nil {
			return err
		}

		vehicleColumns := []string{"VehicleId"}
		vehicleValues := []interface{}{vehicleID}
		if req.Label != nil {
			vehicleColumns = append(vehicleColumns, "Label")
			vehicleValues = append(vehicleValues, *req.Label)
		}
		if req.LicensePlate != nil {
			vehicleColumns = append(vehicleColumns, "LicensePlate")
			vehicleValues = append(vehicleValues, *req.LicensePlate)
		}

		shouldWriteUnavailableReason := false
		unavailableReason := ""
		if req.IsActive != nil {
			if *req.IsActive {
				shouldWriteUnavailableReason = true
			} else {
				shouldWriteUnavailableReason = true
				unavailableReason = normalizeWarehouseVehicleUnavailableReason(req.UnavailableReason)
				if unavailableReason == "" {
					unavailableReason = warehouseVehicleUnavailableReasonManualHold
				}
				if !isWarehouseVehicleUnavailableReason(unavailableReason) {
					return &warehouseFleetMutationRuleError{StatusCode: http.StatusBadRequest, Message: "invalid unavailable_reason"}
				}
			}
		} else if req.UnavailableReason != nil {
			if vehicleState.IsActive {
				return &warehouseFleetMutationRuleError{StatusCode: http.StatusBadRequest, Message: "unavailable_reason requires is_active=false"}
			}
			unavailableReason = normalizeWarehouseVehicleUnavailableReason(req.UnavailableReason)
			if unavailableReason == "" {
				return &warehouseFleetMutationRuleError{StatusCode: http.StatusBadRequest, Message: "unavailable_reason is required when updating an inactive vehicle"}
			}
			if !isWarehouseVehicleUnavailableReason(unavailableReason) {
				return &warehouseFleetMutationRuleError{StatusCode: http.StatusBadRequest, Message: "invalid unavailable_reason"}
			}
			shouldWriteUnavailableReason = true
		}

		mutations := make([]*spanner.Mutation, 0, 2)
		assignedDriver, err := readWarehouseDriverByVehicle(ctx, txn, ops, vehicleID, "")
		if err != nil {
			return err
		}
		if req.IsActive != nil && vehicleState.IsActive != *req.IsActive {
			if assignedDriver != nil {
				if ruleErr := warehouseVehicleAssignmentRuleError(*assignedDriver); ruleErr != nil {
					return ruleErr
				}
				driverColumns := []string{"DriverId"}
				driverValues := []interface{}{assignedDriver.DriverID}
				if *req.IsActive {
					if strings.EqualFold(assignedDriver.TruckStatus, warehouseDriverStatusMaintenance) {
						driverColumns = append(driverColumns, "TruckStatus")
						driverValues = append(driverValues, warehouseDriverStatusAvailable)
					}
				} else if warehouseVehicleReasonRequiresMaintenance(unavailableReason) {
					driverColumns = append(driverColumns, "TruckStatus")
					driverValues = append(driverValues, warehouseDriverStatusMaintenance)
				}
				if len(driverColumns) > 1 {
					mutations = append(mutations, spanner.Update("Drivers", driverColumns, driverValues))
					cacheDriverIDs = append(cacheDriverIDs, assignedDriver.DriverID)
				}
			}
			vehicleColumns = append(vehicleColumns, "IsActive")
			vehicleValues = append(vehicleValues, *req.IsActive)
		}
		if shouldWriteUnavailableReason {
			vehicleColumns = append(vehicleColumns, "UnavailableReason")
			vehicleValues = append(vehicleValues, warehouseNullableString(unavailableReason))
			appliedUnavailableReason = unavailableReason
		}

		if len(vehicleColumns) == 1 && len(mutations) == 0 {
			return &warehouseFleetMutationRuleError{StatusCode: http.StatusBadRequest, Message: "no fields to update"}
		}
		if len(vehicleColumns) > 1 {
			mutations = append(mutations, spanner.Update("Vehicles", vehicleColumns, vehicleValues))
		}
		return txn.BufferWrite(mutations)
	}); err != nil {
		var ruleErr *warehouseFleetMutationRuleError
		if errors.As(err, &ruleErr) {
			http.Error(w, `{"error":"`+ruleErr.Message+`"}`, ruleErr.StatusCode)
			return
		}
		log.Printf("[WH VEHICLES] patch error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	warehouseInvalidateDriverProfiles(ctx, cacheDriverIDs...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "updated", "vehicle_id": vehicleID, "unavailable_reason": appliedUnavailableReason})
}

func normalizeWarehouseVehicleUnavailableReason(reason *string) string {
	if reason == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(*reason))
}

func isWarehouseVehicleUnavailableReason(reason string) bool {
	_, ok := warehouseVehicleUnavailableReasons[reason]
	return ok
}

func warehouseVehicleReasonRequiresMaintenance(reason string) bool {
	switch reason {
	case warehouseVehicleUnavailableReasonMaintenance, warehouseVehicleUnavailableReasonTruckDamaged:
		return true
	default:
		return false
	}
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
