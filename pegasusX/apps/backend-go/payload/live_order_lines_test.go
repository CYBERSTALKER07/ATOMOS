package payload

import "testing"

func TestDecodeLiveOrderLineItems(t *testing.T) {
	raw := []byte(`[{"sku":"sku-1","name":"Milk","quantity":6},{"sku_id":"sku-2","name":"Bread","quantity":2}]`)
	items := decodeLiveOrderLineItems(raw)
	if len(items) != 2 {
		t.Fatalf("items len = %d want 2", len(items))
	}
	if items[0].SKUID != "sku-1" || items[0].ProductName != "Milk" || items[0].Quantity != 6 {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[1].SKUID != "sku-2" || items[1].Quantity != 2 {
		t.Fatalf("second item = %#v", items[1])
	}
}
