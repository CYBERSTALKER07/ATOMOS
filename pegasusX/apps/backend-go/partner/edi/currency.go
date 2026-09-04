package edi

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ediCurrency is empty-currency law for EDI-lite codecs: stored ISO code, else
// the shipped pack. Planned/unknown packs stay empty — never invent UZS.
func ediCurrency(supplierID, stored string) string {
	c, err := auth.CoalesceCurrency(context.Background(), supplierID, stored)
	if err != nil {
		return strings.ToUpper(strings.TrimSpace(stored))
	}
	return c
}
