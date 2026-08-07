package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// Default STAGE location IDs used when credit paths lack an explicit bin.
const (
	DefaultRecvLocationID    = "recv-default"
	DefaultReturnsLocationID = "returns-default"
)

// CreditViaDefaultPutawayInTxn credits quantity onto StockLots (then V2 rollup)
// when WMS lots are the source of truth. Callers must use this instead of direct
// SupplierInventoryV2 writes while LotsEnabled().
func CreditViaDefaultPutawayInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, productID, locationID, zone string,
	qty int64,
) (*PutawayResult, error) {
	if qty <= 0 {
		return nil, nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	productID = strings.TrimSpace(productID)
	locationID = strings.TrimSpace(locationID)
	zone = strings.TrimSpace(zone)
	if supplierID == "" || warehouseID == "" || productID == "" {
		return nil, fmt.Errorf("credit putaway: supplier_id, warehouse_id, and product_id required")
	}
	if locationID == "" {
		locationID = DefaultRecvLocationID
	}
	if zone == "" {
		zone = "RECV"
	}
	if _, err := UpsertBinInTxn(ctx, txn, CreateBinRequest{
		WarehouseID:  warehouseID,
		LocationID:   locationID,
		Zone:         zone,
		LocationType: "STAGE",
		PickSequence: 0,
	}); err != nil {
		return nil, err
	}
	lotCode := fmt.Sprintf("AUTO-%s-%s", productID, time.Now().UTC().Format("20060102"))
	return PutawayInTxn(ctx, txn, PutawayRequest{
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		ProductID:   productID,
		LocationID:  locationID,
		LotCode:     lotCode,
		Quantity:    qty,
	})
}
