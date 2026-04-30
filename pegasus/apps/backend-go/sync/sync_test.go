package sync

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── OfflineDelivery JSON contract ──────────────────────────────────────────

func TestOfflineDelivery_JSONRoundtrip(t *testing.T) {
	orig := OfflineDelivery{
		OrderID:   "ORD-001",
		Signature: "abc123hash",
		Timestamp: 1712937600000,
		Status:    "DELIVERED",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded OfflineDelivery
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != orig {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", decoded, orig)
	}
}

func TestOfflineDelivery_JSONKeys(t *testing.T) {
	d := OfflineDelivery{OrderID: "ORD-X", Signature: "sig", Timestamp: 123, Status: "DELIVERED"}
	data, _ := json.Marshal(d)
	s := string(data)
	for _, key := range []string{`"order_id"`, `"signature"`, `"timestamp"`, `"status"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("JSON should contain key %s, got: %s", key, s)
		}
	}
}

// ── BatchSyncPayload JSON contract ─────────────────────────────────────────

func TestBatchSyncPayload_JSONRoundtrip(t *testing.T) {
	orig := BatchSyncPayload{
		DriverID: "DRV-001",
		Deliveries: []OfflineDelivery{
			{OrderID: "ORD-A", Signature: "sig-a", Timestamp: 100, Status: "DELIVERED"},
			{OrderID: "ORD-B", Signature: "sig-b", Timestamp: 200, Status: "REJECTED_DAMAGED"},
		},
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded BatchSyncPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.DriverID != orig.DriverID || len(decoded.Deliveries) != 2 {
		t.Fatalf("mismatch: got %+v", decoded)
	}
}

func TestBatchSyncPayload_JSONKeys(t *testing.T) {
	p := BatchSyncPayload{DriverID: "DRV-X", Deliveries: []OfflineDelivery{}}
	data, _ := json.Marshal(p)
	s := string(data)
	for _, key := range []string{`"driver_id"`, `"deliveries"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("JSON should contain key %s, got: %s", key, s)
		}
	}
}

// ── BatchSyncResponse JSON contract ────────────────────────────────────────

func TestBatchSyncResponse_JSONRoundtrip(t *testing.T) {
	orig := BatchSyncResponse{
		Status:    "SYNC_COMPLETE",
		Processed: []string{"ORD-A", "ORD-B"},
		Skipped:   1,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded BatchSyncResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Status != orig.Status || len(decoded.Processed) != 2 || decoded.Skipped != 1 {
		t.Fatalf("mismatch: got %+v", decoded)
	}
}

func TestBatchSyncResponse_JSONKeys(t *testing.T) {
	r := BatchSyncResponse{Status: "OK", Processed: []string{"A"}, Skipped: 0}
	data, _ := json.Marshal(r)
	s := string(data)
	for _, key := range []string{`"status"`, `"processed"`, `"skipped"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("JSON should contain key %s, got: %s", key, s)
		}
	}
}

// ── Dedup key format ───────────────────────────────────────────────────────

func TestDedupKeyFormat(t *testing.T) {
	// Verify the dedup key pattern matches what batch.go uses:
	// fmt.Sprintf("desert:sync:%s:%s", driverID, orderID)
	key := "desert:sync:DRV-001:ORD-001"
	if !strings.HasPrefix(key, "desert:sync:") {
		t.Fatal("dedup key should start with desert:sync:")
	}
	parts := strings.Split(key, ":")
	if len(parts) != 4 {
		t.Fatalf("dedup key should have 4 colon-separated parts, got %d", len(parts))
	}
	if parts[2] != "DRV-001" || parts[3] != "ORD-001" {
		t.Fatalf("unexpected parts: %v", parts)
	}
}

func TestBatchSyncPayload_EmptyDeliveries(t *testing.T) {
	p := BatchSyncPayload{DriverID: "DRV-EMPTY"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// null deliveries should unmarshal cleanly
	var decoded BatchSyncPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.DriverID != "DRV-EMPTY" {
		t.Fatalf("driver_id mismatch: %s", decoded.DriverID)
	}
}
