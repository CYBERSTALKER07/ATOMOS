package order

import "testing"

func TestParseExceptionReportBody_LegacyDriverWire(t *testing.T) {
	body := []byte(`{
		"order_id": "ord-1",
		"missing_items": [
			{"sku_id": "SKU-A", "missing_qty": 2},
			{"sku_id": "SKU-B", "missing_qty": 1, "reason": "DAMAGED", "photo_url": "https://cdn/x.jpg"}
		]
	}`)
	req, err := parseExceptionReportBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if req.OrderID != "ord-1" {
		t.Fatalf("order_id=%q", req.OrderID)
	}
	if len(req.Items) != 2 {
		t.Fatalf("items=%d", len(req.Items))
	}
	if req.Items[0].SKU != "SKU-A" || req.Items[0].Quantity != 2 || req.Items[0].Reason != "MISSING" {
		t.Fatalf("item0=%+v", req.Items[0])
	}
	if req.Items[1].SKU != "SKU-B" || req.Items[1].Quantity != 1 || req.Items[1].Reason != "DAMAGED" {
		t.Fatalf("item1=%+v", req.Items[1])
	}
}

func TestParseExceptionReportBody_CanonicalWire(t *testing.T) {
	body := []byte(`{
		"order_id": "ord-2",
		"note": "seal broken",
		"photo_url": "https://cdn/order.jpg",
		"items": [
			{"sku": "SKU-1", "quantity": 1, "reason": "WRONG_ITEM", "photo_url": "https://cdn/line.jpg"}
		]
	}`)
	req, err := parseExceptionReportBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if req.PhotoURL == "" || len(req.Items) != 1 || req.Items[0].SKU != "SKU-1" {
		t.Fatalf("%+v", req)
	}
}
