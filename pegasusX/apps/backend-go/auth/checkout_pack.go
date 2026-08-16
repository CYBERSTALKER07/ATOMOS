package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrMarketPackUnknown    = errors.New("market_pack_unknown")
	ErrMarketPackNotShipped = errors.New("market_pack_not_shipped")
	ErrPackGatewayForbidden = errors.New("pack_gateway_forbidden")
	ErrPackCurrencyMismatch = errors.New("pack_currency_mismatch")
)

// CheckoutPackFromContext returns the shipped pack checkout must use (GS-M1).
// Claims resolve first; missing claims fall back to the env shipped default so
// unit tests without a JWT still price in UZ. Planned/unknown packs fail closed.
// Catalog CheckoutReadsThis stays false until GS-M2; this reader still applies
// currency + PSP from the shipped pack.
func CheckoutPackFromContext(ctx context.Context) (MarketPack, error) {
	if claims, ok := FromContext(ctx); ok {
		return RequireCheckoutPack(claims)
	}
	p, ok := ResolveShippedMarketPack(DefaultMarketCodeFromEnv())
	if !ok {
		return MarketPack{}, ErrMarketPackNotShipped
	}
	return p, nil
}

// RequireCheckoutPack fails closed for unknown or planned packs.
func RequireCheckoutPack(c Claims) (MarketPack, error) {
	asg := ResolveMarketAssignment(c)
	p, ok := ResolveMarketPack(asg.MarketCode)
	if !ok {
		return MarketPack{}, ErrMarketPackUnknown
	}
	if p.Status != MarketPackShipped {
		return MarketPack{}, ErrMarketPackNotShipped
	}
	return p, nil
}

// CanonicalPSP maps runtime aliases onto pack adapter names.
// PEGASUS / GP are not a second live PSP — they are GLOBAL_PAY.
func CanonicalPSP(gateway string) string {
	switch strings.ToUpper(strings.TrimSpace(gateway)) {
	case "PEGASUS", "GLOBALPAY", "GP":
		return "GLOBAL_PAY"
	default:
		return strings.ToUpper(strings.TrimSpace(gateway))
	}
}

// PackAllowsPSP reports whether gateway is in pack.PSPAdapters.
func PackAllowsPSP(pack MarketPack, gateway string) bool {
	want := CanonicalPSP(gateway)
	if want == "" {
		return false
	}
	for _, a := range pack.PSPAdapters {
		if CanonicalPSP(a) == want {
			return true
		}
	}
	return false
}

// AssertPackPSP fails closed for an unknown or planned-pack gateway.
func AssertPackPSP(pack MarketPack, gateway string) error {
	if !PackAllowsPSP(pack, gateway) {
		return ErrPackGatewayForbidden
	}
	return nil
}

// ResolveCheckoutCurrency uses the shipped pack currency.
// Empty request → pack. Non-empty must match pack.
func ResolveCheckoutCurrency(pack MarketPack, requested string) (string, error) {
	want := strings.ToUpper(strings.TrimSpace(pack.CurrencyCode))
	if want == "" {
		return "", ErrMarketPackUnknown
	}
	req := strings.ToUpper(strings.TrimSpace(requested))
	if req == "" || req == want {
		return want, nil
	}
	return "", ErrPackCurrencyMismatch
}

// IsShippedPackCurrency is true when a shipped pack uses this ISO code.
func IsShippedPackCurrency(code string) bool {
	c := strings.ToUpper(strings.TrimSpace(code))
	if c == "" {
		return false
	}
	for _, p := range ListMarketPacks() {
		if p.Status == MarketPackShipped && strings.EqualFold(p.CurrencyCode, c) {
			return true
		}
	}
	return false
}

// CheckoutPackHTTPStatus maps M1 sentinels to status + error code.
func CheckoutPackHTTPStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrMarketPackUnknown), errors.Is(err, ErrMarketPackNotShipped):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, ErrPackGatewayForbidden), errors.Is(err, ErrPackCurrencyMismatch):
		return http.StatusUnprocessableEntity, err.Error()
	default:
		return http.StatusUnprocessableEntity, err.Error()
	}
}
