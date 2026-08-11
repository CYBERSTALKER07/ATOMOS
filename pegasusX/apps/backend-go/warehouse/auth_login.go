package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

type warehouseStaffRecord struct {
	UserID              string
	SupplierID          string
	Name                string
	Phone               string
	PasswordHash        string
	SupplierRole        string
	AssignedWarehouseID string
	IsActive            bool
}

// HandleWarehouseLogin authenticates warehouse staff (phone + PIN) for native clients.
// POST /v1/auth/warehouse/login
func (s *Service) HandleWarehouseLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()

	var req struct {
		Phone    string `json:"phone"`
		PIN      string `json:"pin"`
		Password string `json:"password"`
		IDToken  string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	var staff warehouseStaffRecord
	var lookupPhone string

	idToken := strings.TrimSpace(req.IDToken)
	if idToken != "" && s.firebaseVerifier != nil {
		fbClaims, err := s.firebaseVerifier.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			s.log.Warn("firebase token verification failed", "err", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_id_token"})
			return
		}
		if fbClaims.PhoneNumber == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "phone_number_missing_in_token"})
			return
		}
		lookupPhone = fbClaims.PhoneNumber
	} else {
		lookupPhone = strings.TrimSpace(req.Phone)
		secret := strings.TrimSpace(req.PIN)
		if secret == "" {
			secret = strings.TrimSpace(req.Password)
		}
		if lookupPhone == "" || secret == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
			return
		}
		rec, found, err := s.lookupWarehouseStaffByPhone(r.Context(), lookupPhone)
		if err != nil {
			s.log.ErrorContext(r.Context(), "warehouse staff lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
		if !verifyWarehouseStaffSecret(rec.PasswordHash, secret) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
		staff = rec
	}

	if staff.UserID == "" {
		rec, found, err := s.lookupWarehouseStaffByPhone(r.Context(), lookupPhone)
		if err != nil {
			s.log.ErrorContext(r.Context(), "warehouse staff lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
		staff = rec
	}

	warehouseID := strings.TrimSpace(staff.AssignedWarehouseID)
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	supplierID := strings.TrimSpace(staff.SupplierID)
	if supplierID == "" {
		supplierID = s.resolveSupplierScope(r.Context())
	}

	isConfigured := false
	if warehouseID != "" {
		isConfigured = s.warehouseIsConfigured(r.Context(), warehouseID)
	}

	jwtClaims := auth.Claims{
		Subject:      staff.UserID,
		Role:         auth.RoleWarehouse,
		SupplierID:   supplierID,
		SupplierRole: auth.Role(strings.TrimSpace(staff.SupplierRole)),
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
		IsConfigured: isConfigured,
		PhoneNumber:  staff.Phone,
	}
	if strings.EqualFold(staff.SupplierRole, string(auth.RoleWarehouseAdmin)) {
		jwtClaims.SupplierRole = auth.RoleWarehouseAdmin
	}

	token, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	refresh, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_refresh_failed"})
		return
	}

	resp := map[string]any{
		"token":         token,
		"refresh_token": refresh,
		"warehouse_id":  warehouseID,
		"role":          string(jwtClaims.Role),
		"name":          staff.Name,
		"is_configured": isConfigured,
	}
	if fbToken, err := auth.MintCustomToken(r.Context(), staff.UserID, map[string]interface{}{
		"role":         string(jwtClaims.Role),
		"warehouse_id": warehouseID,
		"supplier_id":  staff.SupplierID,
	}); err == nil && fbToken != "" {
		resp["firebase_token"] = fbToken
		resp["firebaseToken"] = fbToken
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) lookupWarehouseStaffByPhone(ctx context.Context, phone string) (warehouseStaffRecord, bool, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return warehouseStaffRecord{}, false, nil
	}
	if s.spannerClient == nil {
		return warehouseStaffRecord{}, false, errors.New("spanner_not_configured")
	}

	stmt := spanner.Statement{
		SQL: `SELECT UserId, SupplierId, Name, Phone, PasswordHash, SupplierRole, COALESCE(AssignedWarehouseId, ''), IsActive
		      FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
		      WHERE Phone = @phone
		        AND IsActive = true
		        AND SupplierRole IN ('WAREHOUSE', 'WAREHOUSE_ADMIN', 'WAREHOUSE_STAFF', 'PAYLOADER')
		      LIMIT 1`,
		Params: map[string]any{"phone": phone},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return warehouseStaffRecord{}, false, nil
	}
	if err != nil {
		return warehouseStaffRecord{}, false, fmt.Errorf("query warehouse staff by phone: %w", err)
	}

	var rec warehouseStaffRecord
	if err := row.Columns(
		&rec.UserID,
		&rec.SupplierID,
		&rec.Name,
		&rec.Phone,
		&rec.PasswordHash,
		&rec.SupplierRole,
		&rec.AssignedWarehouseID,
		&rec.IsActive,
	); err != nil {
		return warehouseStaffRecord{}, false, fmt.Errorf("scan warehouse staff: %w", err)
	}
	if !rec.IsActive {
		return warehouseStaffRecord{}, false, nil
	}
	return rec, true, nil
}

func verifyWarehouseStaffSecret(storedHash, secret string) bool {
	storedHash = strings.TrimSpace(storedHash)
	secret = strings.TrimSpace(secret)
	if storedHash == "" || secret == "" {
		return false
	}
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(secret)) == nil
	}
	return storedHash == secret
}

// HandleWarehouseRefresh re-issues access tokens from a refresh JWT.
// POST /v1/auth/warehouse/refresh  body: {"refresh_token":"..."}
func (s *Service) HandleWarehouseRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	refresh := strings.TrimSpace(req.RefreshToken)
	if refresh == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token_required"})
		return
	}

	claims, err := auth.Parse(refresh, s.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_refresh_token"})
		return
	}
	if claims.Role != auth.RoleWarehouse && claims.Role != auth.RoleWarehouseAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid_refresh_scope"})
		return
	}

	claims.IsConfigured = s.warehouseIsConfigured(r.Context(), claims.HomeNodeID)

	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	newRefresh, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_refresh_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":         token,
		"refresh_token": newRefresh,
		"warehouse_id":  claims.HomeNodeID,
		"role":          string(claims.Role),
		"is_configured": claims.IsConfigured,
	})
}
