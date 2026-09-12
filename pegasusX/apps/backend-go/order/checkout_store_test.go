package order

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestResolveCheckoutStore_BodyCoordsNoSpanner(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	store, err := svc.resolveCheckoutStore(context.Background(), "ret-1", 41.3, 69.2)
	if err != nil {
		t.Fatal(err)
	}
	if store.Lat != 41.3 || store.Lng != 69.2 {
		t.Fatalf("%+v", store)
	}
	if store.CountryCode != "" {
		t.Fatalf("empty country must stay empty for stubs: %+v", store)
	}
}

func TestResolveCheckoutStore_BodyCannotChangeCountry(t *testing.T) {
	t.Parallel()
	base := proximity.StorePoint{CountryCode: "UZ", Lat: 41.31, Lng: 69.28, LocationID: "loc-a"}
	got := proximity.MergeStorePin(base, proximity.StorePoint{}, 33.6, 73.0)
	if got.CountryCode != "UZ" {
		t.Fatalf("country=%q", got.CountryCode)
	}
	if got.Lat != 33.6 || got.Lng != 73.0 {
		t.Fatalf("coords=%v,%v", got.Lat, got.Lng)
	}
}

func TestResolveCheckoutStore_ActiveLocationFromClaimsWithoutSpanner(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	ctx := auth.WithClaims(context.Background(), auth.Claims{ActiveLocationID: "loc-a"})
	store, err := svc.resolveCheckoutStore(ctx, "ret-1", 41.3, 69.2)
	if err != nil {
		t.Fatal(err)
	}
	if store.LocationID != "" {
		t.Fatalf("no Spanner means overlay cannot load: %+v", store)
	}
}

func TestMapWarehouseResolveError(t *testing.T) {
	t.Parallel()
	if err := mapWarehouseResolveError(auth.ErrGeographyIncomplete); !errors.Is(err, auth.ErrGeographyIncomplete) {
		t.Fatalf("%v", err)
	}
	if err := mapWarehouseResolveError(proximity.ErrZoneMiss); !errors.Is(err, ErrZoneMiss) {
		t.Fatalf("%v", err)
	}
	if err := mapWarehouseResolveError(errors.New("resolver_down")); !errors.Is(err, ErrServiceabilityUnavailable) {
		t.Fatalf("%v", err)
	}
}
