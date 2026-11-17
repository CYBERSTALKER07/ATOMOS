package stocklots

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openStocklotsSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner stocklots integration test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner emulator connect (%s): %v", dbPath, err)
		}
		t.Skipf("spanner emulator unavailable (%s): %v", dbPath, err)
	}

	_, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "StockLots", spanner.Key{"__stocklots_probe__"}, []string{"LotId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("spanner StockLots unavailable (%s): %v", dbPath, probeErr)
		}
		t.Skipf("spanner StockLots table unavailable (%s): %v", dbPath, probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func TestReconcileAndRepairInventoryV2_OrphanStockLots_LiveSpanner(t *testing.T) {
	client := openStocklotsSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sid := "sup-" + uuid.NewString()[:8]
	wid := "wh-" + uuid.NewString()[:8]
	oid := "ord-" + uuid.NewString()[:8]
	rid := "ret-" + uuid.NewString()[:8]

	lotHealthy1 := "lot-" + uuid.NewString()[:8]
	lotHealthy2 := "lot-" + uuid.NewString()[:8]
	lotHealthyQuar := "lot-" + uuid.NewString()[:8]
	lotOrphan1 := "lot-" + uuid.NewString()[:8]
	lotOrphan2 := "lot-" + uuid.NewString()[:8]
	lotOrphanStranded := "lot-" + uuid.NewString()[:8]
	lotOrphanZero := "lot-" + uuid.NewString()[:8]
	lotQuarOnly := "lot-" + uuid.NewString()[:8]

	allLots := []string{
		lotHealthy1, lotHealthy2, lotHealthyQuar,
		lotOrphan1, lotOrphan2, lotOrphanStranded, lotOrphanZero, lotQuarOnly,
	}

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		var muts []*spanner.Mutation
		muts = append(muts,
			spanner.Delete("OrderLotReservations", spanner.Key{oid, lotOrphanStranded}),
			spanner.Delete("Orders", spanner.Key{oid}),
			spanner.Delete("SupplierInventoryV2", spanner.KeyRange{
				Start: spanner.Key{sid, wid},
				End:   spanner.Key{sid, wid},
				Kind:  spanner.ClosedClosed,
			}),
		)
		for _, lid := range allLots {
			muts = append(muts, spanner.Delete("StockLots", spanner.Key{lid}))
		}
		_, _ = client.Apply(cleanCtx, muts)
	})

	_, err := client.Apply(ctx, []*spanner.Mutation{
		// 1. Healthy product: SupplierInventoryV2 matches sum of AVAILABLE lots (60+40=100, 4+6=10);
		// QUARANTINED lot (25) must be ignored.
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       sid,
			"WarehouseId":      wid,
			"ProductId":        "prod-healthy",
			"QuantityOnHand":   int64(100),
			"QuantityReserved": int64(10),
			"ReorderThreshold": int64(5),
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotHealthy1, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-healthy",
			"LocationId": "loc-1", "QuantityOnHand": int64(60), "QuantityReserved": int64(4),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotHealthy2, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-healthy",
			"LocationId": "loc-2", "QuantityOnHand": int64(40), "QuantityReserved": int64(6),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotHealthyQuar, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-healthy",
			"LocationId": "loc-q", "QuantityOnHand": int64(25), "QuantityReserved": int64(0),
			"Status": "QUARANTINED", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),

		// 2. Orphan AVAILABLE StockLots (no SupplierInventoryV2 row): 35+15=50 QoH, 5+0=5 QR.
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotOrphan1, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-orphan",
			"LocationId": "loc-1", "QuantityOnHand": int64(35), "QuantityReserved": int64(5),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotOrphan2, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-orphan",
			"LocationId": "loc-2", "QuantityOnHand": int64(15), "QuantityReserved": int64(0),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),

		// 3. Orphan AVAILABLE StockLot with stranded reservation on a CANCELLED order.
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            oid,
			"SupplierId":         sid,
			"RetailerId":         rid,
			"WarehouseId":        wid,
			"Status":             "CANCELLED",
			"OrderSource":        "RETAILER",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(0),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotOrphanStranded, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-orphan-stranded",
			"LocationId": "loc-3", "QuantityOnHand": int64(40), "QuantityReserved": int64(12),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderLotReservations", map[string]any{
			"OrderId": oid, "LotId": lotOrphanStranded, "Quantity": int64(12), "CreatedAt": spanner.CommitTimestamp,
		}),

		// 4. Zero-quantity orphan AVAILABLE StockLot (still missing SupplierInventoryV2 row).
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotOrphanZero, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-orphan-zero",
			"LocationId": "loc-4", "QuantityOnHand": int64(0), "QuantityReserved": int64(0),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),

		// 5. Non-AVAILABLE orphan lot (QUARANTINED only) must NOT trigger a mismatch.
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotQuarOnly, "SupplierId": sid, "WarehouseId": wid, "ProductId": "prod-quar-only",
			"LocationId": "loc-q", "QuantityOnHand": int64(99), "QuantityReserved": int64(0),
			"Status": "QUARANTINED", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),

		// 6. Stale SupplierInventoryV2 row with zero AVAILABLE lots.
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       sid,
			"WarehouseId":      wid,
			"ProductId":        "prod-stale-v2",
			"QuantityOnHand":   int64(20),
			"QuantityReserved": int64(3),
			"ReorderThreshold": int64(2),
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed live stocklots fixtures: %v", err)
	}

	// Step 1: ReconcileInventoryV2 catches all orphan AVAILABLE StockLots + stale V2 drift.
	rep, err := ReconcileInventoryV2(ctx, client, sid, wid)
	if err != nil {
		t.Fatalf("ReconcileInventoryV2: %v", err)
	}
	if rep.Matched != 1 {
		t.Fatalf("Matched = %d, want 1 (prod-healthy)", rep.Matched)
	}
	if len(rep.Mismatches) != 4 {
		t.Fatalf("len(Mismatches) = %d, want 4; got %+v", len(rep.Mismatches), rep.Mismatches)
	}

	byProd := make(map[string]InventoryReconcileRow, len(rep.Mismatches))
	for _, m := range rep.Mismatches {
		byProd[m.ProductID] = m
	}

	wantMismatches := map[string]InventoryReconcileRow{
		"prod-orphan":          {ProductID: "prod-orphan", V2OnHand: 0, LotsOnHand: 50, V2Reserved: 0, LotsReserved: 5, MissingV2: true},
		"prod-orphan-stranded": {ProductID: "prod-orphan-stranded", V2OnHand: 0, LotsOnHand: 40, V2Reserved: 0, LotsReserved: 12, MissingV2: true},
		"prod-orphan-zero":     {ProductID: "prod-orphan-zero", V2OnHand: 0, LotsOnHand: 0, V2Reserved: 0, LotsReserved: 0, MissingV2: true},
		"prod-stale-v2":        {ProductID: "prod-stale-v2", V2OnHand: 20, LotsOnHand: 0, V2Reserved: 3, LotsReserved: 0, MissingV2: false},
	}
	for pid, want := range wantMismatches {
		got, ok := byProd[pid]
		if !ok {
			t.Fatalf("missing expected mismatch for %s; got %+v", pid, rep.Mismatches)
		}
		if got != want {
			t.Fatalf("mismatch[%s] = %+v, want %+v", pid, got, want)
		}
	}

	// Step 2: RepairInventoryV2 frees stranded terminal reservation, creates missing V2 rollups, and repairs stale V2.
	repairRes, err := RepairInventoryV2(ctx, client, sid, wid)
	if err != nil {
		t.Fatalf("RepairInventoryV2: %v", err)
	}
	if len(repairRes.Before.Mismatches) != 4 {
		t.Fatalf("Before mismatches = %d, want 4", len(repairRes.Before.Mismatches))
	}
	if repairRes.ReleasedOrderReservations != 1 {
		t.Fatalf("ReleasedOrderReservations = %d, want 1", repairRes.ReleasedOrderReservations)
	}
	if repairRes.LotsAdjusted != 1 {
		t.Fatalf("LotsAdjusted = %d, want 1", repairRes.LotsAdjusted)
	}
	if repairRes.ProductsRolledUp != 4 {
		t.Fatalf("ProductsRolledUp = %d, want 4", repairRes.ProductsRolledUp)
	}
	if len(repairRes.After.Mismatches) != 0 {
		t.Fatalf("After mismatches = %+v, want empty", repairRes.After.Mismatches)
	}
	if repairRes.After.Matched != 5 {
		t.Fatalf("After matched = %d, want 5", repairRes.After.Matched)
	}

	// Step 3: Assert direct Spanner state of SupplierInventoryV2, StockLots, and OrderLotReservations.
	wantV2 := map[string][2]int64{
		"prod-healthy":         {100, 10},
		"prod-orphan":          {50, 5},
		"prod-orphan-stranded": {40, 0},
		"prod-orphan-zero":     {0, 0},
		"prod-stale-v2":        {0, 0},
	}
	for pid, exp := range wantV2 {
		row, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(
			ctx, "SupplierInventoryV2", spanner.Key{sid, wid, pid}, []string{"QuantityOnHand", "QuantityReserved"},
		)
		if err != nil {
			t.Fatalf("read SupplierInventoryV2[%s]: %v", pid, err)
		}
		var qoh, qr int64
		if err := row.Columns(&qoh, &qr); err != nil {
			t.Fatalf("decode SupplierInventoryV2[%s]: %v", pid, err)
		}
		if qoh != exp[0] || qr != exp[1] {
			t.Fatalf("SupplierInventoryV2[%s] = (%d, %d), want (%d, %d)", pid, qoh, qr, exp[0], exp[1])
		}
	}

	lotRow, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(
		ctx, "StockLots", spanner.Key{lotOrphanStranded}, []string{"QuantityReserved"},
	)
	if err != nil {
		t.Fatalf("read stranded StockLot: %v", err)
	}
	var lotQR int64
	if err := lotRow.Columns(&lotQR); err != nil {
		t.Fatalf("decode stranded StockLot: %v", err)
	}
	if lotQR != 0 {
		t.Fatalf("stranded StockLot QuantityReserved = %d, want 0", lotQR)
	}

	_, resErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(
		ctx, "OrderLotReservations", spanner.Key{oid, lotOrphanStranded}, []string{"Quantity"},
	)
	if spanner.ErrCode(resErr) != codes.NotFound {
		t.Fatalf("expected OrderLotReservations row deleted, got err=%v", resErr)
	}

	// Step 4: Repeat ReconcileInventoryV2 independently to confirm clean state.
	finalRep, err := ReconcileInventoryV2(ctx, client, sid, wid)
	if err != nil {
		t.Fatalf("final ReconcileInventoryV2: %v", err)
	}
	if finalRep.Matched != 5 || len(finalRep.Mismatches) != 0 {
		t.Fatalf("final ReconcileInventoryV2 = %+v, want Matched=5 Mismatches=0", finalRep)
	}
}
