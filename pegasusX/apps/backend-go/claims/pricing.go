package claims

import (
	"fmt"
	"math"
	"strings"
)

// OrderLine is a priced line on the original order (source of truth for chargebacks).
type OrderLine struct {
	SKU            string
	Quantity       int64
	UnitPriceMinor int64
}

// AggregateClaimLines merges duplicate SKUs in a request so split rows cannot
// bypass the per-SKU quantity cap (e.g. two rows of qty 3 on an order of 5).
func AggregateClaimLines(lines []ClaimLine) []ClaimLine {
	if len(lines) == 0 {
		return nil
	}
	type acc struct {
		qty    int64
		reason string
	}
	bySKU := make(map[string]acc, len(lines))
	order := make([]string, 0, len(lines))
	for _, cl := range lines {
		sku := strings.TrimSpace(cl.SKU)
		if sku == "" || cl.Quantity <= 0 {
			continue
		}
		a, ok := bySKU[sku]
		if !ok {
			order = append(order, sku)
			a.reason = strings.TrimSpace(cl.Reason)
		}
		// Checked add to avoid wrap on hostile payloads.
		if a.qty > math.MaxInt64-cl.Quantity {
			a.qty = math.MaxInt64
		} else {
			a.qty += cl.Quantity
		}
		if a.reason == "" {
			a.reason = strings.TrimSpace(cl.Reason)
		}
		bySKU[sku] = a
	}
	out := make([]ClaimLine, 0, len(order))
	for _, sku := range order {
		a := bySKU[sku]
		out = append(out, ClaimLine{SKU: sku, Quantity: a.qty, Reason: a.reason})
	}
	return out
}

// mulInt64 returns a*b or an error if the product would overflow int64.
func mulInt64(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a < 0 || b < 0 {
		return 0, fmt.Errorf("%w: negative factor", ErrPricingFailed)
	}
	if a > math.MaxInt64/b {
		return 0, fmt.Errorf("%w: amount overflow", ErrPricingFailed)
	}
	return a * b, nil
}

// addInt64 returns a+b or overflow error.
func addInt64(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, fmt.Errorf("%w: sum overflow", ErrPricingFailed)
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, fmt.Errorf("%w: sum overflow", ErrPricingFailed)
	}
	return a + b, nil
}

// buildOrderSKUIndex aggregates order lines by SKU.
// Unit price: weighted average in minor units (rounded half-up) when the same
// SKU appears at different prices; otherwise the single unit price.
func buildOrderSKUIndex(orderLines []OrderLine) (map[string]OrderLine, error) {
	type acc struct {
		qty       int64
		valueSum  int64 // sum(qty * unit_price) for weighted average
	}
	tmp := make(map[string]acc, len(orderLines))
	for _, ol := range orderLines {
		sku := strings.TrimSpace(ol.SKU)
		if sku == "" || ol.Quantity <= 0 {
			continue
		}
		if ol.UnitPriceMinor < 0 {
			return nil, fmt.Errorf("%w: negative unit price for sku %s", ErrPricingFailed, sku)
		}
		lineVal, err := mulInt64(ol.Quantity, ol.UnitPriceMinor)
		if err != nil {
			return nil, err
		}
		a := tmp[sku]
		newQty, err := addInt64(a.qty, ol.Quantity)
		if err != nil {
			return nil, err
		}
		newVal, err := addInt64(a.valueSum, lineVal)
		if err != nil {
			return nil, err
		}
		a.qty = newQty
		a.valueSum = newVal
		tmp[sku] = a
	}
	if len(tmp) == 0 {
		return nil, fmt.Errorf("%w: order has no priced lines", ErrPricingFailed)
	}
	out := make(map[string]OrderLine, len(tmp))
	for sku, a := range tmp {
		// Weighted average unit price: floor((valueSum + qty/2) / qty) half-up.
		unit := a.valueSum / a.qty
		if rem := a.valueSum % a.qty; rem*2 >= a.qty {
			unit++
		}
		out[sku] = OrderLine{SKU: sku, Quantity: a.qty, UnitPriceMinor: unit}
	}
	return out, nil
}

