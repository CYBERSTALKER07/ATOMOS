package warehouse

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// HandleWarehouseLogin authenticates a warehouse staff member with phone + PIN.
// POST /v1/auth/warehouse/login → { phone, pin }
func HandleWarehouseLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phone string `json:"phone"`
			PIN   string `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.Phone == "" || req.PIN == "" {
			http.Error(w, `{"error":"phone and pin are required"}`, http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT ws.WorkerId, ws.Name, ws.PinHash, ws.SupplierId,
			             ws.WarehouseId, ws.IsActive, ws.Role,
			             COALESCE(wh.Name, '') AS WarehouseName
			      FROM WarehouseStaff ws
			      LEFT JOIN Warehouses wh ON ws.WarehouseId = wh.WarehouseId
			      WHERE ws.Phone = @phone LIMIT 1`,
			Params: map[string]interface{}{"phone": req.Phone},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("[WAREHOUSE AUTH] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var workerID, name, pinHash, supplierID, warehouseID string
		var isActive bool
		var role spanner.NullString
		var warehouseName string
		if err := row.Columns(&workerID, &name, &pinHash, &supplierID,
			&warehouseID, &isActive, &role, &warehouseName); err != nil {
			log.Printf("[WAREHOUSE AUTH] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !isActive {
			http.Error(w, `{"error":"account deactivated"}`, http.StatusForbidden)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(req.PIN)); err != nil {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		warehouseRole := "WAREHOUSE_STAFF"
		if role.Valid && role.StringVal != "" {
			warehouseRole = role.StringVal
		}

		// Mint JWT with WAREHOUSE role + warehouse scope
		claims := &auth.PegasusClaims{
			UserID:        workerID,
			Role:          "WAREHOUSE",
			WarehouseID:   warehouseID,
			WarehouseRole: warehouseRole,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString(auth.JWTSecret)
		if err != nil {
			log.Printf("[WAREHOUSE AUTH] token generation error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":          tokenStr,
			"user_id":        workerID,
			"role":           "WAREHOUSE",
			"warehouse_role": warehouseRole,
			"warehouse_id":   warehouseID,
			"warehouse_name": warehouseName,
			"supplier_id":    supplierID,
			"name":           name,
		})
	}
}

// HandleWarehouseRegister creates a new warehouse staff member.
// POST /v1/auth/warehouse/register — called by SUPPLIER to add warehouse staff.
func HandleWarehouseRegister(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			WarehouseId string `json:"warehouse_id"`
			Name        string `json:"name"`
			Phone       string `json:"phone"`
			PIN         string `json:"pin"`
			Role        string `json:"role"` // WAREHOUSE_ADMIN | WAREHOUSE_STAFF | PAYLOADER
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.WarehouseId == "" || req.Name == "" || req.Phone == "" || req.PIN == "" {
			http.Error(w, `{"error":"warehouse_id, name, phone, and pin are required"}`, http.StatusBadRequest)
			return
		}
		if len(req.PIN) < 6 {
			http.Error(w, `{"error":"pin must be at least 6 digits"}`, http.StatusBadRequest)
			return
		}
		validRoles := map[string]bool{"WAREHOUSE_ADMIN": true, "WAREHOUSE_STAFF": true, "PAYLOADER": true}
		if !validRoles[req.Role] {
			req.Role = "WAREHOUSE_STAFF"
		}

		// Verify warehouse belongs to the supplier
		whRow, err := spannerClient.Single().ReadRow(r.Context(), "Warehouses",
			spanner.Key{req.WarehouseId}, []string{"SupplierId"})
		if err != nil {
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		}
		var warehouseSupplierId string
		if err := whRow.Columns(&warehouseSupplierId); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if warehouseSupplierId != claims.ResolveSupplierID() {
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
			return
		}

		// Check duplicate phone
		dupStmt := spanner.Statement{
			SQL:    `SELECT WorkerId FROM WarehouseStaff WHERE Phone = @phone LIMIT 1`,
			Params: map[string]interface{}{"phone": req.Phone},
		}
		dupIter := spannerClient.Single().Query(r.Context(), dupStmt)
		dupRow, dupErr := dupIter.Next()
		dupIter.Stop()
		if dupErr == nil && dupRow != nil {
			http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[WAREHOUSE REGISTER] bcrypt error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		workerId := uuid.New().String()
		m := spanner.Insert("WarehouseStaff",
			[]string{"WorkerId", "SupplierId", "WarehouseId", "Name", "Phone", "PinHash", "Role", "IsActive", "CreatedAt"},
			[]interface{}{workerId, claims.ResolveSupplierID(), req.WarehouseId, req.Name, req.Phone, string(hash), req.Role, true, spanner.CommitTimestamp},
		)
		if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{m})
		}); err != nil {
			log.Printf("[WAREHOUSE REGISTER] spanner insert error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"worker_id":    workerId,
			"warehouse_id": req.WarehouseId,
			"name":         req.Name,
			"role":         req.Role,
		})
	}
}
