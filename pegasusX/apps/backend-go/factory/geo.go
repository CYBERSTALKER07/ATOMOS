package factory

import (
	"context"
	"errors"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func stampFactoryCoords(ctx context.Context, lat, lng float64, requestedCountry string) (proximity.NodeGeography, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return proximity.NodeGeography{}, err
	}
	return proximity.StampNodeGeography(pack, lat, lng, requestedCountry)
}

func stampFactoryEntity(ctx context.Context, f *Factory) error {
	if f == nil {
		return auth.ErrGeographyIncomplete
	}
	if f.Lat != nil && f.Lng != nil {
		geo, err := stampFactoryCoords(ctx, *f.Lat, *f.Lng, f.CountryCode)
		if err != nil {
			return err
		}
		f.CountryCode = geo.CountryCode
		cell := geo.H3Cell
		f.H3Cell = &cell
		return nil
	}
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return err
	}
	country, err := proximity.ResolveNodeCountry(pack, f.CountryCode)
	if err != nil {
		return err
	}
	f.CountryCode = country
	return nil
}

func writeMarketLaw(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrCrossMarketDeferred) || errors.Is(err, auth.ErrGeographyIncomplete) ||
		errors.Is(err, auth.ErrMarketPackUnknown) || errors.Is(err, auth.ErrMarketPackNotShipped) {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return true
	}
	return false
}
