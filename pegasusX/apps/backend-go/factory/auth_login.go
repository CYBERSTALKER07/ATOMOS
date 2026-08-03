package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

type factoryStaffRecord struct {
	UserID            string
	SupplierID        string
	Name              string
	Phone             string
	PasswordHash      string
	SupplierRole      string
	AssignedFactoryID string
	IsActive          bool
}

// HandleFactoryLogin authenticates factory staff for native clients.
// POST /v1/auth/factory/login  body: { "phone", "pin" } or { "phone", "password" }
func (s *Service) HandleFactoryLogin(w http.ResponseWriter, r *http.Request) {
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

	var staff factoryStaffRecord
	var lookupPhone string
	var verified bool

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
		verified = true
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
		if s.spannerClient == nil {
			if !s.verifyFactoryDemoCredentials(lookupPhone, secret) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
				return
			}
			s.issueFactoryDemoToken(w, lookupPhone)
			return
		}
		rec, found, err := s.lookupFactoryStaffByPhone(r.Context(), lookupPhone)
		if err != nil {
			s.log.ErrorContext(r.Context(), "factory staff lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
		if !verifyFactoryStaffSecret(rec.PasswordHash, secret) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
		staff = rec
		verified = true
	}

	if s.spannerClient == nil {
		if !s.verifyFactoryDemoPhone(lookupPhone) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
		s.issueFactoryDemoToken(w, lookupPhone)
		return
	}

	if !verified || staff.UserID == "" {
		rec, found, err := s.lookupFactoryStaffByPhone(r.Context(), lookupPhone)
		if err != nil {
			s.log.ErrorContext(r.Context(), "factory staff lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
		staff = rec
	}

	factoryID := strings.TrimSpace(staff.AssignedFactoryID)
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	supplierID := strings.TrimSpace(staff.SupplierID)
	if supplierID == "" {
		supplierID = s.supplierID
	}

	isConfigured := false
	if factoryID != "" {
		isConfigured = s.factoryIsConfigured(r.Context(), factoryID)
	}

	jwtClaims := auth.Claims{
		Subject:      staff.UserID,
		Role:         auth.RoleFactory,
		SupplierID:   supplierID,
		SupplierRole: auth.Role(strings.TrimSpace(staff.SupplierRole)),
		HomeNodeType: auth.HomeNodeFactory,
		HomeNodeID:   factoryID,
		IsConfigured: isConfigured,
		PhoneNumber:  staff.Phone,
	}
	if strings.EqualFold(staff.SupplierRole, string(auth.RoleFactoryAdmin)) {
		jwtClaims.SupplierRole = auth.RoleFactoryAdmin
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

	factoryName, _ := s.lookupFactoryName(r.Context(), factoryID)

	resp := map[string]any{
		"token":         token,
		"refresh_token": refresh,
		"factory_id":    factoryID,
		"factory_name":  factoryName,
		"role":          string(jwtClaims.Role),
		"factory_role":  string(jwtClaims.SupplierRole),
		"name":          staff.Name,
		"is_configured": isConfigured,
	}
	if fbToken, err := auth.MintCustomToken(r.Context(), staff.UserID, map[string]interface{}{
		"role":        string(jwtClaims.Role),
		"factory_id":  factoryID,
		"supplier_id": staff.SupplierID,
	}); err == nil && fbToken != "" {
		resp["firebase_token"] = fbToken
		resp["firebaseToken"] = fbToken
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) lookupFactoryStaffByPhone(ctx context.Context, phone string) (factoryStaffRecord, bool, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return factoryStaffRecord{}, false, nil
	}
	if s.spannerClient == nil {
		return factoryStaffRecord{}, false, errors.New("spanner_not_configured")
	}

	stmt := spanner.Statement{
		SQL: `SELECT UserId, SupplierId, Name, Phone, PasswordHash, SupplierRole, COALESCE(AssignedFactoryId, ''), IsActive
		      FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
		      WHERE Phone = @phone
		        AND IsActive = true
		        AND SupplierRole IN ('FACTORY', 'FACTORY_ADMIN', 'FACTORY_STAFF')
		      LIMIT 1`,
		Params: map[string]any{"phone": phone},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return factoryStaffRecord{}, false, nil
	}
	if err != nil {
		return factoryStaffRecord{}, false, fmt.Errorf("query factory staff by phone: %w", err)
	}

	var rec factoryStaffRecord
	if err := row.Columns(
		&rec.UserID,
		&rec.SupplierID,
		&rec.Name,
		&rec.Phone,
		&rec.PasswordHash,
		&rec.SupplierRole,
		&rec.AssignedFactoryID,
		&rec.IsActive,
	); err != nil {
		return factoryStaffRecord{}, false, fmt.Errorf("scan factory staff: %w", err)
	}
	if !rec.IsActive {
		return factoryStaffRecord{}, false, nil
	}
	return rec, true, nil
}

func (s *Service) lookupFactoryName(ctx context.Context, factoryID string) (string, error) {
	factoryID = strings.TrimSpace(factoryID)
	if factoryID == "" || s.spannerClient == nil {
		return "", nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Factories", spanner.Key{factoryID}, []string{"Name"})
	if err != nil {
		return "", err
	}
	var name string
	if err := row.Columns(&name); err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

func verifyFactoryStaffSecret(storedHash, secret string) bool {
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

func (s *Service) verifyFactoryDemoCredentials(phone, secret string) bool {
	if !s.verifyFactoryDemoPhone(phone) {
		return false
	}
	expectSecret := strings.TrimSpace(os.Getenv("FACTORY_DEMO_PIN"))
	if expectSecret == "" {
		expectSecret = strings.TrimSpace(os.Getenv("FACTORY_DEMO_PASSWORD"))
	}
	if expectSecret == "" {
		expectSecret = "1234"
	}
	return secret == expectSecret
}

func (s *Service) verifyFactoryDemoPhone(phone string) bool {
	expectPhone := strings.TrimSpace(os.Getenv("FACTORY_DEMO_PHONE"))
	if expectPhone == "" {
		expectPhone = "+998901000099"
	}
	return phone == expectPhone
}

func (s *Service) issueFactoryDemoToken(w http.ResponseWriter, phone string) {
	factoryID := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID"))
	if factoryID == "" {
		factoryID = "factory-demo-1"
	}
	factoryName := strings.TrimSpace(os.Getenv("FACTORY_DEMO_NAME"))
	if factoryName == "" {
		factoryName = "PegasusX Demo Factory"
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	claims := auth.Claims{
		Subject:      "factory-staff-demo",
		Role:         auth.RoleFactory,
		SupplierID:   s.supplierID,
		SupplierRole: auth.RoleFactoryAdmin,
		HomeNodeType: auth.HomeNodeFactory,
		HomeNodeID:   factoryID,
		IsConfigured: true,
		PhoneNumber:  phone,
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	refresh, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_refresh_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"token":         token,
		"refresh_token": refresh,
		"factory_id":    factoryID,
		"factory_name":  factoryName,
		"role":          string(auth.RoleFactory),
		"factory_role":  string(auth.RoleFactoryAdmin),
		"name":          "Factory Demo",
	})
}

// HandleFactoryRefresh re-issues tokens from a refresh JWT.
// POST /v1/auth/factory/refresh
func (s *Service) HandleFactoryRefresh(w http.ResponseWriter, r *http.Request) {
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
	if claims.Role != auth.RoleFactory && claims.Role != auth.RoleFactoryAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid_refresh_scope"})
		return
	}

	claims.IsConfigured = s.factoryIsConfigured(r.Context(), claims.HomeNodeID)

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
		"factory_id":    claims.HomeNodeID,
		"is_configured": claims.IsConfigured,
	})
}
