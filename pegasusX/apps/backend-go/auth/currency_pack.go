package auth

import (
	"context"
	"strings"
)

// PackCurrency returns the shipped pack ISO currency (GS-M5).
func PackCurrency(pack MarketPack) (string, error) {
	return ResolveCheckoutCurrency(pack, "")
}

// CurrencyFromContext resolves empty-currency law: claims → supplier profile → env shipped pack.
// Never invents UZS.
func CurrencyFromContext(ctx context.Context, supplierID string) (string, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return "", err
	}
	return PackCurrency(pack)
}

// CoalesceCurrency uses stored currency when set, otherwise the shipped pack.
func CoalesceCurrency(ctx context.Context, supplierID, stored string) (string, error) {
	if c := strings.ToUpper(strings.TrimSpace(stored)); c != "" {
		return c, nil
	}
	return CurrencyFromContext(ctx, supplierID)
}
