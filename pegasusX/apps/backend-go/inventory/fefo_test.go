package inventory

import (
	"testing"
	"time"
)

func TestFEFO_EarliestExpiredFirst(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	warehouseID := "wh-001"
	supplierID := "sup-001"
	productID := "prod-milk"

	loc1 := Location{
		LocationID:   "loc-01",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		Aisle:        "A",
		Rack:         "01",
		Shelf:        "01",
		Bin:          "01",
		Zone:         "COLD",
		LocationType: "PICK",
		IsActive:     true,
	}
	loc2 := Location{
		LocationID:   "loc-02",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		Aisle:        "A",
		Rack:         "01",
		Shelf:        "01",
		Bin:          "02",
		Zone:         "COLD",
		LocationType: "PICK",
		IsActive:     true,
	}
	loc3 := Location{
		LocationID:   "loc-03",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		Aisle:        "A",
		Rack:         "01",
		Shelf:        "01",
		Bin:          "03",
		Zone:         "COLD",
		LocationType: "PICK",
		IsActive:     true,
	}

	lot1 := StockLot{
		LotID:             "lot-exp-10d",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc1.LocationID,
		LotCode:           "LOT-10D",
		ExpiryDate:        now.Add(10 * 24 * time.Hour),
		QuantityOnHand:    20,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}
	lot2 := StockLot{
		LotID:             "lot-exp-03d",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc2.LocationID,
		LotCode:           "LOT-03D",
		ExpiryDate:        now.Add(3 * 24 * time.Hour),
		QuantityOnHand:    20,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}
	lot3 := StockLot{
		LotID:             "lot-exp-30d",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc3.LocationID,
		LotCode:           "LOT-30D",
		ExpiryDate:        now.Add(30 * 24 * time.Hour),
		QuantityOnHand:    20,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}

	lotsByProduct := map[string][]StockLotLocation{
		productID: {
			{Lot: lot1, Location: loc1},
			{Lot: lot2, Location: loc2},
			{Lot: lot3, Location: loc3},
		},
	}

	req := WavePickRequest{
		WaveID:       "wave-001",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: false,
		Items: []WavePickItem{
			{
				OrderID:   "order-101",
				LineID:    "line-1",
				ProductID: productID,
				Quantity:  15,
			},
		},
	}

	res, err := AllocateWaveFEFOPure(now, req, lotsByProduct)
	if err != nil {
		t.Fatalf("unexpected allocation error: %v", err)
	}

	if res.TotalUnits != 15 {
		t.Errorf("expected total units 15, got %d", res.TotalUnits)
	}
	if len(res.Instructions) != 1 {
		t.Fatalf("expected 1 pick instruction, got %d", len(res.Instructions))
	}

	instr := res.Instructions[0]
	if instr.LotID != "lot-exp-03d" {
		t.Errorf("expected earliest expiring lot 'lot-exp-03d', got %s", instr.LotID)
	}
	if instr.LotCode != "LOT-03D" {
		t.Errorf("expected lot code 'LOT-03D', got %s", instr.LotCode)
	}
	if instr.Quantity != 15 {
		t.Errorf("expected quantity 15, got %d", instr.Quantity)
	}
	if instr.PickSequence != 1 {
		t.Errorf("expected pick sequence 1, got %d", instr.PickSequence)
	}
}

