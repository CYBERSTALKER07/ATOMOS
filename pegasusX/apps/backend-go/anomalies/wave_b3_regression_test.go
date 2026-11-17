package anomalies

import "testing"

func TestBuildCostMap_NewestLayerAndFallbackPrecedence(t *testing.T) {
	layers := []CostLayerSummary{
		{ProductID: "SKU-1", UnitCostMinor: 8_000}, // newest first
		{ProductID: "SKU-1", UnitCostMinor: 5_000}, // older must NOT overwrite
	}
	fallbacks := []ProductPriceRecord{
		{ProductID: "SKU-1", UnitPriceMinor: 12_000}, // cost layer must win over fallback price
		{ProductID: "SKU-2", UnitPriceMinor: 9_000},  // fallback applies when no layer exists
	}
	got := buildCostMap(layers, fallbacks)
	if got["SKU-1"] != 8_000 {
		t.Fatalf("SKU-1 cost=%d, want 8000 (newest layer)", got["SKU-1"])
	}
	if got["SKU-2"] != 9_000 {
		t.Fatalf("SKU-2 cost=%d, want 9000 (fallback)", got["SKU-2"])
	}
}
