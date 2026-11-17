package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestScopeKeyBindsPrincipalAndRoute(t *testing.T) {
	t.Parallel()
	a := ScopeKey("user-a", "POST /v1/orders", "checkout-1")
	b := ScopeKey("user-b", "POST /v1/orders", "checkout-1")
	c := ScopeKey("user-a", "POST /v1/other", "checkout-1")
	if a == "" || len(a) != 64 {
		t.Fatalf("want sha256 hex len 64, got %q", a)
	}
	if a == b {
		t.Fatal("different principals must not share key")
	}
	if a == c {
		t.Fatal("different routes must not share key")
	}
	if ScopeKey("user-a", "POST /v1/orders", "checkout-1") != a {
		t.Fatal("scope key must be deterministic")
	}
}

func TestScopeRequestIsolatesRetailerTenantsAndMultiUsers(t *testing.T) {
	t.Parallel()
	req := func(c auth.Claims) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
		return r.WithContext(auth.WithClaims(r.Context(), c))
	}

	// Same user switching retailer orgs must never collide.
	orgA := ScopeRequest(req(auth.Claims{
		Subject:       "user-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-org-a",
	}), "shared-key")
	orgB := ScopeRequest(req(auth.Claims{
		Subject:       "user-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-org-b",
	}), "shared-key")
	if orgA == orgB {
		t.Fatal("same subject across different RetailerOrgIDs must not share key")
	}

	// Distinct users within the same retailer org must not collide/leak.
	orgAUser2 := ScopeRequest(req(auth.Claims{
		Subject:       "user-2",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-org-a",
	}), "shared-key")
	if orgA == orgAUser2 {
		t.Fatal("distinct users in the same RetailerOrgID must not collide on same raw key")
	}

	// Legacy token (empty RetailerOrgID) vs explicit v2 token on same org + subject should match.
	legacy := ScopeRequest(req(auth.Claims{
		Subject: "ret-legacy",
		Role:    auth.RoleRetailer,
	}), "shared-key")
	v2SameOwner := ScopeRequest(req(auth.Claims{
		Subject:        "ret-legacy",
		RetailerUserID: "ret-legacy",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-legacy",
	}), "shared-key")
	if legacy != v2SameOwner {
		t.Fatalf("legacy and explicit owner claims for same retailer org mismatch: %q vs %q", legacy, v2SameOwner)
	}

	// Distinct supplier tenants with same Subject must not share key.
	sup1 := ScopeRequest(req(auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}), "shared-key")
	sup2 := ScopeRequest(req(auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-2",
	}), "shared-key")
	if sup1 == sup2 {
		t.Fatal("distinct SupplierIDs must not share key")
	}

	// Contextless caller keeps unauthenticated empty principal.
	unauth := ScopeRequest(httptest.NewRequest(http.MethodPost, "/v1/orders", nil), "shared-key")
	if unauth != ScopeKey("", "POST /v1/orders", "shared-key") {
		t.Fatalf("unauthenticated key mismatch: %q", unauth)
	}
	if got := ScopePrincipal(context.Background()); got != "" {
		t.Fatalf("expected empty principal for background ctx, got %q", got)
	}
}
