import re

with open("apps/backend-go/factory/repository_spanner.go", "r") as f:
    content = f.read()

# Add imports for inventory, stocklots, and google.golang.org/api/iterator
if '"google.golang.org/api/iterator"' not in content:
    content = content.replace('"time"', '"time"\n\t"google.golang.org/api/iterator"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/inventory"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"\n\t"google.golang.org/grpc/codes"')

save_transfer_replacement = """func (tx *spannerFactoryTx) SaveTransfer(ctx context.Context, t TransferRow) error {
	mut := spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]interface{}{
		"TransferId":     t.TransferID,
		"FactoryId":      tx.factoryNode,
		"SupplierId":     tx.supplierID,
		"OrderId":        spanner.NullString{StringVal: t.OrderID, Valid: t.OrderID != ""},
		"ManifestId":     spanner.NullString{StringVal: t.ManifestID, Valid: t.ManifestID != ""},
		"State":          t.State,
		"TotalVolumeVU":  float64(t.TotalVU),
		"DriverId":       spanner.NullString{StringVal: t.DriverID, Valid: t.DriverID != ""},
		"VehicleId":      spanner.NullString{StringVal: t.VehicleID, Valid: t.VehicleID != ""},
		"ReassignDepth":  int64(t.ReassignDepth),
		"ExceptionCount": t.ExceptionCount,
		"CreatedAt":      parseTime(t.CreatedAt),
		"UpdatedAt":      parseTime(t.UpdatedAt),
	})
	if err := tx.txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
		return err
	}
	
	if t.State == "COMPLETED" || t.State == "RECEIVED" {
		return tx.autoReceiveTransfer(ctx, t.TransferID)
	}
	return nil
}

func (tx *spannerFactoryTx) autoReceiveTransfer(ctx context.Context, transferID string) error {
	row, err := tx.txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID}, []string{"SupplierId", "WarehouseId", "SupplyRequestId", "State"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil
		}
		return err
	}
	var supplierID, warehouseID, reqID, state spanner.NullString
	if err := row.Columns(&supplierID, &warehouseID, &reqID, &state); err != nil {
		return err
	}
	if !reqID.Valid || reqID.StringVal == "" {
		return nil
	}

	if err := tx.txn.BufferWrite([]*spanner.Mutation{
		spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
			"RequestId": reqID.StringVal,
			"State":     "RECEIVED",
			"UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		return err
	}

	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT ItemId, ProductId, COALESCE(ShippedQuantity, RequestedQuantity)
		      FROM WarehouseSupplyRequestItems WHERE RequestId = @rid`,
		Params: map[string]any{"rid": reqID.StringVal},
	})
	defer iter.Stop()

	for {
		irow, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var itemID, productID string
		var qty int64
		if err := irow.Columns(&itemID, &productID, &qty); err != nil {
			return err
		}
		if qty > 0 {
			if stocklots.LotsEnabled() {
				if _, err := stocklots.UpsertBinInTxn(ctx, tx.txn, stocklots.CreateBinRequest{
					WarehouseID: warehouseID.StringVal, LocationID: "recv-default", Zone: "RECV",
					LocationType: "STAGE", PickSequence: 0,
				}); err != nil {
					return err
				}
				if _, err := stocklots.PutawayInTxn(ctx, tx.txn, stocklots.PutawayRequest{
					SupplierID: supplierID.StringVal, WarehouseID: warehouseID.StringVal, ProductID: productID,
					LocationID: "recv-default", Quantity: qty,
				}); err != nil {
					return err
				}
			} else {
				if err := inventory.CreditSupplierInventoryV2InTxn(ctx, tx.txn, supplierID.StringVal, warehouseID.StringVal, productID, qty); err != nil {
					return err
				}
			}
			
			if err := tx.txn.BufferWrite([]*spanner.Mutation{
				spanner.UpdateMap("WarehouseSupplyRequestItems", map[string]any{
					"RequestId":        reqID.StringVal,
					"ItemId":           itemID,
					"ReceivedQuantity": qty,
				}),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}"""

pattern = re.compile(r'func \(tx \*spannerFactoryTx\) SaveTransfer.*?return tx\.txn\.BufferWrite\(\[\]\*spanner\.Mutation\{mut\}\)\n\}', re.DOTALL)
content = pattern.sub(save_transfer_replacement, content)

with open("apps/backend-go/factory/repository_spanner.go", "w") as f:
    f.write(content)

