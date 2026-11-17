package migration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func newTestSpannerApplyTx(supplierID string, now time.Time) *spannerApplyTx {
	return &spannerApplyTx{
		supplierID:      supplierID,
		sessionID:       "sess-test-chunk",
		now:             now,
		seq:             0,
		currency:        "UZS",
		identities:      make(map[string]string),
		checkedIDs:      make(map[string]bool),
		warehouses:      make(map[string]StoredWarehouse),
		whByName:        make(map[string]string),
		plantMaps:       make(map[string]string),
		products:        make(map[string]StoredProduct),
		prodByBarcode:   make(map[string]string),
		prodByName:      make(map[string]string),
		checkedProdIDs:  make(map[string]bool),
		checkedProdBC:   make(map[string]bool),
		checkedProdName: make(map[string]bool),
		customers:       make(map[string]StoredCustomer),
		custByINN:       make(map[string]string),
		custByPhone:     make(map[string]string),
		custByLegalName: make(map[string]string),
		checkedCustIDs:  make(map[string]bool),
		checkedCustINNs: make(map[string]bool),
		checkedCustPh:   make(map[string]bool),
		checkedCustName: make(map[string]bool),
		stocks:          make(map[string]StoredStock),
		checkedStocks:   make(map[string]bool),
		lotsByID:        make(map[string]StoredLot),
		lotsByKey:       make(map[string]string),
		checkedLotPairs: make(map[string]bool),
		checkedLotKeys:  make(map[string]bool),
		lotOverrides:    make(map[string]map[string]stocklots.StockLotSnapshot),
		locations:       make(map[string]bool),
		snapRecorded:    make(map[string]bool),
		insertedWH:      make(map[string]bool),
		insertedProd:    make(map[string]bool),
		insertedLot:     make(map[string]bool),
		insertedLink:    make(map[string]bool),
		whMuts:          make(map[string]*spanner.Mutation),
		plantMuts:       make(map[string]*spanner.Mutation),
		prodMuts:        make(map[string]*spanner.Mutation),
		retMuts:         make(map[string]*spanner.Mutation),
		linkMuts:        make(map[string]*spanner.Mutation),
		stockMuts:       make(map[string]*spanner.Mutation),
		lotMuts:         make(map[string]*spanner.Mutation),
		orderMuts:       make(map[string]*spanner.Mutation),
		orderCols:       make(map[string]map[string]any),
		idMuts:          make(map[string]*spanner.Mutation),
		rollupPairs:     make(map[string]whProdPair),
		ordersByID:      make(map[string]order.Order),
		ordersByExt:     make(map[string]string),
		checkedOrderExt: make(map[string]bool),
		arCache:         ar.NewMigrationTxnCache(),
	}
}

