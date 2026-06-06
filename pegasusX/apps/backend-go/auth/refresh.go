package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// HandleTokenRefresh re-issues a session JWT from a valid Bearer token.
// POST /v1/auth/refresh — used by driver Android/iOS on 401 recovery.
func HandleTokenRefresh(secret, issuer string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeRefreshJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		if secret == "" {
			writeRefreshJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
			return
		}
		token := BearerToken(r)
		if token == "" {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization_required"})
			return
		}
		claims, err := Parse(token, secret)
		if err != nil {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
			return
		}
		if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(string(claims.Role)) == "" {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token_claims"})
			return
		}
		next, err := Issue(claims, IssueOptions{
			Secret: secret,
			Issuer: issuer,
			TTL:    24 * time.Hour,
		})
		if err != nil {
			writeRefreshJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
			return
		}
		writeRefreshJSON(w, http.StatusOK, map[string]string{"token": next})
	}
}

func writeRefreshJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
