package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// DefaultWSSessionTTL is the short-lived WebSocket ticket lifetime for browser clients.
const DefaultWSSessionTTL = 10 * time.Minute

// WSSessionHandler returns GET handler that mints a token_use=ws ticket from the
// current session claims. Mount behind RequireRole (and Firebase when enabled).
func WSSessionHandler(secret, issuer string, ttl time.Duration) http.HandlerFunc {
	if ttl <= 0 {
		ttl = DefaultWSSessionTTL
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeWSSessionJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		claims, ok := FromContext(r.Context())
		if !ok || IsPendingOrgSelect(claims) {
			writeWSSessionJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if strings.TrimSpace(secret) == "" {
			writeWSSessionJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
			return
		}
		token, expiresAt, err := IssueWSTicket(claims, IssueOptions{
			Secret: secret,
			Issuer: issuer,
			TTL:    ttl,
		})
		if err != nil {
			writeWSSessionJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_websocket_token_failed"})
			return
		}
		writeWSSessionJSON(w, http.StatusOK, map[string]any{
			"token":      token,
			"expires_at": expiresAt.UTC().Format(time.RFC3339Nano),
		})
	}
}

func writeWSSessionJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
