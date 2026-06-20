package order

import (
	"testing"
	"time"

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
	now, err := time.Parse(time.RFC3339, "2026-06-17T10:00:00Z")
	require.NoError(t, err)
	tooSoon := now.AddDate(0, 0, 1)
	_, _, _, _, _, err = ClassifyDelivery(now, DeliveryModeScheduled, &tooSoon, nil, 3, 90)
	require.Error(t, err)

	okDate := now.AddDate(0, 0, 5)
	source, status, _, _, _, err := ClassifyDelivery(now, DeliveryModeScheduled, &okDate, nil, 3, 90)
	require.NoError(t, err)
	require.Equal(t, OrderSourceManualPreorder, source)
	require.Equal(t, StatusScheduled, status)
}
