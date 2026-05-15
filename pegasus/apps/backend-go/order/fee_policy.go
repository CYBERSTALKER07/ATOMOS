package order

import (
	"context"
	"fmt"
	"strings"

	"backend-go/countrycfg"
)

const (
	regionalDegressiveFeeTierBase   = "REGIONAL_DEGRESSIVE_BASE"
	regionalDegressiveFeeTierGrowth = "REGIONAL_DEGRESSIVE_GROWTH"
	regionalDegressiveFeeTierScale  = "REGIONAL_DEGRESSIVE_SCALE"

	regionalDegressiveGrowthReductionBasisPoints int64 = 50
	regionalDegressiveScaleReductionBasisPoints  int64 = 100
)

type regionalDegressiveFeeDefaults struct {
	CurrencyCode          string
	GrowthThresholdAmount int64
	ScaleThresholdAmount  int64
	CapAmount             int64
}

type checkoutFeeComputation struct {
	PolicyVersion   string
	SelectedTierKey string
	BasisPoints     int64
	CapApplied      bool
	FeeAmount       int64
	NetPayoutAmount int64
}

func (s *OrderService) resolveRegionalDegressiveFeeDefaults(ctx context.Context, supplierIDs []string) (map[string]regionalDegressiveFeeDefaults, error) {
	if s == nil || s.CountryConfig == nil || len(supplierIDs) == 0 {
		return nil, nil
	}

	defaultsBySupplier := make(map[string]regionalDegressiveFeeDefaults, len(supplierIDs))
	defaultsByRegion := make(map[string]regionalDegressiveFeeDefaults)

	for _, supplierID := range supplierIDs {
		trimmedSupplierID := strings.TrimSpace(supplierID)
		if trimmedSupplierID == "" {
			continue
		}

		region, err := s.CountryConfig.ResolveSupplierRegion(ctx, trimmedSupplierID)
		if err != nil {
			return nil, fmt.Errorf("resolve supplier %s region: %w", trimmedSupplierID, err)
		}
		if region == nil {
			continue
		}

		if defaults, ok := defaultsByRegion[region.RegionID]; ok {
			defaultsBySupplier[trimmedSupplierID] = defaults
			continue
		}

		regionalConfig, err := s.CountryConfig.GetRegionalConfig(ctx, region.RegionID)
		if err != nil {
			return nil, fmt.Errorf("read regional config %s: %w", region.RegionID, err)
		}

		defaults, ok := buildRegionalDegressiveFeeDefaults(region, regionalConfig)
		if !ok {
			continue
		}

		defaultsByRegion[region.RegionID] = defaults
		defaultsBySupplier[trimmedSupplierID] = defaults
	}

	return defaultsBySupplier, nil
}

func buildRegionalDegressiveFeeDefaults(region *countrycfg.Region, cfg *countrycfg.RegionalConfig) (regionalDegressiveFeeDefaults, bool) {
	if region == nil || cfg == nil {
		return regionalDegressiveFeeDefaults{}, false
	}

	growthThresholdAmount := cfg.DegressiveFeeGrowthThresholdAmount
	scaleThresholdAmount := cfg.DegressiveFeeScaleThresholdAmount
	capAmount := cfg.DegressiveFeeCapAmount

	if growthThresholdAmount < 0 || scaleThresholdAmount < 0 || capAmount < 0 {
		return regionalDegressiveFeeDefaults{}, false
	}
	if growthThresholdAmount > 0 && scaleThresholdAmount > 0 && scaleThresholdAmount < growthThresholdAmount {
		return regionalDegressiveFeeDefaults{}, false
	}

	return regionalDegressiveFeeDefaults{
		CurrencyCode:          strings.ToUpper(strings.TrimSpace(region.CurrencyCode)),
		GrowthThresholdAmount: growthThresholdAmount,
		ScaleThresholdAmount:  scaleThresholdAmount,
		CapAmount:             capAmount,
	}, true
}

