package dispatch

import (
	"strings"
	"testing"
)

func TestBinPack_atomicOrderNotSplitWhenExceedsFleetCap(t *testing.T) {
	fleet := []AvailableDriver{{
		DriverID:    "d1",
		MaxVolumeVU: 10,
	}}
	orders := []DispatchableOrder{{
		OrderID:  "o1",
		VolumeVU: 25,
		Lat:      41.3,
		Lng:      69.2,
	}}

	result := BinPack(orders, fleet, H3CellLookup)
	if len(result.Splits) != 0 {
		t.Fatalf("expected no splits, got %d", len(result.Splits))
	}
	if len(result.Orphans) != 1 || result.Orphans[0].OrderID != "o1" {
		t.Fatalf("expected orphan o1, got %+v", result.Orphans)
	}
	if len(result.Routes) != 0 {
		t.Fatalf("expected no routes, got %d", len(result.Routes))
	}
}

func TestBinPack_wholeOrdersOnlyOnRoute(t *testing.T) {
	fleet := []AvailableDriver{{
		DriverID:    "d1",
		MaxVolumeVU: 10,
	}}
	orders := []DispatchableOrder{
		{OrderID: "o1", VolumeVU: 6, Lat: 41.31, Lng: 69.21},
		{OrderID: "o2", VolumeVU: 5, Lat: 41.31, Lng: 69.21},
	}

	result := BinPack(orders, fleet, H3CellLookup)
	for _, route := range result.Routes {
		if route.LoadedVolume > route.MaxVolume+1e-6 {
			t.Fatalf("route overloaded: loaded=%.2f max=%.2f", route.LoadedVolume, route.MaxVolume)
		}
		orderIDs := make(map[string]struct{})
		for _, stop := range route.Orders {
			if strings.Contains(stop.OrderID, "-CHUNK-") {
				t.Fatalf("unexpected split chunk id %s", stop.OrderID)
			}
			orderIDs[stop.OrderID] = struct{}{}
		}
		if len(orderIDs) != len(route.Orders) {
			t.Fatalf("duplicate stops on route: %+v", route.Orders)
		}
	}
}
