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
	"backend-go/pkg/pin"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ─── Staff Management (warehouse admins manage their own staff) ───────────────

type StaffItem struct {
	WorkerID    string `json:"worker_id"`
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Role        string `json:"role"` // WAREHOUSE_ADMIN | WAREHOUSE_STAFF | PAYLOADER
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	WarehouseID string `json:"warehouse_id"`
}

type CreateStaffReq struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	PIN   string `json:"pin"`
	Role  string `json:"role"` // WAREHOUSE_STAFF | PAYLOADER
}

// HandleOpsStaff — GET/POST for /v1/warehouse/ops/staff
func HandleOpsStaff(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listOpsStaff(w, r, spannerClient)
		case http.MethodPost:
			createOpsStaff(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleOpsStaffDetail — GET/PATCH for /v1/warehouse/ops/staff/{id}
// Also handles POST /v1/warehouse/ops/staff/{id}/rotate-pin
func HandleOpsStaffDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}
		if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
			http.Error(w, `{"error":"warehouse admin required"}`, http.StatusForbidden)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/staff/")

		// Route: POST /v1/warehouse/ops/staff/{id}/rotate-pin
		if strings.HasSuffix(path, "/rotate-pin") {
			workerID := strings.TrimSuffix(path, "/rotate-pin")
			rotateOpsStaffPIN(w, r, spannerClient, ops, workerID)
			return
		}

		workerID := strings.TrimSuffix(path, "/")
		if workerID == "" || strings.Contains(workerID, "/") {
			http.Error(w, "worker_id required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPatch:
			patchOpsStaff(w, r, spannerClient, ops, workerID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listOpsStaff(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT WorkerId, Name, Phone, COALESCE(Role, 'WAREHOUSE_STAFF'),
		             IsActive, CreatedAt, COALESCE(WarehouseId, '')
		      FROM WarehouseStaff
		      WHERE SupplierId = @sid AND WarehouseId = @whId
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
	}

	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var staff []StaffItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[WH STAFF] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var s StaffItem
		var createdAt time.Time
		if err := row.Columns(&s.WorkerID, &s.Name, &s.Phone, &s.Role,
			&s.IsActive, &createdAt, &s.WarehouseID); err != nil {
			log.Printf("[WH STAFF] parse: %v", err)
			continue
		}
		s.CreatedAt = createdAt.Format(time.RFC3339)
		staff = append(staff, s)
	}
	if staff == nil {
		staff = []StaffItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"staff": staff, "total": len(staff)})
}

func createOpsStaff(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"only warehouse admins can create staff"}`, http.StatusForbidden)
		return
	}

	var req CreateStaffReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Phone == "" || req.PIN == "" {
		http.Error(w, `{"error":"name, phone, and pin required"}`, http.StatusBadRequest)
		return
	}
	if len(req.PIN) < 8 {
		http.Error(w, `{"error":"pin must be at least 8 digits"}`, http.StatusBadRequest)
		return
	}
	// Warehouse admins can only create sub-staff, not other admins
	validRoles := map[string]bool{"WAREHOUSE_STAFF": true, "PAYLOADER": true}
	if !validRoles[req.Role] {
		req.Role = "WAREHOUSE_STAFF"
	}

	// Check duplicate phone
	dupStmt := spanner.Statement{
		SQL:    `SELECT WorkerId FROM WarehouseStaff WHERE Phone = @phone LIMIT 1`,
		Params: map[string]interface{}{"phone": req.Phone},
	}
	dupIter := client.Single().Query(r.Context(), dupStmt)
	dupRow, dupErr := dupIter.Next()
	dupIter.Stop()
	if dupErr == nil && dupRow != nil {
		http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
		return
	}

	workerID := uuid.New().String()
	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		bcryptHash, regErr := pin.RegisterExisting(ctx, txn, req.PIN, pin.EntityWarehouseStaff, workerID)
		if regErr != nil {
			return fmt.Errorf("register PIN: %w", regErr)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("WarehouseStaff",
				[]string{"WorkerId", "SupplierId", "WarehouseId", "Name", "Phone", "PinHash", "Role", "IsActive", "CreatedAt"},
				[]interface{}{workerID, ops.SupplierID, ops.WarehouseID, req.Name, req.Phone, bcryptHash, req.Role, true, spanner.CommitTimestamp},
			),
		})
	})
	if err != nil {
		if strings.Contains(err.Error(), "PIN already in use") {
			http.Error(w, `{"error":"pin already in use"}`, http.StatusConflict)
			return
		}
		log.Printf("[WH STAFF] insert error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"worker_id": workerID,
		"name":      req.Name,
		"role":      req.Role,
	})
}

func patchOpsStaff(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps, workerID string) {
	var req struct {
		IsActive *bool   `json:"is_active,omitempty"`
		Role     *string `json:"role,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"WorkerId"}
	vals := []interface{}{workerID}
	if req.IsActive != nil {
		cols = append(cols, "IsActive")
		vals = append(vals, *req.IsActive)
	}
	if req.Role != nil {
		valid := map[string]bool{"WAREHOUSE_STAFF": true, "PAYLOADER": true}
		if !valid[*req.Role] {
			http.Error(w, `{"error":"invalid role"}`, http.StatusBadRequest)
			return
		}
		cols = append(cols, "Role")
		vals = append(vals, *req.Role)
	}
	if len(cols) == 1 {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	m := spanner.Update("WarehouseStaff", cols, vals)
	if _, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[WH STAFF] patch error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "worker_id": workerID})
}

// rotateOpsStaffPIN generates a new globally-unique PIN for a warehouse staff member.
// POST /v1/warehouse/ops/staff/{id}/rotate-pin
func rotateOpsStaffPIN(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, ops *auth.WarehouseOps, workerID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if workerID == "" {
		http.Error(w, `{"error":"worker_id required"}`, http.StatusBadRequest)
		return
	}

	// Verify staff exists and belongs to this warehouse.
	row, err := spannerClient.Single().ReadRow(r.Context(), "WarehouseStaff",
		spanner.Key{workerID}, []string{"SupplierId", "WarehouseId"})
	if err != nil {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}
	var ownerSID, whID string
	if err := row.Columns(&ownerSID, &whID); err != nil || ownerSID != ops.SupplierID || whID != ops.WarehouseID {
		http.Error(w, `{"error":"staff not found"}`, http.StatusNotFound)
		return
	}

	var pinResult *pin.Result
	_, err = spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var rotErr error
		pinResult, rotErr = pin.Rotate(ctx, txn, pin.EntityWarehouseStaff, workerID)
		if rotErr != nil {
			return rotErr
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("WarehouseStaff", []string{"WorkerId", "PinHash"}, []interface{}{workerID, pinResult.BcryptHash}),
		})
	})
	if err != nil {
		log.Printf("[WH STAFF] rotate PIN failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"worker_id": workerID,
		"pin":       pinResult.Plaintext,
	})
}
