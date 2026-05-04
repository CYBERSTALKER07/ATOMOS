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
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// VEHICLE FLEET — ABSTRACT VOLUMETRIC UNITS (VU)
// Trucks are independent entities, decoupled from drivers.
// VehicleClass determines MaxVolumeVU:
//
//	CLASS_A (Damass/Minivan) = 50 VU
//	CLASS_B (Transit Van)    = 150 VU
//	CLASS_C (Box Truck)      = 400 VU
//
// 1.0 VU = 1 standard case of 1L water bottles (universal baseline)
// ═══════════════════════════════════════════════════════════════════════════════

// VehicleClassCapacity maps class code → MaxVolumeVU
var VehicleClassCapacity = map[string]float64{
	"CLASS_A": 50.0,
	"CLASS_B": 150.0,
	"CLASS_C": 400.0,
}

// VehicleClassLabels maps class code → human label
var VehicleClassLabels = map[string]string{
	"CLASS_A": "Damass / Minivan",
	"CLASS_B": "Transit Van",
	"CLASS_C": "Box Truck / Isuzu",
}

// ── Request / Response DTOs ───────────────────────────────────────────────

type CreateVehicleRequest struct {
	VehicleClass string   `json:"vehicle_class"` // CLASS_A | CLASS_B | CLASS_C
	Label        string   `json:"label"`         // Nickname (optional)
	LicensePlate string   `json:"license_plate"`
	LengthCM     *float64 `json:"length_cm,omitempty"` // Physical cargo dimensions (optional)
	WidthCM      *float64 `json:"width_cm,omitempty"`
	HeightCM     *float64 `json:"height_cm,omitempty"`
	// V.O.I.D. Phase VII — home-node override (GLOBAL_ADMIN only).
	HomeNodeType string `json:"home_node_type,omitempty"` // WAREHOUSE | FACTORY
	HomeNodeId   string `json:"home_node_id,omitempty"`
}

type VehicleResponse struct {
	VehicleID    string   `json:"vehicle_id"`
	SupplierId   string   `json:"supplier_id"`
	VehicleClass string   `json:"vehicle_class"`
	ClassLabel   string   `json:"class_label"`
	Label        string   `json:"label"`
	LicensePlate string   `json:"license_plate"`
	MaxVolumeVU  float64  `json:"max_volume_vu"`
	IsActive     bool     `json:"is_active"`
	CreatedAt    string   `json:"created_at"`
	LengthCM     *float64 `json:"length_cm,omitempty"`
	WidthCM      *float64 `json:"width_cm,omitempty"`
	HeightCM     *float64 `json:"height_cm,omitempty"`
}

