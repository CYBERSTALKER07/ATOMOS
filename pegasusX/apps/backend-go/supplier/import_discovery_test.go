package supplier

import (
	"strings"
	"testing"
)

func TestDetectImportAnomalies_Int64MinorUnits(t *testing.T) {
	mappings := []importMappingCandidate{
		{SourceColumn: "UnitPriceTiyin", TargetField: "price_minor"},
	}
	rows := []map[string]any{
		{"UnitPriceTiyin": "10000"},
		{"UnitPriceTiyin": "10500"},
		{"UnitPriceTiyin": "9800"},
		{"UnitPriceTiyin": "150000"}, // >= 10x median (10000)
	}

	anomalies := detectImportAnomalies(rows, mappings)
	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %+v", anomalies)
	}
	if anomalies[0].Kind != "PRICE_VARIANCE_EXTREME" {
		t.Fatalf("kind=%q want PRICE_VARIANCE_EXTREME", anomalies[0].Kind)
	}
	if !strings.Contains(anomalies[0].Detail, "price_minor 150000 deviates more than 1000% from median 10500") {
		t.Fatalf("unexpected integer anomaly detail: %q", anomalies[0].Detail)
	}

	normalRows := []map[string]any{
		{"UnitPriceTiyin": "10000"},
		{"UnitPriceTiyin": "12000"},
		{"UnitPriceTiyin": "9500"},
	}
	if got := detectImportAnomalies(normalRows, mappings); len(got) != 0 {
		t.Fatalf("expected 0 anomalies for normal prices, got %+v", got)
	}
}
