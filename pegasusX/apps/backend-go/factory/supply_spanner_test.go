package factory

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

func extractMutationTableAndPK(m *spanner.Mutation) (string, string) {
	v := reflect.ValueOf(m).Elem()
	tableField := v.FieldByName("table")
	table := tableField.String()

	colsField := v.FieldByName("columns")
	valsField := v.FieldByName("values")

	var colMap = make(map[string]string)
	for i := 0; i < colsField.Len(); i++ {
		c := colsField.Index(i).String()
		val := fmt.Sprintf("%v", valsField.Index(i))
		colMap[c] = val
	}

	switch table {
	case "WarehouseSupplyRequests":
		return table, colMap["RequestId"]
	case "WarehouseSupplyRequestItems":
		return table, fmt.Sprintf("%s#%s", colMap["RequestId"], colMap["ItemId"])
	case "FactoryInternalTransfers":
		return table, colMap["TransferId"]
	case "OutboxEvents":
		return table, colMap["EventId"]
	default:
		return table, fmt.Sprintf("%v", valsField.Index(0))
	}
}

func TestBuildFulfillSupplyRequestMutations_NoDuplicatesForInternalAndExternal(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	rec := spannerSupplyRow{
		RequestID:    "req-101",
		WarehouseID:  "wh-01",
		SupplierID:   "sup-01",
		FactoryID:    "fac-01",
		TransferMode: supplier.TransferModeInternal,
		ProjectedVU:  10,
	}
	lines := []factorySupplyItem{
		{ItemID: "item-1", ProductID: "prod-1", ShippedQuantity: 5},
		{ItemID: "item-2", ProductID: "prod-2", ShippedQuantity: 15},
	}

	// 1. Internal transfer (initialTransferState == "RECEIVED")
	mutsInternal, err := buildFulfillSupplyRequestMutations(
		"req-101", "tr-internal-01", "fac-01", supplier.TransferModeInternal, "RECEIVED",
		"", 20.0, rec, lines, now,
	)
	if err != nil {
		t.Fatalf("build mutations for INTERNAL failed: %v", err)
	}

	seenInternal := make(map[string]int)
	for _, m := range mutsInternal {
		table, pk := extractMutationTableAndPK(m)
		key := table + ":" + pk
		seenInternal[key]++
		if seenInternal[key] > 1 {
			t.Errorf("duplicate mutation buffered for key: %s (table=%s, pk=%s)", key, table, pk)
		}
	}

	// Verify exact count: 1 WarehouseSupplyRequests, 2 WarehouseSupplyRequestItems, 1 FactoryInternalTransfers, 2 OutboxEvents = 6
	if len(mutsInternal) != 6 {
		t.Errorf("expected 6 mutations for INTERNAL with 2 lines, got %d", len(mutsInternal))
	}

	// 2. External transfer (initialTransferState == "IN_TRANSIT")
	mutsExternal, err := buildFulfillSupplyRequestMutations(
		"req-101", "tr-external-02", "fac-01", supplier.TransferModeTruck, "IN_TRANSIT",
		"drv-99", 20.0, rec, lines, now,
	)
	if err != nil {
		t.Fatalf("build mutations for EXTERNAL failed: %v", err)
	}

	seenExternal := make(map[string]int)
	for _, m := range mutsExternal {
		table, pk := extractMutationTableAndPK(m)
		key := table + ":" + pk
		seenExternal[key]++
		if seenExternal[key] > 1 {
			t.Errorf("duplicate mutation buffered for key: %s (table=%s, pk=%s)", key, table, pk)
		}
	}

	// Verify exact count: 1 WarehouseSupplyRequests, 2 WarehouseSupplyRequestItems, 1 FactoryInternalTransfers, 1 OutboxEvent = 5
	if len(mutsExternal) != 5 {
		t.Errorf("expected 5 mutations for EXTERNAL with 2 lines, got %d", len(mutsExternal))
	}
}
