package main

import (
	"testing"

	aibridge "aibridge"
)

func TestHeuristicFallbackMappings_PriseToUnitPrice(t *testing.T) {
	headers := []string{"SKU", "Prise", "Qty"}
	resolved, unresolved := deterministicMappings(headers, supplierInventoryTargetFields)

	if _, ok := resolved["Prise"]; ok {
		t.Fatalf("expected misspelled header to remain unresolved for AI/fallback stage")
	}

	fallback := heuristicFallbackMappings(unresolved, supplierInventoryTargetFields)
	var found bool
	for _, candidate := range fallback {
		if candidate.SourceColumn == "Prise" {
			found = true
			if candidate.TargetField != "unit_price" {
				t.Fatalf("expected Prise -> unit_price, got %s", candidate.TargetField)
			}
			if candidate.Confidence < confidenceThreshold {
				t.Fatalf("expected confidence >= %.2f, got %.2f", confidenceThreshold, candidate.Confidence)
			}
		}
	}
	if !found {
		t.Fatalf("expected fallback mapping for misspelled Prise header")
	}
}

func TestResolveDiscoveryStatus_LowConfidenceRequiresMapping(t *testing.T) {
	status, lowConfidence := resolveDiscoveryStatus([]aibridge.MappingCandidate{
		{SourceColumn: "Column_A", TargetField: "sku_id", Confidence: 1},
		{SourceColumn: "Column_B", TargetField: "unit_price", Confidence: 0.61},
	})

	if status != "MAPPING_REQUIRED" {
		t.Fatalf("expected status MAPPING_REQUIRED, got %s", status)
	}
	if !lowConfidence {
		t.Fatalf("expected lowConfidence=true")
	}
}
