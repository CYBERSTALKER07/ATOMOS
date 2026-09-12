package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	ErrTimezoneInvalid           = errors.New("timezone_invalid")
	ErrShopClosedGraceInvalid    = errors.New("shop_closed_grace_invalid")
	ErrWeatherScopeUnavailable   = errors.New("weather_scope_unavailable")
	ErrFactorySLAHoursInvalid    = errors.New("factory_sla_hours_invalid")
	ErrLaborMaxShiftHoursInvalid = errors.New("labor_max_shift_hours_invalid")
)

// PackTimezoneName returns the shipped pack IANA timezone (GS-M4).
func PackTimezoneName(pack MarketPack) (string, error) {
	if pack.Status != MarketPackShipped {
		return "", ErrMarketPackNotShipped
	}
	tz := strings.TrimSpace(pack.Timezone)
	if tz == "" {
		return "", ErrTimezoneInvalid
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "", ErrTimezoneInvalid
	}
	return tz, nil
}

// PackLocation loads the shipped pack timezone.
func PackLocation(pack MarketPack) (*time.Location, error) {
	name, err := PackTimezoneName(pack)
	if err != nil {
		return nil, err
	}
	return time.LoadLocation(name)
}

// TimezoneNameFromContext resolves IANA TZ: claims → supplier profile → env shipped pack.
func TimezoneNameFromContext(ctx context.Context, supplierID string) (string, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return "", err
	}
	return PackTimezoneName(pack)
}

// TimezoneFromContext resolves the shipped pack *time.Location.
func TimezoneFromContext(ctx context.Context, supplierID string) (*time.Location, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	return PackLocation(pack)
}

// PackShopClosedGrace returns the shipped pack grace window.
func PackShopClosedGrace(pack MarketPack) (time.Duration, error) {
	if pack.Status != MarketPackShipped {
		return 0, ErrMarketPackNotShipped
	}
	if pack.ShopClosedGraceMin <= 0 {
		return 0, ErrShopClosedGraceInvalid
	}
	return time.Duration(pack.ShopClosedGraceMin) * time.Minute, nil
}

// ShopClosedGraceFromContext resolves shop-closed grace from the shipped pack.
func ShopClosedGraceFromContext(ctx context.Context, supplierID string) (time.Duration, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return 0, err
	}
	return PackShopClosedGrace(pack)
}

// PackWeatherScope returns the shipped pack weather geo (empty on planned).
func PackWeatherScope(pack MarketPack) (string, error) {
	if pack.Status != MarketPackShipped {
		return "", ErrMarketPackNotShipped
	}
	scope := strings.TrimSpace(pack.WeatherScope)
	if scope == "" {
		return "", ErrWeatherScopeUnavailable
	}
	return scope, nil
}

// WeatherScopeFromContext resolves weather geo from the shipped pack.
func WeatherScopeFromContext(ctx context.Context, supplierID string) (string, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return "", err
	}
	return PackWeatherScope(pack)
}

// PackFactorySLADefaultHours returns the shipped pack default SLA window.
func PackFactorySLADefaultHours(pack MarketPack) (float64, error) {
	if pack.Status != MarketPackShipped {
		return 0, ErrMarketPackNotShipped
	}
	if pack.FactorySLADefaultHours <= 0 {
		return 0, ErrFactorySLAHoursInvalid
	}
	return float64(pack.FactorySLADefaultHours), nil
}

// FactorySLADefaultHoursFromContext resolves factory SLA hours from the shipped pack.
func FactorySLADefaultHoursFromContext(ctx context.Context, supplierID string) (float64, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return 0, err
	}
	return PackFactorySLADefaultHours(pack)
}

// PackLaborMaxShiftHours returns the shipped pack max shift hours.
func PackLaborMaxShiftHours(pack MarketPack) (int64, error) {
	if pack.Status != MarketPackShipped {
		return 0, ErrMarketPackNotShipped
	}
	if pack.LaborMaxShiftHours <= 0 {
		return 0, ErrLaborMaxShiftHoursInvalid
	}
	return pack.LaborMaxShiftHours, nil
}

// LaborMaxShiftHoursFromContext resolves max shift hours from the shipped pack.
func LaborMaxShiftHoursFromContext(ctx context.Context, supplierID string) (int64, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return 0, err
	}
	return PackLaborMaxShiftHours(pack)
}

// TimezonePackHTTPStatus maps M4 sentinels to status + error code.
func TimezonePackHTTPStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrTimezoneInvalid), errors.Is(err, ErrShopClosedGraceInvalid),
		errors.Is(err, ErrWeatherScopeUnavailable), errors.Is(err, ErrFactorySLAHoursInvalid),
		errors.Is(err, ErrLaborMaxShiftHoursInvalid):
		return http.StatusUnprocessableEntity, err.Error()
	default:
		return FiscalPackHTTPStatus(err)
	}
}
