// Package auth carries JWT claim shapes, context helpers, and role-gating
// middleware. All scope (supplier_id, factory_id, warehouse_id, home_node_id)
// MUST be resolved from the authenticated session — never from request bodies.
//
// Gate 5 Phase 1: prefer TenantFromContext (request-scoped SupplierId). ResolveSupplierID
// remains for claim fallback during migration.
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
	RolePlatformAdmin  Role = "PLATFORM_ADMIN" // Break-glass platform governance
	RoleRetailer       Role = "RETAILER"
	RoleDriver         Role = "DRIVER"
	RolePayload        Role = "PAYLOAD"
	RoleFactory        Role = "FACTORY" // Native factory staff JWT (iOS / portal login)
	RoleFactoryAdmin   Role = "FACTORY_ADMIN"
	RoleFactoryDriver  Role = "FACTORY_DRIVER"
	RoleWarehouseAdmin Role = "WAREHOUSE_ADMIN"
	RoleWarehouse      Role = "WAREHOUSE" // Native warehouse staff JWT (iOS / portal login)
)

// HomeNodeType is the discriminator for Driver / Vehicle scope.
type HomeNodeType string

const (
	HomeNodeWarehouse HomeNodeType = "WAREHOUSE"
	HomeNodeFactory   HomeNodeType = "FACTORY"
)

// TokenUse discriminates intermediate multi-org tokens from full session JWTs.
// Empty / "full" = normal business access. "PendingOrgSelect" is C1.2 only.
const (
	TokenUseFull             = ""
	TokenUsePendingOrgSelect = "PendingOrgSelect"
)

// Claims is the parsed JWT identity. Populated by Middleware and read via
// FromContext. SupplierID is required for every authenticated caller in
// single-tenant mode.
//
// Retail OS (Phase 0+): for Role==RETAILER, Subject is ideally RetailerUserId
// and RetailerOrgID is the tenant RetailerId. Legacy tokens may leave
// RetailerOrgID empty and set Subject=RetailerId (treated as OWNER).
//
// Wave C1.2: TokenUse=PendingOrgSelect means the caller authenticated the
// person but has not selected an org; business routes must reject it.
type Claims struct {
	Subject      string
	Role         Role
	SupplierID   string
	SupplierRole Role         // FACTORY_ADMIN | WAREHOUSE_ADMIN when Role==ADMIN derivative
	HomeNodeType HomeNodeType // WAREHOUSE | FACTORY when applicable
	HomeNodeID   string
	IsRegistered bool // supplier completed business/tax onboarding
	IsConfigured bool // supplier completed billing setup
	PhoneNumber  string
	TraceID      string

	// TokenUse is empty for full tokens; PendingOrgSelect for multi-org intermediate.
	TokenUse string

	// Retailer multi-user identity (Role==RETAILER).
	RetailerOrgID    string   // tenant RetailerId
	RetailerRole     string   // OWNER | ADMIN | MANAGER | BUYER | ...
	RetailerUserID   string   // same as Subject when v2 tokens
	LocationIDs      []string // optional location scope (staff bind)
	ActiveLocationID string   // currently selected store branch (Phase 2)
	CapabilityPacks  []string // enabled pack ids excluding always-on CORE (optional cache)
}

