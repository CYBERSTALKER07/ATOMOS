package order

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

// ReleaseReservationsInTxn decrements QuantityReserved for each line item within an existing RW transaction.
func ReleaseReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string, source OrderSource, lineItems []LineItem) error {
	return ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, "", source, lineItems)
}

// ReleaseReservationsForOrderInTxn releases bag or lot reservations for an order.
func ReleaseReservationsForOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, orderID string, source OrderSource, lineItems []LineItem) error {
	if source == OrderSourceBackorder || strings.TrimSpace(warehouseID) == "" {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	orderID = strings.TrimSpace(orderID)

	if stocklots.LotsEnabled() && orderID != "" {
		return stocklots.ReleaseLotReservationsInTxn(ctx, txn, supplierID, warehouseID, orderID)
	}
	if len(lineItems) == 0 {
		return nil
	}
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
	return ReleaseReservationsFromOrderFieldsWithID(ctx, txn, supplierID, warehouseID, "", orderSource, lineItemsRaw)
}

// ReleaseReservationsFromOrderFieldsWithID releases with an explicit order ID for lot paths.
func ReleaseReservationsFromOrderFieldsWithID(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderID, orderSource string,
	lineItemsRaw []byte,
) error {
	var lineItems []LineItem
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &lineItems); err != nil {
			return fmt.Errorf("parse line items for release: %w", err)
		}
	}
	return ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, orderID, OrderSource(orderSource), lineItems)
}

// releaseOrderReservationsInTxn releases stock for a loaded Order aggregate.
func releaseOrderReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error {
	if o == nil {
		return nil
	}
	return ReleaseReservationsForOrderInTxn(ctx, txn, o.SupplierID, o.WarehouseID, o.OrderID, o.Source, o.LineItems)
}