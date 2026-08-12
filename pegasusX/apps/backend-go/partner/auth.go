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

// AuthOptions configures partner AuthMiddleware (API keys + OAuth access tokens).
type AuthOptions struct {
	Keys      KeyRepository
	JWTSecret string
}

// AuthMiddleware authenticates Bearer pxk_* keys or partner_access JWTs.
// Falls through when the Authorization header is absent / not a partner credential
// so human SessionAuth can still attach claims on shared routers.
func AuthMiddleware(keys KeyRepository) func(http.Handler) http.Handler {
	return AuthMiddlewareOpts(AuthOptions{Keys: keys})
}

// AuthMiddlewareOpts authenticates with optional partner JWT secret for OAuth tokens.
func AuthMiddlewareOpts(opts AuthOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.BearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(token, "pxk_") || strings.HasPrefix(token, "pxs_") {
				p, ok := authenticateAPIKey(r.Context(), opts.Keys, token, w)
				if !ok {
					return
				}
				next.ServeHTTP(w, r.WithContext(withPartnerTenant(r.Context(), p)))
				return
			}
			if looksLikeJWT(token) && strings.TrimSpace(opts.JWTSecret) != "" {
				p, ok := authenticateAccessToken(r.Context(), opts.Keys, opts.JWTSecret, token, w)
				if !ok {
					return
				}
				if p.KeyID != "" {
					next.ServeHTTP(w, r.WithContext(withPartnerTenant(r.Context(), p)))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func withPartnerTenant(ctx context.Context, p Principal) context.Context {
	ctx = WithPrincipal(ctx, p)
	if tid := strings.TrimSpace(p.TenantID); tid != "" {
		ctx = auth.WithTenant(ctx, auth.TenantContext{SupplierID: tid, Source: "partner"})
	}
	return ctx
}

func authenticateAPIKey(ctx context.Context, keys KeyRepository, token string, w http.ResponseWriter) (Principal, bool) {
	prefix, ok := ParseBearerKey(token)
	if !ok || keys == nil {
		writePartnerError(w, http.StatusUnauthorized, "invalid_partner_key")
		return Principal{}, false
	}
	k, found, err := keys.GetByPrefix(ctx, prefix)
	if err != nil || !found {
		writePartnerError(w, http.StatusUnauthorized, "invalid_partner_key")
		return Principal{}, false
	}
	if k.Status != KeyStatusActive {
		writePartnerError(w, http.StatusUnauthorized, "partner_key_revoked")
		return Principal{}, false
	}
	if k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt) {
		writePartnerError(w, http.StatusUnauthorized, "partner_key_expired")
		return Principal{}, false
	}
	if !VerifyAPIKey(token, k.KeyHash) {
		writePartnerError(w, http.StatusUnauthorized, "invalid_partner_key")
		return Principal{}, false
	}
	_ = keys.TouchLastUsed(ctx, k.KeyID)
	return Principal{
		KeyID: k.KeyID, TenantType: k.TenantType, TenantID: k.TenantID, Scopes: k.Scopes,
		Sandbox: IsSandboxKey(token) || IsSandboxRateClass(k.RateLimitClass),
	}, true
}

func authenticateAccessToken(ctx context.Context, keys KeyRepository, secret, token string, w http.ResponseWriter) (Principal, bool) {
	claims, err := ParsePartnerAccessToken(token, secret)
	if err != nil {
		// Not a partner token (wrong secret / use) → fall through for human JWT.
		if err.Error() == "invalid_token_use" || err.Error() == "invalid_token" {
			return Principal{}, true // ok=true with empty KeyID → fall through
		}
		writePartnerError(w, http.StatusUnauthorized, "invalid_partner_token")
		return Principal{}, false
	}
	if strings.TrimSpace(claims.TenantID) == "" || strings.TrimSpace(claims.KeyID) == "" {
		writePartnerError(w, http.StatusUnauthorized, "invalid_partner_token")
		return Principal{}, false
	}
	// Live revoke: require key still ACTIVE when repository is available.
	if keys != nil {
		k, found, gErr := keys.GetByID(ctx, claims.KeyID)
		if gErr != nil || !found || k.Status != KeyStatusActive {
			writePartnerError(w, http.StatusUnauthorized, "partner_key_revoked")
			return Principal{}, false
		}
		if k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt) {
			writePartnerError(w, http.StatusUnauthorized, "partner_key_expired")
			return Principal{}, false
		}
		_ = keys.TouchLastUsed(ctx, k.KeyID)
		scopes := claims.Scopes
		if len(scopes) == 0 {
			scopes = k.Scopes
		}
		return Principal{KeyID: k.KeyID, TenantType: k.TenantType, TenantID: k.TenantID, Scopes: scopes}, true
	}
	return Principal{
		KeyID: claims.KeyID, TenantType: claims.TenantType, TenantID: claims.TenantID, Scopes: claims.Scopes,
	}, true
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
