package order

import (
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/stretchr/testify/require"
)

func TestValidateLineQuantities_MinMax(t *testing.T) {
	min := int64(2)
	max := int64(10)
	policy := WarehouseOpsPolicy{
		OrderLineMinQuantity: &min,
		OrderLineMaxQuantity: &max,
	}
	errs := ValidateLineQuantities([]LineItem{
		{SKU: "sku-ok", Quantity: 5},
		{SKU: "sku-low", Quantity: 1},
		{SKU: "sku-high", Quantity: 11},
	}, policy)
	require.Len(t, errs, 2)
	require.Contains(t, errs["sku-low"], "minimum")
	require.Contains(t, errs["sku-high"], "maximum")
}

func TestComputeOrderDeliveryFee_Tiers(t *testing.T) {
	km5 := 5.0
	rules := &DeliveryFeeRules{
		Currency:     "UZS",
		BaseFeeMinor: 0,
		Tiers: []DeliveryFeeTier{
			{MaxKm: &km5, FeeMinor: 0},
			{MaxKm: nil, FeeMinor: 100000},
		},
	}
	policy := WarehouseOpsPolicy{
		Lat:              41.2995,
		Lng:              69.2401,
		DeliveryFeeRules: rules,
	}
	nearFee, _ := ComputeOrderDeliveryFee(policy, 41.30, 69.24)
	require.Equal(t, int64(0), nearFee)
	farFee, dist := ComputeOrderDeliveryFee(policy, 41.50, 69.50)
	require.Equal(t, int64(100000), farFee)
	require.Greater(t, dist, 5.0)
}

func TestClassifyDelivery_WarehouseLeadWindow(t *testing.T) {
	now := proximity.TashkentTodayStart(time.Now().UTC())
	tooSoon := now.AddDate(0, 0, 2)
	okDate := now.AddDate(0, 0, 4)
	loc := proximity.TashkentLocation

	_, _, _, _, _, err := ClassifyDelivery(now, loc, DeliveryModeScheduled, &tooSoon, nil, 3, 90)
	if err == nil {
		t.Errorf("expected error for T+2 scheduling against 3-day lead")
	}

	source, status, _, _, _, err := ClassifyDelivery(now, loc, DeliveryModeScheduled, &okDate, nil, 3, 90)
	if err != nil {
		t.Fatalf("unexpected error for T+4 scheduling: %v", err)
	}
	require.Equal(t, OrderSourceManualPreorder, source)
	require.Equal(t, StatusScheduled, status)
}

func TestEffectiveMaxQuantity_CapsByLineMax(t *testing.T) {
	max := int64(50)
	policy := WarehouseOpsPolicy{OrderLineMaxQuantity: &max}
	require.Equal(t, int64(50), EffectiveMaxQuantity(100, policy))
	require.Equal(t, int64(30), EffectiveMaxQuantity(30, policy))
}

func TestComputeOrderableQuantities_MinStockBlock(t *testing.T) {
	min := int64(10)
	policy := WarehouseOpsPolicy{OrderLineMinQuantity: &min}
	orderable, errs := ComputeOrderableQuantities(map[string]int64{"sku-a": 5}, policy)
	require.Equal(t, int64(0), orderable["sku-a"])
	require.Contains(t, errs["sku-a"], "minimum")
}

func TestComputeOrderableQuantities_RejectVsBackorder(t *testing.T) {
	max := int64(50)
	policy := WarehouseOpsPolicy{
		DefaultOutOfStockPolicy: outOfStockPolicyReject,
		OrderLineMaxQuantity:    &max,
	}
	avail := map[string]int64{"sku-a": 20}
	perSKU := map[string]string{"sku-a": outOfStockPolicyReject}
	orderable, errs := ComputeOrderableQuantitiesForPolicy(avail, perSKU, policy)
	require.Empty(t, errs)
	require.Equal(t, int64(20), orderable["sku-a"])

	policy.DefaultOutOfStockPolicy = outOfStockPolicyAcceptBackorder
	perSKU["sku-a"] = outOfStockPolicyAcceptBackorder
	orderable, errs = ComputeOrderableQuantitiesForPolicy(avail, perSKU, policy)
	require.Empty(t, errs)
	require.Equal(t, int64(50), orderable["sku-a"])
}
