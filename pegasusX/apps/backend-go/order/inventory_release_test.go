package order

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestReleaseReservationsInTxn_skipsWhenNoReleaseNeeded(t *testing.T) {
	cases := []struct {
		name        string
		source      OrderSource
		warehouseID string
		lineItems   []LineItem
	}{
		{
			name:        "backorder",
			source:      OrderSourceBackorder,
			warehouseID: "wh_1",
			lineItems:   []LineItem{{SKU: "sku_1", Quantity: 1}},
		},
		{
			name:        "empty warehouse",
			source:      OrderSourceManual,
			warehouseID: "",
			lineItems:   []LineItem{{SKU: "sku_1", Quantity: 1}},
		},
		{
			name:        "no line items",
			source:      OrderSourceManual,
			warehouseID: "wh_1",
			lineItems:   nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ReleaseReservationsInTxn(t.Context(), nil, "sup_1", tc.warehouseID, tc.source, tc.lineItems); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWarehouseRejectOrder_transitionsToCancelled(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:     "ord_reject",
			SupplierID:  "sup_1",
			WarehouseID: "wh_1",
			RetailerID:  "ret_1",
			Status:      StatusPending,
			Source:      OrderSourceManual,
			LineItems:   []LineItem{{SKU: "sku_1", Quantity: 2}},
			Currency:    "UZS",
		},
		found: true,
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup_1"})
	ops := &auth.WarehouseOps{WarehouseID: "wh_1", SupplierID: "sup_1"}
	if err := svc.WarehouseRejectOrder(context.Background(), ops, "ord_reject", "OUT_OF_STOCK"); err != nil {
		t.Fatalf("warehouse reject: %v", err)
	}
	if repo.captured.Status != StatusCancelled {
		t.Fatalf("status=%s want CANCELLED", repo.captured.Status)
	}
}
