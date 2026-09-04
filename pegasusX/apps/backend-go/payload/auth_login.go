package payload

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/staffinvite"
	"golang.org/x/crypto/bcrypt"
)

// PayloadStaffRecord is the SupplierUsers PAYLOADER slice used at login (GS-T5).
type PayloadStaffRecord struct {
	UserID       string
	Name         string
	Phone        string
	PasswordHash string
	SupplierID   string
	WarehouseID  string
}

// PayloadStaffLookup finds an active payloader by phone.
type PayloadStaffLookup func(ctx context.Context, phone string) (PayloadStaffRecord, bool, error)

// SetStaffLookup wires GS-T5 staff-row login (bootstrap: SupplierUsers).
func (s *Service) SetStaffLookup(fn PayloadStaffLookup) {
	if s != nil {
		s.staffLookup = fn
	}
}

// HandlePayloaderLogin authenticates payload terminal staff (phone + PIN).
// POST /v1/auth/payloader/login
func (s *Service) HandlePayloaderLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()

	var req struct {
		Phone   string `json:"phone"`
		PIN     string `json:"pin"`
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	phone := strings.TrimSpace(req.Phone)
	pin := strings.TrimSpace(req.PIN)
	idToken := strings.TrimSpace(req.IDToken)
	if idToken != "" && s.firebaseVerifier == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": auth.FirebaseLoginUnavailable})
		return
	}

	if idToken != "" && s.firebaseVerifier != nil {
		claims, err := s.firebaseVerifier.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			s.log.Warn("firebase token verification failed", "err", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_id_token"})
			return
		}
		if claims.PhoneNumber == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "phone_number_missing_in_token"})
			return
		}
		phone = claims.PhoneNumber
		rec, found, err := s.lookupPayloadStaff(r.Context(), phone)
		if err != nil {
			s.log.ErrorContext(r.Context(), "payload login lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "unregistered_phone_number_awaiting_admin_approval"})
			return
		}
		s.issuePayloaderToken(w, r, rec)
		return
	}

	if phone == "" || pin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
		return
	}

	rec, found, err := s.lookupPayloadStaff(r.Context(), phone)
	if err != nil {
		s.log.ErrorContext(r.Context(), "payload login lookup failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
		return
	}
	if found {
		if !verifyPayloadSecret(rec.PasswordHash, pin) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
		s.issuePayloaderToken(w, r, rec)
		return
	}
	if rec, ok := s.ssmrDemoPayloader(phone, pin); ok {
		s.issuePayloaderToken(w, r, rec)
		return
	}
	if s.staffLookup == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payload_lookup_not_configured"})
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
}

func (s *Service) lookupPayloadStaff(ctx context.Context, phone string) (PayloadStaffRecord, bool, error) {
	if s.staffLookup == nil {
		return PayloadStaffRecord{}, false, nil
	}
	return s.staffLookup(ctx, phone)
}

func verifyPayloadSecret(storedHash, secret string) bool {
	storedHash = strings.TrimSpace(storedHash)
	secret = strings.TrimSpace(secret)
	if storedHash == "" || secret == "" {
		return false
	}
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(secret)) == nil
	}
	return false
}

func (s *Service) ssmrDemoPayloader(phone, pin string) (PayloadStaffRecord, bool) {
	if !staffinvite.DemoScaffoldAllowed() {
		return PayloadStaffRecord{}, false
	}
	expectPhone := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PHONE"))
	if expectPhone == "" || phone != expectPhone {
		return PayloadStaffRecord{}, false
	}
	expectPIN := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PIN"))
	if expectPIN == "" || pin != expectPIN {
		return PayloadStaffRecord{}, false
	}
	warehouseID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WAREHOUSE_ID"))
	if warehouseID == "" {
		warehouseID = strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID"))
	}
	if warehouseID == "" {
		warehouseID = strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID"))
	}
	workerID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WORKER_ID"))
	if workerID == "" {
		workerID = "payloader-demo-1"
	}
	return PayloadStaffRecord{
		UserID:      workerID,
		Name:        "SSMR Demo Payloader",
		Phone:       phone,
		SupplierID:  s.seedSupplierID,
		WarehouseID: warehouseID,
	}, true
}

func (s *Service) issuePayloaderToken(w http.ResponseWriter, r *http.Request, rec PayloadStaffRecord) {
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}
	supplierID := strings.TrimSpace(rec.SupplierID)
	if supplierID == "" {
		supplierID = s.seedSupplierID
	}
	workerID := strings.TrimSpace(rec.UserID)
	warehouseID := strings.TrimSpace(rec.WarehouseID)
	name := strings.TrimSpace(rec.Name)
	if name == "" {
		name = workerID
	}
	claims := auth.Claims{
		Subject:      workerID,
		Role:         auth.RolePayload,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
		IsConfigured: true,
		PhoneNumber:  rec.Phone,
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

	resp := map[string]any{
		"token":         token,
		"refresh_token": refresh,
		"worker_id":     workerID,
		"supplier_id":   supplierID,
		"role":          string(auth.RolePayload),
		"name":          name,
		"warehouse_id":  warehouseID,
	}
	if fbToken, err := auth.MintCustomToken(r.Context(), workerID, map[string]interface{}{
		"role":        string(auth.RolePayload),
		"worker_id":   workerID,
		"supplier_id": supplierID,
	}); err != nil {
		s.log.Warn("firebase custom token mint failed", "err", err)
	} else if fbToken != "" {
		resp["firebase_token"] = fbToken
	}

	writeJSON(w, http.StatusOK, resp)
}
