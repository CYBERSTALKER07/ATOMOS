package factory

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	secret := strings.TrimSpace(req.PIN)
	if secret == "" {
		secret = strings.TrimSpace(req.Password)
	}
	if phone == "" || secret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
		return
	}

	expectPhone := strings.TrimSpace(os.Getenv("FACTORY_DEMO_PHONE"))
	if expectPhone == "" {
		expectPhone = "+998901000099"
	}
	expectSecret := strings.TrimSpace(os.Getenv("FACTORY_DEMO_PIN"))
	if expectSecret == "" {
		expectSecret = strings.TrimSpace(os.Getenv("FACTORY_DEMO_PASSWORD"))
	}
	if expectSecret == "" {
		expectSecret = "1234"
	}
	if phone != expectPhone || secret != expectSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}

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
		"factory_id":    claims.HomeNodeID,
	})
}
