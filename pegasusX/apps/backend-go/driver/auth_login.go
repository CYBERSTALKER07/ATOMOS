package driver

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

// DriverLoginRecord is the Drivers-table slice used at login (GS-T5).
type DriverLoginRecord struct {
	DriverID     string
	Name         string
	Phone        string
	PinHash      string
	SupplierID   string
	HomeNodeType string
	HomeNodeID   string
	VehicleID    string
}

// DriverLoginLookup finds an active driver by phone.
type DriverLoginLookup func(ctx context.Context, phone string) (DriverLoginRecord, bool, error)

// SetLoginLookup wires GS-T5 Drivers-table login (bootstrap: Spanner).
func (s *Service) SetLoginLookup(fn DriverLoginLookup) {
	if s != nil {
		s.loginLookup = fn
	}
}

// HandleDriverLogin authenticates drivers for native clients.
// POST /v1/auth/driver/login  body: { "phone", "pin" }
func (s *Service) HandleDriverLogin(w http.ResponseWriter, r *http.Request) {
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
		rec, found, err := s.lookupDriverLogin(r.Context(), phone)
		if err != nil {
			s.log.ErrorContext(r.Context(), "driver login lookup failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "unregistered_phone_number_awaiting_admin_approval"})
			return
		}
		s.issueDriverToken(w, r, rec)
		return
	}

	if phone == "" || pin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
		return
	}

	rec, found, err := s.lookupDriverLogin(r.Context(), phone)
	if err != nil {
		s.log.ErrorContext(r.Context(), "driver login lookup failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
		return
	}
	if found {
		if !verifyDriverPin(rec.PinHash, pin) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
		s.issueDriverToken(w, r, rec)
		return
	}
	if rec, ok := s.ssmrDemoDriver(phone, pin); ok {
		s.issueDriverToken(w, r, rec)
		return
	}
	if s.loginLookup == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "driver_lookup_not_configured"})
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
}

func (s *Service) lookupDriverLogin(ctx context.Context, phone string) (DriverLoginRecord, bool, error) {
	if s.loginLookup == nil {
		return DriverLoginRecord{}, false, nil
	}
	return s.loginLookup(ctx, phone)
}

func verifyDriverPin(storedHash, secret string) bool {
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

func (s *Service) ssmrDemoDriver(phone, pin string) (DriverLoginRecord, bool) {
	if !staffinvite.DemoScaffoldAllowed() {
		return DriverLoginRecord{}, false
	}
	expectPhone := strings.TrimSpace(os.Getenv("DRIVER_DEMO_PHONE"))
	if expectPhone == "" || phone != expectPhone {
		return DriverLoginRecord{}, false
	}
	expectPIN := strings.TrimSpace(os.Getenv("DRIVER_DEMO_PIN"))
	if expectPIN == "" {
		expectPIN = strings.TrimSpace(os.Getenv("DRIVER_DEMO_PASSWORD"))
	}
	if expectPIN == "" || pin != expectPIN {
		return DriverLoginRecord{}, false
	}
	driverID := strings.TrimSpace(os.Getenv("DRIVER_DEMO_ID"))
	if driverID == "" {
		driverID = "drv_factory_1"
	}
	homeNodeID := strings.TrimSpace(os.Getenv("DRIVER_DEMO_HOME_NODE_ID"))
	if homeNodeID == "" {
		homeNodeID = strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID"))
	}
	homeNodeType := strings.TrimSpace(os.Getenv("DRIVER_DEMO_HOME_NODE_TYPE"))
	if homeNodeType == "" {
		homeNodeType = string(auth.HomeNodeFactory)
	}
	return DriverLoginRecord{
		DriverID:     driverID,
		Name:         "SSMR Demo Driver",
		Phone:        phone,
		SupplierID:   s.seedSupplierID,
		HomeNodeType: homeNodeType,
		HomeNodeID:   homeNodeID,
	}, true
}

func (s *Service) issueDriverToken(w http.ResponseWriter, r *http.Request, rec DriverLoginRecord) {
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}
	supplierID := strings.TrimSpace(rec.SupplierID)
	if supplierID == "" {
		supplierID = s.seedSupplierID
	}
	homeNodeType := strings.TrimSpace(rec.HomeNodeType)
	if homeNodeType == "" {
		homeNodeType = string(auth.HomeNodeFactory)
	}
	claims := auth.Claims{
		Subject:      rec.DriverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeType(homeNodeType),
		HomeNodeID:   rec.HomeNodeID,
		PhoneNumber:  rec.Phone,
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}

	name := strings.TrimSpace(rec.Name)
	if name == "" {
		name = rec.DriverID
	}
	resp := map[string]any{
		"token":          token,
		"user_id":        rec.DriverID,
		"driver_id":      rec.DriverID,
		"role":           string(auth.RoleDriver),
		"name":           name,
		"supplier_id":    supplierID,
		"vehicle_id":     rec.VehicleID,
		"home_node_type": homeNodeType,
		"home_node_id":   rec.HomeNodeID,
	}
	if strings.EqualFold(homeNodeType, string(auth.HomeNodeFactory)) {
		resp["driver_mode"] = "FACTORY"
		resp["factory_id"] = rec.HomeNodeID
	} else {
		resp["driver_mode"] = "WAREHOUSE"
		resp["warehouse_id"] = rec.HomeNodeID
	}
	if fbToken, err := auth.MintCustomToken(r.Context(), rec.DriverID, map[string]interface{}{
		"role":           string(auth.RoleDriver),
		"driver_id":      rec.DriverID,
		"supplier_id":    supplierID,
		"home_node_type": homeNodeType,
		"home_node_id":   rec.HomeNodeID,
	}); err == nil && fbToken != "" {
		resp["firebase_token"] = fbToken
		resp["firebaseToken"] = fbToken
	}

	writeJSON(w, http.StatusOK, resp)
}
