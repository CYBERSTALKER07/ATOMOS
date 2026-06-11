package warehouse

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
)

func TestFilterLockedOrders_ExcludesLockedOrderIDs(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &warehouseRepoSpy{
			locks: map[string]DispatchLock{
				"l1": {EntityType: "ORDER", EntityID: "ord-locked"},
			},
		},
	})
	rows := []dispatch.DispatchableOrder{
		{OrderID: "ord-locked"},
		{OrderID: "ord-free"},
	}
	filtered := filterLockedOrders(context.Background(), svc, "wh-1", rows)
	if len(filtered) != 1 || filtered[0].OrderID != "ord-free" {
		t.Fatalf("filtered = %#v want only ord-free", filtered)
	}
}

func TestIsWarehouseDispatchLocked_BlocksWarehouseScope(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &warehouseRepoSpy{
			locks: map[string]DispatchLock{
				"l1": {EntityType: "WAREHOUSE", EntityID: "wh-1"},
			},
		},
	})
	locked, reason := svc.isWarehouseDispatchLocked(context.Background(), "wh-1")
	if !locked || reason != "warehouse_dispatch_locked" {
		t.Fatalf("locked=%v reason=%q", locked, reason)
	}
}

func TestFilterLockedDrivers_ExcludesLockedDriverIDs(t *testing.T) {
	locks := map[string]DispatchLock{
		"l1": {EntityType: "DRIVER", EntityID: "drv-locked"},
	}
	drivers := []PortalDriver{
		{DriverID: "drv-locked"},
		{DriverID: "drv-free"},
	}
	filtered := filterLockedDrivers("wh-1", drivers, locks)
	if len(filtered) != 1 || filtered[0].DriverID != "drv-free" {
		t.Fatalf("filtered = %#v want only drv-free", filtered)
	}
}
