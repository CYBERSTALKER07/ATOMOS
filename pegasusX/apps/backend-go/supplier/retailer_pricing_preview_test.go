package supplier

import "testing"

func TestLineItemsContainProduct(t *testing.T) {
	raw := []byte(`[{"product_id":"prod-1","sku_id":"sku-1"},{"product_id":"prod-2"}]`)
	if !lineItemsContainProduct(raw, "prod-1") {
		t.Fatal("expected prod-1 match")
	}
	if !lineItemsContainProduct(raw, "sku-1") {
		t.Fatal("expected sku-1 match")
	}
	if lineItemsContainProduct(raw, "missing") {
		t.Fatal("expected no match")
	}
}
