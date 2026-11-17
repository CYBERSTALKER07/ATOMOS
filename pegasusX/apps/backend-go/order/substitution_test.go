package order

import (
	"testing"
)

func TestEvaluateLineSubstitution_DisallowedCutsLine(t *testing.T) {
	req := ShortageEvaluationRequest{
		SKU:              "SKU-COCA-1L",
		BrandID:          "BRAND-COCA",
		RequestedQty:     10,
		AvailableQty:     0,
		AllowSubstitutes: false,
		MustHave:         false,
	}

	res := EvaluateLineSubstitution(req, nil)
	if res.Action != ShortageActionCutLine {
		t.Fatalf("want action %s, got %s", ShortageActionCutLine, res.Action)
	}
	if res.Reason != "OUT_OF_STOCK_SUBSTITUTE_DISALLOWED" {
		t.Fatalf("want reason OUT_OF_STOCK_SUBSTITUTE_DISALLOWED, got %s", res.Reason)
	}
}

func TestEvaluateLineSubstitution_MustHaveUnfilledHoldsOrder(t *testing.T) {
	req := ShortageEvaluationRequest{
		SKU:              "SKU-COCA-1L",
		BrandID:          "BRAND-COCA",
		RequestedQty:     10,
		AvailableQty:     0,
		AllowSubstitutes: false,
		MustHave:         true,
	}

	res := EvaluateLineSubstitution(req, nil)
	if res.Action != ShortageActionHoldOrder {
		t.Fatalf("want action %s, got %s", ShortageActionHoldOrder, res.Action)
	}
	if res.Reason != "MUST_HAVE_LINE_UNFILLED" {
		t.Fatalf("want reason MUST_HAVE_LINE_UNFILLED, got %s", res.Reason)
	}
}

func TestEvaluateLineSubstitution_AllowedFindsBrandCandidate(t *testing.T) {
	req := ShortageEvaluationRequest{
		SKU:                "SKU-COCA-1L",
		BrandID:            "BRAND-COCA",
		RequestedQty:       10,
		AvailableQty:       0,
		AllowSubstitutes:   true,
		OriginalPriceMinor: 10000,
	}

	candidates := []SubstituteCandidate{
		{
			SKU:          "SKU-PEPSI-1L",
			BrandID:      "BRAND-PEPSI", // different brand!
			AvailableQty: 20,
			PriceMinor:   9500,
		},
		{
			SKU:          "SKU-COCA-05L",
			BrandID:      "BRAND-COCA", // same brand family
			AvailableQty: 20,
			PriceMinor:   6000,
		},
	}

	res := EvaluateLineSubstitution(req, candidates)
	if res.Action != ShortageActionSubstitute {
		t.Fatalf("want action %s, got %s", ShortageActionSubstitute, res.Action)
	}
	if res.SubstituteSKU != "SKU-COCA-05L" {
		t.Fatalf("want substitute SKU-COCA-05L, got %s", res.SubstituteSKU)
	}
	// Price gap: (6000 - 10000) * 10 = -40000 (total credit to shop)
	if res.PriceGapMinor != -40000 {
		t.Fatalf("want price gap -40000, got %d", res.PriceGapMinor)
	}
}

func TestEvaluateLineSubstitution_EnforcesAllowlistAndSupplier(t *testing.T) {
	req := ShortageEvaluationRequest{
		SKU:                   "SKU-COCA-1L",
		SupplierID:            "sup-1",
		BrandID:               "BRAND-COCA",
		RequestedQty:          5,
		AvailableQty:          0,
		AllowSubstitutes:      true,
		OriginalPriceMinor:    10000,
		AllowedSubstituteSKUs: []string{"SKU-COCA-2L"},
		EnforceAllowlist:      true,
	}

	candidates := []SubstituteCandidate{
		{
			SKU:          "SKU-COCA-05L",
			SupplierID:   "sup-1",
			BrandID:      "BRAND-COCA",
			AvailableQty: 50,
			PriceMinor:   9000, // Not in allowlist
		},
		{
			SKU:          "SKU-COCA-2L",
			SupplierID:   "sup-2", // Wrong supplier
			BrandID:      "BRAND-COCA",
			AvailableQty: 50,
			PriceMinor:   11000,
		},
		{
			SKU:          "SKU-COCA-2L",
			SupplierID:   "sup-1",
			BrandID:      "BRAND-COCA",
			Name:         "Coca-Cola 2L",
			AvailableQty: 10,
			PriceMinor:   12000,
		},
	}

	res := EvaluateLineSubstitution(req, candidates)
	if res.Action != ShortageActionSubstitute {
		t.Fatalf("want SUBSTITUTE, got %s (%s)", res.Action, res.Reason)
	}
	if res.SubstituteSKU != "SKU-COCA-2L" {
		t.Fatalf("want SKU-COCA-2L, got %s", res.SubstituteSKU)
	}
	if res.PriceGapMinor != 10000 {
		t.Fatalf("want PriceGapMinor (12000-10000)*5 = 10000, got %d", res.PriceGapMinor)
	}

	// Empty allowlist with EnforceAllowlist blocks substitution and holds MustHave order.
	req.AllowedSubstituteSKUs = nil
	req.MustHave = true
	resBlocked := EvaluateLineSubstitution(req, candidates)
	if resBlocked.Action != ShortageActionHoldOrder {
		t.Fatalf("want HOLD_ORDER when allowlist empty and MustHave=true, got %s", resBlocked.Action)
	}
}

func TestEvaluateOrderShortages_RecomputesTotalOnCutAndSubstitute(t *testing.T) {
	items := []LineItem{
		{SKU: "SKU-A", Name: "Brand A", BrandID: "BRAND-1", Quantity: 4, UnitPrice: 2500, AllowSubstitutes: true},
		{SKU: "SKU-B", Name: "Brand B", BrandID: "BRAND-2", Quantity: 3, UnitPrice: 3000, AllowSubstitutes: false, MustHave: false},
		{SKU: "SKU-C", Name: "Brand C", BrandID: "BRAND-3", Quantity: 2, UnitPrice: 5000, AllowSubstitutes: false, MustHave: false},
	}
	avail := map[string]int64{
		"SKU-A":     0,
		"SKU-A-ALT": 10,
		"SKU-B":     1, // short -> cut
		"SKU-C":     5, // fulfill
	}
	cands := map[string][]SubstituteCandidate{
		"SKU-A": {
			{SKU: "SKU-A-ALT", SupplierID: "sup-1", BrandID: "BRAND-1", Name: "Brand A Alt", AvailableQty: 10, PriceMinor: 2200},
		},
	}

	plan := EvaluateOrderShortages(items, avail, cands, OrderShortageOptions{SupplierID: "sup-1"})
	if plan.HoldOrder {
		t.Fatalf("did not expect HOLD_ORDER")
	}
	if len(plan.Substitutions) != 1 || plan.Substitutions[0].PriceGapMinor != -1200 {
		t.Fatalf("unexpected substitutions: %+v", plan.Substitutions)
	}
	if len(plan.CutLines) != 1 || plan.CutLines[0].Reason != "OUT_OF_STOCK_SUBSTITUTE_DISALLOWED" {
		t.Fatalf("unexpected cut lines: %+v", plan.CutLines)
	}
	// SKU-A substituted at 4*2200=8800 + SKU-C 2*5000=10000 = 18800
	if plan.UpdatedTotalMinor != 18800 {
		t.Fatalf("want recomputed total 18800, got %d", plan.UpdatedTotalMinor)
	}
	if len(plan.Lines) != 2 {
		t.Fatalf("want 2 surviving lines, got %d", len(plan.Lines))
	}
}
