package demand

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLineItemsJsonParsingAndVelocityAggregation(t *testing.T) {
	// Sample JSON matching real Spanner orders
	rawJSON := []byte(`[
		{"sku": "sku-1", "product_id": "", "delivered_qty": 56.0, "quantity": 56.0},
		{"sku": "", "product_id": "sku-2", "delivered_qty": 0.0, "quantity": 28.0},
		{"sku": "sku-3", "qty": 84.0}
	]`)

	var lines []struct {
		SKU          string  `json:"sku"`
		ProductID    string  `json:"product_id"`
		DeliveredQty float64 `json:"delivered_qty"`
		Quantity     float64 `json:"quantity"`
		Qty          float64 `json:"qty"`
	}
	if err := json.Unmarshal(rawJSON, &lines); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	totals := map[string]float64{}
	for _, l := range lines {
		sku := strings.TrimSpace(l.SKU)
		if sku == "" {
			sku = strings.TrimSpace(l.ProductID)
		}
		if sku == "" {
			continue
		}
		qty := l.DeliveredQty
		if qty == 0 {
			qty = l.Quantity
		}
		if qty == 0 {
			qty = l.Qty
		}
		if qty > 0 {
			totals[sku] += qty
		}
	}

	// Verify 28-day base velocity: sum / 28.0
	if v := totals["sku-1"] / 28.0; v != 2.0 {
		t.Fatalf("sku-1 velocity: got %v, want 2.0", v)
	}
	if v := totals["sku-2"] / 28.0; v != 1.0 {
		t.Fatalf("sku-2 velocity (falling back to quantity): got %v, want 1.0", v)
	}
	if v := totals["sku-3"] / 28.0; v != 3.0 {
		t.Fatalf("sku-3 velocity (falling back to qty): got %v, want 3.0", v)
	}
}
