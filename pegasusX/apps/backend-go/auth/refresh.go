package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// IsTokenRevoked reports whether claims' JTI or Subject is revoked (fail-closed on store errors).
func IsTokenRevoked(ctx context.Context, claims Claims) bool {
	return tokenRevoked(ctx, claims)
}

// ValidateAndRotateRefreshClaims validates revocation, role/ticket scope, and expiry window
// for refresh requests, then single-use revokes the consumed refresh JTI.
func ValidateAndRotateRefreshClaims(ctx context.Context, claims Claims, maxWindow time.Duration) error {
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(string(claims.Role)) == "" {
		return errors.New("invalid_refresh_token")
	}
	if IsWSTicket(claims) {
		return errors.New("ws_ticket_not_refreshable")
	}
	if IsPendingOrgSelect(claims) {
		return errors.New("pending_org_select")
	}
	revoked, storeErr := checkTokenRevoked(ctx, claims)
	if storeErr != nil {
		return errors.New("revocation_store_unavailable")
	}
	if revoked {
		return errors.New("token_revoked")
	}
	userRevoked, uerr := checkUserRevoked(ctx, claims)
	if uerr != nil {
		return errors.New("revocation_store_unavailable")
	}
	if userRevoked {
		return errors.New("token_revoked")
	}
	if maxWindow > 0 && (claims.ExpiresAt.IsZero() || time.Since(claims.ExpiresAt) > maxWindow) {
		return errors.New("token_expired_beyond_refresh_window")
	}
	if jti := strings.TrimSpace(claims.JTI); jti != "" {
		revokeTTL := 24 * time.Hour
		if !claims.ExpiresAt.IsZero() {
			if rem := time.Until(claims.ExpiresAt) + time.Hour; rem > revokeTTL {
				revokeTTL = rem
			}
		}
		if err := GetRevocationStore().Revoke(ctx, jti, revokeTTL); err != nil {
			return errors.New("revoke_failed")
		}
	}
	return nil
}

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

		// Enforce maximum refresh window: tokens expired > 7 days cannot be refreshed.
		const maxRefreshWindow = 7 * 24 * time.Hour
		if claims.ExpiresAt.IsZero() || time.Since(claims.ExpiresAt) > maxRefreshWindow {
			writeRefreshJSON(w, http.StatusUnauthorized, map[string]string{"error": "token_expired_beyond_refresh_window"})
			return
		}
		
		jti := strings.TrimSpace(claims.JTI)
		if jti != "" {
			revokeTTL := 24 * time.Hour
			if rem := time.Until(claims.ExpiresAt.Add(maxRefreshWindow)); rem > revokeTTL {
				revokeTTL = rem
			}
			if err := GetRevocationStore().Revoke(r.Context(), jti, revokeTTL); err != nil {
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
