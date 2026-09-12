package retailer

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// stampPackCurrency applies GS-M5 / L2: empty → shipped pack; mismatch → error.
// Never invents UZS outside the pack.
func stampPackCurrency(ctx context.Context, requested string) (string, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return "", err
	}
	return auth.ResolveCheckoutCurrency(pack, requested)
}

// coalescePackCurrency uses stored currency when set, otherwise the shipped pack.
func coalescePackCurrency(ctx context.Context, stored string) string {
	if c := strings.ToUpper(strings.TrimSpace(stored)); c != "" {
		return c
	}
	cur, err := auth.CurrencyFromContext(ctx, "")
	if err != nil {
		return ""
	}
	return cur
}
