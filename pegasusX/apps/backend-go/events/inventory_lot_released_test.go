package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryLotReleasedEventContract(t *testing.T) {
	if EventInventoryLotReleased != "inventory.lot.released" {
		t.Fatalf("EventInventoryLotReleased = %q, want %q", EventInventoryLotReleased, "inventory.lot.released")
	}

	ev := InventoryLotReleasedEvent{
		BaseEvent: BaseEvent{
			Type:      EventInventoryLotReleased,
			Version:   1,
			Timestamp: "2026-11-16T03:00:00Z",
		},
		OrderID:     "ord-1",
		SupplierID:  "sup-1",
		WarehouseID: "wh-1",
		ReleasedLots: []ReleasedLotQuantity{
			{LotID: "lot-1", Quantity: 5},
			{LotID: "lot-2", Quantity: 3},
		},
	}

	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal InventoryLotReleasedEvent: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal InventoryLotReleasedEvent: %v", err)
	}
	if decoded["type"] != EventInventoryLotReleased {
		t.Fatalf("type = %v, want %s", decoded["type"], EventInventoryLotReleased)
	}
	if decoded["order_id"] != "ord-1" || decoded["supplier_id"] != "sup-1" || decoded["warehouse_id"] != "wh-1" {
		t.Fatalf("unexpected id fields: %v", decoded)
	}
	lots, ok := decoded["released_lots"].([]any)
	if !ok || len(lots) != 2 {
		t.Fatalf("released_lots = %v, want 2 items", decoded["released_lots"])
	}

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	if !strings.Contains(string(envelopeSrc), `"inventory.lot.released"`) {
		t.Fatalf("%s missing %q in EventType union", envelopePath, EventInventoryLotReleased)
	}
}
