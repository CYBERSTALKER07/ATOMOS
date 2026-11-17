package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"path"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ScopeKey binds an idempotency key to principal + route so callers cannot
// replay another actor's cached response by reusing the raw header value.
func ScopeKey(principalID, routePattern, rawKey string) string {
	principalID = strings.TrimSpace(principalID)
	routePattern = strings.TrimSpace(routePattern)
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(principalID + "|" + routePattern + "|" + rawKey))
	return hex.EncodeToString(sum[:])
}

// ScopePrincipal derives a tenant-isolated idempotency namespace from context.
// Retailer sessions bind to RetailerOrgID (plus actor when present) so
// multi-org users and distinct tenants cannot collide or replay across orgs.
func ScopePrincipal(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if claims, ok := auth.FromContext(ctx); ok {
		sub := strings.TrimSpace(claims.Subject)
		if claims.Role == auth.RoleRetailer || strings.TrimSpace(claims.RetailerOrgID) != "" {
			orgID := auth.ResolveRetailerOrgID(claims)
			actorID := auth.ResolveRetailerUserID(claims)
			if orgID != "" && actorID != "" {
				return "retailer:" + orgID + ":user:" + actorID
			}
			if orgID != "" {
				return "retailer:" + orgID
			}
		}
		supplierID := strings.TrimSpace(claims.SupplierID)
		if supplierID == "" {
			if t, hasTenant := auth.TenantFromContext(ctx); hasTenant {
				supplierID = strings.TrimSpace(t.SupplierID)
			}
		}
		if supplierID != "" {
			if sub != "" {
				return "supplier:" + supplierID + ":user:" + sub
			}
			return "supplier:" + supplierID
		}
		if sub != "" {
			return "user:" + sub
		}
	}
	if t, ok := auth.PartnerIdentityFromContext(ctx); ok && (t.KeyID != "" || t.TenantID != "" || t.SupplierID != "") {
		partnerID := strings.TrimSpace(t.TenantID)
		if partnerID == "" {
			partnerID = strings.TrimSpace(t.SupplierID)
		}
		keyID := strings.TrimSpace(t.KeyID)
		if partnerID != "" && keyID != "" {
			return "partner:" + partnerID + ":" + keyID
		}
		if keyID != "" {
			return "partner:" + keyID
		}
		return "tenant:" + partnerID
	}
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		return "tenant:" + strings.TrimSpace(t.SupplierID)
	}
	return ""
}

// CleanRoutePattern normalizes the HTTP method and request URL path so
// trailing or duplicate slashes map to one canonical route pattern.
func CleanRoutePattern(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	cleanPath := path.Clean(strings.TrimSpace(r.URL.Path))
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	return r.Method + " " + cleanPath
}

// ScopeRequest binds rawKey to the authenticated tenant/principal and canonical route.
func ScopeRequest(r *http.Request, rawKey string) string {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return ""
	}
	if r == nil {
		return ScopeKey("", "", rawKey)
	}
	return ScopeKey(ScopePrincipal(r.Context()), CleanRoutePattern(r), rawKey)
}
