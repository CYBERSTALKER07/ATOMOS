package auth

import (
	"context"
	"errors"
	"net/http"
)

var ErrBreachRadiusInvalid = errors.New("breach_radius_invalid")

// PackBreachRadiusMeters returns the shipped pack's one doorstep radius (GS-M3).
// Planned packs and unset/zero radius fail closed — never 500 m.
func PackBreachRadiusMeters(pack MarketPack) (float64, error) {
	if pack.Status != MarketPackShipped {
		return 0, ErrMarketPackNotShipped
	}
	if pack.BreachRadiusMeters <= 0 {
		return 0, ErrBreachRadiusInvalid
	}
	return pack.BreachRadiusMeters, nil
}

// BreachRadiusFromContext resolves one breach radius: claims → supplier profile → env shipped pack.
func BreachRadiusFromContext(ctx context.Context, supplierID string) (float64, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return 0, err
	}
	return PackBreachRadiusMeters(pack)
}

// BreachPackHTTPStatus maps M3 sentinels to status + error code.
func BreachPackHTTPStatus(err error) (int, string) {
	if errors.Is(err, ErrBreachRadiusInvalid) {
		return http.StatusUnprocessableEntity, err.Error()
	}
	return FiscalPackHTTPStatus(err)
}
