package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

// releaseOrderReservations returns reserved stock when an order is cancelled.
func (s *Service) releaseOrderReservations(ctx context.Context, o *Order) error {
	if s == nil || s.spannerClient == nil || o == nil {
		return nil
	}
	if o.Source == OrderSourceBackorder || o.WarehouseID == "" || len(o.LineItems) == 0 {
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, item := range o.LineItems {
			sku := strings.TrimSpace(item.SKU)
			if sku == "" || item.Quantity <= 0 {
				continue
			}
			row, err := txn.ReadRow(ctx, "SupplierInventoryV2",
				spanner.Key{o.SupplierID, o.WarehouseID, sku},
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
				"SupplierId":       o.SupplierID,
				"WarehouseId":      o.WarehouseID,
				"ProductId":        sku,
				"QuantityReserved": next,
				"UpdatedAt":        spanner.CommitTimestamp,
			})}); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil && s.cache != nil {
		s.cache.Invalidate(ctx, "catalog:products:"+o.SupplierID)
	}
	return err
}
