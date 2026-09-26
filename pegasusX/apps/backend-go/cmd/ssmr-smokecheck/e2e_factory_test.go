package main

import (
	"encoding/json"
	"testing"
)

func TestFactorySmokeOrderIDFitsSpanner(t *testing.T) {
	for i := 0; i < 20; i++ {
		id := factorySmokeOrderID(i)
		if id == "" {
			t.Fatal("empty factory smoke order id")
		}
		if len(id) > 36 {
			t.Fatalf("order_id %q len %d exceeds FactoryInternalTransfers.OrderId STRING(36)", id, len(id))
		}
	}
}

func TestFactoryDispatchBodyPinsTransferIDs(t *testing.T) {
	raw := factoryDispatchBody("ssmr-smoke-a", []string{"tr_a", "tr_b"})
	var got struct {
		Reason      string   `json:"reason"`
		TransferIDs []string `json:"transfer_ids"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Reason != "ssmr-smoke-a" {
		t.Fatalf("reason=%q", got.Reason)
	}
	if len(got.TransferIDs) != 2 || got.TransferIDs[0] != "tr_a" || got.TransferIDs[1] != "tr_b" {
		t.Fatalf("transfer_ids=%v", got.TransferIDs)
	}
}
