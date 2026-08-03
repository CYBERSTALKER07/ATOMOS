package order

import "testing"

func TestApplyNegotiatedLineItems(t *testing.T) {
	t.Parallel()
	existing := []LineItem{
		{SKU: "SSMR-SKU-1", Quantity: 2, UnitPrice: 1000},
		{SKU: "SSMR-SKU-2", Quantity: 3, UnitPrice: 500},
	}
	proposed := []ProposedNegotiationItem{
		{SKUID: "SSMR-SKU-1", OriginalQty: 2, ProposedQty: 1},
	}
	updated, total, err := applyNegotiatedLineItems(existing, proposed)
	if err != nil {
		t.Fatal(err)
	}
	if updated[0].Quantity != 1 {
		t.Fatalf("sku1 qty=%d want 1", updated[0].Quantity)
	}
	if updated[1].Quantity != 3 {
		t.Fatalf("sku2 should be unchanged, qty=%d", updated[1].Quantity)
	}
	// 1*1000 + 3*500 = 2500
	if total != 2500 {
		t.Fatalf("total=%d want 2500", total)
	}
}

func TestApplyNegotiatedLineItems_rejectNegative(t *testing.T) {
	t.Parallel()
	_, _, err := applyNegotiatedLineItems(
		[]LineItem{{SKU: "A", Quantity: 1, UnitPrice: 10}},
		[]ProposedNegotiationItem{{SKUID: "A", ProposedQty: -1}},
	)
	if err == nil {
		t.Fatal("expected error for negative proposed qty")
	}
}

func TestNegotiationFeatureDisabled(t *testing.T) {
	t.Parallel()
	if NegotiationFeatureEnabled() {
		t.Fatal("quantity negotiation must stay product-disabled")
	}
	if !quantityNegotiationDisabled() {
		t.Fatal("quantityNegotiationDisabled() must be true")
	}
}
