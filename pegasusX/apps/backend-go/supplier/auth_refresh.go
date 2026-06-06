package supplier

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleSupplierRefresh re-issues supplier portal tokens from a refresh JWT.
// POST /v1/auth/supplier/refresh  body: {"refresh_token":"..."}
func (s *Service) HandleSupplierRefresh(w http.ResponseWriter, r *http.Request) {
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
	if claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid_refresh_scope"})
		return
	}

	token, err := auth.Issue(claims, auth.IssueOptions{
		Secret: s.jwtSecret,
		Issuer: s.jwtIssuer,
		TTL:    s.jwtTTL,
		Now:    s.now,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	auth.SetSessionCookie(w, token, s.jwtTTL, s.cookieSecure)

	writeJSON(w, http.StatusOK, map[string]string{
		"token":         token,
		"refresh_token": refresh,
		"supplier_id":   claims.SupplierID,
	})
}

func (s *Service) issueRefreshToken(supplierID string, isConfigured bool) (string, error) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		supplierID = s.supplierID
	}
	return auth.Issue(auth.Claims{
		Subject:      supplierID,
		Role:         auth.RoleAdmin,
		SupplierID:   supplierID,
		IsConfigured: isConfigured,
	}, auth.IssueOptions{
		Secret: s.jwtSecret,
		Issuer: s.jwtIssuer,
		TTL:    7 * 24 * time.Hour,
		Now:    s.now,
	})
}
