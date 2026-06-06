package warehouse

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleWarehouseLogin authenticates warehouse staff (phone + PIN) for native clients.
// POST /v1/auth/warehouse/login
func (s *Service) HandleWarehouseLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()

	var req struct {
		Phone string `json:"phone"`
		PIN   string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	pin := strings.TrimSpace(req.PIN)
	if phone == "" || pin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
		return
	}

	expectPhone := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_PHONE"))
	if expectPhone == "" {
		expectPhone = "+998901000088"
	}
	expectPIN := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_PIN"))
	if expectPIN == "" {
		expectPIN = "1234"
	}
	if phone != expectPhone || pin != expectPIN {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}

	warehouseID := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID"))
	if warehouseID == "" {
		warehouseID = "wh-demo-1"
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	claims := auth.Claims{
		Subject:      "wh-staff-demo",
		Role:         auth.RoleWarehouse,
		SupplierID:   s.supplierID,
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

	writeJSON(w, http.StatusOK, map[string]string{
		"token":         token,
		"refresh_token": refresh,
		"warehouse_id":  warehouseID,
		"role":          string(auth.RoleWarehouse),
		"name":          "Warehouse Demo",
	})
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

	writeJSON(w, http.StatusOK, map[string]string{
		"token":         token,
		"refresh_token": newRefresh,
		"warehouse_id":  claims.HomeNodeID,
		"role":          string(claims.Role),
	})
}
