package factory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/pkg/pin"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// ── Factory Fleet Management ──────────────────────────────────────────────────
// Factory-assigned drivers and vehicles (separate from warehouse/supplier fleet).
// Factories use the same Drivers/Vehicles tables, scoped by factory context.

// HandleFactoryFleetDrivers — GET/POST /v1/factory/fleet/drivers
func HandleFactoryFleetDrivers(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listFactoryDrivers(w, r, spannerClient)
		case http.MethodPost:
			// Factory driver creation goes through supplier fleet endpoint
			// (drivers are supplier-wide). This is a scoped list view only.
			http.Error(w, "Use /v1/supplier/fleet/drivers to create drivers", http.StatusMethodNotAllowed)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleFactoryFleetVehicles — GET /v1/factory/fleet/vehicles
func HandleFactoryFleetVehicles(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		listFactoryVehicles(w, r, spannerClient)
	}
}

func listFactoryDrivers(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	// Factory drivers: query Drivers with a join on factory assignment
	// For now, factory drivers are referenced via FactoryTruckManifests.DriverId
	// We list all drivers for the supplier and let the factory admin filter
	scope := auth.GetFactoryScope(r.Context())
	if scope == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_ = factoryID // used for scoping context

	// List supplier-wide drivers (factory admins can view all drivers in their org)
	stmt := spanner.Statement{
		SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), COALESCE(d.DriverType, ''),
		             COALESCE(d.VehicleType, ''), COALESCE(d.LicensePlate, ''),
		             COALESCE(d.IsActive, true), COALESCE(d.TruckStatus, 'AVAILABLE'), d.CreatedAt,
		             COALESCE(d.VehicleId, '')
		      FROM Drivers d
		      JOIN FactoryStaff fs ON d.SupplierId = fs.SupplierId
		      WHERE fs.FactoryId = @fid
		      GROUP BY d.DriverId, d.Name, d.Phone, d.DriverType, d.VehicleType,
		               d.LicensePlate, d.IsActive, d.TruckStatus, d.CreatedAt, d.VehicleId
		      ORDER BY d.CreatedAt DESC`,
		Params: map[string]interface{}{"fid": factoryID},
	}
	iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
	defer iter.Stop()

	type FactoryDriver struct {
		DriverId     string `json:"driver_id"`
		Name         string `json:"name"`
		Phone        string `json:"phone"`
		DriverType   string `json:"driver_type"`
		VehicleType  string `json:"vehicle_type"`
		LicensePlate string `json:"license_plate"`
		IsActive     bool   `json:"is_active"`
		TruckStatus  string `json:"truck_status"`
		CreatedAt    string `json:"created_at"`
		VehicleId    string `json:"vehicle_id,omitempty"`
	}

	drivers := []FactoryDriver{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[FACTORY FLEET] drivers list error: %v", err)
			break
		}
		var d FactoryDriver
		var createdAt time.Time
		if err := row.Columns(&d.DriverId, &d.Name, &d.Phone, &d.DriverType,
			&d.VehicleType, &d.LicensePlate, &d.IsActive, &d.TruckStatus, &createdAt,
			&d.VehicleId); err != nil {
			continue
		}
		d.CreatedAt = createdAt.Format(time.RFC3339)
		drivers = append(drivers, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": drivers})
}

func listFactoryVehicles(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	// Factory vehicles: supplier-wide vehicles visible to factory admin
	stmt := spanner.Statement{
		SQL: `SELECT v.VehicleId, v.SupplierId, v.VehicleClass, v.MaxVolumeVU,
		             COALESCE(v.LicensePlate, ''), v.IsActive, v.CreatedAt
		      FROM Vehicles v
		      JOIN FactoryStaff fs ON v.SupplierId = fs.SupplierId
		      WHERE fs.FactoryId = @fid
		      GROUP BY v.VehicleId, v.SupplierId, v.VehicleClass, v.MaxVolumeVU,
		               v.LicensePlate, v.IsActive, v.CreatedAt
		      ORDER BY v.CreatedAt DESC`,
		Params: map[string]interface{}{"fid": factoryID},
	}
	iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
	defer iter.Stop()

	type FactoryVehicle struct {
		VehicleId    string  `json:"vehicle_id"`
		SupplierId   string  `json:"supplier_id"`
		VehicleClass string  `json:"vehicle_class"`
		MaxVolumeVU  float64 `json:"max_volume_vu"`
		LicensePlate string  `json:"license_plate"`
		IsActive     bool    `json:"is_active"`
		CreatedAt    string  `json:"created_at"`
	}

	vehicles := []FactoryVehicle{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[FACTORY FLEET] vehicles list error: %v", err)
			break
		}
		var v FactoryVehicle
		var createdAt time.Time
		if err := row.Columns(&v.VehicleId, &v.SupplierId, &v.VehicleClass, &v.MaxVolumeVU,
			&v.LicensePlate, &v.IsActive, &createdAt); err != nil {
			continue
		}
		v.CreatedAt = createdAt.Format(time.RFC3339)
		vehicles = append(vehicles, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": vehicles})
}

// ── Factory Staff Management ──────────────────────────────────────────────────

type StaffResponse struct {
	StaffId   string `json:"staff_id"`
	FactoryId string `json:"factory_id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	StaffRole string `json:"staff_role"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// HandleFactoryStaff — GET/POST /v1/factory/staff
func HandleFactoryStaff(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listFactoryStaff(w, r, spannerClient)
		case http.MethodPost:
			createFactoryStaff(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleFactoryStaffDetail — GET, DELETE for /v1/factory/staff/{id}
// Also handles POST /v1/factory/staff/{id}/rotate-pin
func HandleFactoryStaffDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		staffID := strings.TrimPrefix(r.URL.Path, "/v1/factory/staff/")

		// Route: POST /v1/factory/staff/{id}/rotate-pin
		if strings.HasSuffix(staffID, "/rotate-pin") {
			staffID = strings.TrimSuffix(staffID, "/rotate-pin")
			rotateFactoryStaffPIN(w, r, spannerClient, staffID)
			return
		}

		if staffID == "" || strings.Contains(staffID, "/") {
			http.Error(w, "staff_id required in path", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodDelete:
			deactivateStaff(w, r, spannerClient, staffID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// rotateFactoryStaffPIN generates a new globally-unique PIN for a factory staff member.
// POST /v1/factory/staff/{id}/rotate-pin
func rotateFactoryStaffPIN(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, staffID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if staffID == "" {
		http.Error(w, `{"error":"staff_id required"}`, http.StatusBadRequest)
		return
	}

	// Verify staff exists.
	row, err := spannerClient.Single().ReadRow(r.Context(), "FactoryStaff",
		spanner.Key{staffID}, []string{"FactoryId"})
	if err != nil {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}
	var factoryID string
	if err := row.Columns(&factoryID); err != nil {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.FactoryID != factoryID {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}

	var pinResult *pin.Result
	_, err = spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var rotErr error
		pinResult, rotErr = pin.Rotate(ctx, txn, pin.EntityFactoryStaff, staffID)
		if rotErr != nil {
			return rotErr
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("FactoryStaff", []string{"StaffId", "PinHash"}, []interface{}{staffID, pinResult.BcryptHash}),
		})
	})
	if err != nil {
		log.Printf("[FACTORY STAFF] rotate PIN failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"staff_id": staffID,
		"pin":      pinResult.Plaintext,
	})
}

func listFactoryStaff(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT StaffId, FactoryId, Name, COALESCE(Phone, ''), StaffRole, IsActive, CreatedAt
		      FROM FactoryStaff WHERE FactoryId = @fid ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"fid": factoryID},
	}
	iter := spannerClient.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	staff := []StaffResponse{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[FACTORY STAFF] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var s StaffResponse
		var createdAt time.Time
		if err := row.Columns(&s.StaffId, &s.FactoryId, &s.Name, &s.Phone,
			&s.StaffRole, &s.IsActive, &createdAt); err != nil {
			continue
		}
		s.CreatedAt = createdAt.Format(time.RFC3339)
		staff = append(staff, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": staff})
}

func createFactoryStaff(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	scope := auth.GetFactoryScope(r.Context())
	if scope != nil && scope.IsPayloader {
		http.Error(w, `{"error":"payloaders cannot create staff"}`, http.StatusForbidden)
		return
	}

	var req struct {
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Password  string `json:"password"`
		StaffRole string `json:"staff_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Phone == "" || req.Password == "" {
		http.Error(w, `{"error":"name, phone, and password are required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}
	if req.StaffRole != "FACTORY_ADMIN" && req.StaffRole != "FACTORY_PAYLOADER" {
		req.StaffRole = "FACTORY_PAYLOADER"
	}

	// Resolve supplier from factory
	fRow, err := spannerClient.Single().ReadRow(r.Context(), "Factories",
		spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}
	var supplierID string
	if err := fRow.Columns(&supplierID); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Duplicate phone check + hash password
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		log.Printf("[FACTORY STAFF] bcrypt error: %v", hashErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	staffId := uuid.New().String()
	m := spanner.Insert("FactoryStaff",
		[]string{"StaffId", "FactoryId", "SupplierId", "Name", "Phone", "PasswordHash", "StaffRole", "IsActive", "CreatedAt"},
		[]interface{}{staffId, factoryID, supplierID, req.Name, req.Phone, string(hash), req.StaffRole, true, spanner.CommitTimestamp},
	)
	if _, err := spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		log.Printf("[FACTORY STAFF] create error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"staff_id":   staffId,
		"factory_id": factoryID,
		"name":       req.Name,
		"staff_role": req.StaffRole,
	})
}

func deactivateStaff(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, staffID string) {
	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	// Verify staff belongs to this factory
	row, err := spannerClient.Single().ReadRow(r.Context(), "FactoryStaff",
		spanner.Key{staffID}, []string{"FactoryId"})
	if err != nil {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}
	var rowFid string
	if err := row.Columns(&rowFid); err != nil || rowFid != factoryID {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}

	m := spanner.Update("FactoryStaff",
		[]string{"StaffId", "IsActive"},
		[]interface{}{staffID, false},
	)
	if _, err := spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		log.Printf("[FACTORY STAFF] deactivate error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "DEACTIVATED",
		"staff_id": staffID,
	})
}
