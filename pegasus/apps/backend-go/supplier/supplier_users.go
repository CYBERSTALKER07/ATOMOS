package supplier

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// ── Supplier Users (Org Sub-Accounts) — GLOBAL_ADMIN manages the org pyramid ─
//
// Endpoints:
//   GET    /v1/supplier/org/members         → list all SupplierUsers in the organization
//   POST   /v1/supplier/org/members/invite  → create a new sub-account (GLOBAL_ADMIN only)
//   PUT    /v1/supplier/org/members/{id}    → update role/warehouse/active (GLOBAL_ADMIN only)
//   DELETE /v1/supplier/org/members/{id}    → soft-delete (IsActive=false) (GLOBAL_ADMIN only)
//
// On first invite, the root Suppliers registrant is auto-mirrored into
// SupplierUsers as GLOBAL_ADMIN so the org has a canonical user list.

// ── DTOs ──────────────────────────────────────────────────────────────────────

type OrgMember struct {
	UserId              string `json:"user_id"`
	SupplierId          string `json:"supplier_id"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Phone               string `json:"phone"`
	SupplierRole        string `json:"supplier_role"`
	AssignedWarehouseId string `json:"assigned_warehouse_id"`
	AssignedFactoryId   string `json:"assigned_factory_id"`
	IsActive            bool   `json:"is_active"`
	CreatedAt           string `json:"created_at"`
}

// ── Route Handlers ────────────────────────────────────────────────────────────

// HandleOrgMembers routes GET for /v1/supplier/org/members
func HandleOrgMembers(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listOrgMembers(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleOrgInvite creates a new sub-account. POST /v1/supplier/org/members/invite
func HandleOrgInvite(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		inviteOrgMember(w, r, spannerClient)
	}
}

// HandleOrgMemberAction routes PUT and DELETE for /v1/supplier/org/members/{id}
func HandleOrgMemberAction(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remainder := strings.TrimPrefix(r.URL.Path, "/v1/supplier/org/members/")
		if remainder == "" || remainder == "invite" || strings.Contains(remainder, "/") {
			http.Error(w, `{"error":"user_id required in path"}`, http.StatusBadRequest)
			return
		}
		userID := remainder

		switch r.Method {
		case http.MethodPut:
			updateOrgMember(w, r, spannerClient, userID)
		case http.MethodDelete:
			deactivateOrgMember(w, r, spannerClient, userID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── Implementations ───────────────────────────────────────────────────────────

func listOrgMembers(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	supplierID, err := resolveSupplierID(r.Context(), client, claims)
	if err != nil || supplierID == "" {
		http.Error(w, `{"error":"supplier_resolve_failed"}`, http.StatusInternalServerError)
		return
	}

	// NODE_ADMIN sees only their warehouse colleagues
	scope := auth.GetWarehouseScope(r.Context())

	sql := `SELECT UserId, SupplierId, Name, COALESCE(Email, ''), COALESCE(Phone, ''),
	               SupplierRole, COALESCE(AssignedWarehouseId, ''), COALESCE(AssignedFactoryId, ''),
	               IsActive, CreatedAt
	        FROM SupplierUsers
	        WHERE SupplierId = @supplierID`
	params := map[string]interface{}{"supplierID": supplierID}

	if scope != nil && scope.IsNodeAdmin && scope.WarehouseID != "" {
		sql += ` AND (AssignedWarehouseId = @warehouseID OR AssignedWarehouseId IS NULL)`
		params["warehouseID"] = scope.WarehouseID
	}

	sql += ` ORDER BY CreatedAt ASC`

	iter := client.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	members := []OrgMember{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[ORG] list query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var m OrgMember
		var createdAt time.Time
		if err := row.Columns(&m.UserId, &m.SupplierId, &m.Name, &m.Email, &m.Phone,
			&m.SupplierRole, &m.AssignedWarehouseId, &m.AssignedFactoryId, &m.IsActive, &createdAt); err != nil {
			log.Printf("[ORG] parse error: %v", err)
			continue
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		members = append(members, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": members, "count": len(members)})
}

func inviteOrgMember(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}

	supplierID, err := resolveSupplierID(r.Context(), client, claims)
	if err != nil || supplierID == "" {
		http.Error(w, `{"error":"supplier_resolve_failed"}`, http.StatusInternalServerError)
		return
	}

	var req struct {
		Name                string `json:"name"`
		Email               string `json:"email"`
		Phone               string `json:"phone"`
		Password            string `json:"password"`
		SupplierRole        string `json:"supplier_role"`
		AssignedWarehouseId string `json:"assigned_warehouse_id"`
		AssignedFactoryId   string `json:"assigned_factory_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)

	if req.Name == "" || req.Password == "" {
		http.Error(w, `{"error":"name and password are required"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" && req.Phone == "" {
		http.Error(w, `{"error":"email or phone is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}
	if req.SupplierRole != "GLOBAL_ADMIN" && req.SupplierRole != "NODE_ADMIN" &&
		req.SupplierRole != "FACTORY_ADMIN" && req.SupplierRole != "FACTORY_PAYLOADER" {
		http.Error(w, `{"error":"supplier_role must be GLOBAL_ADMIN, NODE_ADMIN, FACTORY_ADMIN, or FACTORY_PAYLOADER"}`, http.StatusBadRequest)
		return
	}
	if req.SupplierRole == "NODE_ADMIN" && req.AssignedWarehouseId == "" {
		http.Error(w, `{"error":"assigned_warehouse_id is required for NODE_ADMIN"}`, http.StatusBadRequest)
		return
	}
	if (req.SupplierRole == "FACTORY_ADMIN" || req.SupplierRole == "FACTORY_PAYLOADER") && req.AssignedFactoryId == "" {
		http.Error(w, `{"error":"assigned_factory_id is required for factory roles"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Validate warehouse belongs to this supplier
	if req.AssignedWarehouseId != "" {
		var whSupplierId string
		_ = client.Single().Query(ctx, spanner.Statement{
			SQL:    "SELECT SupplierId FROM Warehouses WHERE WarehouseId = @wid",
			Params: map[string]interface{}{"wid": req.AssignedWarehouseId},
		}).Do(func(row *spanner.Row) error { return row.Columns(&whSupplierId) })
		if whSupplierId != supplierID {
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusBadRequest)
			return
		}
	}

	// Validate factory belongs to this supplier
	if req.AssignedFactoryId != "" {
		var facSupplierId string
		_ = client.Single().Query(ctx, spanner.Statement{
			SQL:    "SELECT SupplierId FROM Factories WHERE FactoryId = @fid",
			Params: map[string]interface{}{"fid": req.AssignedFactoryId},
		}).Do(func(row *spanner.Row) error { return row.Columns(&facSupplierId) })
		if facSupplierId != supplierID {
			http.Error(w, `{"error":"factory does not belong to your organization"}`, http.StatusBadRequest)
			return
		}
	}

	// Check uniqueness (email or phone across SupplierUsers)
	if req.Email != "" {
		var existingID string
		_ = client.Single().Query(ctx, spanner.Statement{
			SQL:    "SELECT UserId FROM SupplierUsers WHERE Email = @email LIMIT 1",
			Params: map[string]interface{}{"email": req.Email},
		}).Do(func(row *spanner.Row) error { return row.Columns(&existingID) })
		if existingID != "" {
			http.Error(w, `{"error":"email already registered"}`, http.StatusConflict)
			return
		}
	}
	if req.Phone != "" {
		var existingID string
		_ = client.Single().Query(ctx, spanner.Statement{
			SQL:    "SELECT UserId FROM SupplierUsers WHERE Phone = @phone LIMIT 1",
			Params: map[string]interface{}{"phone": req.Phone},
		}).Do(func(row *spanner.Row) error { return row.Columns(&existingID) })
		if existingID != "" {
			http.Error(w, `{"error":"phone already registered"}`, http.StatusConflict)
			return
		}
	}

	// Auto-mirror root supplier on first invite
	ensureRootMirrored(ctx, client, claims, supplierID)

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"password hashing failed"}`, http.StatusInternalServerError)
		return
	}

	userID := uuid.New().String()

	cols := []string{"UserId", "SupplierId", "Name", "Email", "Phone", "PasswordHash",
		"SupplierRole", "IsActive", "CreatedAt"}
	vals := []interface{}{userID, supplierID, req.Name, req.Email, req.Phone, string(hash),
		req.SupplierRole, true, spanner.CommitTimestamp}

	if req.AssignedWarehouseId != "" {
		cols = append(cols, "AssignedWarehouseId")
		vals = append(vals, req.AssignedWarehouseId)
	}
	if req.AssignedFactoryId != "" {
		cols = append(cols, "AssignedFactoryId")
		vals = append(vals, req.AssignedFactoryId)
	}

	m := spanner.Insert("SupplierUsers", cols, vals)
	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[ORG] invite insert error: %v", err)
		http.Error(w, `{"error":"invite_failed"}`, http.StatusInternalServerError)
		return
	}

	// Create Firebase Auth user (graceful)
	if auth.FirebaseAuthClient != nil {
		fbUid, fbErr := auth.CreateFirebaseUser(ctx, req.Email, req.Password, req.Name, req.Phone, "SUPPLIER", map[string]interface{}{
			"supplier_id":   supplierID,
			"supplier_role": req.SupplierRole,
			"warehouse_id":  req.AssignedWarehouseId,
			"factory_id":    req.AssignedFactoryId,
		})
		if fbErr == nil && fbUid != "" {
			_, _ = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("SupplierUsers",
						[]string{"UserId", "FirebaseUid"},
						[]interface{}{userID, fbUid}),
				})
			})
		}
	}

	cache.Invalidate(ctx, cache.SupplierProfile(supplierID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":       userID,
		"supplier_id":   supplierID,
		"name":          req.Name,
		"supplier_role": req.SupplierRole,
		"warehouse_id":  req.AssignedWarehouseId,
		"factory_id":    req.AssignedFactoryId,
		"status":        "invited",
	})
}

