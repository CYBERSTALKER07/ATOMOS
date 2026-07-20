package order

import (
	"context"
	"log/slog"
	"testing"
)

func TestUnifiedCheckout_CreatesSingleSupplierOrder(t *testing.T) {
	t.Parallel()

	repo := &testRepo{}
	warehouse := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		Warehouse:    warehouse,
		SupplierID:   "sup-1",
		SupplierName: "Test Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	// Scaffold path (no Spanner): client unit prices still accepted for unit tests.
	resp, err := svc.UnifiedCheckout(context.Background(), "ret-1", UnifiedCheckoutRequest{
		Latitude:  41.31,
		Longitude: 69.24,
		Items: []UnifiedCheckoutLineItem{{
			SkuID:     "sku-1",
			Quantity:  2,
			UnitPrice: 15000,
		}},
	})
	if err != nil {
		t.Fatalf("UnifiedCheckout() err = %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.Status)
	}
	if resp.InvoiceID == "" {
		t.Fatal("invoice_id empty")
	}
	if resp.Total != 30000 {
		t.Fatalf("total = %d, want 30000", resp.Total)
	}
	if len(resp.SupplierOrders) != 1 {
		t.Fatalf("supplier_orders len = %d, want 1", len(resp.SupplierOrders))
	}
	so := resp.SupplierOrders[0]
	if so.OrderID == "" || so.SupplierID != "sup-1" || so.Total != 30000 || so.ItemCount != 1 {
		t.Fatalf("unexpected supplier order: %+v", so)
	}
	if repo.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", repo.createCalls)
	}
}

func TestAuthoritativeCheckoutLines_RejectsBadItems(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1", Currency: "UZS", Log: slog.Default()})
	_, err := svc.authoritativeCheckoutLines(context.Background(), "ret-1", []UnifiedCheckoutLineItem{{
		SkuID: "", Quantity: 1, UnitPrice: 100,
	}})
	if err == nil {
		t.Fatal("expected error for empty sku")
	}
}

func TestCheckoutSnapshot_ForbidsForeignRetailer(t *testing.T) {
	t.Parallel()

	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			RetailerID: "ret-owner",
			TotalMinor: 5000,
			Currency:   "UZS",
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Currency: "UZS"})

	_, _, err := svc.CheckoutSnapshot(context.Background(), "ord-1", "ret-other")
	if err == nil || err != ErrOrderForbidden {
		t.Fatalf("CheckoutSnapshot() err = %v, want %v", err, ErrOrderForbidden)
	}
}
