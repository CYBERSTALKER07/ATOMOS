package proximity

import (
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func tashkentStore() StorePoint {
	return StorePoint{CountryCode: "UZ", Lat: 41.3111, Lng: 69.2797, H3Cell: MatchingH3Cell(41.3111, 69.2797)}
}

func uzWarehouses() []WarehouseCandidate {
	return []WarehouseCandidate{
		{WarehouseID: "wh-a", CountryCode: "UZ", Lat: 41.2000, Lng: 69.1000, IsActive: true, IsOnShift: true},
		{WarehouseID: "wh-b", CountryCode: "UZ", Lat: 41.3120, Lng: 69.2800, IsActive: true, IsOnShift: true},
	}
}

func TestResolveServingWarehouse_ClosestWhenCoverageEmpty(t *testing.T) {
	t.Parallel()
	id, err := ResolveServingWarehouse("UZ", tashkentStore(), uzWarehouses(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != "wh-b" {
		t.Fatalf("got %s want wh-b", id)
	}
}

func TestResolveServingWarehouse_CoverageCityBeatsCloser(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	aCells := []string{store.H3Cell}
	bCells := []string{MatchingH3Cell(40.0, 68.0)}
	cands := []WarehouseCandidate{
		{WarehouseID: "wh-a", CountryCode: "UZ", Lat: 41.2000, Lng: 69.1000, CoverageCells: aCells, IsActive: true, IsOnShift: true},
		{WarehouseID: "wh-b", CountryCode: "UZ", Lat: 41.3120, Lng: 69.2800, CoverageCells: bCells, IsActive: true, IsOnShift: true},
	}
	id, err := ResolveServingWarehouse("UZ", store, cands, nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != "wh-a" {
		t.Fatalf("got %s want wh-a", id)
	}
}

func TestResolveServingWarehouse_PKStoreOnUZPack(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	store.CountryCode = "PK"
	_, err := ResolveServingWarehouse("UZ", store, uzWarehouses(), nil)
	if !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveServingWarehouse_EmptyCountryIncomplete(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	store.CountryCode = ""
	_, err := ResolveServingWarehouse("UZ", store, uzWarehouses(), nil)
	if !errors.Is(err, auth.ErrGeographyIncomplete) {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveServingWarehouse_RequiresOnShift(t *testing.T) {
	t.Parallel()
	cands := uzWarehouses()
	cands[1].IsOnShift = false
	id, err := ResolveServingWarehouse("UZ", tashkentStore(), cands, nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != "wh-a" {
		t.Fatalf("got %s want wh-a", id)
	}
}

func TestMergeStorePin_BodyCannotChangeCountry(t *testing.T) {
	t.Parallel()
	base := StorePoint{CountryCode: "UZ", Lat: 41.3, Lng: 69.2}
	overlay := StorePoint{LocationID: "loc-a", CountryCode: "UZ", Lat: 41.31, Lng: 69.28}
	got := MergeStorePin(base, overlay, 40.0, 68.0)
	if got.CountryCode != "UZ" {
		t.Fatalf("country=%q", got.CountryCode)
	}
	if got.Lat != 40.0 || got.Lng != 68.0 {
		t.Fatalf("coords=%v,%v", got.Lat, got.Lng)
	}
	if got.LocationID != "loc-a" {
		t.Fatalf("location=%q", got.LocationID)
	}
}

func TestResolveSupplyFactory_ClosestWithoutPrimary(t *testing.T) {
	t.Parallel()
	factories := []FactoryCandidate{
		{FactoryID: "fac-far", CountryCode: "UZ", Lat: 40.0, Lng: 68.0, IsActive: true},
		{FactoryID: "fac-near", CountryCode: "UZ", Lat: 41.31, Lng: 69.28, IsActive: true},
	}
	id, err := ResolveSupplyFactory("UZ", 41.3111, 69.2797, "", nil, factories)
	if err != nil {
		t.Fatal(err)
	}
	if id != "fac-near" {
		t.Fatalf("got %s", id)
	}
}

func TestResolveSupplyFactory_LanePriorityBeatsCloser(t *testing.T) {
	t.Parallel()
	factories := []FactoryCandidate{
		{FactoryID: "fac-far", CountryCode: "UZ", Lat: 40.0, Lng: 68.0, IsActive: true},
		{FactoryID: "fac-near", CountryCode: "UZ", Lat: 41.31, Lng: 69.28, IsActive: true},
	}
	lanes := []SupplyLane{{FactoryID: "fac-far", Priority: 10, IsActive: true}}
	id, err := ResolveSupplyFactory("UZ", 41.3111, 69.2797, "", lanes, factories)
	if err != nil {
		t.Fatal(err)
	}
	if id != "fac-far" {
		t.Fatalf("got %s", id)
	}
}

func TestResolveSupplyFactory_PrimaryWins(t *testing.T) {
	t.Parallel()
	factories := []FactoryCandidate{
		{FactoryID: "fac-far", CountryCode: "UZ", Lat: 40.0, Lng: 68.0, IsActive: true},
		{FactoryID: "fac-near", CountryCode: "UZ", Lat: 41.31, Lng: 69.28, IsActive: true},
	}
	id, err := ResolveSupplyFactory("UZ", 41.3111, 69.2797, "fac-far", nil, factories)
	if err != nil || id != "fac-far" {
		t.Fatalf("id=%s err=%v", id, err)
	}
}

func TestPerimeterCells_ExplicitOnly(t *testing.T) {
	t.Parallel()
	got := PerimeterCells([]WarehouseCandidate{
		{WarehouseID: "a", CoverageCells: []string{"cell-1"}, IsActive: true, IsOnShift: true},
		{WarehouseID: "b", CoverageCells: nil, IsActive: true, IsOnShift: true},
	})
	if len(got) != 1 || got[0] != "cell-1" {
		t.Fatalf("%#v", got)
	}
}

func TestResolveServingWarehouse_LocationPinBeatsCloser(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	store.LocationID = "loc-s"
	store.RetailerID = "ret-1"
	pins := []ServicePin{{WarehouseID: "wh-a", TargetType: PinTargetLocation, TargetID: "loc-s", Priority: 1}}
	id, err := ResolveServingWarehouse("UZ", store, uzWarehouses(), pins)
	if err != nil {
		t.Fatal(err)
	}
	if id != "wh-a" {
		t.Fatalf("got %s want wh-a", id)
	}
}

func TestResolveServingWarehouse_UnpinReturnsClosest(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	store.LocationID = "loc-s"
	id, err := ResolveServingWarehouse("UZ", store, uzWarehouses(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != "wh-b" {
		t.Fatalf("got %s want wh-b", id)
	}
}

func TestResolveServingWarehouse_RegionPinClosestAmongSet(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	store.RegionID = "reg-tashkent"
	pins := []ServicePin{
		{WarehouseID: "wh-a", TargetType: PinTargetRegion, TargetID: "reg-tashkent", Priority: 1},
	}
	id, err := ResolveServingWarehouse("UZ", store, uzWarehouses(), pins)
	if err != nil || id != "wh-a" {
		t.Fatalf("id=%s err=%v", id, err)
	}
}

func TestResolveServingWarehouse_OffShiftPinFallsThrough(t *testing.T) {
	t.Parallel()
	store := tashkentStore()
	store.LocationID = "loc-s"
	cands := uzWarehouses()
	cands[0].IsOnShift = false
	pins := []ServicePin{{WarehouseID: "wh-a", TargetType: PinTargetLocation, TargetID: "loc-s", Priority: 9}}
	id, err := ResolveServingWarehouse("UZ", store, cands, pins)
	if err != nil || id != "wh-b" {
		t.Fatalf("ineligible pin must fall through id=%s err=%v", id, err)
	}
}

func TestEffectiveCoverageMode(t *testing.T) {
	t.Parallel()
	wh := WarehouseCandidate{WarehouseID: "wh-a"}
	if got := EffectiveCoverageMode(wh, nil); got != CoverageModeCountryClosest {
		t.Fatalf("%s", got)
	}
	wh.CoverageCells = []string{"cell"}
	if got := EffectiveCoverageMode(wh, nil); got != CoverageModeCityCells {
		t.Fatalf("%s", got)
	}
	if got := EffectiveCoverageMode(wh, []ServicePin{{WarehouseID: "wh-a", TargetType: PinTargetRetailer, TargetID: "r"}}); got != CoverageModePinned {
		t.Fatalf("%s", got)
	}
}
