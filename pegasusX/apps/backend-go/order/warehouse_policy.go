package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
)

var ErrLineQuantityOutOfRange = errors.New("line_quantity_out_of_range")

// DeliveryFeeTier is one distance band for delivery surcharges.
type DeliveryFeeTier struct {
	MaxKm    *float64 `json:"max_km,omitempty"`
	FeeMinor int64    `json:"fee_minor"`
}

// DeliveryFeeRules configures warehouse delivery surcharges by distance.
type DeliveryFeeRules struct {
	Currency     string            `json:"currency"`
	BaseFeeMinor int64             `json:"base_fee_minor"`
	Tiers        []DeliveryFeeTier `json:"tiers"`
}

// WarehouseOpsPolicy is the checkout-facing warehouse configuration slice.
type WarehouseOpsPolicy struct {
	WarehouseID                string
	Name                       string
	RegionID                   string
	DefaultOutOfStockPolicy    string
	ShowStockCountsToRetailers bool
	IsOnShift                  bool
	PreorderMinLeadDays        int64
	PreorderMaxLeadDays        int64
	OrderLineMinQuantity       *int64
	OrderLineMaxQuantity       *int64
	DeliveryFeeRules           *DeliveryFeeRules
	Lat                        float64
	Lng                        float64
}

// LoadWarehouseOpsPolicy reads warehouse checkout policy from Spanner.
func LoadWarehouseOpsPolicy(ctx context.Context, client *spanner.Client, warehouseID string) (WarehouseOpsPolicy, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if client == nil || warehouseID == "" {
		return WarehouseOpsPolicy{}, errors.New("warehouse policy: missing client or warehouse_id")
	}
	row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{
			"WarehouseId", "Name", "RegionId", "Lat", "Lng",
			"DefaultOutOfStockPolicy", "ShowStockCountsToRetailers", "IsOnShift",
			"PreorderMinLeadDays", "PreorderMaxLeadDays",
			"OrderLineMinQuantity", "OrderLineMaxQuantity", "DeliveryFeeRules",
		})
	if err != nil {
		return WarehouseOpsPolicy{}, fmt.Errorf("read warehouse %s: %w", warehouseID, err)
	}
	var (
		policy                     WarehouseOpsPolicy
		regionID                   spanner.NullString
		defaultPolicy              spanner.NullString
		orderLineMin, orderLineMax spanner.NullInt64
		feeRulesRaw                spanner.NullJSON
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
	); err != nil {
		return WarehouseOpsPolicy{}, err
	}
	policy.RegionID = regionID.StringVal
	policy.DefaultOutOfStockPolicy = strings.ToUpper(strings.TrimSpace(defaultPolicy.StringVal))
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
		rules, err := parseDeliveryFeeRulesJSON([]byte(feeRulesRaw.String()))
		if err != nil {
			return WarehouseOpsPolicy{}, err
		}
		policy.DeliveryFeeRules = rules
	}
	if policy.PreorderMinLeadDays <= 0 {
		policy.PreorderMinLeadDays = 3
	}
	if policy.PreorderMaxLeadDays <= 0 {
		policy.PreorderMaxLeadDays = 90
	}
	return policy, nil
}

func parseDeliveryFeeRulesJSON(raw json.RawMessage) (*DeliveryFeeRules, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rules DeliveryFeeRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, fmt.Errorf("invalid delivery_fee_rules: %w", err)
	}
	if rules.Currency == "" {
		rules.Currency = "UZS"
	}
	return &rules, nil
}

func computeDeliveryFeeMinor(rules *DeliveryFeeRules, distanceKm float64) int64 {
	if rules == nil {
		return 0
	}
	if distanceKm < 0 {
		distanceKm = 0
	}
	fee := rules.BaseFeeMinor
	tiers := append([]DeliveryFeeTier(nil), rules.Tiers...)
	sort.SliceStable(tiers, func(i, j int) bool {
		ki, kj := math.MaxFloat64, math.MaxFloat64
		if tiers[i].MaxKm != nil {
			ki = *tiers[i].MaxKm
		}
		if tiers[j].MaxKm != nil {
			kj = *tiers[j].MaxKm
		}
		return ki < kj
	})
	for _, tier := range tiers {
		if tier.MaxKm == nil {
			fee = tier.FeeMinor
			break
		}
		if distanceKm <= *tier.MaxKm {
			fee = tier.FeeMinor
			break
		}
	}
	if fee < 0 {
		return 0
	}
	return fee
}

// ValidateLineQuantities checks each line against warehouse min/max limits.
func ValidateLineQuantities(items []LineItem, policy WarehouseOpsPolicy) map[string]string {
	errs := make(map[string]string)
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		qty := item.Quantity
		if policy.OrderLineMinQuantity != nil && qty < *policy.OrderLineMinQuantity {
			errs[sku] = fmt.Sprintf("minimum quantity is %d", *policy.OrderLineMinQuantity)
		}
		if policy.OrderLineMaxQuantity != nil && qty > *policy.OrderLineMaxQuantity {
			errs[sku] = fmt.Sprintf("maximum quantity is %d", *policy.OrderLineMaxQuantity)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func mergeLineErrors(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// ComputeOrderDeliveryFee returns fee minor units and distance km warehouse→delivery pin.
func ComputeOrderDeliveryFee(policy WarehouseOpsPolicy, deliveryLat, deliveryLng float64) (feeMinor int64, distanceKm float64) {
	if policy.DeliveryFeeRules == nil {
		return 0, 0
	}
	if policy.Lat == 0 && policy.Lng == 0 {
		return policy.DeliveryFeeRules.BaseFeeMinor, 0
	}
	distanceKm = haversineKm(policy.Lat, policy.Lng, deliveryLat, deliveryLng)
	return computeDeliveryFeeMinor(policy.DeliveryFeeRules, distanceKm), distanceKm
}
