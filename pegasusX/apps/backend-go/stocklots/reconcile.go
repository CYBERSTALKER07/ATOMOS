package stocklots

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// InventoryReconcileReport compares V2 vs lot sums.
type InventoryReconcileReport struct {
	WarehouseID string                  `json:"warehouse_id"`
	SupplierID  string                  `json:"supplier_id"`
	Matched     int                     `json:"matched"`
	Mismatches  []InventoryReconcileRow `json:"mismatches"`
}

// InventoryReconcileRow is one SKU drift.
type InventoryReconcileRow struct {
	ProductID    string `json:"product_id"`
	V2OnHand     int64  `json:"v2_on_hand"`
	LotsOnHand   int64  `json:"lots_on_hand"`
	V2Reserved   int64  `json:"v2_reserved"`
	LotsReserved int64  `json:"lots_reserved"`
}

// ReconcileInventoryV2 asserts SupplierInventoryV2 ≡ sum AVAILABLE lots.
func ReconcileInventoryV2(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (*InventoryReconcileReport, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT v.ProductId, v.QuantityOnHand, v.QuantityReserved,
		             COALESCE(l.qoh, 0), COALESCE(l.qr, 0)
		      FROM SupplierInventoryV2 v
		      LEFT JOIN (
		        SELECT ProductId, SUM(QuantityOnHand) AS qoh, SUM(QuantityReserved) AS qr
		        FROM StockLots
		        WHERE SupplierId = @sid AND WarehouseId = @wid AND Status = 'AVAILABLE'
		        GROUP BY ProductId
		      ) l ON l.ProductId = v.ProductId
		      WHERE v.SupplierId = @sid AND v.WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	rep := &InventoryReconcileReport{WarehouseID: warehouseID, SupplierID: supplierID, Mismatches: []InventoryReconcileRow{}}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid string
		var v2q, v2r, lq, lr int64
		if err := row.Columns(&pid, &v2q, &v2r, &lq, &lr); err != nil {
			return nil, err
		}
		if v2q == lq && v2r == lr {
			rep.Matched++
			continue
		}
		rep.Mismatches = append(rep.Mismatches, InventoryReconcileRow{
			ProductID: pid, V2OnHand: v2q, LotsOnHand: lq, V2Reserved: v2r, LotsReserved: lr,
		})
	}
	return rep, nil
}