func TestFEFO_MultiLotSplit(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	warehouseID := "wh-001"
	supplierID := "sup-001"
	productID := "prod-bread"

	locA := Location{LocationID: "loc-A", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "01", IsActive: true}
	locB := Location{LocationID: "loc-B", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "02", IsActive: true}
	locC := Location{LocationID: "loc-C", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "03", IsActive: true}

	lotA := StockLot{
		LotID:             "lot-A-exp2d",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        locA.LocationID,
		LotCode:           "LOT-A",
		ExpiryDate:        now.Add(2 * 24 * time.Hour),
		QuantityOnHand:    10,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}
	lotB := StockLot{
		LotID:             "lot-B-exp5d",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        locB.LocationID,
		LotCode:           "LOT-B",
		ExpiryDate:        now.Add(5 * 24 * time.Hour),
		QuantityOnHand:    10,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}
	lotC := StockLot{
		LotID:             "lot-C-exp10d",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        locC.LocationID,
		LotCode:           "LOT-C",
		ExpiryDate:        now.Add(10 * 24 * time.Hour),
		QuantityOnHand:    10,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}

	lotsByProduct := map[string][]StockLotLocation{
		productID: {
			{Lot: lotA, Location: locA},
			{Lot: lotB, Location: locB},
			{Lot: lotC, Location: locC},
		},
	}

	req := WavePickRequest{
		WaveID:       "wave-002",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: false,
		Items: []WavePickItem{
			{OrderID: "ord-1", LineID: "l-1", ProductID: productID, Quantity: 15},
		},
	}

	res, err := AllocateWaveFEFOPure(now, req, lotsByProduct)
	if err != nil {
		t.Fatalf("unexpected allocation error: %v", err)
	}

	if res.TotalUnits != 15 {
		t.Errorf("expected total units 15, got %d", res.TotalUnits)
	}
	if len(res.Instructions) != 2 {
		t.Fatalf("expected 2 pick instructions for split, got %d", len(res.Instructions))
	}

	// Should take 10 from lotA (expiring day 2) and 5 from lotB (expiring day 5)
	first := res.Instructions[0]
	second := res.Instructions[1]

	if first.LotID != "lot-A-exp2d" || first.Quantity != 10 {
		t.Errorf("expected first instruction 10 units from lot-A-exp2d, got %d from %s", first.Quantity, first.LotID)
	}
	if second.LotID != "lot-B-exp5d" || second.Quantity != 5 {
		t.Errorf("expected second instruction 5 units from lot-B-exp5d, got %d from %s", second.Quantity, second.LotID)
	}
}

func TestFEFO_ExpiredLotsExcluded(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	warehouseID := "wh-001"
	supplierID := "sup-001"
	productID := "prod-yogurt"

	loc1 := Location{LocationID: "loc-01", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "01", IsActive: true}
	loc2 := Location{LocationID: "loc-02", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "02", IsActive: true}
	loc3 := Location{LocationID: "loc-03", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "03", IsActive: true}

	lotExpired := StockLot{
		LotID:             "lot-expired",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc1.LocationID,
		LotCode:           "LOT-EXPIRED",
		ExpiryDate:        now.Add(-48 * time.Hour), // Expired 2 days ago
		QuantityOnHand:    50,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}
	lotQuarantine := StockLot{
		LotID:             "lot-quarantine",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc2.LocationID,
		LotCode:           "LOT-QUARANTINE",
		ExpiryDate:        now.Add(5 * 24 * time.Hour),
		QuantityOnHand:    50,
		QuantityAllocated: 0,
		Status:            "QUARANTINE",
	}
	lotValid := StockLot{
		LotID:             "lot-valid",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc3.LocationID,
		LotCode:           "LOT-VALID",
		ExpiryDate:        now.Add(10 * 24 * time.Hour),
		QuantityOnHand:    20,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}

	lotsByProduct := map[string][]StockLotLocation{
		productID: {
			{Lot: lotExpired, Location: loc1},
			{Lot: lotQuarantine, Location: loc2},
			{Lot: lotValid, Location: loc3},
		},
	}

	req := WavePickRequest{
		WaveID:       "wave-003",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: false,
		Items: []WavePickItem{
			{OrderID: "ord-1", LineID: "l-1", ProductID: productID, Quantity: 15},
		},
	}

	res, err := AllocateWaveFEFOPure(now, req, lotsByProduct)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Instructions) != 1 {
		t.Fatalf("expected 1 instruction from valid lot, got %d", len(res.Instructions))
	}
	if res.Instructions[0].LotID != "lot-valid" {
		t.Errorf("expected allocation from lot-valid, got %s", res.Instructions[0].LotID)
	}
	if res.Instructions[0].Quantity != 15 {
		t.Errorf("expected 15 units allocated, got %d", res.Instructions[0].Quantity)
	}

	// Now try requesting more than valid lot has (e.g. 25) with AllowPartial=false -> must fail because expired/quarantine cannot be used
	reqOver := WavePickRequest{
		WaveID:       "wave-003-over",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: false,
		Items: []WavePickItem{
			{OrderID: "ord-1", LineID: "l-1", ProductID: productID, Quantity: 25},
		},
	}
	_, errOver := AllocateWaveFEFOPure(now, reqOver, lotsByProduct)
	if errOver == nil {
		t.Fatal("expected error when available unexpired stock is insufficient, got nil")
	}
}

