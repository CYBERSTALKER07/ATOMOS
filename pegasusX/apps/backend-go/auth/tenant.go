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
// Explicit TENANT_CONTEXT_ENFORCED wins; otherwise default true for PEGASUSX_ENV=ssmr|production.
func TenantContextEnforced() bool {
	if v := strings.TrimSpace(os.Getenv("TENANT_CONTEXT_ENFORCED")); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")))
	return env == "ssmr" || env == "production"
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
// ResolveSupplierID from claims, else fallback (seed during Gate 5 migration).
// When TenantContextEnforced and the caller is authenticated (JWT claims present)
// but has no tenant/claim supplier, returns "" instead of seed (Week 11 fail-closed).
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