func computeCheckoutFee(grossAmount int64, currency string, baseFeeBasisPoints int64, policyVersion string, regionalDefaults *regionalDegressiveFeeDefaults) checkoutFeeComputation {
	if grossAmount < 0 {
		grossAmount = 0
	}

	switch normalizeFeePolicyVersion(policyVersion) {
	case FeePolicyVersionRegionalDegressiveV1:
		if computation, ok := computeRegionalDegressiveCheckoutFee(grossAmount, currency, baseFeeBasisPoints, regionalDefaults); ok {
			return computation
		}
	}

	return computeLegacyCheckoutFee(grossAmount, baseFeeBasisPoints)
}

func computeLegacyCheckoutFee(grossAmount int64, baseFeeBasisPoints int64) checkoutFeeComputation {
	basisPoints := sanitizeFeeBasisPoints(baseFeeBasisPoints)
	feeAmount := (grossAmount * basisPoints) / 10000
	if feeAmount < 0 {
		feeAmount = 0
	}
	netPayoutAmount := grossAmount - feeAmount
	if netPayoutAmount < 0 {
		netPayoutAmount = 0
	}

	return checkoutFeeComputation{
		PolicyVersion:   FeePolicyVersionLegacyCheckout,
		SelectedTierKey: FeeTierLegacyFlat,
		BasisPoints:     basisPoints,
		CapApplied:      false,
		FeeAmount:       feeAmount,
		NetPayoutAmount: netPayoutAmount,
	}
}

func computeRegionalDegressiveCheckoutFee(grossAmount int64, currency string, baseFeeBasisPoints int64, regionalDefaults *regionalDegressiveFeeDefaults) (checkoutFeeComputation, bool) {
	if regionalDefaults == nil {
		return checkoutFeeComputation{}, false
	}

	trimmedCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if trimmedCurrency == "" || trimmedCurrency != regionalDefaults.CurrencyCode {
		return checkoutFeeComputation{}, false
	}

	basisPoints := sanitizeFeeBasisPoints(baseFeeBasisPoints)
	selectedTierKey := regionalDegressiveFeeTierBase

	if regionalDefaults.ScaleThresholdAmount > 0 && grossAmount >= regionalDefaults.ScaleThresholdAmount {
		basisPoints = reduceFeeBasisPoints(basisPoints, regionalDegressiveScaleReductionBasisPoints)
		selectedTierKey = regionalDegressiveFeeTierScale
	} else if regionalDefaults.GrowthThresholdAmount > 0 && grossAmount >= regionalDefaults.GrowthThresholdAmount {
		basisPoints = reduceFeeBasisPoints(basisPoints, regionalDegressiveGrowthReductionBasisPoints)
		selectedTierKey = regionalDegressiveFeeTierGrowth
	}

	feeAmount := (grossAmount * basisPoints) / 10000
	capApplied := false
	if regionalDefaults.CapAmount > 0 && feeAmount > regionalDefaults.CapAmount {
		feeAmount = regionalDefaults.CapAmount
		capApplied = true
	}
	if feeAmount < 0 {
		feeAmount = 0
	}

	netPayoutAmount := grossAmount - feeAmount
	if netPayoutAmount < 0 {
		netPayoutAmount = 0
	}

	return checkoutFeeComputation{
		PolicyVersion:   FeePolicyVersionRegionalDegressiveV1,
		SelectedTierKey: selectedTierKey,
		BasisPoints:     basisPoints,
		CapApplied:      capApplied,
		FeeAmount:       feeAmount,
		NetPayoutAmount: netPayoutAmount,
	}, true
}

func sanitizeFeeBasisPoints(value int64) int64 {
	if value < 0 {
		return 0
	}
	if value > 10000 {
		return 10000
	}
	return value
}

func reduceFeeBasisPoints(baseFeeBasisPoints, reductionBasisPoints int64) int64 {
	sanitizedBase := sanitizeFeeBasisPoints(baseFeeBasisPoints)
	if reductionBasisPoints <= 0 {
		return sanitizedBase
	}
	if sanitizedBase <= reductionBasisPoints {
		return 0
	}
	return sanitizedBase - reductionBasisPoints
}