func updateOrgMember(w http.ResponseWriter, r *http.Request, client *spanner.Client, targetUserID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}

	supplierID, err := resolveSupplierID(r.Context(), client, claims)
	if err != nil || supplierID == "" {
		http.Error(w, `{"error":"supplier_resolve_failed"}`, http.StatusInternalServerError)
		return
	}

	// Verify target belongs to same supplier
	var targetSid string
	var targetCurrentRole string
	iter := client.Single().Query(r.Context(), spanner.Statement{
		SQL:    "SELECT SupplierId, SupplierRole FROM SupplierUsers WHERE UserId = @uid",
		Params: map[string]interface{}{"uid": targetUserID},
	})
	defer iter.Stop()
	row, err2 := iter.Next()
	if err2 != nil {
		http.Error(w, `{"error":"user not found in your organization"}`, http.StatusNotFound)
		return
	}
	if err2 := row.Columns(&targetSid, &targetCurrentRole); err2 != nil {
		http.Error(w, `{"error":"user not found in your organization"}`, http.StatusNotFound)
		return
	}
	iter.Stop()
	if targetSid != supplierID {
		http.Error(w, `{"error":"user not found in your organization"}`, http.StatusNotFound)
		return
	}

	var req struct {
		SupplierRole        *string `json:"supplier_role,omitempty"`
		AssignedWarehouseId *string `json:"assigned_warehouse_id,omitempty"`
		AssignedFactoryId   *string `json:"assigned_factory_id,omitempty"`
		IsActive            *bool   `json:"is_active,omitempty"`
		Name                *string `json:"name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	// ── Self-protection: cannot demote or deactivate yourself ──
	isSelf := targetUserID == claims.UserID
	if isSelf {
		if req.SupplierRole != nil && *req.SupplierRole != "GLOBAL_ADMIN" {
			http.Error(w, `{"error":"cannot demote your own account"}`, http.StatusForbidden)
			return
		}
		if req.IsActive != nil && !*req.IsActive {
			http.Error(w, `{"error":"cannot deactivate your own account"}`, http.StatusForbidden)
			return
		}
	}

	// ── Root lock: cannot demote or deactivate the original registrant ──
	// The root user has UserId == SupplierId (mirrored at T=0)
	isRoot := targetUserID == supplierID
	if isRoot && !isSelf {
		if req.SupplierRole != nil && *req.SupplierRole != "GLOBAL_ADMIN" {
			http.Error(w, `{"error":"cannot demote the organization founder"}`, http.StatusForbidden)
			return
		}
		if req.IsActive != nil && !*req.IsActive {
			http.Error(w, `{"error":"cannot deactivate the organization founder"}`, http.StatusForbidden)
			return
		}
	}

	cols := []string{"UserId"}
	vals := []interface{}{targetUserID}

	if req.SupplierRole != nil {
		if *req.SupplierRole != "GLOBAL_ADMIN" && *req.SupplierRole != "NODE_ADMIN" &&
			*req.SupplierRole != "FACTORY_ADMIN" && *req.SupplierRole != "FACTORY_PAYLOADER" {
			http.Error(w, `{"error":"supplier_role must be GLOBAL_ADMIN, NODE_ADMIN, FACTORY_ADMIN, or FACTORY_PAYLOADER"}`, http.StatusBadRequest)
			return
		}
		cols = append(cols, "SupplierRole")
		vals = append(vals, *req.SupplierRole)

		// ── Scope-clearing on role transitions ──
		// GLOBAL_ADMIN: unscoped — clear both assignments
		if *req.SupplierRole == "GLOBAL_ADMIN" {
			cols = append(cols, "AssignedWarehouseId", "AssignedFactoryId")
			vals = append(vals, "", "")
		}
		// NODE_ADMIN: clear factory scope (warehouse must be set explicitly)
		if *req.SupplierRole == "NODE_ADMIN" {
			cols = append(cols, "AssignedFactoryId")
			vals = append(vals, "")
		}
		// FACTORY_*: clear warehouse scope (factory must be set explicitly)
		if *req.SupplierRole == "FACTORY_ADMIN" || *req.SupplierRole == "FACTORY_PAYLOADER" {
			cols = append(cols, "AssignedWarehouseId")
			vals = append(vals, "")
		}
	}
	if req.AssignedWarehouseId != nil {
		if *req.AssignedWarehouseId != "" {
			var whSid string
			_ = client.Single().Query(r.Context(), spanner.Statement{
				SQL:    "SELECT SupplierId FROM Warehouses WHERE WarehouseId = @wid",
				Params: map[string]interface{}{"wid": *req.AssignedWarehouseId},
			}).Do(func(row *spanner.Row) error { return row.Columns(&whSid) })
			if whSid != supplierID {
				http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusBadRequest)
				return
			}
		}
		cols = append(cols, "AssignedWarehouseId")
		vals = append(vals, *req.AssignedWarehouseId)
	}
	if req.AssignedFactoryId != nil {
		if *req.AssignedFactoryId != "" {
			var facSid string
			_ = client.Single().Query(r.Context(), spanner.Statement{
				SQL:    "SELECT SupplierId FROM Factories WHERE FactoryId = @fid",
				Params: map[string]interface{}{"fid": *req.AssignedFactoryId},
			}).Do(func(row *spanner.Row) error { return row.Columns(&facSid) })
			if facSid != supplierID {
				http.Error(w, `{"error":"factory does not belong to your organization"}`, http.StatusBadRequest)
				return
			}
		}
		cols = append(cols, "AssignedFactoryId")
		vals = append(vals, *req.AssignedFactoryId)
	}
	if req.IsActive != nil {
		cols = append(cols, "IsActive")
		vals = append(vals, *req.IsActive)
	}
	if req.Name != nil {
		cols = append(cols, "Name")
		vals = append(vals, strings.TrimSpace(*req.Name))
	}

	if len(cols) == 1 {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	mut := spanner.Update("SupplierUsers", cols, vals)
	if _, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{mut})
	}); err != nil {
		log.Printf("[ORG] update error for %s: %v", targetUserID, err)
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	cache.Invalidate(r.Context(), cache.SupplierProfile(supplierID))

	// Read back the updated member for enriched response (avoids frontend re-fetch)
	var updatedRole, updatedWarehouse, updatedFactory, updatedName string
	var updatedActive bool
	readIter := client.Single().Query(r.Context(), spanner.Statement{
		SQL: `SELECT SupplierRole, COALESCE(AssignedWarehouseId, ''), COALESCE(AssignedFactoryId, ''),
		             IsActive, COALESCE(Name, '')
		      FROM SupplierUsers WHERE UserId = @uid`,
		Params: map[string]interface{}{"uid": targetUserID},
	})
	readRow, readErr := readIter.Next()
	if readErr == nil {
		_ = readRow.Columns(&updatedRole, &updatedWarehouse, &updatedFactory, &updatedActive, &updatedName)
	}
	readIter.Stop()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":                "updated",
		"user_id":               targetUserID,
		"supplier_role":         updatedRole,
		"assigned_warehouse_id": updatedWarehouse,
		"assigned_factory_id":   updatedFactory,
		"is_active":             updatedActive,
		"name":                  updatedName,
	})
}

func deactivateOrgMember(w http.ResponseWriter, r *http.Request, client *spanner.Client, targetUserID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}

	if targetUserID == claims.UserID {
		http.Error(w, `{"error":"cannot deactivate your own account"}`, http.StatusForbidden)
		return
	}

	supplierID, err := resolveSupplierID(r.Context(), client, claims)
	if err != nil || supplierID == "" {
		http.Error(w, `{"error":"supplier_resolve_failed"}`, http.StatusInternalServerError)
		return
	}

	// Root lock: the original registrant (UserId == SupplierId) cannot be deactivated
	if targetUserID == supplierID {
		http.Error(w, `{"error":"cannot deactivate the organization founder"}`, http.StatusForbidden)
		return
	}

	var targetSid string
	_ = client.Single().Query(r.Context(), spanner.Statement{
		SQL:    "SELECT SupplierId FROM SupplierUsers WHERE UserId = @uid",
		Params: map[string]interface{}{"uid": targetUserID},
	}).Do(func(row *spanner.Row) error { return row.Columns(&targetSid) })
	if targetSid != supplierID {
		http.Error(w, `{"error":"user not found in your organization"}`, http.StatusNotFound)
		return
	}

	mut := spanner.Update("SupplierUsers",
		[]string{"UserId", "IsActive"},
		[]interface{}{targetUserID, false})
	if _, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{mut})
	}); err != nil {
		log.Printf("[ORG] deactivate error for %s: %v", targetUserID, err)
		http.Error(w, `{"error":"deactivate_failed"}`, http.StatusInternalServerError)
		return
	}

	cache.Invalidate(r.Context(), cache.SupplierProfile(supplierID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated", "user_id": targetUserID})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// resolveSupplierID returns the SupplierId for the authenticated user.
// For SupplierUsers (sub-accounts), UserID != SupplierId — we look it up.
// For root Suppliers table users, UserID == SupplierId.
func resolveSupplierID(ctx context.Context, client *spanner.Client, claims *auth.PegasusClaims) (string, error) {
	var sid string
	_ = client.Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT SupplierId FROM SupplierUsers WHERE UserId = @uid LIMIT 1",
		Params: map[string]interface{}{"uid": claims.UserID},
	}).Do(func(row *spanner.Row) error { return row.Columns(&sid) })
	if sid != "" {
		return sid, nil
	}

	// Fallback: root supplier — UserID IS the SupplierId
	var exists bool
	_ = client.Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT true FROM Suppliers WHERE SupplierId = @sid LIMIT 1",
		Params: map[string]interface{}{"sid": claims.ResolveSupplierID()},
	}).Do(func(row *spanner.Row) error { return row.Columns(&exists) })
	if exists {
		return claims.ResolveSupplierID(), nil
	}

	return "", nil
}

// ensureRootMirrored auto-creates a SupplierUsers row for the root Suppliers
// registrant as GLOBAL_ADMIN if they don't already exist in SupplierUsers.
func ensureRootMirrored(ctx context.Context, client *spanner.Client, claims *auth.PegasusClaims, supplierID string) {
	var existingUID string
	_ = client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT UserId FROM SupplierUsers WHERE SupplierId = @sid AND UserId = @sid LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID},
	}).Do(func(row *spanner.Row) error { return row.Columns(&existingUID) })
	if existingUID != "" {
		return
	}

	var name, email, phone, pwHash string
	_ = client.Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT Name, COALESCE(Email, ''), COALESCE(Phone, ''), COALESCE(PasswordHash, '') FROM Suppliers WHERE SupplierId = @sid",
		Params: map[string]interface{}{"sid": supplierID},
	}).Do(func(row *spanner.Row) error { return row.Columns(&name, &email, &phone, &pwHash) })
	if name == "" {
		return
	}

	m := spanner.InsertOrUpdate("SupplierUsers",
		[]string{"UserId", "SupplierId", "Name", "Email", "Phone", "PasswordHash",
			"SupplierRole", "IsActive", "CreatedAt"},
		[]interface{}{supplierID, supplierID, name, email, phone, pwHash,
			"GLOBAL_ADMIN", true, spanner.CommitTimestamp},
	)
	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[ORG] auto-mirror root supplier %s failed: %v", supplierID, err)
	} else {
		cache.Invalidate(ctx, cache.SupplierProfile(supplierID))
		log.Printf("[ORG] Auto-mirrored root supplier %s into SupplierUsers as GLOBAL_ADMIN", supplierID)
	}
}
