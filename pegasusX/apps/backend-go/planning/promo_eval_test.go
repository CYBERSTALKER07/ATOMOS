package planning

import (
	"encoding/json"
	"testing"
)

func TestProjectPromoPandL_DefaultElasticity(t *testing.T) {
	got := projectPromoPandL(PromoSimulateInput{
		PromotionID:   "promo-1",
		DiscountPct:   20,
		ExpectedUnits: 1000,
		AvgUnitMargin: 500,
	})
	if got.ElasticityUsed != DefaultPromoElasticity {
		t.Fatalf("ElasticityUsed=%v want %v", got.ElasticityUsed, DefaultPromoElasticity)
	}
	// volumeLift = 1 + min(20,50)/100 * 0.5 = 1.1 → 1100
	if got.ProjectedVolume != 1100 {
		t.Fatalf("ProjectedVolume=%d want 1100", got.ProjectedVolume)
	}
	if !got.SandboxOnly {
		t.Fatal("SandboxOnly must remain true")
	}
}

func TestProjectPromoPandL_CustomElasticityChangesVolume(t *testing.T) {
	base := projectPromoPandL(PromoSimulateInput{
		DiscountPct:   20,
		ExpectedUnits: 1000,
		AvgUnitMargin: 500,
	})
	high := projectPromoPandL(PromoSimulateInput{
		DiscountPct:   20,
		ExpectedUnits: 1000,
		AvgUnitMargin: 500,
		Elasticity:    1.0,
	})
	if high.ElasticityUsed != 1.0 {
		t.Fatalf("ElasticityUsed=%v want 1.0", high.ElasticityUsed)
	}
	// volumeLift = 1 + 0.20*1.0 = 1.2 → 1200
	if high.ProjectedVolume != 1200 {
		t.Fatalf("ProjectedVolume=%d want 1200", high.ProjectedVolume)
	}
	if high.ProjectedVolume <= base.ProjectedVolume {
		t.Fatalf("higher elasticity must lift volume: high=%d base=%d", high.ProjectedVolume, base.ProjectedVolume)
	}
}

func TestAggregatePromoLines_EmptyPromotionID(t *testing.T) {
	raw, _ := json.Marshal([]promoLineJSON{
		{PromotionID: "promo-a", Quantity: 10, UnitPriceMinor: 100},
	})
	v, m := aggregatePromoLines(raw, "", 1000)
	if v != 0 || m != 0 {
		t.Fatalf("empty promotionID must return zeros, got volume=%d margin=%d", v, m)
	}
}

func TestAggregatePromoLines_OnlyMatchingPromotion(t *testing.T) {
	raw, _ := json.Marshal([]promoLineJSON{
		{PromotionID: "promo-a", Quantity: 10, UnitPriceMinor: 100},
		{PromotionID: "promo-b", Quantity: 5, UnitPriceMinor: 200},
		{PromotionID: "promo-a", Quantity: 3, UnitPriceMinor: 50},
	})
	v, m := aggregatePromoLines(raw, "promo-a", 9999)
	if v != 13 {
		t.Fatalf("volume=%d want 13", v)
	}
	// 10*100 + 3*50 = 1150
	if m != 1150 {
		t.Fatalf("margin=%d want 1150", m)
	}
}

func TestAggregatePromoLines_WrongPromotion(t *testing.T) {
	raw, _ := json.Marshal([]promoLineJSON{
		{PromotionID: "promo-a", Quantity: 10, UnitPriceMinor: 100},
	})
	v, m := aggregatePromoLines(raw, "promo-other", 5000)
	if v != 0 || m != 0 {
		t.Fatalf("wrong promotion must return zeros, got volume=%d margin=%d", v, m)
	}
}

func TestAggregatePromoLines_ProportionalFallback(t *testing.T) {
	raw, _ := json.Marshal([]promoLineJSON{
		{PromotionID: "promo-a", Quantity: 2},
		{PromotionID: "", Quantity: 8},
	})
	v, m := aggregatePromoLines(raw, "promo-a", 1000)
	if v != 2 {
		t.Fatalf("volume=%d want 2", v)
	}
	// 1000 * 2/10 = 200
	if m != 200 {
		t.Fatalf("margin=%d want 200 (proportional)", m)
	}
}

func TestAggregatePromoLines_LineTotalPreferred(t *testing.T) {
	raw, _ := json.Marshal([]promoLineJSON{
		{PromotionID: "promo-a", Quantity: 4, UnitPriceMinor: 100, LineTotalMinor: 350},
	})
	v, m := aggregatePromoLines(raw, "promo-a", 0)
	if v != 4 || m != 350 {
		t.Fatalf("want volume=4 margin=350, got volume=%d margin=%d", v, m)
	}
}
