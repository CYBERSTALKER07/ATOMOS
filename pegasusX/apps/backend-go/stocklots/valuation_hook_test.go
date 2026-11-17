package stocklots

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func TestValuationHooks_InvokedInTxnOnPickAndAdjustment(t *testing.T) {
	ctx := context.Background()
	client := openStocklotsSpannerTestClient(t)

	suffix := uuid.NewString()[:8]
	supID := "sup-vhk-" + suffix
	whID := "wh-vhk-" + suffix
	prodID := "prod-vhk-" + suffix
	lotID := "lot-vhk-" + suffix
	adjID := "adj-vhk-" + suffix
	waveID := "wave-vhk-" + suffix
	taskID := "task-vhk-" + suffix
	manID := "man-vhk-" + suffix
	ordID := "ord-vhk-" + suffix
	now := time.Now().UTC()

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("PickTasks", spanner.KeyRange{Start: spanner.Key{waveID}, End: spanner.Key{waveID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("PickWaves", spanner.Key{waveID}),
			spanner.Delete("InventoryAdjustments", spanner.Key{adjID}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
		})
		_, _ = client.PartitionedUpdate(cleanCtx, spanner.Statement{
			SQL:    `DELETE FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supID},
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": supID, "WarehouseId": whID, "ProductId": prodID,
			"LocationId": "bin-1", "LotCode": "LOT-VHK",
			"QuantityOnHand": int64(20), "QuantityReserved": int64(5), "Status": "AVAILABLE",
			"ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("InventoryAdjustments", map[string]any{
			"AdjustmentId": adjID, "WarehouseId": whID, "ProductId": prodID, "LotId": lotID,
			"DeltaQty": int64(-3), "ReasonCode": "DAMAGE", "Status": "PENDING",
			"ActorId": "counter-1", "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("PickWaves", map[string]any{
			"WaveId": waveID, "WarehouseId": whID, "SupplierId": supID, "ManifestId": manID,
			"Strategy": "MANIFEST", "Status": "RELEASED", "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("PickTasks", map[string]any{
			"WaveId": waveID, "TaskId": taskID, "OrderId": ordID, "ProductId": prodID,
			"LotId": lotID, "LocationId": "bin-1", "QuantityRequested": int64(5),
			"QuantityPicked": int64(0), "Status": "PENDING", "PickSequence": int64(1),
			"CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed stocklots hook test fixtures: %v", err)
	}

	var capturedPick []PickValuationArgs
	var capturedAdj []AdjustmentValuationArgs
	SetPickValuationHook(func(_ context.Context, _ *spanner.ReadWriteTransaction, args PickValuationArgs) error {
		capturedPick = append(capturedPick, args)
		return nil
	})
	SetAdjustmentValuationHook(func(_ context.Context, _ *spanner.ReadWriteTransaction, args AdjustmentValuationArgs) error {
		capturedAdj = append(capturedAdj, args)
		return nil
	})
	t.Cleanup(func() {
		SetPickValuationHook(nil)
		SetAdjustmentValuationHook(nil)
	})

	if _, err := client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if _, err := ConfirmPickTaskInTxn(txCtx, txn, waveID, taskID, "picker-1", 5); err != nil {
			return err
		}
		if _, err := ApproveAdjustmentInTxn(txCtx, txn, adjID, "approver-1"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("hooked RW transaction failed: %v", err)
	}

	if len(capturedPick) != 1 || capturedPick[0].EntityID != supID || capturedPick[0].Quantity != 5 || capturedPick[0].OrderID != ordID {
		t.Fatalf("capturedPick = %+v, want 1 pick arg for entity=%s qty=5 order=%s", capturedPick, supID, ordID)
	}
	if len(capturedAdj) != 1 || capturedAdj[0].EntityID != supID || capturedAdj[0].DeltaQty != -3 || capturedAdj[0].ApprovedBy != "approver-1" {
		t.Fatalf("capturedAdj = %+v, want 1 adj arg for entity=%s delta=-3", capturedAdj, supID)
	}
}
