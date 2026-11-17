package main

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// smokeOperatingCurrency is empty-currency law for SSMR smokecheck: stored ISO
// (seed supplier currency), else the shipped pack. Planned/unknown packs stay
// empty — never invent UZS. Callers skip the marker when empty.
func smokeOperatingCurrency(ctx context.Context, stored string) string {
	c, err := auth.CoalesceCurrency(ctx, "", stored)
	if err != nil {
		return strings.ToUpper(strings.TrimSpace(stored))
	}
	return c
}
