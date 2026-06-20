package warehouse

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
)

// LoadOpsPolicy reads warehouse ops policy for handlers and order enforcement.
func LoadOpsPolicy(ctx context.Context, client *spanner.Client, warehouseID string) (WarehouseOpsPolicy, error) {
	if client == nil {
		return WarehouseOpsPolicy{}, fmt.Errorf("spanner client required")
	}
	row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{
			"WarehouseId", "Name", "RegionId", "Lat", "Lng",
			"DefaultOutOfStockPolicy", "ShowStockCountsToRetailers", "IsOnShift",
			"PreorderMinLeadDays", "PreorderMaxLeadDays",
			"OrderLineMinQuantity", "OrderLineMaxQuantity", "DeliveryFeeRules",
			"OperatingSchedule",
		})
	if err != nil {
		return WarehouseOpsPolicy{}, err
	}
	return scanWarehouseOpsPolicyRow(row)
}

func scanWarehouseOpsPolicyRow(row *spanner.Row) (WarehouseOpsPolicy, error) {
	var (
		policy                     WarehouseOpsPolicy
		regionID                   spanner.NullString
		defaultPolicy              spanner.NullString
		orderLineMin, orderLineMax spanner.NullInt64
		feeRulesRaw                spanner.NullJSON
		schedule                   spanner.NullJSON
		lat, lng                   spanner.NullFloat64
	)
	if err := row.Columns(
		&policy.WarehouseID,
		&policy.Name,
		&regionID,
		&lat, &lng,
		&defaultPolicy,
		&policy.ShowStockCountsToRetailers,
		&policy.IsOnShift,
		&policy.PreorderMinLeadDays,
		&policy.PreorderMaxLeadDays,
		&orderLineMin,
		&orderLineMax,
		&feeRulesRaw,
		&schedule,
	); err != nil {
		return WarehouseOpsPolicy{}, err
	}
	policy.RegionID = regionID.StringVal
	policy.DefaultOutOfStockPolicy = ResolveOutOfStockPolicy(defaultPolicy.StringVal, "")
	if lat.Valid {
		policy.Lat = lat.Float64
	}
	if lng.Valid {
		policy.Lng = lng.Float64
	}
	if orderLineMin.Valid {
		v := orderLineMin.Int64
		policy.OrderLineMinQuantity = &v
	}
	if orderLineMax.Valid {
		v := orderLineMax.Int64
		policy.OrderLineMaxQuantity = &v
	}
	if feeRulesRaw.Valid {
		rules, err := ParseDeliveryFeeRulesJSON([]byte(feeRulesRaw.String()))
		if err != nil {
			return WarehouseOpsPolicy{}, err
		}
		policy.DeliveryFeeRules = rules
	}
	if schedule.Valid {
		_ = json.Unmarshal([]byte(schedule.String()), &policy.OperatingSchedule)
	}
	if policy.PreorderMinLeadDays <= 0 {
		policy.PreorderMinLeadDays = defaultPreorderMinLeadDays
	}
	if policy.PreorderMaxLeadDays <= 0 {
		policy.PreorderMaxLeadDays = defaultPreorderMaxLeadDays
	}
	return policy, nil
}

func opsPolicyToJSONMap(policy WarehouseOpsPolicy, sched any, expressEnabled bool, expressStockFloor int64) map[string]any {
	out := map[string]any{
		"warehouse_id":                   policy.WarehouseID,
		"name":                           policy.Name,
		"region_id":                      policy.RegionID,
		"default_out_of_stock_policy":    policy.DefaultOutOfStockPolicy,
		"show_stock_counts_to_retailers":   policy.ShowStockCountsToRetailers,
		"operating_schedule":               sched,
		"is_on_shift":                    policy.IsOnShift,
		"ops_always_available":           true,
		"express_enabled":                expressEnabled,
		"express_stock_floor":            expressStockFloor,
		"preorder_min_lead_days":         policy.PreorderMinLeadDays,
		"preorder_max_lead_days":         policy.PreorderMaxLeadDays,
	}
	if policy.OrderLineMinQuantity != nil {
		out["order_line_min_quantity"] = *policy.OrderLineMinQuantity
	}
	if policy.OrderLineMaxQuantity != nil {
		out["order_line_max_quantity"] = *policy.OrderLineMaxQuantity
	}
	if policy.DeliveryFeeRules != nil {
		out["delivery_fee_rules"] = policy.DeliveryFeeRules
	}
	return out
}
