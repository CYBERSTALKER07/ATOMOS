package schemadrift

import (
	"strings"
	"testing"
)

func TestShopClosedOrdersColumns_coverGraceProximity(t *testing.T) {
	need := map[string]bool{
		"ShopClosedAt":           false,
		"ShopClosedGraceEndsAt":  false,
		"PartialDelivery":        false,
		"ProximityUnlockedAt":    false,
		"ProximityMethod":        false,
		"ShopClosedReason":       false,
		"ShopClosedResolution":   false,
	}
	for _, col := range ShopClosedOrdersColumns {
		if _, ok := need[col]; ok {
			need[col] = true
		}
	}
	var missing []string
	for col, ok := range need {
		if !ok {
			missing = append(missing, col)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("ShopClosedOrdersColumns missing required keys: %s", strings.Join(missing, ", "))
	}
}
