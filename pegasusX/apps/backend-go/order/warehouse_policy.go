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
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

var (
	ErrLineQuantityOutOfRange = errors.New("line_quantity_out_of_range")
	ErrOrderAcceptanceClosed  = errors.New("order_acceptance_closed")
)

const uncappedOrderableQuantity int64 = 99999

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
	OperatingSchedule          OperatingSchedule
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
			"OperatingSchedule",
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
		scheduleRaw                spanner.NullJSON
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
		&scheduleRaw,
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
	if scheduleRaw.Valid {
		policy.OperatingSchedule = ParseOperatingSchedule(json.RawMessage(scheduleRaw.String()))
	}
	if policy.DefaultOutOfStockPolicy == "" {
		policy.DefaultOutOfStockPolicy = outOfStockPolicyReject
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
		if cur, err := auth.CurrencyFromContext(context.Background(), ""); err == nil {
			rules.Currency = cur
		}
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

// EffectiveMaxQuantity returns min(available, warehouse line max) for UI caps.
func EffectiveMaxQuantity(available int64, policy WarehouseOpsPolicy) int64 {
	if available < 0 {
		available = 0
	}
	if policy.OrderLineMaxQuantity != nil && *policy.OrderLineMaxQuantity < available {
		return *policy.OrderLineMaxQuantity
	}
	return available
}

// ComputeOrderableQuantities derives per-SKU checkout caps using warehouse default REJECT policy.
func ComputeOrderableQuantities(available map[string]int64, policy WarehouseOpsPolicy) (map[string]int64, map[string]string) {
	perSKU := make(map[string]string, len(available))
	def := policy.DefaultOutOfStockPolicy
	if def == "" {
		def = outOfStockPolicyReject
	}
	for sku := range available {
		perSKU[sku] = def
	}
	return ComputeOrderableQuantitiesForPolicy(available, perSKU, policy)
}

// ComputeOrderableQuantitiesForPolicy caps by stock when REJECT, or line max only when ACCEPT_BACKORDER.
func ComputeOrderableQuantitiesForPolicy(
	available map[string]int64,
	perSKUPolicy map[string]string,
	policy WarehouseOpsPolicy,
) (map[string]int64, map[string]string) {
	orderable := make(map[string]int64, len(available))
	lineErrs := make(map[string]string)
	warehouseDefault := policy.DefaultOutOfStockPolicy
	if warehouseDefault == "" {
		warehouseDefault = outOfStockPolicyReject
	}
	for sku, avail := range available {
		skuPolicy := resolveOutOfStockPolicy(warehouseDefault, perSKUPolicy[sku])
		if skuPolicy == outOfStockPolicyAcceptBackorder {
			cap := uncappedOrderableQuantity
			if policy.OrderLineMaxQuantity != nil {
				cap = *policy.OrderLineMaxQuantity
			}
			orderable[sku] = cap
			continue
		}
		effective := EffectiveMaxQuantity(avail, policy)
		if policy.OrderLineMinQuantity != nil && effective < *policy.OrderLineMinQuantity {
			lineErrs[sku] = fmt.Sprintf("only %d available; minimum quantity is %d", effective, *policy.OrderLineMinQuantity)
			orderable[sku] = 0
			continue
		}
		orderable[sku] = effective
	}
	if len(lineErrs) == 0 {
		lineErrs = nil
	}
	return orderable, lineErrs
}

func lineMaxOnlyCap(policy WarehouseOpsPolicy) int64 {
	if policy.OrderLineMaxQuantity != nil {
		return *policy.OrderLineMaxQuantity
	}
	return uncappedOrderableQuantity
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
	distanceKm = proximity.HaversineDistance(policy.Lat, policy.Lng, deliveryLat, deliveryLng)
	return computeDeliveryFeeMinor(policy.DeliveryFeeRules, distanceKm), distanceKm
}