// IsPendingOrgSelect reports whether claims are intermediate multi-org tokens.
func IsPendingOrgSelect(c Claims) bool {
	return strings.EqualFold(strings.TrimSpace(c.TokenUse), TokenUsePendingOrgSelect)
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
// PendingOrgSelect intermediate tokens are rejected with ORG_SELECT_REQUIRED
// so multi-org login cannot touch business routes until select-org.
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
			if IsPendingOrgSelect(c) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden","code":"ORG_SELECT_REQUIRED"}`))
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

// ResolveRetailerOrgID returns the retailer tenant id for a RETAILER session.
// Prefer RetailerOrgID (v2); fall back to Subject for legacy single-owner tokens.
func ResolveRetailerOrgID(c Claims) string {
	if org := strings.TrimSpace(c.RetailerOrgID); org != "" {
		return org
	}
	return strings.TrimSpace(c.Subject)
}

// ResolveRetailerUserID returns the person id for a RETAILER session.
// Prefer RetailerUserID / Subject when OrgID is set; for legacy tokens the
// person and org are the same id (owner bootstrap).
func ResolveRetailerUserID(c Claims) string {
	if uid := strings.TrimSpace(c.RetailerUserID); uid != "" {
		return uid
	}
	return strings.TrimSpace(c.Subject)
}

// EffectiveRetailerRole returns the staff role. Empty/legacy → OWNER.
func EffectiveRetailerRole(c Claims) string {
	role := strings.ToUpper(strings.TrimSpace(c.RetailerRole))
	if role == "" {
		return "OWNER"
	}
	return role
}

// Retailer permission keys (Phase 0 coarse matrix).
const (
	PermCapManage      = "cap.manage"
	PermStaffManage    = "staff.manage"
	PermLocationManage = "location.manage"
	PermOrderPlace     = "order.place"
	PermOrderCancel    = "order.cancel"
	PermDockReceive    = "dock.receive"
	PermStockView      = "stock.view"
	PermStockAdjust    = "stock.adjust"
	PermStockCount     = "stock.count"
	PermPosSell        = "pos.sell"
	PermPosVoid        = "pos.void"
	PermShiftOpen      = "shift.open"
	PermShiftClose     = "shift.close"
	PermReportsView    = "reports.view"
	PermAssistRespond  = "assist.respond"
	PermSectionManage  = "section.manage"
	PermClaimFile      = "claim.file"
)

// retailerRolePerms is the Phase 0 permission template matrix.
var retailerRolePerms = map[string]map[string]struct{}{
	"OWNER": {
		PermCapManage: {}, PermStaffManage: {}, PermLocationManage: {},
		PermOrderPlace: {}, PermOrderCancel: {}, PermDockReceive: {},
		PermStockView: {}, PermStockAdjust: {}, PermStockCount: {},
		PermPosSell: {}, PermPosVoid: {}, PermShiftOpen: {}, PermShiftClose: {},
		PermReportsView: {}, PermAssistRespond: {}, PermSectionManage: {}, PermClaimFile: {},
	},
	"ADMIN": {
		PermCapManage: {}, PermStaffManage: {}, PermLocationManage: {},
		PermOrderPlace: {}, PermOrderCancel: {}, PermDockReceive: {},
		PermStockView: {}, PermStockAdjust: {}, PermStockCount: {},
		PermPosSell: {}, PermPosVoid: {}, PermShiftOpen: {}, PermShiftClose: {},
		PermReportsView: {}, PermAssistRespond: {}, PermSectionManage: {}, PermClaimFile: {},
	},
	"MANAGER": {
		PermOrderPlace: {}, PermOrderCancel: {}, PermDockReceive: {},
		PermStockView: {}, PermStockAdjust: {}, PermStockCount: {},
		PermPosSell: {}, PermPosVoid: {}, PermShiftOpen: {}, PermShiftClose: {},
		PermReportsView: {}, PermAssistRespond: {}, PermSectionManage: {}, PermClaimFile: {},
	},
	"BUYER": {
		PermOrderPlace: {}, PermOrderCancel: {}, PermStockView: {}, PermReportsView: {},
	},
	"RECEIVER": {
		PermDockReceive: {}, PermStockView: {}, PermClaimFile: {},
	},
	"CASHIER": {
		PermPosSell: {}, PermStockView: {}, PermShiftOpen: {},
	},
	"STOCK_CLERK": {
		PermDockReceive: {}, PermStockView: {}, PermStockAdjust: {}, PermStockCount: {}, PermShiftOpen: {}, PermClaimFile: {},
	},
	"SECTION_LEAD": {
		PermStockView: {}, PermStockCount: {}, PermShiftOpen: {}, PermAssistRespond: {}, PermSectionManage: {},
	},
	"VIEWER": {
		PermStockView: {}, PermReportsView: {},
	},
}

// HasRetailerPerm reports whether the claims grant the given permission key.
// Non-RETAILER roles always return false. OWNER always true. Legacy empty role → OWNER.
func HasRetailerPerm(c Claims, perm string) bool {
	if c.Role != RoleRetailer {
		return false
	}
	role := EffectiveRetailerRole(c)
	if role == "OWNER" {
		return true
	}
	set, ok := retailerRolePerms[role]
	if !ok {
		return false
	}
	_, ok = set[perm]
	return ok
}

// ListRetailerPerms returns sorted permission keys for the effective role.
func ListRetailerPerms(c Claims) []string {
	role := EffectiveRetailerRole(c)
	set, ok := retailerRolePerms[role]
	if !ok {
		set = retailerRolePerms["VIEWER"]
	}
	if role == "OWNER" {
		set = retailerRolePerms["OWNER"]
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	// stable-ish order not required for auth; callers may sort
	return out
}

// RequireRetailerPerm 403s when the authenticated retailer lacks perm.
// Must be used inside a RETAILER role gate.
func RequireRetailerPerm(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if c.Role != RoleRetailer {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			if !HasRetailerPerm(c, perm) {
				http.Error(w, `{"error":"forbidden","detail":"missing_permission","permission":"`+perm+`"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
