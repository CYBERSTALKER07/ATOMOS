package migration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	// PermissionApplyFinancials is the canonical permission stamp required for
	// financial cutover actions (nasiya_ar, set_on_hand, rollback, cutover, release_open_orders).
	PermissionApplyFinancials = "migration.apply_financials"
)

var (
	// ErrAuthBypassForbidden indicates ALLOW_AUTH_BYPASS is enabled while invoking a migration route.
	ErrAuthBypassForbidden = errors.New("auth_bypass_forbidden")
	// ErrMigrationUnauthenticated indicates the caller has no verified session claims.
	ErrMigrationUnauthenticated = errors.New("unauthenticated")
	// ErrMigrationForbidden indicates the caller role or tenant scope cannot access the migration resource.
	ErrMigrationForbidden = errors.New("migration_forbidden")
	// ErrFinancialPermissionRequired indicates the caller lacks org-level admin authority for financial actions.
	ErrFinancialPermissionRequired = errors.New("financial_permission_required")
	// ErrCrossCellForbidden indicates an attempted cross-cell access across HomeCells.
	ErrCrossCellForbidden = errors.New("cross_cell_forbidden")
)

// AuthzRequest describes the caller's intended migration operation for fail-closed authorization.
type AuthzRequest struct {
	AllowAuthBypass  bool
	Pack             string
	Action           string
	TargetSupplierID string
	TargetHomeCell   string
}

// IsAuthBypassActive reports whether auth bypass is enabled either via router wiring or ALLOW_AUTH_BYPASS env.
func IsAuthBypassActive(allowAuthBypassCfg bool) bool {
	if allowAuthBypassCfg {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_AUTH_BYPASS")), "true")
}

// GuardRequestAuthorization enforces fail-closed security gates for all migration endpoints:
// 1. Reject whenever ALLOW_AUTH_BYPASS is active (403 auth_bypass_forbidden).
// 2. Require full authenticated claims (no PendingOrgSelect or WS upgrade tickets).
// 3. Enforce supplier org-admin or platform-admin role, plus strict PermissionApplyFinancials for financial actions.
// 4. Enforce single-tenant SupplierID and HomeCell isolation for non-platform admins.
func GuardRequestAuthorization(ctx context.Context, req AuthzRequest) (auth.Claims, error) {
	if IsAuthBypassActive(req.AllowAuthBypass) {
		return auth.Claims{}, ErrAuthBypassForbidden
	}

	claims, ok := auth.FromContext(ctx)
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		return auth.Claims{}, ErrMigrationUnauthenticated
	}
	if auth.IsPendingOrgSelect(claims) || auth.IsWSTicket(claims) {
		return auth.Claims{}, ErrMigrationForbidden
	}

	switch claims.Role {
	case auth.RoleAdmin, auth.RolePlatformAdmin:
	default:
		return auth.Claims{}, ErrMigrationForbidden
	}

	if requiresFinancialAuthority(req.Pack, req.Action) && !HasFinancialMigrationAuthority(claims) {
		return auth.Claims{}, ErrFinancialPermissionRequired
	}

	if claims.Role != auth.RolePlatformAdmin {
		callerSupplier := strings.TrimSpace(claims.SupplierID)
		if callerSupplier == "" {
			return auth.Claims{}, ErrMigrationForbidden
		}
		if target := strings.TrimSpace(req.TargetSupplierID); target != "" && target != callerSupplier {
			return auth.Claims{}, ErrMigrationForbidden
		}
		callerCell := strings.TrimSpace(auth.EffectiveHomeCell(claims))
		if targetCell := strings.TrimSpace(req.TargetHomeCell); targetCell != "" && callerCell != "" && !strings.EqualFold(targetCell, callerCell) {
			return auth.Claims{}, ErrCrossCellForbidden
		}
	}

	return claims, nil
}

// HasFinancialMigrationAuthority reports whether claims permit approving/applying financial
// packs (nasiya_ar), destructive set_on_hand stock overwrites, rollbacks, cutover, or open-order releases.
// Derivative scoped staff roles (e.g. WAREHOUSE_ADMIN, FACTORY_ADMIN) are intentionally excluded.
func HasFinancialMigrationAuthority(claims auth.Claims) bool {
	if claims.Role == auth.RolePlatformAdmin {
		return true
	}
	if claims.Role != auth.RoleAdmin {
		return false
	}
	sub := claims.SupplierRole
	return sub == "" || sub == auth.RoleAdmin
}

func requiresFinancialAuthority(pack, action string) bool {
	normalizedPack := strings.ToLower(strings.TrimSpace(pack))
	normalizedAction := strings.ToLower(strings.TrimSpace(action))

	if normalizedPack == "nasiya_ar" && normalizedAction != "read" {
		return true
	}
	switch normalizedAction {
	case "set_on_hand", "rollback", "cutover", "release_open_orders":
		return true
	default:
		return false
	}
}

// AuthzHTTPStatus maps migration authorization errors to deterministic HTTP status codes and API error codes.
func AuthzHTTPStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrAuthBypassForbidden):
		return http.StatusForbidden, "auth_bypass_forbidden"
	case errors.Is(err, ErrMigrationUnauthenticated):
		return http.StatusUnauthorized, "unauthenticated"
	case errors.Is(err, ErrFinancialPermissionRequired):
		return http.StatusForbidden, "financial_permission_required"
	case errors.Is(err, ErrCrossCellForbidden):
		return http.StatusForbidden, "cross_cell_forbidden"
	case errors.Is(err, ErrMigrationForbidden):
		return http.StatusForbidden, "migration_forbidden"
	default:
		return http.StatusForbidden, "migration_forbidden"
	}
}

// WriteAuthzError writes a structured JSON authorization error response.
func WriteAuthzError(w http.ResponseWriter, err error) {
	status, code := AuthzHTTPStatus(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": err.Error(),
	})
}
