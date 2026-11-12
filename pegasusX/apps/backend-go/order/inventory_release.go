package order

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
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

	if len(lineItems) == 0 {
		return nil
	}
	// Aggregate quantities by SKU to avoid double-read/overwrite issues for duplicate SKUs in a single order
	skuQuantities := make(map[string]int64)
	for _, item := range lineItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		skuQuantities[sku] += item.Quantity
	}
	if len(skuQuantities) == 0 {
		return nil
	}

	skus := make([]string, 0, len(skuQuantities))
	for sku := range skuQuantities {
		skus = append(skus, sku)
	}
	sort.Strings(skus)
	keySets := make([]spanner.KeySet, 0, len(skus))
	for _, sku := range skus {
		keySets = append(keySets, spanner.Key{supplierID, warehouseID, sku})
	}

	reservedBySKU := make(map[string]int64, len(skus))
	invIter := txn.Read(ctx, "SupplierInventoryV2", spanner.KeySets(keySets...), []string{"ProductId", "QuantityReserved"})
	defer invIter.Stop()
	for {
		row, err := invIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("read reserved qty batch: %w", err)
		}
		var sku string
		var qr int64
		if err := row.Columns(&sku, &qr); err != nil {
			return err
		}
		reservedBySKU[sku] = qr
	}

	var muts []*spanner.Mutation
	for _, sku := range skus {
		qr, exists := reservedBySKU[sku]
		if !exists {
			continue
		}
		next := qr - skuQuantities[sku]
		if next < 0 {
			next = 0
		}
		muts = append(muts, spanner.UpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        sku,
			"QuantityReserved": next,
			"UpdatedAt":        spanner.CommitTimestamp,
		}))
	}
	if len(muts) > 0 {
		if err := txn.BufferWrite(muts); err != nil {
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
