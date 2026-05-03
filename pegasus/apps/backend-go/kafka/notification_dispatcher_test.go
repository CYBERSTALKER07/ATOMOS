package kafka

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewPayloadSyncFrame(t *testing.T) {
	ts := time.Date(2026, time.May, 3, 11, 0, 0, 0, time.UTC)
	frame := newPayloadSyncFrame(PayloadSyncEvent{
		SupplierID:  "supplier-1",
		WarehouseID: "warehouse-9",
		ManifestID:  "manifest-7",
		Reason:      "REBALANCED",
		Timestamp:   ts,
	})

	if frame.Type != EventPayloadSync {
		t.Fatalf("type = %q, want %q", frame.Type, EventPayloadSync)
	}
	if frame.Channel != "SYNC" {
		t.Fatalf("channel = %q, want %q", frame.Channel, "SYNC")
	}
	if frame.ManifestID != "manifest-7" {
		t.Fatalf("manifest_id = %q, want manifest-7", frame.ManifestID)
	}
	if frame.WarehouseID != "warehouse-9" {
		t.Fatalf("warehouse_id = %q, want warehouse-9", frame.WarehouseID)
	}
	if frame.Reason != "REBALANCED" {
		t.Fatalf("reason = %q, want REBALANCED", frame.Reason)
	}
	if !frame.Timestamp.Equal(ts) {
		t.Fatalf("timestamp = %s, want %s", frame.Timestamp, ts)
	}

	data, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}

	if raw["type"] != EventPayloadSync {
		t.Fatalf("json type = %#v, want %q", raw["type"], EventPayloadSync)
	}
	if raw["channel"] != "SYNC" {
		t.Fatalf("json channel = %#v, want %q", raw["channel"], "SYNC")
	}
	if raw["manifest_id"] != "manifest-7" {
		t.Fatalf("json manifest_id = %#v, want manifest-7", raw["manifest_id"])
	}
	if _, ok := raw["supplier_id"]; ok {
		t.Fatal("json unexpectedly contains supplier_id")
	}
}
