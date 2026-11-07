package factory

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/uber/h3-go/v4"
)

func TestStampFactoryEntity_DefaultsPackAndRes7(t *testing.T) {
	t.Parallel()
	lat, lng := 41.3111, 69.2797
	f := Factory{Lat: &lat, Lng: &lng}
	if err := stampFactoryEntity(context.Background(), &f); err != nil {
		t.Fatal(err)
	}
	if f.CountryCode != "UZ" {
		t.Fatalf("country=%q", f.CountryCode)
	}
	if f.H3Cell == nil {
		t.Fatal("h3 empty")
	}
	cell := h3.Cell(h3.IndexFromString(*f.H3Cell))
	if cell.Resolution() != 7 {
		t.Fatalf("res=%d", cell.Resolution())
	}
}

func TestStampFactoryEntity_RejectsForeignCountry(t *testing.T) {
	t.Parallel()
	lat, lng := 41.3111, 69.2797
	f := Factory{Lat: &lat, Lng: &lng, CountryCode: "US"}
	if err := stampFactoryEntity(context.Background(), &f); !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}
