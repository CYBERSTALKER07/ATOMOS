package synthesis

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// preorderOperatingCurrency is empty-currency law for AI_PREORDER inserts:
// stored ISO (signal currency), else the shipped pack. Planned/unknown packs
// stay empty — never invent UZS.
func preorderOperatingCurrency(ctx context.Context, supplierID, stored string) string {
	c, err := auth.CoalesceCurrency(ctx, supplierID, stored)
	if err != nil {
		return strings.ToUpper(strings.TrimSpace(stored))
	}
	return c
}
