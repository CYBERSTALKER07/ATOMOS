package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
)

var (
	ErrFiscalAdapterUnimplemented = errors.New("fiscal_adapter_unimplemented")
	ErrFakeFiscalForbidden        = errors.New("fake_fiscal_forbidden")
)

// FiscalPackFromContext returns the shipped pack fiscal must use (GS-M2).
// Claims first, then the supplier profile, then the env shipped default.
// Planned/unknown packs fail closed — do not fiscalize as silent UZ.
func FiscalPackFromContext(ctx context.Context, supplierID string) (MarketPack, error) {
	if claims, ok := FromContext(ctx); ok {
		return RequireCheckoutPack(claims)
	}
	return FiscalPackForSupplier(supplierID)
}

// FiscalPackForSupplier resolves a shipped pack from the supplier profile, else env.
func FiscalPackForSupplier(supplierID string) (MarketPack, error) {
	if p, ok := lookupMarketProfile(supplierID); ok {
		pack, found := ResolveMarketPack(p.MarketCode)
		if !found {
			return MarketPack{}, ErrMarketPackUnknown
		}
		if pack.Status != MarketPackShipped {
			return MarketPack{}, ErrMarketPackNotShipped
		}
		return pack, nil
	}
	p, ok := ResolveShippedMarketPack(DefaultMarketCodeFromEnv())
	if !ok {
		return MarketPack{}, ErrMarketPackNotShipped
	}
	return p, nil
}

// CanonicalFiscalAdapter maps catalog / env aliases onto pack adapter names.
func CanonicalFiscalAdapter(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "MY_SOLIQ", "MYSOLIQ", "SOLIQ", "OFD":
		return "MY_SOLIQ"
	case "PEPPOL":
		return "PEPPOL"
	case "COMMERCIAL", "PLATFORM", "PEGASUS":
		return "COMMERCIAL"
	case "PLANNED":
		return "PLANNED"
	case "FAKE":
		return "FAKE"
	case "GLOBAL_PAY":
		return "GLOBAL_PAY"
	default:
		return strings.ToUpper(strings.TrimSpace(name))
	}
}

// PackFiscalAdapter returns the shipped pack's fiscal adapter.
// PEPPOL / PLANNED / unknown fail closed (not implemented).
func PackFiscalAdapter(pack MarketPack) (string, error) {
	if pack.Status != MarketPackShipped {
		return "", ErrMarketPackNotShipped
	}
	switch CanonicalFiscalAdapter(pack.FiscalAdapter) {
	case "MY_SOLIQ":
		return "MY_SOLIQ", nil
	case "COMMERCIAL":
		return "COMMERCIAL", nil
	default:
		return "", ErrFiscalAdapterUnimplemented
	}
}

// AssertFiscalRuntime applies M2 fail-closed rules to the cell runtime adapter.
// PEGASUS/FAKE on a MY_SOLIQ pack is allowed outside production — that disagreement
// is why checkout_reads_this stays false. FAKE is forbidden in production.
func AssertFiscalRuntime(pack MarketPack, runtime string) error {
	if _, err := PackFiscalAdapter(pack); err != nil {
		return err
	}
	if CanonicalFiscalAdapter(runtime) == "FAKE" && isProductionFiscalEnv() {
		return ErrFakeFiscalForbidden
	}
	return nil
}

func isProductionFiscalEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV"))) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

// BuyerAcceptancePollerAllowed is true only for a shipped MY_SOLIQ pack (BF-352).
func BuyerAcceptancePollerAllowed() bool {
	pack, err := FiscalPackForSupplier("")
	if err != nil {
		return false
	}
	ad, err := PackFiscalAdapter(pack)
	return err == nil && ad == "MY_SOLIQ"
}

// FiscalPackHTTPStatus maps M2 sentinels to status + error code.
func FiscalPackHTTPStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrMarketPackUnknown), errors.Is(err, ErrMarketPackNotShipped):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, ErrFiscalAdapterUnimplemented), errors.Is(err, ErrFakeFiscalForbidden),
		errors.Is(err, ErrBreachRadiusInvalid), errors.Is(err, ErrTimezoneInvalid),
		errors.Is(err, ErrShopClosedGraceInvalid), errors.Is(err, ErrPayoutRailUnknown):
		return http.StatusUnprocessableEntity, err.Error()
	default:
		return CheckoutPackHTTPStatus(err)
	}
}
