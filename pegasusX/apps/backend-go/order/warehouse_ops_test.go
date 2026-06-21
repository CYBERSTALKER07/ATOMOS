package order

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestWarehouseMarkDelayed_InvalidState(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:     "ord_1",
			SupplierID:  "sup_1",
			WarehouseID: "wh_1",
			RetailerID:  "ret_1",
			Status:      StatusCompleted,
			Currency:    "UZS",
		},
		found: true,
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup_1"})
	ops := &auth.WarehouseOps{WarehouseID: "wh_1", SupplierID: "sup_1"}
	err := svc.WarehouseMarkDelayed(context.Background(), ops, "ord_1", "OPS_HOLD", nil)
	if err == nil {
		t.Fatal("expected invalid transition error")
	}
}

func TestWarehouseMarkDelayed_SupplierAdminResolvesFromOrder(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:     "ord_admin",
			SupplierID:  "sup_1",
			WarehouseID: "wh_1",
			RetailerID:  "ret_1",
			Status:      StatusPending,
			Currency:    "UZS",
		},
		found: true,
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup_1"})
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "sup_1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup_1",
	})
	if err := svc.WarehouseMarkDelayed(ctx, nil, "ord_admin", "SSMR_DELAY", map[string]any{"proposed_delivery_date": "2026-07-01"}); err != nil {
		t.Fatalf("supplier admin delay: %v", err)
	}
	if repo.captured.Status != StatusDelayed {
		t.Fatalf("status=%s want DELAYED", repo.captured.Status)
	}
}

func TestWarehousePayloadOverflow_ClearsAssignment(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:     "ord_2",
			SupplierID:  "sup_1",
			WarehouseID: "wh_1",
			RetailerID:  "ret_1",
			Status:      StatusLoaded,
			DriverID:    "drv_1",
			ManifestID:  "mf_1",
			Currency:    "UZS",
		},
		found: true,
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup_1"})
	ops := &auth.WarehouseOps{WarehouseID: "wh_1", SupplierID: "sup_1"}
	if err := svc.WarehousePayloadOverflow(context.Background(), ops, "ord_2", "OVERFLOW"); err != nil {
		t.Fatalf("overflow: %v", err)
	}
	if repo.captured.Status != StatusPending {
		t.Fatalf("status=%s want PENDING", repo.captured.Status)
	}
	if repo.captured.DriverID != "" || repo.captured.ManifestID != "" {
		t.Fatalf("assignment not cleared: driver=%q manifest=%q", repo.captured.DriverID, repo.captured.ManifestID)
	}
}
