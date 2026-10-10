package promotion

import (
	"fmt"
	"strings"
	"time"
)

const maxDiscountBps int64 = 10000

// PickBestPromotion selects the highest-discount active promotion for one catalog row.
func PickBestPromotion(
	now time.Time,
	retailerID string,
	productID string,
	categoryID string,
	quantity int64,
	listUnitPrice int64,
	orderSubtotalMinor int64,
	promotions []Promotion,
) (*Promotion, int64, int64) {
	var best *Promotion
	var bestUnit int64
	var bestDiscountBps int64
	for i := range promotions {
		p := &promotions[i]
		if !promotionActiveAt(p, now) {
			continue
		}
		if !retailerEligible(p, retailerID) {
			continue
		}
		if !scopeMatches(p, productID, categoryID) {
			continue
		}
		if p.MinOrderAmountMinor > 0 && orderSubtotalMinor < p.MinOrderAmountMinor {
			continue
		}
		var bestTier *PromotionTier
		for j := range p.Tiers {
			t := &p.Tiers[j]
			if quantity >= t.MinQuantity {
				if bestTier == nil || t.DiscountBps > bestTier.DiscountBps {
					bestTier = t
				}
			}
		}
		if bestTier == nil {
			continue
		}
		unit := discountedUnit(listUnitPrice, bestTier.DiscountBps)
		if best == nil || unit < bestUnit || (unit == bestUnit && p.Priority > best.Priority) {
			best = p
			bestUnit = unit
			bestDiscountBps = bestTier.DiscountBps
		}
	}
	if best == nil {
		return nil, listUnitPrice, 0
	}
	return best, bestUnit, bestDiscountBps
}

// ApplyQuote prices all lines for a supplier cart slice.
func ApplyQuote(
	now time.Time,
	supplierID string,
	retailerID string,
	lines []LineInput,
	promotions []Promotion,
) (QuoteResult, error) {
	if len(lines) == 0 {
		return QuoteResult{}, fmt.Errorf("lines must not be empty")
	}
	subtotal := int64(0)
	for _, line := range lines {
		if line.Quantity <= 0 || line.UnitPrice < 0 {
			return QuoteResult{}, fmt.Errorf("invalid line quantity or price")
		}
		subtotal += line.UnitPrice * line.Quantity
	}

	quoted := make([]QuotedLine, 0, len(lines))
	currency := strings.TrimSpace(lines[0].Currency)
	total := int64(0)
	discountTotal := int64(0)

	for _, line := range lines {
		if line.PriceIsOverride {
			lineTotal := line.UnitPrice * line.Quantity
			total += lineTotal
			quoted = append(quoted, QuotedLine{
				ProductID:     line.ProductID,
				Quantity:      line.Quantity,
				ListUnitPrice: line.UnitPrice,
				UnitPrice:     line.UnitPrice,
				LineTotal:     lineTotal,
				Currency:      line.Currency,
			})
			continue
		}
		best, unit, appliedBps := PickBestPromotion(
			now,
			retailerID,
			line.ProductID,
			line.CategoryID,
			line.Quantity,
			line.UnitPrice,
			subtotal,
			promotions,
		)
		lineTotal := unit * line.Quantity
		total += lineTotal
		lineDiscount := (line.UnitPrice - unit) * line.Quantity
		if lineDiscount > 0 {
			discountTotal += lineDiscount
		}
		q := QuotedLine{
			ProductID:     line.ProductID,
			Quantity:      line.Quantity,
			ListUnitPrice: line.UnitPrice,
			UnitPrice:     unit,
			LineTotal:     lineTotal,
			Currency:      line.Currency,
		}
		if best != nil {
			q.DiscountBps = appliedBps
			q.PromotionID = best.PromotionID
			q.PromotionName = best.Name
			q.PromotionLabel = formatPromotionLabel(best, appliedBps)
		}
		quoted = append(quoted, q)
	}

	return QuoteResult{
		SupplierID:    supplierID,
		RetailerID:    retailerID,
		Lines:         quoted,
		SubtotalMinor: subtotal,
		DiscountMinor: discountTotal,
		TotalMinor:    total,
		Currency:      currency,
	}, nil
}

