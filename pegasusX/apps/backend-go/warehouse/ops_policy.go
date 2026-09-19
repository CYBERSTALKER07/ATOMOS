package warehouse

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	defaultPreorderMinLeadDays = int64(3)
	defaultPreorderMaxLeadDays = int64(90)
)

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
	OperatingSchedule          map[string]any
	Lat                        float64
	Lng                        float64
}

func normalizePreorderLeadDays(minLead, maxLead int64) (int64, int64, error) {
	if minLead <= 0 {
		minLead = defaultPreorderMinLeadDays
	}
	if maxLead <= 0 {
		maxLead = defaultPreorderMaxLeadDays
	}
	if minLead > maxLead {
		return 0, 0, fmt.Errorf("preorder_min_lead_days must be <= preorder_max_lead_days")
	}
	return minLead, maxLead, nil
}

func validateOrderLineLimits(minQty, maxQty *int64) error {
	if minQty != nil && *minQty < 1 {
		return errors.New("order_line_min_quantity must be >= 1")
	}
	if maxQty != nil && *maxQty < 1 {
		return errors.New("order_line_max_quantity must be >= 1")
	}
	if minQty != nil && maxQty != nil && *minQty > *maxQty {
		return errors.New("order_line_min_quantity must be <= order_line_max_quantity")
	}
	return nil
}

func parseDeliveryFeeRules(raw json.RawMessage) (*DeliveryFeeRules, error) {
	return ParseDeliveryFeeRulesJSON(raw)
}

// ParseDeliveryFeeRulesJSON unmarshals and validates delivery fee rules JSON.
func ParseDeliveryFeeRulesJSON(raw json.RawMessage) (*DeliveryFeeRules, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rules DeliveryFeeRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, fmt.Errorf("invalid delivery_fee_rules: %w", err)
	}
	if err := validateDeliveryFeeRules(&rules); err != nil {
		return nil, err
	}
	return &rules, nil
}

func validateDeliveryFeeRules(rules *DeliveryFeeRules) error {
	if rules == nil {
		return nil
	}
	rules.Currency = strings.TrimSpace(rules.Currency)
	if rules.Currency == "" {
		rules.Currency = packCurrencyDefault()
	}
	if rules.BaseFeeMinor < 0 {
		return errors.New("delivery_fee_rules.base_fee_minor must be >= 0")
	}
	type tierSort struct {
		idx int
		km  float64
	}
	sorted := make([]tierSort, 0, len(rules.Tiers))
	for i, t := range rules.Tiers {
		if t.FeeMinor < 0 {
			return errors.New("delivery_fee_rules.tiers.fee_minor must be >= 0")
		}
		if t.MaxKm != nil && *t.MaxKm < 0 {
			return errors.New("delivery_fee_rules.tiers.max_km must be >= 0")
		}
		km := math.MaxFloat64
		if t.MaxKm != nil {
			km = *t.MaxKm
		}
		sorted = append(sorted, tierSort{idx: i, km: km})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].km == sorted[j].km {
			return sorted[i].idx < sorted[j].idx
		}
		return sorted[i].km < sorted[j].km
	})
	for i := 1; i < len(sorted); i++ {
		if sorted[i].km == sorted[i-1].km && sorted[i].km != math.MaxFloat64 {
			return errors.New("delivery_fee_rules.tiers must have unique max_km values")
		}
	}
	return nil
}

// ComputeDeliveryFeeMinor returns surcharge for distanceKm using warehouse rules.
func ComputeDeliveryFeeMinor(rules *DeliveryFeeRules, distanceKm float64) int64 {
	if rules == nil {
		return 0
	}
	if distanceKm < 0 {
		distanceKm = 0
	}
	fee := rules.BaseFeeMinor
	for _, tier := range rules.Tiers {
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
