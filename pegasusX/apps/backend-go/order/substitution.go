package order

import (
	"fmt"
	"sort"
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
	SKU                   string   `json:"sku"`
	SupplierID            string   `json:"supplier_id,omitempty"`
	BrandID               string   `json:"brand_id"`
	RequestedQty          int64    `json:"requested_qty"`
	AvailableQty          int64    `json:"available_qty"`
	AllowSubstitutes      bool     `json:"allow_substitutes"`
	MustHave              bool     `json:"must_have"`
	OriginalPriceMinor    int64    `json:"original_price_minor"`
	AllowedSubstituteSKUs []string `json:"allowed_substitute_skus,omitempty"`
	EnforceAllowlist      bool     `json:"enforce_allowlist,omitempty"`
}

// SubstituteCandidate is an available SKU evaluated for brand-family substitution.
type SubstituteCandidate struct {
	SKU          string `json:"sku"`
	SupplierID   string `json:"supplier_id,omitempty"`
	BrandID      string `json:"brand_id"`
	Name         string `json:"name,omitempty"`
	AvailableQty int64  `json:"available_qty"`
	PriceMinor   int64  `json:"price_minor"`
}

// ShortageEvaluationResult is the outcome of the shortage and substitution evaluation.
type ShortageEvaluationResult struct {
	Action               ShortageAction `json:"action"`
	Reason               string         `json:"reason"`
	OriginalSKU          string         `json:"original_sku,omitempty"`
	SubstituteSKU        string         `json:"substitute_sku,omitempty"`
	SubstituteName       string         `json:"substitute_name,omitempty"`
	SubstitutePriceMinor int64          `json:"substitute_price_minor,omitempty"`
	PriceGapMinor        int64          `json:"price_gap_minor"` // exact int64: (SubstitutePriceMinor - OriginalPriceMinor) * RequestedQty
}

// EvaluateLineSubstitution enforces PegasusX B2B Operating Model Invariants:
//  1. In stock: Line is fulfilled if stock >= requested.
//  2. If stock < requested and AllowSubstitutes is true:
//     Matches candidates strictly within the same BrandID family (and SupplierID when provided,
//     plus retailer allowlist when specified) with sufficient stock, computing integer PriceGapMinor
//     as (cand.PriceMinor - req.OriginalPriceMinor) * req.RequestedQty.
//  3. If no substitute is available or allowed:
//     - If MustHave is true: HOLD_ORDER (order waits; does not ride truck half-complete).
//     - If MustHave is false: CUT_LINE with reason OUT_OF_STOCK_SUBSTITUTE_DISALLOWED (or no family substitute).
func EvaluateLineSubstitution(req ShortageEvaluationRequest, candidates []SubstituteCandidate) ShortageEvaluationResult {
	reqSKU := strings.TrimSpace(req.SKU)
	if req.AvailableQty >= req.RequestedQty && req.RequestedQty > 0 {
		return ShortageEvaluationResult{
			Action:      ShortageActionFulfill,
			Reason:      "IN_STOCK",
			OriginalSKU: reqSKU,
		}
	}

	brandID := strings.TrimSpace(req.BrandID)
	reqSupplier := strings.TrimSpace(req.SupplierID)
	hasAllowlist := req.EnforceAllowlist || len(req.AllowedSubstituteSKUs) > 0
	var allowSet map[string]struct{}
	if hasAllowlist {
		allowSet = make(map[string]struct{}, len(req.AllowedSubstituteSKUs))
		for _, allowed := range req.AllowedSubstituteSKUs {
			if trimmed := strings.TrimSpace(allowed); trimmed != "" {
				allowSet[trimmed] = struct{}{}
			}
		}
	}

	if req.AllowSubstitutes && brandID != "" && req.RequestedQty > 0 && len(candidates) > 0 {
		valid := make([]SubstituteCandidate, 0, len(candidates))
		for _, cand := range candidates {
			candSKU := strings.TrimSpace(cand.SKU)
			if candSKU == "" || strings.EqualFold(candSKU, reqSKU) {
				continue
			}
			candBrand := strings.TrimSpace(cand.BrandID)
			if !strings.EqualFold(candBrand, brandID) {
				continue
			}
			if reqSupplier != "" && strings.TrimSpace(cand.SupplierID) != reqSupplier {
				continue
			}
			if hasAllowlist {
				if _, ok := allowSet[candSKU]; !ok {
					continue
				}
			}
			if cand.AvailableQty < req.RequestedQty {
				continue
			}
			valid = append(valid, cand)
		}
		if len(valid) > 0 {
			sort.Slice(valid, func(i, j int) bool {
				diffI := valid[i].PriceMinor - req.OriginalPriceMinor
				if diffI < 0 {
					diffI = -diffI
				}
				diffJ := valid[j].PriceMinor - req.OriginalPriceMinor
				if diffJ < 0 {
					diffJ = -diffJ
				}
				if diffI != diffJ {
					return diffI < diffJ
				}
				if valid[i].PriceMinor != valid[j].PriceMinor {
					return valid[i].PriceMinor < valid[j].PriceMinor
				}
				return valid[i].SKU < valid[j].SKU
			})
			best := valid[0]
			priceGapMinor := (best.PriceMinor - req.OriginalPriceMinor) * req.RequestedQty
			return ShortageEvaluationResult{
				Action:               ShortageActionSubstitute,
				Reason:               "SAME_BRAND_FAMILY_SUBSTITUTE",
				OriginalSKU:          reqSKU,
				SubstituteSKU:        strings.TrimSpace(best.SKU),
				SubstituteName:       strings.TrimSpace(best.Name),
				SubstitutePriceMinor: best.PriceMinor,
				PriceGapMinor:        priceGapMinor,
			}
		}
	}

	if req.MustHave {
		return ShortageEvaluationResult{
			Action:      ShortageActionHoldOrder,
			Reason:      "MUST_HAVE_LINE_UNFILLED",
			OriginalSKU: reqSKU,
		}
	}

	reason := "OUT_OF_STOCK_SUBSTITUTE_DISALLOWED"
	if req.AllowSubstitutes {
		reason = "OUT_OF_STOCK_NO_FAMILY_SUBSTITUTE_FOUND"
	}

	return ShortageEvaluationResult{
		Action:      ShortageActionCutLine,
		Reason:      reason,
		OriginalSKU: reqSKU,
	}
}

