package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/pkg/pin"

	"cloud.google.com/go/spanner"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

var errFactoryRegisterPhoneConflict = errors.New("factory register phone already exists")

// HandleFactoryLogin authenticates a factory staff member with phone + PIN
// (primary) or phone + password (legacy fallback for pre-migration staff).
// POST /v1/auth/factory/login → { phone, pin } or { phone, password }
func HandleFactoryLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phone    string `json:"phone"`
			PIN      string `json:"pin"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.Phone == "" || (req.PIN == "" && req.Password == "") {
			http.Error(w, `{"error":"phone and pin (or password) are required"}`, http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT fs.StaffId, fs.Name, fs.PasswordHash, COALESCE(fs.PinHash, '') AS PinHash,
			             fs.StaffRole, fs.FactoryId,
			             fs.SupplierId, fs.IsActive, COALESCE(f.Name, '') AS FactoryName
			      FROM FactoryStaff fs
			      JOIN Factories f ON fs.FactoryId = f.FactoryId
			      WHERE fs.Phone = @phone LIMIT 1`,
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
			log.Printf("[FACTORY AUTH] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var staffID, name, passwordHash, pinHash, staffRole, factoryID, supplierID, factoryName string
		var isActive bool
		if err := row.Columns(&staffID, &name, &passwordHash, &pinHash, &staffRole, &factoryID,
			&supplierID, &isActive, &factoryName); err != nil {
			log.Printf("[FACTORY AUTH] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !isActive {
			http.Error(w, `{"error":"account deactivated"}`, http.StatusForbidden)
			return
		}

		// Dual-auth: try PIN first (new path), fall back to password (legacy).
		authOK := false
		if req.PIN != "" && pinHash != "" {
			if bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(req.PIN)) == nil {
				authOK = true
			}
		}
		if !authOK && req.Password != "" {
			if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) == nil {
				authOK = true
			}
		}
		if !authOK {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		// Mint JWT with FACTORY role + factory scope
		claims := &auth.PegasusClaims{
			UserID:      staffID,
			Role:        "FACTORY",
			FactoryID:   factoryID,
			FactoryRole: staffRole,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString(auth.JWTSecret)
		if err != nil {
			log.Printf("[FACTORY AUTH] token generation error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":        tokenStr,
			"user_id":      staffID,
			"role":         "FACTORY",
			"factory_role": staffRole,
			"factory_id":   factoryID,
			"factory_name": factoryName,
			"supplier_id":  supplierID,
			"name":         name,
		})
	}
}

// HandleFactoryRegister creates a new factory staff member.
// POST /v1/auth/factory/register — called by SUPPLIER (GLOBAL_ADMIN) to add staff.
func HandleFactoryRegister(spannerClient *spanner.Client) http.HandlerFunc {
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
			FactoryId string `json:"factory_id"`
			Name      string `json:"name"`
			Phone     string `json:"phone"`
			Password  string `json:"password"`
			StaffRole string `json:"staff_role"` // FACTORY_ADMIN | FACTORY_PAYLOADER
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.FactoryId == "" || req.Name == "" || req.Phone == "" || req.Password == "" {
			http.Error(w, `{"error":"factory_id, name, phone, and password are required"}`, http.StatusBadRequest)
			return
		}
		if len(req.Password) < 8 {
			http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
			return
		}
		if req.StaffRole != "FACTORY_ADMIN" && req.StaffRole != "FACTORY_PAYLOADER" {
			req.StaffRole = "FACTORY_PAYLOADER"
		}

		// Verify factory belongs to the supplier
		fRow, err := spannerClient.Single().ReadRow(r.Context(), "Factories",
			spanner.Key{req.FactoryId}, []string{"SupplierId"})
		if err != nil {
			http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
			return
		}
		var factorySupplierId string
		if err := fRow.Columns(&factorySupplierId); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if factorySupplierId != claims.ResolveSupplierID() {
			http.Error(w, `{"error":"factory does not belong to your organization"}`, http.StatusForbidden)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[FACTORY REGISTER] bcrypt error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		staffId := uuid.New().String()

		var pinResult *pin.Result
		_, txnErr := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			dupIter := txn.Query(ctx, spanner.Statement{
				SQL:    `SELECT StaffId FROM FactoryStaff WHERE Phone = @phone LIMIT 1`,
				Params: map[string]interface{}{"phone": req.Phone},
			})
			defer dupIter.Stop()
			if _, dupErr := dupIter.Next(); dupErr == nil {
				return errFactoryRegisterPhoneConflict
			} else if dupErr != iterator.Done {
				return fmt.Errorf("check duplicate phone: %w", dupErr)
			}

			var pinErr error
			pinResult, pinErr = pin.GenerateUnique(ctx, txn, pin.EntityFactoryStaff, staffId)
			if pinErr != nil {
				return fmt.Errorf("generate PIN: %w", pinErr)
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("FactoryStaff",
					[]string{"StaffId", "FactoryId", "SupplierId", "Name", "Phone", "PasswordHash", "PinHash", "StaffRole", "IsActive", "CreatedAt"},
					[]interface{}{staffId, req.FactoryId, claims.ResolveSupplierID(), req.Name, req.Phone, string(hash), pinResult.BcryptHash, req.StaffRole, true, spanner.CommitTimestamp},
				),
			})
		})
		if txnErr != nil {
			if errors.Is(txnErr, errFactoryRegisterPhoneConflict) {
				http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
				return
			}
			log.Printf("[FACTORY REGISTER] spanner insert error: %v", txnErr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"staff_id":   staffId,
			"factory_id": req.FactoryId,
			"name":       req.Name,
			"staff_role": req.StaffRole,
			"pin":        pinResult.Plaintext,
		})
	}
}