// ClaimedQtyBySKU sums quantities on prior claims that still reserve inventory
// liability (OPEN, UNDER_REVIEW, APPROVED, RESOLVED). REJECTED does not count.
func ClaimedQtyBySKU(prior []Claim, excludeClaimID string) map[string]int64 {
	out := make(map[string]int64)
	excludeClaimID = strings.TrimSpace(excludeClaimID)
	for _, c := range prior {
		if excludeClaimID != "" && c.ClaimID == excludeClaimID {
			continue
		}
		switch c.Status {
		case StatusRejected:
			continue
		case StatusOpen, StatusUnderReview, StatusApproved, StatusResolved:
			// count
		default:
			continue
		}
		for _, li := range c.LineItems {
			sku := strings.TrimSpace(li.SKU)
			if sku == "" || li.Quantity <= 0 {
				continue
			}
			if out[sku] > math.MaxInt64-li.Quantity {
				out[sku] = math.MaxInt64
			} else {
				out[sku] += li.Quantity
			}
		}
	}
	return out
}

// PriceClaimLines binds claim SKUs/qty to order unit prices and returns the total.
//
// Rules (marketplace-style):
//  1. Request lines are aggregated by SKU first.
//  2. SKU must exist on the order.
//  3. Claim qty cannot exceed remaining claimable qty (order qty − prior claims).
//  4. Unit price always comes from the order (weighted avg if mixed prices).
//  5. Amount = sum(qty × unit_price) in minor units with overflow checks.
func PriceClaimLines(orderLines []OrderLine, claimLines []ClaimLine) ([]ClaimLine, int64, error) {
	return PriceClaimLinesWithPrior(orderLines, claimLines, nil)
}

// PriceClaimLinesWithPrior is PriceClaimLines with already-claimed qty deducted
// from available order quantity so multi-claim sequences cannot over-charge.
func PriceClaimLinesWithPrior(orderLines []OrderLine, claimLines []ClaimLine, priorClaimed map[string]int64) ([]ClaimLine, int64, error) {
	bySKU, err := buildOrderSKUIndex(orderLines)
	if err != nil {
		return nil, 0, err
	}
	agg := AggregateClaimLines(claimLines)
	if len(agg) == 0 {
		return nil, 0, ErrInvalidLineItems
	}

	out := make([]ClaimLine, 0, len(agg))
	var total int64
	for _, cl := range agg {
		ol, ok := bySKU[cl.SKU]
		if !ok {
			return nil, 0, fmt.Errorf("%w: sku %s not on order", ErrPricingFailed, cl.SKU)
		}
		already := int64(0)
		if priorClaimed != nil {
			already = priorClaimed[cl.SKU]
		}
		available := ol.Quantity - already
		if available < 0 {
			available = 0
		}
		if cl.Quantity > available {
			return nil, 0, fmt.Errorf("%w: qty %d exceeds remaining claimable %d for sku %s (order=%d already_claimed=%d)",
				ErrPricingFailed, cl.Quantity, available, cl.SKU, ol.Quantity, already)
		}
		lineAmt, err := mulInt64(cl.Quantity, ol.UnitPriceMinor)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, ClaimLine{
			SKU:            cl.SKU,
			Quantity:       cl.Quantity,
			Reason:         cl.Reason,
			UnitPriceMinor: ol.UnitPriceMinor,
			AmountMinor:    lineAmt,
		})
		total, err = addInt64(total, lineAmt)
		if err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

// CapAmount ensures chargeback never exceeds order/session ceilings.
func CapAmount(amount, cap int64) int64 {
	if amount < 0 {
		return 0
	}
	if cap > 0 && amount > cap {
		return cap
	}
	return amount
}
