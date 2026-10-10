package main

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// simOperatingCurrency is empty-currency law for ecosystem-simulator fixtures:
// stored ISO (seed supplier currency), else the shipped pack. Planned/unknown
// packs stay empty — never invent UZS.
func simOperatingCurrency(ctx context.Context, stored string) string {
	c, err := auth.CoalesceCurrency(ctx, "", stored)
	if err != nil {
		return strings.ToUpper(strings.TrimSpace(stored))
	}
	return c
}

func (s *Simulator) operatingCurrency(ctx context.Context) string {
	stored := ""
	if s != nil && s.cfg != nil {
		stored = s.cfg.SeedSupplierCurrency
	}
	return simOperatingCurrency(ctx, stored)
}
