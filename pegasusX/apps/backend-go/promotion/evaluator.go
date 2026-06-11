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
) (*Promotion, int64) {
	var best *Promotion
	var bestUnit int64
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
		if p.MinLineQuantity > 0 && quantity < p.MinLineQuantity {
			continue
		}
		if p.MinOrderAmountMinor > 0 && orderSubtotalMinor < p.MinOrderAmountMinor {
			continue
		}
		unit := discountedUnit(listUnitPrice, p.DiscountBps)
		if best == nil || unit < bestUnit || (unit == bestUnit && p.Priority > best.Priority) {
			best = p
			bestUnit = unit
		}
	}
	if best == nil {
		return nil, listUnitPrice
	}
	return best, bestUnit
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
		best, unit := PickBestPromotion(
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
			q.DiscountBps = best.DiscountBps
			q.PromotionID = best.PromotionID
			q.PromotionName = best.Name
			q.PromotionLabel = formatPromotionLabel(best)
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
	best, unit := PickBestPromotion(now, retailerID, productID, categoryID, 1, listPriceMinor, listPriceMinor, promotions)
	if best == nil || unit >= listPriceMinor {
		if best != nil && best.MinLineQuantity > 1 {
			label := formatPromotionLabel(best)
			offer.PromotionLabel = &label
			offer.PromotionID = &best.PromotionID
			offer.PromotionName = &best.Name
			if best.EndsAt != nil {
				ends := best.EndsAt.UTC().Format(time.RFC3339)
				offer.PromotionEndsAt = &ends
			}
		}
		return offer
	}
	offer.SalePriceMinor = &unit
	offer.DiscountBps = &best.DiscountBps
	offer.PromotionID = &best.PromotionID
	offer.PromotionName = &best.Name
	label := formatPromotionLabel(best)
	offer.PromotionLabel = &label
	if best.EndsAt != nil {
		ends := best.EndsAt.UTC().Format(time.RFC3339)
		offer.PromotionEndsAt = &ends
	}
	return offer
}

func promotionActiveAt(p *Promotion, now time.Time) bool {
	if p == nil || !p.IsActive || p.DiscountBps <= 0 || p.DiscountBps > maxDiscountBps {
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

func formatPromotionLabel(p *Promotion) string {
	pct := float64(p.DiscountBps) / 100.0
	if p.MinLineQuantity > 1 {
		return fmt.Sprintf("%.2g%% off when you buy %d+", pct, p.MinLineQuantity)
	}
	if p.MinOrderAmountMinor > 0 {
		return fmt.Sprintf("%.2g%% off orders over %d", pct, p.MinOrderAmountMinor)
	}
	return fmt.Sprintf("%.2g%% off", pct)
}
