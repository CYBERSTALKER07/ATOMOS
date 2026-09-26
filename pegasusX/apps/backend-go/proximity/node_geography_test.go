package proximity

import (
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/uber/h3-go/v4"
)

func TestStampNodeGeography_DefaultsPackCountryAndRes7(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	geo, err := StampNodeGeography(pack, 41.3111, 69.2797, "")
	if err != nil {
		t.Fatal(err)
	}
	if geo.CountryCode != "UZ" {
		t.Fatalf("country=%q", geo.CountryCode)
	}
	cell := h3.Cell(h3.IndexFromString(geo.H3Cell))
	if !cell.IsValid() || cell.Resolution() != MatchingResolution {
		t.Fatalf("h3=%q res=%d", geo.H3Cell, cell.Resolution())
	}
}

func TestStampNodeGeography_RejectsForeignCountry(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	_, err := StampNodeGeography(pack, 41.3111, 69.2797, "US")
	if !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}

func TestMatchingH3Cell_NotRes9(t *testing.T) {
	t.Parallel()
	got := MatchingH3Cell(41.3111, 69.2797)
	legacy := H3CellFromLatLng(41.3111, 69.2797)
	if got == "" || got == legacy {
		t.Fatalf("res7=%q res9=%q", got, legacy)
	}
	cell := h3.Cell(h3.IndexFromString(got))
	if cell.Resolution() != 7 {
		t.Fatalf("res=%d", cell.Resolution())
	}
}