type UpdateVehicleRequest struct {
	Label        *string `json:"label,omitempty"`
	LicensePlate *string `json:"license_plate,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

var errScopedVehicleNotFound = errors.New("vehicle not found")

// ── Handlers ──────────────────────────────────────────────────────────────

// HandleVehicles routes GET (list) and POST (create) for /v1/supplier/fleet/vehicles
func HandleVehicles(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listVehicles(w, r, spannerClient)
		case http.MethodPost:
			createVehicle(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleVehicleDetail routes GET, PATCH, DELETE for /v1/supplier/fleet/vehicles/{id}
func HandleVehicleDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/fleet/vehicles/")
		if path == "" || strings.Contains(path, "/") {
			http.Error(w, "vehicle_id required in path", http.StatusBadRequest)
			return
		}
		vehicleID := path

		switch r.Method {
		case http.MethodGet:
			getVehicle(w, r, spannerClient, vehicleID)
		case http.MethodPatch:
			updateVehicle(w, r, spannerClient, vehicleID)
		case http.MethodDelete:
			deactivateVehicle(w, r, spannerClient, vehicleID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── Private Handlers ──────────────────────────────────────────────────────

func createVehicle(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req CreateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	maxVU, validClass := VehicleClassCapacity[req.VehicleClass]
	if !validClass {
		http.Error(w, `{"error":"invalid vehicle_class — must be CLASS_A, CLASS_B, or CLASS_C"}`, http.StatusBadRequest)
		return
	}
	if err := validatePhysicalDimensions(req.LengthCM, req.WidthCM, req.HeightCM, false); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnprocessableEntity)
		return
	}

	// If physical cargo dimensions supplied, compute a more precise VU
	if req.LengthCM != nil && req.WidthCM != nil && req.HeightCM != nil {
		computed := CalculateVU(*req.LengthCM, *req.WidthCM, *req.HeightCM)
		if computed > 0 {
			maxVU = computed
		}
	}

	vehicleID := "VEH-" + uuid.New().String()[:8]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	warehouseID := auth.EffectiveWarehouseID(r.Context())

	// V.O.I.D. Phase VII — derive canonical (HomeNodeType, HomeNodeId) from the
	// caller's JWT scope; reject body overrides that try to escape scope.
	homeNodeType, homeNodeID, scopeOK := auth.ApplyHomeNodeOverride(claims, req.HomeNodeType, req.HomeNodeId)
	if !scopeOK {
		http.Error(w, `{"error":"home_node_id outside caller scope"}`, http.StatusForbidden)
		return
	}
	if homeNodeType == auth.HomeNodeTypeWarehouse && warehouseID == "" {
		warehouseID = homeNodeID
	}

	cols := []string{"VehicleId", "SupplierId", "VehicleClass", "Label", "LicensePlate", "MaxVolumeVU", "IsActive", "CreatedAt", "WarehouseId", "HomeNodeType", "HomeNodeId"}
	vals := []interface{}{vehicleID, supplierID, req.VehicleClass, req.Label, req.LicensePlate, maxVU, true, spanner.CommitTimestamp, nullStr(warehouseID), nullStr(homeNodeType), nullStr(homeNodeID)}
	if req.LengthCM != nil {
		cols = append(cols, "LengthCM", "WidthCM", "HeightCM")
		vals = append(vals, *req.LengthCM, *req.WidthCM, *req.HeightCM)
	}

	vehicleEvent := kafkaEvents.VehicleCreatedEvent{
		VehicleID:    vehicleID,
		SupplierId:   supplierID,
		VehicleClass: req.VehicleClass,
		Label:        req.Label,
		LicensePlate: req.LicensePlate,
		MaxVolumeVU:  maxVU,
		HomeNodeType: homeNodeType,
		HomeNodeId:   homeNodeID,
		CreatedBy:    claims.UserID,
		Timestamp:    time.Now().UTC(),
	}
	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("Vehicles", cols, vals),
		}); err != nil {
			return err
		}
		return outbox.EmitJSON(txn, "Vehicle", vehicleID, kafkaEvents.EventVehicleCreated, kafkaEvents.TopicMain, vehicleEvent, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[VEHICLES] Spanner insert failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(VehicleResponse{
		VehicleID:    vehicleID,
		SupplierId:   supplierID,
		VehicleClass: req.VehicleClass,
		ClassLabel:   VehicleClassLabels[req.VehicleClass],
		Label:        req.Label,
		LicensePlate: req.LicensePlate,
		MaxVolumeVU:  maxVU,
		IsActive:     true,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		LengthCM:     req.LengthCM,
		WidthCM:      req.WidthCM,
		HeightCM:     req.HeightCM,
	})
}

func listVehicles(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, VehicleClass, COALESCE(Label, ''), COALESCE(LicensePlate, ''),
		             MaxVolumeVU, IsActive, CreatedAt, LengthCM, WidthCM, HeightCM
		      FROM Vehicles
		      WHERE SupplierId = @supplierId
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"supplierId": supplierID},
	}
	stmt = auth.AppendWarehouseFilterStmt(r.Context(), stmt, "Vehicles")

	iter := spannerClient.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	vehicles := []VehicleResponse{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[VEHICLES] list query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var v VehicleResponse
		var createdAt time.Time
		if err := row.Columns(&v.VehicleID, &v.VehicleClass, &v.Label, &v.LicensePlate,
			&v.MaxVolumeVU, &v.IsActive, &createdAt, &v.LengthCM, &v.WidthCM, &v.HeightCM); err != nil {
			log.Printf("[VEHICLES] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		v.SupplierId = supplierID
		v.ClassLabel = VehicleClassLabels[v.VehicleClass]
		v.CreatedAt = createdAt.Format(time.RFC3339)
		vehicles = append(vehicles, v)
	}

	// Also fetch assigned driver for each vehicle
	type assignment struct {
		VehicleID  string `json:"vehicle_id"`
		DriverID   string `json:"driver_id"`
		DriverName string `json:"driver_name"`
	}
	assignStmt := spanner.Statement{
		SQL: `SELECT COALESCE(VehicleId, ''), DriverId, Name
		      FROM Drivers
		      WHERE SupplierId = @supplierId AND VehicleId IS NOT NULL`,
		Params: map[string]interface{}{"supplierId": supplierID},
	}
	assignIter := spannerClient.Single().Query(r.Context(), assignStmt)
	defer assignIter.Stop()

	assignMap := map[string]assignment{}
	for {
		row, err := assignIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[VEHICLES] assignment query error for supplier %s: %v", supplierID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var a assignment
		if err := row.Columns(&a.VehicleID, &a.DriverID, &a.DriverName); err != nil {
			log.Printf("[VEHICLES] assignment parse error for supplier %s: %v", supplierID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		assignMap[a.VehicleID] = a
	}

	type VehicleWithDriver struct {
		VehicleResponse
		AssignedDriverID   string `json:"assigned_driver_id"`
		AssignedDriverName string `json:"assigned_driver_name"`
	}

	result := make([]VehicleWithDriver, len(vehicles))
	for i, v := range vehicles {
		result[i] = VehicleWithDriver{VehicleResponse: v}
		if a, ok := assignMap[v.VehicleID]; ok {
			result[i].AssignedDriverID = a.DriverID
			result[i].AssignedDriverName = a.DriverName
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getVehicle(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, vehicleID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	stmt := spanner.Statement{
		SQL: `SELECT v.VehicleId, v.VehicleClass, COALESCE(v.Label, ''), COALESCE(v.LicensePlate, ''),
		             MaxVolumeVU, IsActive, CreatedAt, LengthCM, WidthCM, HeightCM
		      FROM Vehicles v
		      WHERE v.VehicleId = @vehicleId AND v.SupplierId = @supplierId`,
		Params: map[string]interface{}{"vehicleId": vehicleID, "supplierId": supplierID},
	}
	stmt = auth.AppendWarehouseFilterStmt(r.Context(), stmt, "v")

	iter := spannerClient.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		http.Error(w, `{"error":"vehicle not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[VEHICLES] detail query error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var v VehicleResponse
	var createdAt time.Time
	if err := row.Columns(&v.VehicleID, &v.VehicleClass, &v.Label, &v.LicensePlate,
		&v.MaxVolumeVU, &v.IsActive, &createdAt, &v.LengthCM, &v.WidthCM, &v.HeightCM); err != nil {
		log.Printf("[VEHICLES] detail parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	v.SupplierId = supplierID
	v.ClassLabel = VehicleClassLabels[v.VehicleClass]
	v.CreatedAt = createdAt.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func ensureVehicleMutationScope(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, vehicleID string) error {
	stmt := spanner.Statement{
		SQL: `SELECT v.VehicleId
		      FROM Vehicles v
		      WHERE v.VehicleId = @vehicleId AND v.SupplierId = @supplierId`,
		Params: map[string]interface{}{"vehicleId": vehicleID, "supplierId": supplierID},
	}
	stmt = auth.AppendWarehouseFilterStmt(ctx, stmt, "v")

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return errScopedVehicleNotFound
	}
	if err != nil {
		return fmt.Errorf("read vehicle %s in scope: %w", vehicleID, err)
	}
	return nil
}

func updateVehicle(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, vehicleID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req UpdateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cols := []string{"VehicleId", "SupplierId"}
	vals := []interface{}{vehicleID, supplierID}

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

	if len(cols) == 2 {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := ensureVehicleMutationScope(ctx, txn, supplierID, vehicleID); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Vehicles", cols, vals),
		}); err != nil {
			return fmt.Errorf("buffer vehicle %s update: %w", vehicleID, err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errScopedVehicleNotFound) {
			http.Error(w, `{"error":"vehicle not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("[VEHICLES] update failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	invalidateVehicleDriverProfiles(ctx, spannerClient, supplierID, vehicleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "UPDATED", "vehicle_id": vehicleID})
}

func deactivateVehicle(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, vehicleID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := ensureVehicleMutationScope(ctx, txn, supplierID, vehicleID); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Vehicles",
				[]string{"VehicleId", "SupplierId", "IsActive"},
				[]interface{}{vehicleID, supplierID, false},
			),
		}); err != nil {
			return fmt.Errorf("buffer vehicle %s deactivation: %w", vehicleID, err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errScopedVehicleNotFound) {
			http.Error(w, `{"error":"vehicle not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("[VEHICLES] deactivate failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	invalidateVehicleDriverProfiles(ctx, spannerClient, supplierID, vehicleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "DEACTIVATED", "vehicle_id": vehicleID})
}

func invalidateVehicleDriverProfiles(ctx context.Context, spannerClient *spanner.Client, supplierID, vehicleID string) {
	stmt := spanner.Statement{
		SQL: `SELECT DriverId
		      FROM Drivers
		      WHERE SupplierId = @supplierId AND VehicleId = @vehicleId`,
		Params: map[string]interface{}{"supplierId": supplierID, "vehicleId": vehicleID},
	}

	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	keys := make([]string, 0, 2)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[VEHICLES] cache invalidation lookup failed: %v", err)
			return
		}

		var driverID string
		if err := row.Columns(&driverID); err != nil {
			log.Printf("[VEHICLES] cache invalidation parse failed: %v", err)
			return
		}
		if driverID != "" {
			keys = append(keys, cache.DriverProfile(driverID))
		}
	}

	if len(keys) > 0 {
		cache.Invalidate(ctx, keys...)
	}
}
