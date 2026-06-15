package payload

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

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
		// In a real DB, check if payload worker with claims.PhoneNumber exists
		// For scaffold, we check against the demo phone
		expectPhone := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PHONE"))
		if expectPhone == "" {
			expectPhone = "+998901110022"
		}
		if claims.PhoneNumber != expectPhone {
			// Unregistered phone numbers are BLOCKED pending admin approval.
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "unregistered_phone_number_awaiting_admin_approval"})
			return
		}
	} else {
		phone := strings.TrimSpace(req.Phone)
		pin := strings.TrimSpace(req.PIN)
		if phone == "" || pin == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
			return
		}

		expectPhone := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PHONE"))
		if expectPhone == "" {
			expectPhone = "+998901110022"
		}
		expectPIN := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PIN"))
		if expectPIN == "" {
			expectPIN = "33333333"
		}
		if phone != expectPhone || pin != expectPIN {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
	}

	warehouseID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WAREHOUSE_ID"))
	if warehouseID == "" {
		warehouseID = "warehouse-demo-1"
	}
	warehouseName := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WAREHOUSE_NAME"))
	if warehouseName == "" {
		warehouseName = "PegasusX Demo Warehouse"
	}
	workerID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WORKER_ID"))
	if workerID == "" {
		workerID = "payloader-demo-1"
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	claims := auth.Claims{
		Subject:      workerID,
		Role:         auth.RolePayload,
		SupplierID:   s.supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
		IsConfigured: true,
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
		"token":          token,
		"refresh_token":  refresh,
		"worker_id":      workerID,
		"supplier_id":    s.supplierID,
		"role":           string(auth.RolePayload),
		"name":           "Demo Payloader",
		"warehouse_id":   warehouseID,
		"warehouse_name": warehouseName,
		"warehouse_lat":  41.3111,
		"warehouse_lng":  69.2797,
	}
	if fbToken, err := auth.MintCustomToken(r.Context(), workerID, map[string]interface{}{
		"role":        string(auth.RolePayload),
		"worker_id":   workerID,
		"supplier_id": s.supplierID,
	}); err != nil {
		s.log.Warn("firebase custom token mint failed", "err", err)
	} else if fbToken != "" {
		resp["firebase_token"] = fbToken
	}

	writeJSON(w, http.StatusOK, resp)
}
