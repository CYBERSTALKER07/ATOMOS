package migration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestGuardRequestAuthorization_FailClosedWhenAuthBypassActive(t *testing.T) {
	claims := auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}
	ctx := auth.WithClaims(context.Background(), claims)

	_, err := GuardRequestAuthorization(ctx, AuthzRequest{
		AllowAuthBypass:  true,
		TargetSupplierID: "sup-1",
		Action:           "apply",
	})
	if !errors.Is(err, ErrAuthBypassForbidden) {
		t.Fatalf("expected ErrAuthBypassForbidden when config AllowAuthBypass=true, got %v", err)
	}

	t.Setenv("ALLOW_AUTH_BYPASS", "true")
	_, err = GuardRequestAuthorization(ctx, AuthzRequest{
		AllowAuthBypass:  false,
		TargetSupplierID: "sup-1",
		Action:           "read",
	})
	if !errors.Is(err, ErrAuthBypassForbidden) {
		t.Fatalf("expected ErrAuthBypassForbidden when ALLOW_AUTH_BYPASS env is true, got %v", err)
	}
}

func TestGuardRequestAuthorization_ClaimsTokensRolesAndTenantIsolation(t *testing.T) {
	t.Setenv("ALLOW_AUTH_BYPASS", "false")

	// Missing claims -> 401
	if _, err := GuardRequestAuthorization(context.Background(), AuthzRequest{Action: "read"}); !errors.Is(err, ErrMigrationUnauthenticated) {
		t.Fatalf("expected ErrMigrationUnauthenticated, got %v", err)
	}

	// WS ticket or PendingOrgSelect -> forbidden
	wsCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
		TokenUse:   auth.TokenUseWS,
	})
	if _, err := GuardRequestAuthorization(wsCtx, AuthzRequest{Action: "read"}); !errors.Is(err, ErrMigrationForbidden) {
		t.Fatalf("expected WS ticket rejection, got %v", err)
	}

	pendingCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
		TokenUse:   auth.TokenUsePendingOrgSelect,
	})
	if _, err := GuardRequestAuthorization(pendingCtx, AuthzRequest{Action: "read"}); !errors.Is(err, ErrMigrationForbidden) {
		t.Fatalf("expected PendingOrgSelect rejection, got %v", err)
	}

	// Non-supplier role -> forbidden
	retailerCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:       "ret-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-1",
	})
	if _, err := GuardRequestAuthorization(retailerCtx, AuthzRequest{Action: "read"}); !errors.Is(err, ErrMigrationForbidden) {
		t.Fatalf("expected retailer role rejection, got %v", err)
	}

	// Cross-tenant access -> forbidden
	adminCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
		HomeCell:   "cell-uz",
	})
	if _, err := GuardRequestAuthorization(adminCtx, AuthzRequest{Action: "read", TargetSupplierID: "sup-2"}); !errors.Is(err, ErrMigrationForbidden) {
		t.Fatalf("expected cross-tenant rejection, got %v", err)
	}

	// Cross-cell access -> 403 cross_cell_forbidden
	if _, err := GuardRequestAuthorization(adminCtx, AuthzRequest{Action: "read", TargetSupplierID: "sup-1", TargetHomeCell: "cell-eu"}); !errors.Is(err, ErrCrossCellForbidden) {
		t.Fatalf("expected cross-cell rejection, got %v", err)
	}

	// Same tenant + same cell -> ok
	got, err := GuardRequestAuthorization(adminCtx, AuthzRequest{Action: "read", TargetSupplierID: "sup-1", TargetHomeCell: "cell-uz"})
	if err != nil {
		t.Fatalf("unexpected error for matching org admin: %v", err)
	}
	if got.Subject != "admin-1" {
		t.Fatalf("unexpected claims returned: %+v", got)
	}
}

func TestGuardRequestAuthorization_FinancialPermissionGate(t *testing.T) {
	t.Setenv("ALLOW_AUTH_BYPASS", "")

	// Derivative RoleWarehouseAdmin cannot apply nasiya_ar, set_on_hand, rollback, cutover, or release_open_orders.
	whAdminCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:      "wh-admin-1",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-1",
		SupplierRole: auth.RoleWarehouseAdmin,
	})

	if _, err := GuardRequestAuthorization(whAdminCtx, AuthzRequest{Pack: "stock", Action: "apply", TargetSupplierID: "sup-1"}); err != nil {
		t.Fatalf("warehouse admin should be allowed non-financial stock merge apply, got %v", err)
	}
	if _, err := GuardRequestAuthorization(whAdminCtx, AuthzRequest{Pack: "nasiya_ar", Action: "read", TargetSupplierID: "sup-1"}); err != nil {
		t.Fatalf("warehouse admin read on nasiya_ar should be allowed, got %v", err)
	}

	for _, tc := range []AuthzRequest{
		{Pack: "nasiya_ar", Action: "approve", TargetSupplierID: "sup-1"},
		{Pack: "nasiya_ar", Action: "apply", TargetSupplierID: "sup-1"},
		{Pack: "stock", Action: "set_on_hand", TargetSupplierID: "sup-1"},
		{Pack: "stock", Action: "rollback", TargetSupplierID: "sup-1"},
		{Pack: "", Action: "cutover", TargetSupplierID: "sup-1"},
		{Pack: "open_orders", Action: "release_open_orders", TargetSupplierID: "sup-1"},
	} {
		if _, err := GuardRequestAuthorization(whAdminCtx, tc); !errors.Is(err, ErrFinancialPermissionRequired) {
			t.Fatalf("expected ErrFinancialPermissionRequired for %+v, got %v", tc, err)
		}
	}

	// Org-level RoleAdmin passes financial gates.
	orgAdminCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "org-admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	})
	if _, err := GuardRequestAuthorization(orgAdminCtx, AuthzRequest{Pack: "nasiya_ar", Action: "apply", TargetSupplierID: "sup-1"}); err != nil {
		t.Fatalf("org admin should pass nasiya_ar apply, got %v", err)
	}
}

func TestWriteAuthzErrorFormatsDeterministicPayload(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteAuthzError(rec, ErrAuthBypassForbidden)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"auth_bypass_forbidden"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
