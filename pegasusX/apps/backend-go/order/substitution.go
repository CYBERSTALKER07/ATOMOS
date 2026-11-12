package order

import (
	"strings"
)

// ShortageAction defines how the system handles unfillable or short-allocated lines.
type ShortageAction string

const (
	ShortageActionFulfill    ShortageAction = "FULFILL"
	ShortageActionCutLine    ShortageAction = "CUT_LINE"
	ShortageActionHoldOrder  ShortageAction = "HOLD_ORDER"
	ShortageActionSubstitute ShortageAction = "SUBSTITUTE"
	ShortageActionSplitOrder ShortageAction = "SPLIT_ORDER"
)

// ShortageEvaluationRequest contains the line-item constraints and retailer preferences.
type ShortageEvaluationRequest struct {
	SKU                string `json:"sku"`
	BrandID            string `json:"brand_id"`
	RequestedQty       int64  `json:"requested_qty"`
	AvailableQty       int64  `json:"available_qty"`
	AllowSubstitutes   bool   `json:"allow_substitutes"`
	MustHave           bool   `json:"must_have"`
	OriginalPriceMinor int64  `json:"original_price_minor"`
}

// SubstituteCandidate is an available SKU evaluated for brand-family substitution.
type SubstituteCandidate struct {
	SKU          string `json:"sku"`
	BrandID      string `json:"brand_id"`
	AvailableQty int64  `json:"available_qty"`
	PriceMinor   int64  `json:"price_minor"`
}

// ShortageEvaluationResult is the outcome of the shortage and substitution evaluation.
type ShortageEvaluationResult struct {
	Action        ShortageAction `json:"action"`
	Reason        string         `json:"reason"`
	SubstituteSKU string         `json:"substitute_sku,omitempty"`
	PriceGapMinor int64          `json:"price_gap_minor,omitempty"` // substitute price minus original price
}

// EvaluateLineSubstitution enforces PegasusX B2B Operating Model Invariants:
// 1) Out of stock: Line is fulfilled if stock >= requested.
// 2) If stock < requested and AllowSubstitutes is true:
//    Matches candidate strictly within the same BrandID family with sufficient stock.
// 3) If no substitute is available or allowed:
//    - If MustHave is true: HOLD_ORDER (order waits or splits; does not ride truck half-complete).
//    - If MustHave is false: CUT_LINE with reason OUT_OF_STOCK_SUBSTITUTE_DISALLOWED.
func EvaluateLineSubstitution(req ShortageEvaluationRequest, candidates []SubstituteCandidate) ShortageEvaluationResult {
	if req.AvailableQty >= req.RequestedQty && req.RequestedQty > 0 {
		return ShortageEvaluationResult{
			Action: ShortageActionFulfill,
			Reason: "IN_STOCK",
		}
	}

	brandID := strings.TrimSpace(req.BrandID)

	// Evaluate substitution if allowed
	if req.AllowSubstitutes && brandID != "" && len(candidates) > 0 {
		for _, cand := range candidates {
			candBrand := strings.TrimSpace(cand.BrandID)
			if candBrand == brandID && cand.SKU != req.SKU && cand.AvailableQty >= req.RequestedQty {
				return ShortageEvaluationResult{
					Action:        ShortageActionSubstitute,
					Reason:        "SAME_BRAND_FAMILY_SUBSTITUTE",
					SubstituteSKU: cand.SKU,
					PriceGapMinor: cand.PriceMinor - req.OriginalPriceMinor,
				}
			}
		}
	}

	// Must-have line that cannot be filled or substituted holds the order
	if req.MustHave {
		return ShortageEvaluationResult{
			Action: ShortageActionHoldOrder,
			Reason: "MUST_HAVE_LINE_UNFILLED",
		}
	}

	// If substitutes disallowed or unavailable, line is cut
	reason := "OUT_OF_STOCK_SUBSTITUTE_DISALLOWED"
	if req.AllowSubstitutes {
		reason = "OUT_OF_STOCK_NO_FAMILY_SUBSTITUTE_FOUND"
	}

	return ShortageEvaluationResult{
		Action: ShortageActionCutLine,
		Reason: reason,
	}
}
