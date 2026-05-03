package supplier

import (
	"fmt"
	"time"

	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

const inventoryAuditReasonReturnToStock = "RETURN_TO_STOCK"

type returnRestockAuditEntry struct {
	ProductID   string
	SupplierID  string
	AdjustedBy  string
	PreviousQty int64
	Delta       int64
}

func newReturnRestockAuditEntry(productID, supplierID, adjustedBy string, previousQty, delta int64) returnRestockAuditEntry {
	if adjustedBy == "" {
		adjustedBy = supplierID
	}
	return returnRestockAuditEntry{
		ProductID:   productID,
		SupplierID:  supplierID,
		AdjustedBy:  adjustedBy,
		PreviousQty: previousQty,
		Delta:       delta,
	}
}

func (e returnRestockAuditEntry) NewQty() int64 {
	return e.PreviousQty + e.Delta
}

func (e returnRestockAuditEntry) Mutation() *spanner.Mutation {
	return spanner.Insert("InventoryAuditLog",
		[]string{"AuditId", "ProductId", "SupplierId", "AdjustedBy", "PreviousQty", "NewQty", "Delta", "Reason", "AdjustedAt"},
		[]interface{}{
			fmt.Sprintf("AUD-%s", uuid.New().String()[:8]),
			e.ProductID,
			e.SupplierID,
			e.AdjustedBy,
			e.PreviousQty,
			e.NewQty(),
			e.Delta,
			inventoryAuditReasonReturnToStock,
			spanner.CommitTimestamp,
		},
	)
}

func buildReturnResolvedEvent(lineItemID, orderID, skuID string, quantity int64, resolution, supplierID, notes string, timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"type":         ws.EventReturnResolved,
		"line_item_id": lineItemID,
		"order_id":     orderID,
		"sku_id":       skuID,
		"quantity":     quantity,
		"resolution":   resolution,
		"supplier_id":  supplierID,
		"notes":        notes,
		"timestamp":    timestamp.UnixMilli(),
	}
}

func reconcileReturnOutcome(action string) (lineItemStatus string, eventResolution string, restock bool) {
	switch action {
	case "RESTOCK":
		return "RETURNED_TO_STOCK", inventoryAuditReasonReturnToStock, true
	case "WRITE_OFF_DAMAGED":
		return "WRITE_OFF", "WRITE_OFF", false
	default:
		return "", "", false
	}
}
