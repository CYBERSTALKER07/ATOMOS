package auth

import (
	"context"
	"strings"
)

// PackCountryCode is the shipped pack's tax/country stamp (GS-M7).
// Do not infer country from currency (KZT→KZ else UZ).
func PackCountryCode(pack MarketPack) (string, error) {
	if pack.Status != MarketPackShipped {
		return "", ErrMarketPackNotShipped
	}
	code := NormalizeMarketCode(pack.Code)
	if code == "" {
		return "", ErrMarketPackUnknown
	}
	return code, nil
}

// CountryFromContext resolves pack country: claims → supplier profile → env shipped pack.
func CountryFromContext(ctx context.Context, supplierID string) (string, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return "", err
	}
	return PackCountryCode(pack)
}

// NormalizeCountryCode is ISO-like uppercase (UZ, KZ). Empty stays empty.
func NormalizeCountryCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
