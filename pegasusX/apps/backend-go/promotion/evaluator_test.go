package promotion

import (
	"testing"
	"time"
)

func TestPickBestPromotion_MinLineQuantity(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	promos := []Promotion{{
		PromotionID:     "p1",
		Tiers:           []PromotionTier{{MinQuantity: 10, DiscountBps: 1000}},
		ScopeType:       ScopeTypeProduct,
		ScopeProductID:  "prod-1",
		RetailerScope:   RetailerScopeAll,

		IsActive: true,
	}}

	best, unit, _ := PickBestPromotion(now, "ret-1", "prod-1", "cat-1", 5, 10000, 50000, promos)
	if best != nil {
		t.Fatalf("expected no promo below min quantity")
	}
	if unit != 10000 {
		t.Fatalf("unit=%d want list price", unit)
	}

	best, unit, _ = PickBestPromotion(now, "ret-1", "prod-1", "cat-1", 10, 10000, 100000, promos)
	if best == nil || unit != 9000 {
		t.Fatalf("expected 10%% discount at quantity threshold, unit=%d", unit)
	}
}

func TestApplyQuote_OrderThreshold(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	promos := []Promotion{{
		PromotionID:         "p2",
		Tiers: []PromotionTier{{MinQuantity: 1, DiscountBps: 500}},
		ScopeType:           ScopeTypeAllProducts,
		RetailerScope:       RetailerScopeAll,
		MinOrderAmountMinor: 100000,
		IsActive:            true,
	}}
	lines := []LineInput{{ProductID: "a", CategoryID: "c", Quantity: 2, UnitPrice: 40000, Currency: "UZS"}}
	quote, err := ApplyQuote(now, "sup-1", "ret-1", lines, promos)
	if err != nil {
		t.Fatal(err)
	}
	if quote.DiscountMinor != 0 {
		t.Fatalf("discount should be zero below order threshold: %d", quote.DiscountMinor)
	}

	lines[0].Quantity = 3
	quote, err = ApplyQuote(now, "sup-1", "ret-1", lines, promos)
	if err != nil {
		t.Fatal(err)
	}
	if quote.DiscountMinor <= 0 {
		t.Fatalf("expected discount above order threshold")
	}
}
