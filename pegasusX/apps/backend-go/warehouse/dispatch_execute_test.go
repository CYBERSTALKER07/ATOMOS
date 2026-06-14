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

func TestManualCapacityWarnings_FlagsOverload(t *testing.T) {
	warnings := manualCapacityWarnings([]dispatch.DispatchRoute{{
		DriverID:     "drv-1",
		MaxVolume:    100,
		LoadedVolume: 96,
		Orders: []dispatch.GeoOrder{
			{OrderID: "ord-heavy", Volume: 60},
			{OrderID: "ord-light", Volume: 36},
		},
	}})
	if len(warnings) != 1 {
		t.Fatalf("warnings = %#v want 1 overload", warnings)
	}
	if warnings[0].EffectiveMaxVU != 95 {
		t.Fatalf("effective max = %v want 95", warnings[0].EffectiveMaxVU)
	}
	if len(warnings[0].SuggestedUnselectOrderIDs) == 0 {
		t.Fatalf("expected suggested unselect ids, got %#v", warnings[0])
	}
}

func TestManualCapacityWarnings_AllowsWithinBuffer(t *testing.T) {
	warnings := manualCapacityWarnings([]dispatch.DispatchRoute{{
		DriverID:     "drv-1",
		MaxVolume:    100,
		LoadedVolume: 90,
	}})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v want none", warnings)
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
