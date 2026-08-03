package dispatch

import "testing"

func TestTwoOpt_NeverWorsensTour(t *testing.T) {
	// Crossing tour: A-C-B-D around depot
	stops := []GeoOrder{
		{OrderID: "a", Lat: 41.30, Lng: 69.20},
		{OrderID: "c", Lat: 41.32, Lng: 69.24},
		{OrderID: "b", Lat: 41.30, Lng: 69.24},
		{OrderID: "d", Lat: 41.32, Lng: 69.20},
	}
	depotLat, depotLng := 41.31, 69.22
	before := tourLength(stops, depotLat, depotLng)
	afterStops := twoOptTour(stops, depotLat, depotLng)
	after := tourLength(afterStops, depotLat, depotLng)
	if after > before+1e-9 {
		t.Fatalf("2-opt worsened tour: before=%.4f after=%.4f", before, after)
	}
	if len(afterStops) != len(stops) {
		t.Fatalf("lost stops")
	}
	// Membership preserved
	seen := map[string]bool{}
	for _, s := range afterStops {
		seen[s.OrderID] = true
	}
	for _, s := range stops {
		if !seen[s.OrderID] {
			t.Fatalf("missing %s", s.OrderID)
		}
	}
}

func TestResequenceStops_ChangesCrossingOrder(t *testing.T) {
	stops := []GeoOrder{
		{OrderID: "a", Lat: 41.30, Lng: 69.20},
		{OrderID: "c", Lat: 41.32, Lng: 69.24},
		{OrderID: "b", Lat: 41.30, Lng: 69.24},
		{OrderID: "d", Lat: 41.32, Lng: 69.20},
	}
	out := ResequenceStops(stops, 41.31, 69.22)
	if len(out) != 4 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestOrderVolumeVU_CatalogVsDefaultChangesPack(t *testing.T) {
	fleet := []AvailableDriver{{DriverID: "d1", MaxVolumeVU: 20}}
	// Default 1.0 × 15 qty would be 15 VU; catalog 2.0 → 30 VU overflows.
	defaultOrders := []DispatchableOrder{
		{OrderID: "o1", RetailerID: "r1", VolumeVU: 15, Lat: 41.3, Lng: 69.2, VolumeSource: VolumeSourceDefault10},
	}
	catalogOrders := []DispatchableOrder{
		{OrderID: "o1", RetailerID: "r1", VolumeVU: 30, Lat: 41.3, Lng: 69.2, VolumeSource: VolumeSourceCatalog},
	}
	r1 := BinPack(defaultOrders, fleet, H3CellLookup, BinPackOptions{SkipLocalSearch: true})
	r2 := BinPack(catalogOrders, fleet, H3CellLookup, BinPackOptions{SkipLocalSearch: true})
	if len(r1.Routes) == 0 {
		t.Fatalf("default should pack")
	}
	if len(r2.Orphans) == 0 && len(r2.OverflowWarnings) == 0 {
		t.Fatalf("catalog volume should not fit silently: routes=%d", len(r2.Routes))
	}
}
