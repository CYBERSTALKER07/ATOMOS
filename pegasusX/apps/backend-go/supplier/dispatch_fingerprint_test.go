package supplier

import "testing"

func TestComputeSupplierPlanFingerprintStable(t *testing.T) {
	orders := []map[string]any{
		{"order_id": "o-2"},
		{"order_id": "o-1"},
	}
	routes := []map[string]any{
		{"driver_id": "d-1", "order_ids": []any{"o-2", "o-1"}},
	}
	a := computeSupplierPlanFingerprint(orders, routes)
	b := computeSupplierPlanFingerprint(orders, routes)
	if a == "" || a != b {
		t.Fatalf("fingerprints=%q %q", a, b)
	}
}
