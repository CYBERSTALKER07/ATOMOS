package dispatch

import "testing"

func TestWindowsCompatible_MorningAndEveningDoNotMerge(t *testing.T) {
	morning := DispatchableOrder{ReceivingWindowOpen: "08:00", ReceivingWindowClose: "11:00"}
	evening := DispatchableOrder{ReceivingWindowOpen: "17:00", ReceivingWindowClose: "20:00"}
	if windowsCompatible(morning, evening) {
		t.Fatal("expected morning and evening windows to be incompatible")
	}
}

func TestWindowsCompatible_OverlappingWindowsMerge(t *testing.T) {
	a := DispatchableOrder{ReceivingWindowOpen: "09:00", ReceivingWindowClose: "12:00"}
	b := DispatchableOrder{ReceivingWindowOpen: "10:00", ReceivingWindowClose: "13:00"}
	if !windowsCompatible(a, b) {
		t.Fatal("expected overlapping windows to be compatible")
	}
}

func TestRouteWindowsCompatible_RejectsMismatch(t *testing.T) {
	route := []GeoOrder{{OrderID: "A", ReceivingWindowOpen: "08:00", ReceivingWindowClose: "10:00"}}
	candidate := DispatchableOrder{OrderID: "B", ReceivingWindowOpen: "18:00", ReceivingWindowClose: "20:00"}
	if routeWindowsCompatible(route, candidate) {
		t.Fatal("expected route to reject incompatible evening delivery")
	}
}

func TestBinPack_SeparatesMorningAndEveningInSameCell(t *testing.T) {
	orders := []DispatchableOrder{
		{OrderID: "M1", Lat: 41.3, Lng: 69.2, VolumeVU: 10, ReceivingWindowOpen: "08:00", ReceivingWindowClose: "11:00"},
		{OrderID: "E1", Lat: 41.3, Lng: 69.2, VolumeVU: 10, ReceivingWindowOpen: "17:00", ReceivingWindowClose: "20:00"},
	}
	fleet := []AvailableDriver{{DriverID: "D1", MaxVolumeVU: 100}}
	lookup := func(lat, lng float64) string { return "cell-1" }

	result := BinPack(orders, fleet, lookup)
	if len(result.Routes) != 2 {
		t.Fatalf("expected 2 routes for incompatible windows, got %d", len(result.Routes))
	}
}
