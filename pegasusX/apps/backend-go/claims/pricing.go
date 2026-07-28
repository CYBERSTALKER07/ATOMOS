package claims

import (
	"fmt"
	"strings"
)

// OrderLine is a priced line on the original order (source of truth for chargebacks).
type OrderLine struct {
	SKU            string
	Quantity       int64
	UnitPriceMinor int64
}

// PriceClaimLines binds claim SKUs/qty to order unit prices and returns the total.
// Rules (marketplace-style):
//  1. SKU must exist on the order.
//  2. Claim qty cannot exceed original order qty for that SKU.
//  3. Unit price always comes from the order, never the client.
//  4. Amount = sum(qty * unit_price) in minor units (tiyin).
func PriceClaimLines(orderLines []OrderLine, claimLines []ClaimLine) ([]ClaimLine, int64, error) {
	bySKU := make(map[string]OrderLine, len(orderLines))
	for _, ol := range orderLines {
		sku := strings.TrimSpace(ol.SKU)
		if sku == "" || ol.Quantity <= 0 {
			continue
		}
		// Aggregate if SKU appears twice.
		if existing, ok := bySKU[sku]; ok {
			existing.Quantity += ol.Quantity
			// Prefer first non-zero unit price.
			if existing.UnitPriceMinor == 0 {
				existing.UnitPriceMinor = ol.UnitPriceMinor
			}
			bySKU[sku] = existing
			continue
		}
		bySKU[sku] = OrderLine{SKU: sku, Quantity: ol.Quantity, UnitPriceMinor: ol.UnitPriceMinor}
	}
	if len(bySKU) == 0 {
		return nil, 0, fmt.Errorf("%w: order has no priced lines", ErrPricingFailed)
	}

	out := make([]ClaimLine, 0, len(claimLines))
	var total int64
	for _, cl := range claimLines {
		sku := strings.TrimSpace(cl.SKU)
		if sku == "" || cl.Quantity <= 0 {
			return nil, 0, ErrInvalidLineItems
		}
		ol, ok := bySKU[sku]
		if !ok {
			return nil, 0, fmt.Errorf("%w: sku %s not on order", ErrPricingFailed, sku)
		}
		if cl.Quantity > ol.Quantity {
			return nil, 0, fmt.Errorf("%w: qty %d exceeds order qty %d for sku %s", ErrPricingFailed, cl.Quantity, ol.Quantity, sku)
		}
		if ol.UnitPriceMinor < 0 {
			return nil, 0, fmt.Errorf("%w: negative unit price for sku %s", ErrPricingFailed, sku)
		}
		lineAmt := cl.Quantity * ol.UnitPriceMinor
		reason := strings.TrimSpace(cl.Reason)
		out = append(out, ClaimLine{
			SKU:            sku,
			Quantity:       cl.Quantity,
			Reason:         reason,
			UnitPriceMinor: ol.UnitPriceMinor,
			AmountMinor:    lineAmt,
		})
		total += lineAmt
	}
	if total < 0 {
		return nil, 0, fmt.Errorf("%w: negative total", ErrPricingFailed)
	}
	return out, total, nil
}
