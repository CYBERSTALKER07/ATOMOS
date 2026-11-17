package stocklots

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// StockLotSnapshot represents post-mutation lot state for in-transaction rollups
// before Spanner BufferWrite mutations are visible to SQL queries.
type StockLotSnapshot struct {
	SupplierID       string
	WarehouseID      string
	ProductID        string
	QuantityOnHand   int64
	QuantityReserved int64
	Status           string
}

// RollupInventoryV2InTxn recomputes SupplierInventoryV2 QoH/Reserved from AVAILABLE StockLots.
func RollupInventoryV2InTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, productID string) error {
	return RollupInventoryV2WithLotOverridesInTxn(ctx, txn, supplierID, warehouseID, productID, nil)
}

// RollupInventoryV2WithLotOverridesInTxn recomputes SupplierInventoryV2 QoH/Reserved
// from StockLots while overlaying any in-transaction buffered StockLot mutations.
func RollupInventoryV2WithLotOverridesInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, productID string,
	overrides map[string]StockLotSnapshot,
) error {
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	productID = strings.TrimSpace(productID)
	if supplierID == "" || warehouseID == "" || productID == "" {
		return fmt.Errorf("rollup: supplier_id, warehouse_id, product_id required")
	}

	var rows []StockLotSnapshot
	seen := make(map[string]bool, len(overrides))
	if len(overrides) == 0 {
		stmt := spanner.Statement{
			SQL: `SELECT LotId, QuantityOnHand, QuantityReserved, Status
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
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("rollup query: %w", err)
			}
			var lid, st string
			var lqoh, lqr int64
			if err := row.Columns(&lid, &lqoh, &lqr, &st); err != nil {
				return err
			}
			seen[lid] = true
			rows = append(rows, StockLotSnapshot{
				SupplierID: supplierID, WarehouseID: warehouseID, ProductID: productID,
				QuantityOnHand: lqoh, QuantityReserved: lqr, Status: st,
			})
		}
	} else {
		stmt := spanner.Statement{
			SQL: `SELECT LotId, QuantityOnHand, QuantityReserved, Status
			      FROM StockLots
			      WHERE SupplierId = @sid AND WarehouseId = @wid AND ProductId = @pid`,
			Params: map[string]any{
				"sid": supplierID,
				"wid": warehouseID,
				"pid": productID,
			},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("rollup query: %w", err)
			}
			var lid, st string
			var lqoh, lqr int64
			if err := row.Columns(&lid, &lqoh, &lqr, &st); err != nil {
				return err
			}
			seen[lid] = true
			if ov, ok := overrides[lid]; ok {
				rows = append(rows, ov)
				continue
			}
			rows = append(rows, StockLotSnapshot{
				SupplierID: supplierID, WarehouseID: warehouseID, ProductID: productID,
				QuantityOnHand: lqoh, QuantityReserved: lqr, Status: st,
			})
		}
		for lid, ov := range overrides {
			if !seen[lid] {
				rows = append(rows, ov)
			}
		}
	}

	qoh, qr := sumAvailableLotSnapshots(rows, supplierID, warehouseID, productID)
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
		if colErr := existing.Columns(&thresh, &policy, &h3); colErr != nil {
			return colErr
		}
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

func sumAvailableLotSnapshots(rows []StockLotSnapshot, supplierID, warehouseID, productID string) (qoh, qr int64) {
	for _, r := range rows {
		if r.SupplierID != "" && r.SupplierID != supplierID {
			continue
		}
		if r.WarehouseID != "" && r.WarehouseID != warehouseID {
			continue
		}
		if r.ProductID != "" && r.ProductID != productID {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(r.Status), "AVAILABLE") {
			continue
		}
		qoh += r.QuantityOnHand
		qr += r.QuantityReserved
	}
	return qoh, qr
}