// OrderShortageOptions configures order-level shortage evaluation across lines.
type OrderShortageOptions struct {
	SupplierID       string
	EnforceAllowlist bool
}

// OrderShortagePlan summarizes line adjustments, order hold state, and recomputed totals.
type OrderShortagePlan struct {
	Lines             []LineItem
	Substitutions     []ShortageEvaluationResult
	CutLines          []ShortageEvaluationResult
	HoldOrder         bool
	HoldReason        string
	UpdatedTotalMinor int64
	HasModifications  bool
}

// EvaluateOrderShortages evaluates all lines in order using remaining available inventory per SKU
// so multiple lines do not double-allocate the same stock or substitute SKU.
func EvaluateOrderShortages(
	items []LineItem,
	availableBySKU map[string]int64,
	candidatesBySKU map[string][]SubstituteCandidate,
	opts OrderShortageOptions,
) OrderShortagePlan {
	remainingStock := make(map[string]int64, len(availableBySKU))
	for sku, qty := range availableBySKU {
		remainingStock[strings.TrimSpace(sku)] = qty
	}

	plan := OrderShortagePlan{
		Lines: make([]LineItem, 0, len(items)),
	}

	var recomputedTotal int64
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		avail := remainingStock[sku]

		// Inject live remaining stock for candidates before evaluating.
		rawCands := candidatesBySKU[sku]
		adjustedCands := make([]SubstituteCandidate, 0, len(rawCands))
		for _, cand := range rawCands {
			candSKU := strings.TrimSpace(cand.SKU)
			if liveAvail, ok := remainingStock[candSKU]; ok {
				cand.AvailableQty = liveAvail
			} else if candSKU != "" {
				remainingStock[candSKU] = cand.AvailableQty
			}
			adjustedCands = append(adjustedCands, cand)
		}

		res := EvaluateLineSubstitution(ShortageEvaluationRequest{
			SKU:                   sku,
			SupplierID:            opts.SupplierID,
			BrandID:               item.BrandID,
			RequestedQty:          item.Quantity,
			AvailableQty:          avail,
			AllowSubstitutes:      item.AllowSubstitutes,
			MustHave:              item.MustHave,
			OriginalPriceMinor:    item.UnitPrice,
			AllowedSubstituteSKUs: item.AllowedSubstituteSKUs,
			EnforceAllowlist:      opts.EnforceAllowlist,
		}, adjustedCands)

		switch res.Action {
		case ShortageActionFulfill:
			remainingStock[sku] = avail - item.Quantity
			updated := item
			updated.ShortageAction = ShortageActionFulfill
			recomputedTotal += updated.UnitPrice * updated.Quantity
			plan.Lines = append(plan.Lines, updated)

		case ShortageActionSubstitute:
			remainingStock[res.SubstituteSKU] -= item.Quantity
			updated := item
			updated.OriginalSKU = sku
			updated.SKU = res.SubstituteSKU
			updated.SubstituteSKU = res.SubstituteSKU
			if res.SubstituteName != "" {
				updated.Name = res.SubstituteName
			}
			updated.UnitPrice = res.SubstitutePriceMinor
			updated.PriceGapMinor = res.PriceGapMinor
			updated.ShortageAction = ShortageActionSubstitute
			updated.SubstitutionReason = res.Reason
			recomputedTotal += res.SubstitutePriceMinor * updated.Quantity
			plan.Lines = append(plan.Lines, updated)
			plan.Substitutions = append(plan.Substitutions, res)
			plan.HasModifications = true

		case ShortageActionHoldOrder:
			plan.HoldOrder = true
			plan.HoldReason = fmt.Sprintf("%s: %s", res.Reason, sku)
			plan.HasModifications = true
			// Retain untouched items on held orders so warehouse/retailer can resolve without losing lines.
			plan.Lines = append([]LineItem(nil), items...)
			var heldTotal int64
			for _, orig := range items {
				heldTotal += orig.UnitPrice * orig.Quantity
			}
			plan.UpdatedTotalMinor = heldTotal
			return plan

		case ShortageActionCutLine:
			plan.CutLines = append(plan.CutLines, res)
			plan.HasModifications = true
		}
	}

	plan.UpdatedTotalMinor = recomputedTotal
	return plan
}
