package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ReplenishmentBulkProductID is the synthetic SKU credited when factory transfers arrive.
const ReplenishmentBulkProductID = "replenishment-bulk-vu"

// CreditSupplierInventoryV2InTxn increments on-hand stock for a warehouse SKU inside an active txn.
func CreditSupplierInventoryV2InTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, productID string, qty int64) error {
	if qty <= 0 {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	productID = strings.TrimSpace(productID)
	if supplierID == "" || warehouseID == "" || productID == "" {
		return fmt.Errorf("inventory credit: supplier_id, warehouse_id, and product_id required")
	}
	row, err := txn.ReadRow(ctx, "SupplierInventoryV2", spanner.Key{supplierID, warehouseID, productID},
		[]string{"QuantityOnHand", "QuantityReserved"})
	if err != nil {
		if spanner.ErrCode(err) != 5 {
			return fmt.Errorf("inventory credit read: %w", err)
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        productID,
			"QuantityOnHand":   qty,
			"QuantityReserved": int64(0),
			"ReorderThreshold": int64(0),
			"UpdatedAt":        spanner.CommitTimestamp,
		})})
	}
	var qoh, qr int64
	if err := row.Columns(&qoh, &qr); err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("SupplierInventoryV2", map[string]any{
		"SupplierId":     supplierID,
		"WarehouseId":    warehouseID,
		"ProductId":      productID,
		"QuantityOnHand": qoh + qty,
		"UpdatedAt":      spanner.CommitTimestamp,
	})})
}

// CreditBulkVUInTxn increments warehouse bulk replenishment stock inside an active txn.
func CreditBulkVUInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, supplierID string, units int64) error {
	if units <= 0 {
		return nil
	}
	warehouseID = strings.TrimSpace(warehouseID)
	supplierID = strings.TrimSpace(supplierID)
	if warehouseID == "" || supplierID == "" {
		return fmt.Errorf("inventory credit: warehouse_id and supplier_id required")
	}

	stmt := spanner.Statement{
		SQL: `SELECT InventoryId, QuantityOnHand, Version
		      FROM InventoryLevels@{FORCE_INDEX=Idx_InventoryLevels_ByWarehouseProduct}
		      WHERE WarehouseId = @wid AND ProductId = @pid
		      LIMIT 1`,
		Params: map[string]any{
			"wid": warehouseID,
			"pid": ReplenishmentBulkProductID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		invID := deterministicInventoryID(warehouseID, ReplenishmentBulkProductID)
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("InventoryLevels", map[string]any{
			"InventoryId":      invID,
			"ProductId":        ReplenishmentBulkProductID,
			"WarehouseId":      warehouseID,
			"SupplierId":       supplierID,
			"QuantityOnHand":   units,
			"QuantityReserved": int64(0),
			"ReorderThreshold": int64(0),
			"Version":          int64(1),
			"UpdatedAt":        spanner.CommitTimestamp,
		})})
	}
	if err != nil {
		return fmt.Errorf("inventory credit lookup: %w", err)
	}

	var invID string
	var onHand, version int64
	if err := row.Columns(&invID, &onHand, &version); err != nil {
		return fmt.Errorf("inventory credit scan: %w", err)
	}
	return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("InventoryLevels", map[string]any{
		"InventoryId":    invID,
		"QuantityOnHand": onHand + units,
		"Version":        version + 1,
		"UpdatedAt":      spanner.CommitTimestamp,
	})})
}

func deterministicInventoryID(warehouseID, productID string) string {
	seed := fmt.Sprintf("inventory|%s|%s", strings.TrimSpace(warehouseID), strings.TrimSpace(productID))
	sum := sha256.Sum256([]byte(seed))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
}
