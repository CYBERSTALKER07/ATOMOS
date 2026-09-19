package auth

import (
	"context"
	"strings"
)

// CallerSupplierID returns the request tenant supplier from TenantContext or JWT claims.
func CallerSupplierID(ctx context.Context) (string, bool) {
	if t, ok := TenantFromContext(ctx); ok {
		return t.SupplierID, true
	}
	return ResolveSupplierID(ctx)
}

// IsPlatformAdmin reports whether the caller has RolePlatformAdmin.
func IsPlatformAdmin(ctx context.Context) bool {
	c, ok := FromContext(ctx)
	return ok && c.Role == RolePlatformAdmin
}

// EntitySupplierAllowed is fail-closed tenant ownership for path-param detail GETs.
// Platform admins bypass; everyone else must match entitySupplierID to caller scope.
func EntitySupplierAllowed(ctx context.Context, entitySupplierID string) bool {
	if IsPlatformAdmin(ctx) {
		return true
	}
	sid, ok := CallerSupplierID(ctx)
	if !ok || strings.TrimSpace(sid) == "" {
		return false
	}
	return strings.TrimSpace(sid) == strings.TrimSpace(entitySupplierID)
}

// HomeNodeMatches reports whether JWT home-node id equals the path resource id
// for warehouse/factory staff detail reads.
func HomeNodeMatches(ctx context.Context, resourceID string, wantType HomeNodeType) bool {
	c, ok := FromContext(ctx)
	if !ok {
		return false
	}
	if c.HomeNodeType != wantType {
		return false
	}
	return strings.TrimSpace(c.HomeNodeID) != "" &&
		strings.TrimSpace(c.HomeNodeID) == strings.TrimSpace(resourceID)
}
