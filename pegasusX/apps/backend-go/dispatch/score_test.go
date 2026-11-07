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

func TestScoreCandidate_ColdChainRefuse(t *testing.T) {
	order := DispatchableOrder{OrderID: "cold", VolumeVU: 10, Lat: 41.31, Lng: 69.21, RequiresColdChain: true}
	ctx := ScoreContext{DepotLat: 41.3, DepotLng: 69.2}
	warm := AvailableDriver{DriverID: "w", MaxVolumeVU: 100, HasRefrigeration: false}
	reefer := AvailableDriver{DriverID: "c", MaxVolumeVU: 100, HasRefrigeration: true}
	if ScoreCandidate(nil, order, warm, ctx) != -1 {
		t.Fatal("cold order on non-reefer must score -1")
	}
	if ScoreCandidate(nil, order, reefer, ctx) <= 0 {
		t.Fatal("cold order on reefer must score")
	}
}

func TestScoreCandidate_RoadMatrixPreferred(t *testing.T) {
	t.Setenv("DISPATCH_SCORE_USE_OSRM", "true")
	// Far by road but close by haversine? Use fixed RoadKm that makes near stop expensive.
	orderNear := DispatchableOrder{OrderID: "n", VolumeVU: 10, Lat: 41.31, Lng: 69.21, PriorityScore: 50}
	orderFar := DispatchableOrder{OrderID: "f", VolumeVU: 10, Lat: 41.35, Lng: 69.25, PriorityScore: 50}
	driver := AvailableDriver{DriverID: "d", MaxVolumeVU: 100, DriverScore: 50}
	ctx := ScoreContext{
		DepotLat: 41.3, DepotLng: 69.2, MatrixSource: MatrixSourceOSRM,
		RoadKm: func(fromLat, fromLng, toLat, toLng float64) (float64, bool) {
			// Pretend near is 20km road, far is 2km road → far should win empty-mile.
			dLat := toLat - fromLat
			if dLat < 0.02 {
				return 20.0, true
			}
			return 2.0, true
		},
	}
	if ResolveMatrixSource(ctx) != MatrixSourceOSRM {
		t.Fatalf("matrix source=%s", ResolveMatrixSource(ctx))
	}
	// Empty-mile from depot: orderNear → 20km, orderFar → 2km; far should score higher.
	if ScoreCandidate(nil, orderFar, driver, ctx) <= ScoreCandidate(nil, orderNear, driver, ctx) {
		t.Fatal("lower road empty-mile should score higher when OSRM matrix on")
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
