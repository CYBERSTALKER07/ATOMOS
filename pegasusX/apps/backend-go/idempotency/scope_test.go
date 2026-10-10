package idempotency

import "testing"

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
