package dispatch

import (
	"testing"
)

func TestBinPack_BalancedRouting(t *testing.T) {
	fleet := []AvailableDriver{
		{DriverID: "d1", MaxVolumeVU: 100},
		{DriverID: "d2", MaxVolumeVU: 100},
		{DriverID: "d3", MaxVolumeVU: 100},
	}
	orders := []DispatchableOrder{
		{OrderID: "o1", RetailerID: "r1", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o2", RetailerID: "r2", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o3", RetailerID: "r3", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o4", RetailerID: "r4", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o5", RetailerID: "r5", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o6", RetailerID: "r6", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
	}

	result := BinPack(orders, fleet, H3CellLookup)
	if len(result.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(result.Routes))
	}

	for _, route := range result.Routes {
		if len(route.Orders) != 2 {
			t.Errorf("expected 2 orders per truck for balanced routing, got %d", len(route.Orders))
		}
	}
}

func TestBinPack_FallbackToSmallestTruckButPreserveRetailer(t *testing.T) {
	fleet := []AvailableDriver{
		{DriverID: "small", MaxVolumeVU: 30},
		{DriverID: "large", MaxVolumeVU: 100},
	}
	orders := []DispatchableOrder{
		{OrderID: "o1", RetailerID: "r1", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o2", RetailerID: "r2", VolumeVU: 10, Lat: 41.3, Lng: 69.2},
		{OrderID: "o3", RetailerID: "r3", VolumeVU: 20, Lat: 41.3, Lng: 69.2},
		{OrderID: "o4", RetailerID: "r3", VolumeVU: 15, Lat: 41.3, Lng: 69.2}, // total 35 > small truck (30*0.95 = 28.5)
	}

	result := BinPack(orders, fleet, H3CellLookup)
	
	for _, route := range result.Routes {
		if route.DriverID == "small" {
			if route.LoadedVolume > 30*TetrisBuffer {
				t.Errorf("Small truck overloaded: %.2f", route.LoadedVolume)
			}
			for _, o := range route.Orders {
				if o.RetailerID == "r3" {
					t.Errorf("r3 (volume 35) should not be placed in the small truck")
				}
			}
		}
		if route.DriverID == "large" {
			foundR3 := false
			for _, o := range route.Orders {
				if o.RetailerID == "r3" {
					foundR3 = true
				}
			}
			if !foundR3 {
				t.Errorf("r3 not placed in the large truck")
			}
		}
	}
}