// CatalogOffer builds a display offer for quantity=1 browsing.
func CatalogOffer(
	now time.Time,
	retailerID string,
	productID string,
	categoryID string,
	listPriceMinor int64,
	promotions []Promotion,
) ProductOffer {
	offer := ProductOffer{
		ProductID:      productID,
		ListPriceMinor: listPriceMinor,
	}
	best, unit, appliedBps := PickBestPromotion(now, retailerID, productID, categoryID, 1, listPriceMinor, listPriceMinor, promotions)
	if best == nil || unit >= listPriceMinor {
		if best != nil && len(best.Tiers) > 0 {
			lowestTier := best.Tiers[0]
			for _, t := range best.Tiers {
				if t.MinQuantity < lowestTier.MinQuantity {
					lowestTier = t
				}
			}
			if lowestTier.MinQuantity > 1 {
				label := formatPromotionLabel(best, lowestTier.DiscountBps)
				offer.PromotionLabel = &label
				offer.PromotionID = &best.PromotionID
				offer.PromotionName = &best.Name
				if best.EndsAt != nil {
					ends := best.EndsAt.UTC().Format(time.RFC3339)
					offer.PromotionEndsAt = &ends
				}
			}
		}
		return offer
	}
	offer.SalePriceMinor = &unit
	offer.DiscountBps = &appliedBps
	offer.PromotionID = &best.PromotionID
	offer.PromotionName = &best.Name
	label := formatPromotionLabel(best, appliedBps)
	offer.PromotionLabel = &label
	if best.EndsAt != nil {
		ends := best.EndsAt.UTC().Format(time.RFC3339)
		offer.PromotionEndsAt = &ends
	}
	return offer
}

func promotionActiveAt(p *Promotion, now time.Time) bool {
	if p == nil || !p.IsActive || len(p.Tiers) == 0 {
		return false
	}
	if p.StartsAt != nil && now.Before(p.StartsAt.UTC()) {
		return false
	}
	if p.EndsAt != nil && !now.Before(p.EndsAt.UTC()) {
		return false
	}
	return true
}

func retailerEligible(p *Promotion, retailerID string) bool {
	if p.RetailerScope == RetailerScopeAll || len(p.RetailerIDs) == 0 {
		return true
	}
	for _, id := range p.RetailerIDs {
		if id == retailerID {
			return true
		}
	}
	return false
}

func scopeMatches(p *Promotion, productID, categoryID string) bool {
	switch p.ScopeType {
	case ScopeTypeAllProducts:
		return true
	case ScopeTypeProduct:
		return p.ScopeProductID != "" && p.ScopeProductID == productID
	case ScopeTypeCategory:
		return p.ScopeCategoryID != "" && p.ScopeCategoryID == categoryID
	default:
		return false
	}
}

func discountedUnit(listUnitPrice int64, discountBps int64) int64 {
	if discountBps <= 0 {
		return listUnitPrice
	}
	if discountBps >= maxDiscountBps {
		return 0
	}
	return (listUnitPrice * (maxDiscountBps - discountBps)) / maxDiscountBps
}

func formatPromotionLabel(p *Promotion, appliedBps int64) string {
	pct := float64(appliedBps) / 100.0
	minQ := int64(1)
	for _, t := range p.Tiers {
		if t.DiscountBps == appliedBps {
			minQ = t.MinQuantity
			break
		}
	}
	if minQ > 1 {
		return fmt.Sprintf("%.2g%% off when you buy %d+", pct, minQ)
	}
	if p.MinOrderAmountMinor > 0 {
		return fmt.Sprintf("%.2g%% off orders over %d", pct, p.MinOrderAmountMinor)
	}
	return fmt.Sprintf("%.2g%% off", pct)
}
