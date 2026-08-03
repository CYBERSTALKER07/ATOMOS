package order

import (
	"context"
	"encoding/json"
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

// ReleaseReservationsFromOrderFields unmarshals LineItemsJson and releases reserved stock in-txn.
// Used by shop-closed cancel paths that write CANCELLED via raw UpdateMap (not UpdateOrder).
func ReleaseReservationsFromOrderFields(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderSource string,
	lineItemsRaw []byte,
) error {
	var lineItems []LineItem
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &lineItems); err != nil {
			return fmt.Errorf("parse line items for release: %w", err)
		}
	}
	return ReleaseReservationsInTxn(ctx, txn, supplierID, warehouseID, OrderSource(orderSource), lineItems)
}

// releaseOrderReservationsInTxn releases stock for a loaded Order aggregate.
func releaseOrderReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error {
	if o == nil {
		return nil
	}
	return ReleaseReservationsInTxn(ctx, txn, o.SupplierID, o.WarehouseID, o.Source, o.LineItems)
}