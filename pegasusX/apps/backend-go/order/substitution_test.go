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
		SKU:              "SKU-COCA-1L",
		BrandID:          "BRAND-COCA",
		RequestedQty:     10,
		AvailableQty:     0,
		AllowSubstitutes: true,
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
	// Price gap: 6000 - 10000 = -4000 (credit to shop)
	if res.PriceGapMinor != -4000 {
		t.Fatalf("want price gap -4000, got %d", res.PriceGapMinor)
	}
}
