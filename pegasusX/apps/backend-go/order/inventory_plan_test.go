package order

import "testing"

func TestResolveOutOfStockPolicy(t *testing.T) {
	if got := resolveOutOfStockPolicy("REJECT", "ACCEPT_BACKORDER"); got != outOfStockPolicyAcceptBackorder {
		t.Fatalf("product override expected ACCEPT_BACKORDER, got %s", got)
	}
	if got := resolveOutOfStockPolicy("ACCEPT_BACKORDER", ""); got != outOfStockPolicyAcceptBackorder {
		t.Fatalf("warehouse default expected ACCEPT_BACKORDER, got %s", got)
	}
	if got := resolveOutOfStockPolicy("ACCEPT_BACKORDER", "REJECT"); got != outOfStockPolicyReject {
		t.Fatalf("product reject override expected REJECT, got %s", got)
	}
}
