package auth

import (
	"context"
	"net/http"
	"os"
	"strings"
)

// TenantContext is the request-scoped trading-partner key (SupplierId).
// Phase 1 of Gate 5 uses a single key shared by human JWT and partner principal.
type TenantContext struct {
	SupplierID string
	// Source is "jwt", "partner", or "worker" for audit/debug.
	Source string
}

type tenantCtxKey struct{}

// WithTenant attaches TenantContext to ctx. Prefer middleware; services read via TenantFromContext.
func WithTenant(ctx context.Context, t TenantContext) context.Context {
	t.SupplierID = strings.TrimSpace(t.SupplierID)
	t.Source = strings.TrimSpace(t.Source)
	return context.WithValue(ctx, tenantCtxKey{}, t)
}

// TenantFromContext returns the request tenant when present and non-empty.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	if ctx == nil {
		return TenantContext{}, false
	}
	t, ok := ctx.Value(tenantCtxKey{}).(TenantContext)
	if !ok || strings.TrimSpace(t.SupplierID) == "" {
		return TenantContext{}, false
	}
	t.SupplierID = strings.TrimSpace(t.SupplierID)
	return t, true
}

// TenantContextEnforced reports whether missing tenant on authenticated routes must fail closed.
// Explicit TENANT_CONTEXT_ENFORCED wins; otherwise default true for sandbox (incl. ssmr alias) and production.
func TenantContextEnforced() bool {
	if v := strings.TrimSpace(os.Getenv("TENANT_CONTEXT_ENFORCED")); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return IsEnforcedEnv()
}

// SeedFallbackAllowed reports whether PreferTenant may return a bootstrap seed supplier id.
// Default false under sandbox/production (G4.A). Break-glass: ALLOW_SEED_FALLBACK=true.
// Explicit false always wins. Local/dev defaults true when not enforced.
func SeedFallbackAllowed() bool {
	if v := strings.TrimSpace(os.Getenv("ALLOW_SEED_FALLBACK")); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	// Fail closed in enforced envs unless break-glass env is set.
	if TenantContextEnforced() {
		return false
	}
	return true
}

// AttachTenantFromClaims populates TenantContext from JWT claims when not already set
// (partner middleware may have attached first).
func AttachTenantFromClaims(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := TenantFromContext(r.Context()); ok {
			next.ServeHTTP(w, r)
			return
		}
		if sid, ok := ResolveSupplierID(r.Context()); ok {
			next.ServeHTTP(w, r.WithContext(WithTenant(r.Context(), TenantContext{
				SupplierID: sid,
				Source:     "jwt",
			})))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PreferTenantSupplierID returns TenantContext.SupplierID when present, else
// ResolveSupplierID from claims, else optional seed fallback.
// G4.A fail-closed rules:
//  1. Authenticated + TenantContextEnforced without supplier → never seed ("").
//  2. Seed fallback only when SeedFallbackAllowed() (false under sandbox/production by default).
func PreferTenantSupplierID(ctx context.Context, fallback string) string {
	if t, ok := TenantFromContext(ctx); ok {
		return t.SupplierID
	}
	if sid, ok := ResolveSupplierID(ctx); ok && strings.TrimSpace(sid) != "" {
		return strings.TrimSpace(sid)
	}
	if TenantContextEnforced() {
		if _, hasClaims := FromContext(ctx); hasClaims {
			return ""
		}
	}
	if !SeedFallbackAllowed() {
		return ""
	}
	return strings.TrimSpace(fallback)
}

// RequireTenant fails closed when enforcement is on and an authenticated caller
// lacks TenantContext. Unauthenticated public routes pass through.
func RequireTenant(enforced bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enforced {
				next.ServeHTTP(w, r)
				return
			}
			_, hasClaims := FromContext(r.Context())
			_, hasTenant := TenantFromContext(r.Context())
			if !hasClaims && !hasTenant {
				// Public / unauthenticated path.
				next.ServeHTTP(w, r)
				return
			}
			// Break-glass governance: PLATFORM_ADMIN is a cross-tenant role whose
			// JWT carries no SupplierID. Requiring a tenant would 401 every
			// platform-admin route in enforced envs, so exempt it here.
			if claims, ok := FromContext(r.Context()); ok && claims.Role == RolePlatformAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if !hasTenant {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"tenant_required","code":"TENANT_REQUIRED"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
