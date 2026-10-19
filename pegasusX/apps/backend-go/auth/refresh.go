package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// HandleTokenRefresh re-issues a session JWT from a valid Bearer token.
// POST /v1/auth/refresh — used by driver Android/iOS on 401 recovery.
// Revoked tokens (denylist jti) cannot be refreshed.
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
		claims, err := ParseIgnoreExpiry(token, secret)
		if err != nil {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
			return
		}
		revoked, storeErr := checkTokenRevoked(r.Context(), claims)
		if storeErr != nil {
			writeRefreshJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "revocation_store_unavailable"})
			return
		}
		if revoked {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "token_revoked"})
			return
		}
		if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(string(claims.Role)) == "" {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token_claims"})
			return
		}
		if IsWSTicket(claims) {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "ws_ticket_not_refreshable"})
			return
		}
		if IsPendingOrgSelect(claims) {
			writeRefreshJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "ORG_SELECT_REQUIRED"})
			return
		}
		
		jti := strings.TrimSpace(claims.JTI)
		if jti != "" {
			ttl := time.Until(claims.ExpiresAt)
			if ttl < time.Second {
				ttl = time.Second
			}
			if err := GetRevocationStore().Revoke(r.Context(), jti, ttl); err != nil {
				writeRefreshJSON(w, http.StatusInternalServerError, map[string]string{"error": "revoke_failed"})
				return
			}
		}

		// New jti on every refresh so the previous token can be denylisted independently.
		claims.JTI = ""
		claims.TokenUse = TokenUseFull
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

// HandleLogout revokes the current Bearer/cookie JWT until its natural expiry.
// POST /v1/auth/logout
func HandleLogout(secret string) http.HandlerFunc {
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
			if c, err := r.Cookie(CookieName); err == nil {
				token = c.Value
			}
		}
		if token == "" {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization_required"})
			return
		}
		claims, err := Parse(token, secret)
		if err != nil {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
			return
		}
		jti := strings.TrimSpace(claims.JTI)
		if jti == "" {
			// Legacy token: clear cookie and report ok; cannot denylist without jti.
			ClearSessionCookie(w)
			writeRefreshJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
			return
		}
		ttl := time.Until(claims.ExpiresAt)
		if ttl < time.Second {
			ttl = time.Second
		}
		if err := GetRevocationStore().Revoke(r.Context(), jti, ttl); err != nil {
			writeRefreshJSON(w, http.StatusInternalServerError, map[string]string{"error": "revoke_failed"})
			return
		}
		ClearSessionCookie(w)
		writeRefreshJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
	}
}

// ClearSessionCookie expires the supplier session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeRefreshJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
