package retailer

import (
	"context"
	"errors"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func stampRetailerCoords(ctx context.Context, lat, lng float64, requestedCountry string) (proximity.NodeGeography, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return proximity.NodeGeography{}, err
	}
	return proximity.StampNodeGeography(pack, lat, lng, requestedCountry)
}

func resolveRetailerCountry(ctx context.Context, requestedCountry string) (string, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return "", err
	}
	return proximity.ResolveNodeCountry(pack, requestedCountry)
}

func stampRetailerOptionalCoords(ctx context.Context, lat, lng float64, requestedCountry string) (country, h3 string, err error) {
	if lat == 0 && lng == 0 {
		country, err = resolveRetailerCountry(ctx, requestedCountry)
		return country, "", err
	}
	geo, err := stampRetailerCoords(ctx, lat, lng, requestedCountry)
	if err != nil {
		return "", "", err
	}
	return geo.CountryCode, geo.H3Cell, nil
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
