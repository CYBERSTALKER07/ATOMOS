package order

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestValidateFulfillmentForManifestTxn_Mismatch(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	supplierID := "sup-manifest-" + suffix
	wh1 := "wh-1-" + suffix
	wh2 := "wh-2-" + suffix
	orderID := "ord-manifest-" + suffix
	now := time.Now()

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId":      wh1,
			"SupplierId":       supplierID,
			"Name":             wh1,
			"CoverageRadiusKm": 10.0,
			"IsActive":         true,
			"IsOnShift":        true,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         "ret-1",
			"WarehouseId":        wh1,
			"Status":             string(StatusPending),
			"OrderSource":        string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson":      "[]",
			"TotalMinor":         100,
			"Currency":           "UZS",
			"Version":            1,
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
		spanner.InsertMap("OrderLineAllocations", map[string]any{
			"OrderId":     orderID,
			"OrderLineId": "sku-1:0",
			"WarehouseId": wh1,
			"Sku":         "sku-1",
			"Qty":         1,
			"CreatedAt":   spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderStockReservationMarkers", map[string]any{
			"OrderId":    orderID,
			"ReservedAt": spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
	defer client.Apply(ctx, []*spanner.Mutation{
		spanner.Delete("OrderStockReservationMarkers", spanner.Key{orderID}),
		spanner.Delete("OrderLineAllocations", spanner.Key{orderID, "sku-1:0", wh1}),
		spanner.Delete("Orders", spanner.Key{orderID}),
		spanner.Delete("Warehouses", spanner.Key{wh1}),
	})

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return ValidateFulfillmentForManifestTxn(ctx, txn, orderID, wh2)
	})
	if err == nil {
		t.Fatal("expected warehouse mismatch error")
	}
}