func TestFEFO_SerpentinePathSorting(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	warehouseID := "wh-001"
	supplierID := "sup-001"

	locB2 := Location{LocationID: "loc-B-02-01-01", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "B", Rack: "02", Shelf: "01", Bin: "01", IsActive: true}
	locA2 := Location{LocationID: "loc-A-01-02-01", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "02", Bin: "01", IsActive: true}
	locA12 := Location{LocationID: "loc-A-01-01-02", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "02", IsActive: true}
	locA11 := Location{LocationID: "loc-A-01-01-01", WarehouseID: warehouseID, SupplierID: supplierID, Aisle: "A", Rack: "01", Shelf: "01", Bin: "01", IsActive: true}

	lotsByProduct := map[string][]StockLotLocation{
		"prod-1": {{Lot: StockLot{LotID: "lot-1", Status: "AVAILABLE", ExpiryDate: now.Add(24 * time.Hour), QuantityOnHand: 10}, Location: locB2}},
		"prod-2": {{Lot: StockLot{LotID: "lot-2", Status: "AVAILABLE", ExpiryDate: now.Add(24 * time.Hour), QuantityOnHand: 10}, Location: locA2}},
		"prod-3": {{Lot: StockLot{LotID: "lot-3", Status: "AVAILABLE", ExpiryDate: now.Add(24 * time.Hour), QuantityOnHand: 10}, Location: locA12}},
		"prod-4": {{Lot: StockLot{LotID: "lot-4", Status: "AVAILABLE", ExpiryDate: now.Add(24 * time.Hour), QuantityOnHand: 10}, Location: locA11}},
	}

	req := WavePickRequest{
		WaveID:       "wave-004",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: false,
		Items: []WavePickItem{
			{OrderID: "o1", LineID: "l1", ProductID: "prod-1", Quantity: 1},
			{OrderID: "o1", LineID: "l2", ProductID: "prod-2", Quantity: 1},
			{OrderID: "o1", LineID: "l3", ProductID: "prod-3", Quantity: 1},
			{OrderID: "o1", LineID: "l4", ProductID: "prod-4", Quantity: 1},
		},
	}

	res, err := AllocateWaveFEFOPure(now, req, lotsByProduct)
	if err != nil {
		t.Fatalf("unexpected allocation error: %v", err)
	}

	if len(res.Instructions) != 4 {
		t.Fatalf("expected 4 instructions, got %d", len(res.Instructions))
	}

	expectedOrder := []struct {
		aisle string
		rack  string
		shelf string
		bin   string
		seq   int
	}{
		{aisle: "A", rack: "01", shelf: "01", bin: "01", seq: 1},
		{aisle: "A", rack: "01", shelf: "01", bin: "02", seq: 2},
		{aisle: "A", rack: "01", shelf: "02", bin: "01", seq: 3},
		{aisle: "B", rack: "02", shelf: "01", bin: "01", seq: 4},
	}

	for i, exp := range expectedOrder {
		instr := res.Instructions[i]
		if instr.Aisle != exp.aisle || instr.Rack != exp.rack || instr.Shelf != exp.shelf || instr.Bin != exp.bin {
			t.Errorf("instruction %d location mismatch: got (%s,%s,%s,%s), expected (%s,%s,%s,%s)",
				i, instr.Aisle, instr.Rack, instr.Shelf, instr.Bin, exp.aisle, exp.rack, exp.shelf, exp.bin)
		}
		if instr.PickSequence != exp.seq {
			t.Errorf("instruction %d sequence mismatch: got %d, expected %d", i, instr.PickSequence, exp.seq)
		}
	}
}

