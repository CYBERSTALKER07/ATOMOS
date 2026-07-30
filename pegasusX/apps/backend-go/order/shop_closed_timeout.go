package order

import "github.com/pegasusx/pegasusx/apps/backend-go/credit"

// TimeoutDecision is the auto-decision when grace expires.
type TimeoutDecision string

const (
	DecisionCreditLeave       TimeoutDecision = "CREDIT_LEAVE"
	DecisionReturnToWarehouse TimeoutDecision = "RETURN_TO_WAREHOUSE"
	DecisionForceBypass       TimeoutDecision = "FORCE_BYPASS" // rare, config-gated
)

// TimeoutConfig contains knobs for the timeout matrix.
type TimeoutConfig struct {
	MaxAutoCreditMinor       int64 // e.g. 5_000_000 tiyin
	MaxRiskTierForAutoCredit int   // 1 = low, 2 = medium, ...
	AllowForceBypass         bool  // usually false in production
}

// DecideShopClosedTimeout applies the decision matrix (exact specs).
func DecideShopClosedTimeout(order *Order, profile *credit.Profile, cfg TimeoutConfig) TimeoutDecision {
	// 1. Credit leave if possible
	if profile != nil &&
		profile.Status == "ACTIVE" &&
		profile.AvailableCreditMinor >= order.TotalMinor &&
		order.TotalMinor <= cfg.MaxAutoCreditMinor &&
		riskTierLevel(profile.RiskTier) <= cfg.MaxRiskTierForAutoCredit {

		return DecisionCreditLeave
	}

	if cfg.AllowForceBypass {
		return DecisionForceBypass
	}

	// 2. Default: return to warehouse
	return DecisionReturnToWarehouse
}

func riskTierLevel(rt credit.RiskTier) int {
	switch rt {
	case credit.RiskTierLow:
		return 1
	case credit.RiskTierMedium:
		return 2
	case credit.RiskTierHigh:
		return 3
	default:
		return 99 // BLOCK or unknown
	}
}

// ShopClosedReason codes for driver mark-closed (Orders.ShopClosedReason).
const (
	ShopClosedReasonNoAnswer = "NO_ANSWER"
	ShopClosedReasonClosed   = "CLOSED"
	ShopClosedReasonRefused  = "REFUSED"
	ShopClosedReasonOther    = "OTHER"
)

// ValidShopClosedReason reports whether reason is a known code.
func ValidShopClosedReason(reason string) bool {
	switch reason {
	case ShopClosedReasonNoAnswer, ShopClosedReasonClosed, ShopClosedReasonRefused, ShopClosedReasonOther, "":
		return true
	default:
		return false
	}
}

// NormalizeShopClosedReason returns a canonical reason code (default CLOSED).
func NormalizeShopClosedReason(reason string) string {
	if reason == "" {
		return ShopClosedReasonClosed
	}
	if ValidShopClosedReason(reason) {
		return reason
	}
	return ShopClosedReasonOther
}

// ShopClosedResolution codes written to Orders.ShopClosedResolution.
const (
	ShopClosedResolutionRescheduled = "RESCHEDULED"
	ShopClosedResolutionCreditLeave = "CREDIT_LEAVE"
	ShopClosedResolutionCancelled   = "CANCELLED"
	ShopClosedResolutionBypass      = "BYPASS"
	ShopClosedResolutionReturned    = "RETURNED"
)
