package stocklots

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// RollupInventoryV2InTxn recomputes SupplierInventoryV2 QoH/Reserved from AVAILABLE StockLots.
func RollupInventoryV2InTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, productID string) error {
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	productID = strings.TrimSpace(productID)
	if supplierID == "" || warehouseID == "" || productID == "" {
		return fmt.Errorf("rollup: supplier_id, warehouse_id, product_id required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(QuantityOnHand), 0), COALESCE(SUM(QuantityReserved), 0)
		      FROM StockLots
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND ProductId = @pid
		        AND Status = 'AVAILABLE'`,
		Params: map[string]any{
			"sid": supplierID,
			"wid": warehouseID,
			"pid": productID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
				"SupplierId":       supplierID,
				"WarehouseId":      warehouseID,
				"ProductId":        productID,
				"QuantityOnHand":   int64(0),
				"QuantityReserved": int64(0),
				"ReorderThreshold": int64(0),
				"UpdatedAt":        spanner.CommitTimestamp,
			})})
		}
		return fmt.Errorf("rollup query: %w", err)
	}
	var qoh, qr int64
	if err := row.Columns(&qoh, &qr); err != nil {
		return err
	}
	existing, err := txn.ReadRow(ctx, "SupplierInventoryV2",
		spanner.Key{supplierID, warehouseID, productID},
		[]string{"ReorderThreshold", "OutOfStockPolicy", "H3Cell"})
	if err != nil && spanner.ErrCode(err) != 5 {
		return err
	}
	update := map[string]any{
		"SupplierId":       supplierID,
		"WarehouseId":      warehouseID,
		"ProductId":        productID,
		"QuantityOnHand":   qoh,
		"QuantityReserved": qr,
		"UpdatedAt":        spanner.CommitTimestamp,
	}
	if err == nil {
		var thresh int64
		var policy, h3 spanner.NullString
		_ = existing.Columns(&thresh, &policy, &h3)
		update["ReorderThreshold"] = thresh
		if policy.Valid {
			update["OutOfStockPolicy"] = policy.StringVal
		}
		if h3.Valid {
			update["H3Cell"] = h3.StringVal
		}
	} else {
		update["ReorderThreshold"] = int64(0)
	}
	return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierInventoryV2", update)})
}
