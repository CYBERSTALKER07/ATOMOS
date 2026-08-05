package partner

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type principalCtxKey struct{}

// WithPrincipal attaches a partner principal to context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, p)
}

// PrincipalFromContext returns the partner principal if present.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalCtxKey{}).(Principal)
	return p, ok
}

// AuthMiddleware authenticates Bearer pxk_* keys (or falls through if not a partner key).
// Partner routes should use RequirePartner which fails closed when principal is missing.
func AuthMiddleware(keys KeyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.BearerToken(r)
			if token == "" || !strings.HasPrefix(token, "pxk_") {
				next.ServeHTTP(w, r)
				return
			}
			prefix, ok := ParseBearerKey(token)
			if !ok || keys == nil {
				writePartnerError(w, http.StatusUnauthorized, "invalid_partner_key")
				return
			}
			k, found, err := keys.GetByPrefix(r.Context(), prefix)
			if err != nil || !found {
				writePartnerError(w, http.StatusUnauthorized, "invalid_partner_key")
				return
			}
			if k.Status != KeyStatusActive {
				writePartnerError(w, http.StatusUnauthorized, "partner_key_revoked")
				return
			}
			if k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt) {
				writePartnerError(w, http.StatusUnauthorized, "partner_key_expired")
				return
			}
			if !VerifyAPIKey(token, k.KeyHash) {
				writePartnerError(w, http.StatusUnauthorized, "invalid_partner_key")
				return
			}
			_ = keys.TouchLastUsed(r.Context(), k.KeyID)
			p := Principal{KeyID: k.KeyID, TenantType: k.TenantType, TenantID: k.TenantID, Scopes: k.Scopes}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

// RequirePartner fails closed without a partner principal and optional scopes.
func RequirePartner(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFromContext(r.Context())
			if !ok || strings.TrimSpace(p.TenantID) == "" {
				writePartnerError(w, http.StatusUnauthorized, "partner_auth_required")
				return
			}
			for _, s := range scopes {
				if !HasScope(p.Scopes, s) {
					writePartnerError(w, http.StatusForbidden, "insufficient_scope")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writePartnerError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","code":"` + code + `"}`))
}
