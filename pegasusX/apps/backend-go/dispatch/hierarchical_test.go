package dispatch

import (
	"testing"
)

func TestBinPackHierarchical_MergesCells(t *testing.T) {
	t.Setenv("DISPATCH_HIERARCHICAL_H3", "true")
	t.Setenv("DISPATCH_HIERARCHICAL_MIN_ORDERS", "4")
	fleet := []AvailableDriver{
		{DriverID: "d1", MaxVolumeVU: 200},
		{DriverID: "d2", MaxVolumeVU: 200},
	}
	orders := []DispatchableOrder{
		{OrderID: "o1", RetailerID: "r1", VolumeVU: 10, Lat: 41.30, Lng: 69.20},
		{OrderID: "o2", RetailerID: "r2", VolumeVU: 10, Lat: 41.31, Lng: 69.21},
		{OrderID: "o3", RetailerID: "r3", VolumeVU: 10, Lat: 41.50, Lng: 69.50},
		{OrderID: "o4", RetailerID: "r4", VolumeVU: 10, Lat: 41.51, Lng: 69.51},
	}
	res := BinPackHierarchical(orders, fleet, H3CellLookup, BinPackOptions{SkipLocalSearch: true})
	if res == nil || len(res.Routes) == 0 {
		t.Fatalf("expected hierarchical routes")
	}
	assigned := 0
	for _, r := range res.Routes {
		assigned += len(r.Orders)
	}
	if assigned+len(res.Orphans) != 4 {
		t.Fatalf("assigned=%d orphans=%d", assigned, len(res.Orphans))
	}
}
