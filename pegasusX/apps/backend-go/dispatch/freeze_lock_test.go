package dispatch

import "testing"

func TestIsWarehouseDispatchFrozen_BlocksWarehouseScope(t *testing.T) {
	locks := map[string]FreezeLock{
		"l1": {EntityType: "WAREHOUSE", EntityID: "wh-1"},
	}
	frozen, reason := IsWarehouseDispatchFrozen(locks, "wh-1")
	if !frozen || reason != "warehouse_dispatch_locked" {
		t.Fatalf("frozen=%v reason=%q", frozen, reason)
	}
}

func TestFilterFreezeLockedOrders_RemovesBlockedOrders(t *testing.T) {
	locks := map[string]FreezeLock{
		"l1": {EntityType: "ORDER", EntityID: "o-blocked"},
	}
	rows := []DispatchableOrder{
		{OrderID: "o-blocked"},
		{OrderID: "o-open"},
	}
	filtered := FilterFreezeLockedOrders(locks, rows)
	if len(filtered) != 1 || filtered[0].OrderID != "o-open" {
		t.Fatalf("filtered=%v", filtered)
	}
}

func TestFilterFreezeLockedDriverIDs_RemovesBlockedDrivers(t *testing.T) {
	locks := map[string]FreezeLock{
		"l1": {EntityType: "DRIVER", EntityID: "d-blocked"},
	}
	filtered := FilterFreezeLockedDriverIDs(locks, []string{"d-blocked", "d-open"})
	if len(filtered) != 1 || filtered[0] != "d-open" {
		t.Fatalf("filtered=%v", filtered)
	}
}
