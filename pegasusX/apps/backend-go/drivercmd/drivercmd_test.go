package drivercmd

import (
	"encoding/json"
	"testing"
)

func TestPrepareJSON_MintsAndPreservesCommandID(t *testing.T) {
	raw := []byte(`{"type":"ROUTE_REPLANNED","trace_id":"tr-1","order_id":"ord-1"}`)
	out, cmdID, evType, traceID, ok := PrepareJSON(raw)
	if !ok {
		t.Fatal("expected valid envelope")
	}
	if cmdID == "" || evType != "ROUTE_REPLANNED" || traceID != "tr-1" {
		t.Fatalf("unexpected extracted fields: cmdID=%q evType=%q traceID=%q", cmdID, evType, traceID)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("unmarshal enriched: %v", err)
	}
	if decoded["command_id"] != cmdID {
		t.Fatalf("expected stamped command_id %q, got %v", cmdID, decoded["command_id"])
	}

	// Second call preserves the existing command_id deterministically.
	out2, cmdID2, _, _, ok2 := PrepareJSON(out)
	if !ok2 || cmdID2 != cmdID || string(out2) == "" {
		t.Fatalf("expected command_id preserved: got %q want %q", cmdID2, cmdID)
	}
}

func TestPrepareJSON_SkipsInvalidOrUntypedPayloads(t *testing.T) {
	if _, _, _, _, ok := PrepareJSON([]byte("not json")); ok {
		t.Fatal("expected false for invalid json")
	}
	if _, _, _, _, ok := PrepareJSON([]byte(`{"order_id":"ord-1"}`)); ok {
		t.Fatal("expected false when event type missing")
	}
}
