package dispatch

import "testing"

func TestScoreCandidate_PriorityWins(t *testing.T) {
	driver := AvailableDriver{DriverID: "d1", MaxVolumeVU: 100, DriverScore: 50}
	route := &DispatchRoute{DriverID: "d1", MaxVolume: 95, LoadedVolume: 0}
	ctx := ScoreContext{DepotLat: 41.3, DepotLng: 69.2, CellLookup: H3CellLookup, NowMinutes: 10 * 60}

	low := DispatchableOrder{OrderID: "a", VolumeVU: 10, Lat: 41.31, Lng: 69.21, PriorityScore: 40}
	high := DispatchableOrder{OrderID: "b", VolumeVU: 10, Lat: 41.31, Lng: 69.21, PriorityScore: 180}

	if ScoreCandidate(route, high, driver, ctx) <= ScoreCandidate(route, low, driver, ctx) {
		t.Fatalf("higher PriorityScore should win")
	}
}

func TestScoreCandidate_DriverScoreWins(t *testing.T) {
	order := DispatchableOrder{OrderID: "o1", VolumeVU: 10, Lat: 41.31, Lng: 69.21, PriorityScore: 50}
	ctx := ScoreContext{DepotLat: 41.3, DepotLng: 69.2, CellLookup: H3CellLookup}
	weak := AvailableDriver{DriverID: "w", MaxVolumeVU: 100, DriverScore: 20}
	strong := AvailableDriver{DriverID: "s", MaxVolumeVU: 100, DriverScore: 95}
	if ScoreCandidate(nil, order, strong, ctx) <= ScoreCandidate(nil, order, weak, ctx) {
		t.Fatalf("higher DriverScore should win for new route")
	}
}

func TestBinPack_UsesPriorityForAssignment(t *testing.T) {
	fleet := []AvailableDriver{
		{DriverID: "good", MaxVolumeVU: 100, DriverScore: 90},
		{DriverID: "bad", MaxVolumeVU: 100, DriverScore: 10},
	}
	orders := []DispatchableOrder{
		{OrderID: "vip", RetailerID: "r1", VolumeVU: 20, Lat: 41.3, Lng: 69.2, PriorityScore: 180},
	}
	res := BinPack(orders, fleet, H3CellLookup, BinPackOptions{
		Score:           ScoreContext{DepotLat: 41.3, DepotLng: 69.2},
		SkipLocalSearch: true,
	})
	if len(res.Routes) != 1 {
		t.Fatalf("routes=%d", len(res.Routes))
	}
	if res.Routes[0].DriverID != "good" {
		t.Fatalf("expected good driver, got %s", res.Routes[0].DriverID)
	}
}
