package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

// ReleaseReservationsInTxn decrements QuantityReserved for each line item within an existing RW transaction.
func ReleaseReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string, source OrderSource, lineItems []LineItem) error {
	if source == OrderSourceBackorder || strings.TrimSpace(warehouseID) == "" || len(lineItems) == 0 {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	for _, item := range lineItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		row, err := txn.ReadRow(ctx, "SupplierInventoryV2",
			spanner.Key{supplierID, warehouseID, sku},
			[]string{"QuantityReserved"})
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				continue
			}
			return fmt.Errorf("read reserved qty %s: %w", sku, err)
		}
		var qr int64
		if err := row.Columns(&qr); err != nil {
			return err
		}
		next := qr - item.Quantity
		if next < 0 {
			next = 0
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        sku,
			"QuantityReserved": next,
			"UpdatedAt":        spanner.CommitTimestamp,
		})}); err != nil {
			return err
		}
	}
	return nil
}