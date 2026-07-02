package planning

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBaselineFromSignalIngest(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(map[string]any{
		"product_id": "sku-1",
		"units":      42,
		"confidence": 0.8,
	})
	in := SignalIngestInput{
		SignalID:    "sig-1",
		Source:      "ssmr_test",
		WarehouseID: "wh-1",
		Payload:     payload,
	}
	baseline, ok := baselineFromSignalIngest("sup-1", in, now)
	if !ok {
		t.Fatal("expected baseline")
	}
	if baseline.ProductID != "sku-1" || baseline.BaselineQty != 42 || baseline.WarehouseID != "wh-1" {
		t.Fatalf("unexpected baseline: %+v", baseline)
	}
	if baseline.BaselineSource != "signal_ingest" {
		t.Fatalf("expected signal_ingest source, got %q", baseline.BaselineSource)
	}

	_, ok = baselineFromSignalIngest("sup-1", SignalIngestInput{
		Source:  "no_product",
		Payload: json.RawMessage(`{"units":10}`),
	}, now)
	if ok {
		t.Fatal("expected skip without product_id and warehouse_id")
	}
}