func TestImportHistoricalOrder_MultiLineMergeInSameChunkPreservesColumns(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	tx := newTestSpannerApplyTx("sup-test", now)

	// Line 1 for external order EXT-1001
	inserted1, updated1, orderID1, err := tx.ImportHistoricalOrder(ctx, order.HistoricalOrderRequest{
		SupplierID:      "sup-test",
		RetailerID:      "ret-test-42",
		ExternalOrderNo: "EXT-1001",
		RawStatus:       "DELIVERED",
		Currency:        "UZS",
		TotalMinor:      2000,
		LineItems: []order.LineItem{
			{SKU: "SKU-A", Quantity: 2, UnitPrice: 1000},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("import line 1 failed: %v", err)
	}
	if !inserted1 || updated1 {
		t.Fatalf("expected inserted=true, updated=false; got inserted=%v, updated=%v", inserted1, updated1)
	}

	// Line 2 for same external order in the same chunk
	inserted2, updated2, orderID2, err := tx.ImportHistoricalOrder(ctx, order.HistoricalOrderRequest{
		SupplierID:      "sup-test",
		RetailerID:      "ret-test-42",
		ExternalOrderNo: "EXT-1001",
		RawStatus:       "DELIVERED",
		Currency:        "UZS",
		TotalMinor:      1500,
		LineItems: []order.LineItem{
			{SKU: "SKU-B", Quantity: 3, UnitPrice: 500},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("import line 2 failed: %v", err)
	}
	if inserted2 || !updated2 {
		t.Fatalf("expected inserted=false, updated=true; got inserted=%v, updated=%v", inserted2, updated2)
	}
	if orderID1 != orderID2 {
		t.Fatalf("expected same orderID, got %s vs %s", orderID1, orderID2)
	}

	// Verify orderCols has preserved ALL columns from the first line plus updated values
	cols, exists := tx.orderCols[orderID1]
	if !exists {
		t.Fatalf("orderCols missing for orderID %s", orderID1)
	}

	// Check preserved columns
	if cols["SupplierId"] != "sup-test" {
		t.Errorf("SupplierId wiped out: %v", cols["SupplierId"])
	}
	if cols["RetailerId"] != "ret-test-42" {
		t.Errorf("RetailerId wiped out: %v", cols["RetailerId"])
	}
	if cols["Status"] != string(order.StatusCompleted) {
		t.Errorf("Status wiped out: %v", cols["Status"])
	}
	if cols["Currency"] != "UZS" {
		t.Errorf("Currency wiped out: %v", cols["Currency"])
	}
	if cols["OrderSource"] != string(order.OrderSourceMigration) {
		t.Errorf("OrderSource wiped out: %v", cols["OrderSource"])
	}
	if cols["ExternalOrderNo"] != "EXT-1001" {
		t.Errorf("ExternalOrderNo wiped out: %v", cols["ExternalOrderNo"])
	}

	// Check summed totals (2000 + 3*500 = 3500)
	expectedTotal := int64(3500)
	if cols["TotalMinor"] != expectedTotal {
		t.Errorf("expected TotalMinor=%d in cols, got %v", expectedTotal, cols["TotalMinor"])
	}
	if cols["OriginalTotalMinor"] != expectedTotal {
		t.Errorf("expected OriginalTotalMinor=%d in cols, got %v", expectedTotal, cols["OriginalTotalMinor"])
	}

	// Check ordersByID in memory
	stored := tx.ordersByID[orderID1]
	if stored.TotalMinor != expectedTotal {
		t.Errorf("expected stored TotalMinor=%d, got %d", expectedTotal, stored.TotalMinor)
	}
	if len(stored.LineItems) != 2 {
		t.Fatalf("expected 2 line items in stored order, got %d", len(stored.LineItems))
	}

	// Check LineItemsJson
	var lines []order.LineItem
	if raw, ok := cols["LineItemsJson"].([]byte); ok {
		if err := json.Unmarshal(raw, &lines); err != nil {
			t.Fatalf("unmarshal LineItemsJson failed: %v", err)
		}
	} else {
		t.Fatalf("LineItemsJson not []byte in cols: %T", cols["LineItemsJson"])
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in LineItemsJson, got %d", len(lines))
	}
	if lines[0].SKU != "SKU-A" || lines[1].SKU != "SKU-B" {
		t.Errorf("unexpected lines: %+v", lines)
	}
}

func TestImportMigratedOpenOrder_MultiLineMergeInSameChunkPreservesColumns(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	tx := newTestSpannerApplyTx("sup-test", now)

	// Line 1 for external open order EXT-OPEN-999
	inserted1, updated1, orderID1, err := tx.ImportMigratedOpenOrder(ctx, order.MigratedOpenOrderRequest{
		SupplierID:      "sup-test",
		RetailerID:      "ret-test-88",
		ExternalOrderNo: "EXT-OPEN-999",
		Currency:        "UZS",
		TotalMinor:      10000,
		LineItems: []order.LineItem{
			{SKU: "SKU-X", Quantity: 4, UnitPrice: 2500},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("import open line 1 failed: %v", err)
	}
	if !inserted1 || updated1 {
		t.Fatalf("expected inserted=true, updated=false; got inserted=%v, updated=%v", inserted1, updated1)
	}

	// Line 2 for same external open order in the same chunk
	inserted2, updated2, orderID2, err := tx.ImportMigratedOpenOrder(ctx, order.MigratedOpenOrderRequest{
		SupplierID:      "sup-test",
		RetailerID:      "ret-test-88",
		ExternalOrderNo: "EXT-OPEN-999",
		Currency:        "UZS",
		TotalMinor:      3000,
		LineItems: []order.LineItem{
			{SKU: "SKU-Y", Quantity: 1, UnitPrice: 3000},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("import open line 2 failed: %v", err)
	}
	if inserted2 || !updated2 {
		t.Fatalf("expected inserted=false, updated=true; got inserted=%v, updated=%v", inserted2, updated2)
	}
	if orderID1 != orderID2 {
		t.Fatalf("expected same orderID, got %s vs %s", orderID1, orderID2)
	}

	// Verify orderCols has preserved ALL columns from the first line plus updated values
	cols, exists := tx.orderCols[orderID1]
	if !exists {
		t.Fatalf("orderCols missing for orderID %s", orderID1)
	}

	if cols["SupplierId"] != "sup-test" {
		t.Errorf("SupplierId wiped out: %v", cols["SupplierId"])
	}
	if cols["RetailerId"] != "ret-test-88" {
		t.Errorf("RetailerId wiped out: %v", cols["RetailerId"])
	}
	if cols["Status"] != string(order.StatusMigrationOpen) {
		t.Errorf("Status wiped out: %v", cols["Status"])
	}
	if cols["Currency"] != "UZS" {
		t.Errorf("Currency wiped out: %v", cols["Currency"])
	}
	if cols["OrderSource"] != string(order.OrderSourceMigration) {
		t.Errorf("OrderSource wiped out: %v", cols["OrderSource"])
	}

	expectedTotal := int64(13000)
	if cols["TotalMinor"] != expectedTotal {
		t.Errorf("expected TotalMinor=%d in cols, got %v", expectedTotal, cols["TotalMinor"])
	}
	if cols["OriginalTotalMinor"] != expectedTotal {
		t.Errorf("expected OriginalTotalMinor=%d in cols, got %v", expectedTotal, cols["OriginalTotalMinor"])
	}

	stored := tx.ordersByID[orderID1]
	if stored.TotalMinor != expectedTotal {
		t.Errorf("expected stored TotalMinor=%d, got %d", expectedTotal, stored.TotalMinor)
	}
	if len(stored.LineItems) != 2 {
		t.Fatalf("expected 2 line items in stored order, got %d", len(stored.LineItems))
	}
}