func TestFEFO_ShortfallPartial(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	warehouseID := "wh-001"
	supplierID := "sup-001"
	productID := "prod-cheese"

	loc := Location{
		LocationID:  "loc-01",
		WarehouseID: warehouseID,
		SupplierID:  supplierID,
		Aisle:       "A",
		Rack:        "01",
		Shelf:       "01",
		Bin:         "01",
		IsActive:    true,
	}

	lot := StockLot{
		LotID:             "lot-cheese-1",
		SupplierID:        supplierID,
		WarehouseID:       warehouseID,
		ProductID:         productID,
		LocationID:        loc.LocationID,
		LotCode:           "LOT-CHEESE",
		ExpiryDate:        now.Add(5 * 24 * time.Hour),
		QuantityOnHand:    10,
		QuantityAllocated: 0,
		Status:            "AVAILABLE",
	}

	lotsByProduct := map[string][]StockLotLocation{
		productID: {
			{Lot: lot, Location: loc},
		},
	}

	// 1. Partial not allowed -> should return error
	reqStrict := WavePickRequest{
		WaveID:       "wave-strict",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: false,
		Items: []WavePickItem{
			{OrderID: "o1", LineID: "l1", ProductID: productID, Quantity: 25},
		},
	}
	_, errStrict := AllocateWaveFEFOPure(now, reqStrict, lotsByProduct)
	if errStrict == nil {
		t.Fatal("expected error when AllowPartial=false and stock is insufficient")
	}

	// 2. Partial allowed -> should allocate available 10 and record shortfall of 15
	reqPartial := WavePickRequest{
		WaveID:       "wave-partial",
		WarehouseID:  warehouseID,
		SupplierID:   supplierID,
		AllowPartial: true,
		Items: []WavePickItem{
			{OrderID: "o1", LineID: "l1", ProductID: productID, Quantity: 25},
		},
	}
	resPartial, errPartial := AllocateWaveFEFOPure(now, reqPartial, lotsByProduct)
	if errPartial != nil {
		t.Fatalf("unexpected error when AllowPartial=true: %v", errPartial)
	}

	if resPartial.TotalUnits != 10 {
		t.Errorf("expected total units 10, got %d", resPartial.TotalUnits)
	}
	if len(resPartial.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(resPartial.Instructions))
	}
	if resPartial.Instructions[0].Quantity != 10 {
		t.Errorf("expected instruction quantity 10, got %d", resPartial.Instructions[0].Quantity)
	}

	if len(resPartial.Shortfalls) != 1 {
		t.Fatalf("expected 1 shortfall record, got %d", len(resPartial.Shortfalls))
	}
	sf := resPartial.Shortfalls[0]
	if sf.ProductID != productID {
		t.Errorf("expected shortfall product %s, got %s", productID, sf.ProductID)
	}
	if sf.Requested != 25 {
		t.Errorf("expected requested 25, got %d", sf.Requested)
	}
	if sf.Allocated != 10 {
		t.Errorf("expected allocated 10, got %d", sf.Allocated)
	}
	if sf.Shortfall != 15 {
		t.Errorf("expected shortfall 15, got %d", sf.Shortfall)
	}
}

func TestFEFO_ValidationEdgeCases(t *testing.T) {
	now := time.Now()

	// Missing warehouse ID
	_, err := AllocateWaveFEFOPure(now, WavePickRequest{}, nil)
	if err == nil {
		t.Error("expected error for empty warehouse ID")
	}

	// Empty items
	res, err := AllocateWaveFEFOPure(now, WavePickRequest{WarehouseID: "wh-1", Items: nil}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Instructions) != 0 || res.TotalUnits != 0 {
		t.Errorf("expected empty result for empty items, got %+v", res)
	}
}
