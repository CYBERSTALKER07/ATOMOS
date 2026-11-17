package stocklots

import (
	"testing"
)

func TestSumAvailableLotSnapshots_OverridesExcludesQuarantined(t *testing.T) {
	// Baseline: 2 lots in warehouse, each 50 QoH, 5 reserved.
	lot1 := StockLotSnapshot{
		SupplierID:       "sup-1",
		WarehouseID:      "wh-1",
		ProductID:        "prod-1",
		QuantityOnHand:   50,
		QuantityReserved: 5,
		Status:           "AVAILABLE",
	}
	lot2 := StockLotSnapshot{
		SupplierID:       "sup-1",
		WarehouseID:      "wh-1",
		ProductID:        "prod-1",
		QuantityOnHand:   50,
		QuantityReserved: 5,
		Status:           "AVAILABLE",
	}

	// Before quarantine: total available should be 100 QoH, 10 QR
	qoh, qr := sumAvailableLotSnapshots([]StockLotSnapshot{lot1, lot2}, "sup-1", "wh-1", "prod-1")
	if qoh != 100 || qr != 10 {
		t.Fatalf("expected 100 QoH and 10 QR before quarantine, got %d, %d", qoh, qr)
	}

	// In-txn mutation: lot1 is quarantined
	lot1Quarantined := lot1
	lot1Quarantined.Status = "QUARANTINED"

	// Overrides overlay applied during QuarantineLotInTxn
	overriddenRows := []StockLotSnapshot{lot1Quarantined, lot2}
	qohAfter, qrAfter := sumAvailableLotSnapshots(overriddenRows, "sup-1", "wh-1", "prod-1")
	if qohAfter != 50 || qrAfter != 5 {
		t.Fatalf("expected 50 QoH and 5 QR after quarantine override, got %d, %d", qohAfter, qrAfter)
	}
}

func TestSumAvailableLotSnapshots_OverridesIncludesReleased(t *testing.T) {
	// Baseline: lot1 is currently QUARANTINED in database
	lot1Quarantined := StockLotSnapshot{
		SupplierID:       "sup-1",
		WarehouseID:      "wh-1",
		ProductID:        "prod-1",
		QuantityOnHand:   30,
		QuantityReserved: 0,
		Status:           "QUARANTINED",
	}
	lot2Available := StockLotSnapshot{
		SupplierID:       "sup-1",
		WarehouseID:      "wh-1",
		ProductID:        "prod-1",
		QuantityOnHand:   40,
		QuantityReserved: 0,
		Status:           "AVAILABLE",
	}

	// Before release: only lot2 is available
	qoh, qr := sumAvailableLotSnapshots([]StockLotSnapshot{lot1Quarantined, lot2Available}, "sup-1", "wh-1", "prod-1")
	if qoh != 40 || qr != 0 {
		t.Fatalf("expected 40 QoH before release, got %d", qoh)
	}

	// In-txn mutation: lot1 is released back to AVAILABLE
	lot1Released := lot1Quarantined
	lot1Released.Status = "AVAILABLE"

	// Overrides overlay applied during ReleaseLotInTxn
	overriddenRows := []StockLotSnapshot{lot1Released, lot2Available}
	qohAfter, qrAfter := sumAvailableLotSnapshots(overriddenRows, "sup-1", "wh-1", "prod-1")
	if qohAfter != 70 || qrAfter != 0 {
		t.Fatalf("expected 70 QoH after release override, got %d", qohAfter)
	}
}

func TestSumAvailableLotSnapshots_RecallCampaignMultipleLots(t *testing.T) {
	lots := []StockLotSnapshot{
		{SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "prod-milk", QuantityOnHand: 100, Status: "AVAILABLE"},
		{SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "prod-milk", QuantityOnHand: 120, Status: "AVAILABLE"},
		{SupplierID: "sup-1", WarehouseID: "wh-2", ProductID: "prod-milk", QuantityOnHand: 80, Status: "AVAILABLE"},
		{SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "prod-bread", QuantityOnHand: 200, Status: "AVAILABLE"},
	}

	// Recall campaign targets first lot in wh-1
	recalledLot0 := lots[0]
	recalledLot0.Status = "RECALLED"

	activeWh1 := []StockLotSnapshot{recalledLot0, lots[1]}
	qohWh1, _ := sumAvailableLotSnapshots(activeWh1, "sup-1", "wh-1", "prod-milk")
	if qohWh1 != 120 {
		t.Fatalf("expected wh-1 milk QoH to be 120 after recall override, got %d", qohWh1)
	}

	// Other warehouse and product rollups remain unaffected
	qohWh2, _ := sumAvailableLotSnapshots([]StockLotSnapshot{lots[2]}, "sup-1", "wh-2", "prod-milk")
	if qohWh2 != 80 {
		t.Fatalf("expected wh-2 milk QoH to remain 80, got %d", qohWh2)
	}
}
