// Package auth carries JWT claim shapes, context helpers, and role-gating
// middleware. All scope (supplier_id, factory_id, warehouse_id, home_node_id)
// MUST be resolved from the authenticated session — never from request bodies.
//
// In pegasusX (single-supplier tenant) ResolveSupplierID returns the seeded
// SupplierId for every authenticated caller regardless of role.
package auth

import (
	"context"
	"net/http"
	"strings"
)

// Role enumerates every JWT role string. Mirrors packages/types Role union.
type Role string

const (
	RoleAdmin          Role = "ADMIN" // Supplier portal session (single-tenant)
	RoleRetailer       Role = "RETAILER"
	RoleDriver         Role = "DRIVER"
	RolePayload        Role = "PAYLOAD"
	RoleFactory        Role = "FACTORY" // Native factory staff JWT (iOS / portal login)
	RoleFactoryAdmin   Role = "FACTORY_ADMIN"
	RoleWarehouseAdmin Role = "WAREHOUSE_ADMIN"
	RoleWarehouse      Role = "WAREHOUSE" // Native warehouse staff JWT (iOS / portal login)
)

// HomeNodeType is the discriminator for Driver / Vehicle scope.
type HomeNodeType string

const (
	HomeNodeWarehouse HomeNodeType = "WAREHOUSE"
	HomeNodeFactory   HomeNodeType = "FACTORY"
)

// Claims is the parsed JWT identity. Populated by Middleware and read via
// FromContext. SupplierID is required for every authenticated caller in
// single-tenant mode.
type Claims struct {
	Subject      string
	Role         Role
	SupplierID   string
	SupplierRole Role         // FACTORY_ADMIN | WAREHOUSE_ADMIN when Role==ADMIN derivative
	HomeNodeType HomeNodeType // WAREHOUSE | FACTORY when applicable
	HomeNodeID   string
	IsConfigured bool // supplier completed billing setup
	TraceID      string
}

type ctxKey int

const (
	ctxKeyClaims ctxKey = iota
	ctxKeyTraceID
)

// WithClaims attaches claims to the request context. Use only from middleware.
func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, ctxKeyClaims, c)
}

// FromContext returns the claims attached by Middleware, if any.
func FromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ctxKeyClaims).(Claims)
	return c, ok
}

// ResolveSupplierID returns the supplier scope for the caller. In single-tenant
// pegasusX the supplier scope is constant across roles, so it falls through to
// the seeded SupplierID. The boolean return is false when no claims are present.
func ResolveSupplierID(ctx context.Context) (string, bool) {
	c, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	return c.SupplierID, c.SupplierID != ""
}

// RequireRole returns a middleware that 401s unauthenticated callers and 403s
// callers whose Role is not in the allowed set.
func RequireRole(allowed ...Role) func(http.Handler) http.Handler {
	allow := make(map[Role]struct{}, len(allowed))
	for _, r := range allowed {
		allow[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if _, allowed := allow[c.Role]; !allowed {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BearerToken extracts the raw token from an Authorization: Bearer <token>
// header. Returns empty string when missing or malformed.
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[len("Bearer "):])
}
