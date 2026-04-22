package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// HandleAdminRegister creates a new admin account (email + password).
func HandleAdminRegister(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email       string `json:"email"`
			Password    string `json:"password"`
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.DisplayName = strings.TrimSpace(req.DisplayName)

		if req.Email == "" || req.Password == "" || req.DisplayName == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "email, password, and display_name are required"})
			return
		}
		if len(req.Password) < 8 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Password must be at least 8 characters"})
			return
		}

		// Check if email already exists
		ctx := r.Context()
		stmt := spanner.Statement{
			SQL:    "SELECT AdminId FROM Admins WHERE Email = @email LIMIT 1",
			Params: map[string]interface{}{"email": req.Email},
		}
		iter := spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()
		_, err := iter.Next()
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email already registered"})
			return
		}
		if err != iterator.Done {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Password hashing failed"})
			return
		}

		adminId := uuid.New().String()
		m := spanner.InsertMap("Admins", map[string]interface{}{
			"AdminId":      adminId,
			"Email":        req.Email,
			"PasswordHash": string(hash),
			"DisplayName":  req.DisplayName,
			"CreatedAt":    spanner.CommitTimestamp,
		})
		if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{m}); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Insert failed: %v", err)})
			return
		}

		token, err := GenerateTestToken(adminId, "ADMIN")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Token generation failed"})
			return
		}

		// Create Firebase Auth user + mint custom token (graceful — skipped if Firebase is nil)
		var firebaseToken string
		fbUid, fbErr := CreateFirebaseUser(ctx, req.Email, req.Password, req.DisplayName, "", "ADMIN", map[string]interface{}{
			"admin_id": adminId,
		})
		if fbErr == nil && fbUid != "" {
			// Store Firebase UID in Spanner
			_, _ = spannerClient.Apply(ctx, []*spanner.Mutation{
				spanner.Update("Admins", []string{"AdminId", "FirebaseUid"}, []interface{}{adminId, fbUid}),
			})
			firebaseToken, _ = MintCustomToken(ctx, fbUid, map[string]interface{}{"role": "ADMIN", "admin_id": adminId})
		}

		resp := map[string]interface{}{
			"token":        token,
			"role":         "ADMIN",
			"admin_id":     adminId,
			"display_name": req.DisplayName,
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

// HandleAdminLogin authenticates by email + password.
// It first checks the Admins table, then falls through to the Suppliers table
// because the Admin Portal IS the Supplier Portal (same product user).
func HandleAdminLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		ctx := r.Context()

		// ── 1. Try Admins table first ──────────────────────────────────
		var adminId, passwordHash, displayName string
		_ = spannerClient.Single().Query(ctx, spanner.Statement{
			SQL:    "SELECT AdminId, PasswordHash, DisplayName FROM Admins WHERE Email = @email LIMIT 1",
			Params: map[string]interface{}{"email": req.Email},
		}).Do(func(row *spanner.Row) error {
			return row.Columns(&adminId, &passwordHash, &displayName)
		})

		if adminId != "" && passwordHash != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
				return
			}
			token, err := GenerateTestToken(adminId, "ADMIN")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Token generation failed"})
				return
			}
			// Mint Firebase custom token for the admin
			var firebaseToken string
			fbUid := LookupFirebaseUID(ctx, req.Email)
			if fbUid != "" {
				firebaseToken, _ = MintCustomToken(ctx, fbUid, map[string]interface{}{"role": "ADMIN", "admin_id": adminId})
			}
			resp := map[string]interface{}{
				"token":        token,
				"role":         "ADMIN",
				"display_name": displayName,
			}
			if firebaseToken != "" {
				resp["firebase_token"] = firebaseToken
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// ── 2. Try SupplierUsers table (sub-accounts with explicit roles) ──
		var suUserID, suSupplierId, suName, suPwHash, suRole, suWarehouseID, suFactoryID string
		_ = spannerClient.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT UserId, SupplierId, Name, PasswordHash, SupplierRole,
			             COALESCE(AssignedWarehouseId, ''),
			             COALESCE(AssignedFactoryId, '')
			      FROM SupplierUsers WHERE Email = @email AND IsActive = true LIMIT 1`,
			Params: map[string]interface{}{"email": req.Email},
		}).Do(func(row *spanner.Row) error {
			return row.Columns(&suUserID, &suSupplierId, &suName, &suPwHash, &suRole, &suWarehouseID, &suFactoryID)
		})

		if suUserID != "" && suPwHash != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(suPwHash), []byte(req.Password)); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
				return
			}
			// Determine factory role from supplier role for claims
			factoryRole := ""
			if suRole == "FACTORY_ADMIN" || suRole == "FACTORY_PAYLOADER" {
				factoryRole = suRole
			}
			token, err := MintIdentityToken(&LabClaims{
				UserID:       suUserID,
				Role:         "SUPPLIER",
				SupplierRole: suRole,
				WarehouseID:  suWarehouseID,
				FactoryID:    suFactoryID,
				FactoryRole:  factoryRole,
			})
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Token generation failed"})
				return
			}
			var firebaseToken string
			fbUid := LookupFirebaseUID(ctx, req.Email)
			if fbUid != "" {
				firebaseToken, _ = MintCustomToken(ctx, fbUid, map[string]interface{}{
					"role":          "SUPPLIER",
					"supplier_id":   suSupplierId,
					"supplier_role": suRole,
					"warehouse_id":  suWarehouseID,
					"factory_id":    suFactoryID,
					"factory_role":  factoryRole,
				})
			}
			resp := map[string]interface{}{
				"token":         token,
				"role":          "SUPPLIER",
				"supplier_role": suRole,
				"warehouse_id":  suWarehouseID,
				"factory_id":    suFactoryID,
				"display_name":  suName,
			}
			if firebaseToken != "" {
				resp["firebase_token"] = firebaseToken
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// ── 3. Fall through to Suppliers table (root registrant = implicit GLOBAL_ADMIN) ──
		var supplierId, supplierPwHash, supplierName string
		_ = spannerClient.Single().Query(ctx, spanner.Statement{
			SQL:    "SELECT SupplierId, COALESCE(PasswordHash, ''), Name FROM Suppliers WHERE Email = @email LIMIT 1",
			Params: map[string]interface{}{"email": req.Email},
		}).Do(func(row *spanner.Row) error {
			return row.Columns(&supplierId, &supplierPwHash, &supplierName)
		})

		if supplierId == "" || supplierPwHash == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(supplierPwHash), []byte(req.Password)); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}

		token, err := MintIdentityToken(&LabClaims{
			UserID:       supplierId,
			Role:         "SUPPLIER",
			SupplierRole: "GLOBAL_ADMIN",
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Token generation failed"})
			return
		}

		// Mint Firebase custom token for the supplier
		var firebaseToken string
		fbUid := LookupFirebaseUID(ctx, req.Email)
		if fbUid != "" {
			firebaseToken, _ = MintCustomToken(ctx, fbUid, map[string]interface{}{
				"role":          "SUPPLIER",
				"supplier_id":   supplierId,
				"supplier_role": "GLOBAL_ADMIN",
			})
		}
		resp := map[string]interface{}{
			"token":         token,
			"role":          "SUPPLIER",
			"supplier_role": "GLOBAL_ADMIN",
			"display_name":  supplierName,
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// SeedDefaultAdmin inserts the default admin if no admins exist yet.
func SeedDefaultAdmin(ctx context.Context, spannerClient *spanner.Client) {
	iter := spannerClient.Single().Query(ctx, spanner.Statement{SQL: "SELECT AdminId FROM Admins LIMIT 1"})
	defer iter.Stop()
	if _, err := iter.Next(); err == iterator.Done {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		m := spanner.InsertMap("Admins", map[string]interface{}{
			"AdminId":      "ADMIN-001",
			"Email":        "admin@thelab.uz",
			"PasswordHash": string(hash),
			"DisplayName":  "Platform Admin",
			"CreatedAt":    spanner.CommitTimestamp,
		})
		if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{m}); err == nil {
			fmt.Println("[ADMIN SEED] Default admin created: admin@thelab.uz / admin123")
		}
	}
}
