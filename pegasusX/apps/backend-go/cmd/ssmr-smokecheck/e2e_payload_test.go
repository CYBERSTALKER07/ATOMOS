package main

import (
	"encoding/json"
	"testing"
)

func TestPayloadSealBodyRequiresManifestID(t *testing.T) {
	raw := payloadSealBody("mf-1", "ord-1", "veh-1")
	var got struct {
		ManifestID string `json:"manifest_id"`
		OrderID    string `json:"order_id"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ManifestID != "mf-1" || got.OrderID != "ord-1" {
		t.Fatalf("got %+v", got)
	}
}
