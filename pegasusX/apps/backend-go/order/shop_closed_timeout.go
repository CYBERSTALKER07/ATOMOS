package order

import "strings"

// ShopClosedTimeoutResolution is the auto-decision when grace expires
// without a retailer response (design §4.2 step 4).
type ShopClosedTimeoutResolution string

const (
	// TimeoutCreditLeave: low/medium risk + credit available → leave on credit.
	TimeoutCreditLeave ShopClosedTimeoutResolution = "CREDIT_LEAVE"
	// TimeoutReturnToWarehouse: high risk / no credit / fiscal blocked.
	TimeoutReturnToWarehouse ShopClosedTimeoutResolution = "RETURN_TO_WAREHOUSE"
	// TimeoutForceBypass: supervised override path (config + risk gate).
	TimeoutForceBypass ShopClosedTimeoutResolution = "FORCE_BYPASS"
	// TimeoutReschedule: optional soft path when order value is tiny and credit blocked.
	TimeoutReschedule ShopClosedTimeoutResolution = "RESCHEDULE"
)

// ShopClosedRiskTier mirrors credit risk tiers for timeout policy.
type ShopClosedRiskTier string

const (
	ShopClosedRiskLow    ShopClosedRiskTier = "LOW"
	ShopClosedRiskMedium ShopClosedRiskTier = "MEDIUM"
	ShopClosedRiskHigh   ShopClosedRiskTier = "HIGH"
	ShopClosedRiskBlock  ShopClosedRiskTier = "BLOCK"
)

// ShopClosedTimeoutInput is the pure decision input (no I/O).
type ShopClosedTimeoutInput struct {
	// RiskTier from retailer credit profile (LOW|MEDIUM|HIGH|BLOCK).
	RiskTier ShopClosedRiskTier
	// ProfileStatus ACTIVE|FROZEN|CLOSED|BLACKLISTED (empty = treat as no profile).
	ProfileStatus string
	// AvailableCreditMinor from credit profile (integer minor units).
	AvailableCreditMinor int64
	// OrderTotalMinor order amount in minor units.
	OrderTotalMinor int64
	// CreditAllowed is precomputed CheckOrder.Allowed (profile ACTIVE + headroom).
	CreditAllowed bool
	// ForceBypassEnabled when supplier policy allows supervised auto-bypass on timeout.
	ForceBypassEnabled bool
	// LowValueThresholdMinor: below this and credit blocked → RESCHEDULE instead of return
	// (default applied by Decide when ≤ 0: 0 means never use reschedule soft path).
	LowValueThresholdMinor int64
}

// ShopClosedTimeoutDecision is the matrix outcome.
type ShopClosedTimeoutDecision struct {
	Resolution ShopClosedTimeoutResolution
	Reason     string
}

// DecideShopClosedTimeout evaluates the timeout auto-decision matrix.
//
// Matrix (design §4.2 + economic differentiator):
//
//	BLOCK / FROZEN / BLACKLISTED / CLOSED → RETURN_TO_WAREHOUSE
//	HIGH risk                             → RETURN_TO_WAREHOUSE
//	credit available + LOW|MEDIUM         → CREDIT_LEAVE
//	ForceBypassEnabled + MEDIUM+          → FORCE_BYPASS (only when credit blocked)
//	else low value + soft threshold       → RESCHEDULE
//	else                                  → RETURN_TO_WAREHOUSE
//
// Retailer response that arrives while still PENDING always wins over this
// decision at apply time (caller must re-check Resolution IS NULL).
func DecideShopClosedTimeout(in ShopClosedTimeoutInput) ShopClosedTimeoutDecision {
	status := strings.ToUpper(strings.TrimSpace(in.ProfileStatus))
	tier := normalizeShopClosedRisk(in.RiskTier)

	switch status {
	case "FROZEN", "BLACKLISTED", "CLOSED":
		return ShopClosedTimeoutDecision{
			Resolution: TimeoutReturnToWarehouse,
			Reason:     "profile_" + strings.ToLower(status),
		}
	case "", "NO_PROFILE":
		// No profile: cannot leave on credit.
		if in.ForceBypassEnabled && tier != ShopClosedRiskBlock && tier != ShopClosedRiskHigh {
			return ShopClosedTimeoutDecision{Resolution: TimeoutForceBypass, Reason: "no_profile_force_bypass"}
		}
		if in.LowValueThresholdMinor > 0 && in.OrderTotalMinor > 0 && in.OrderTotalMinor <= in.LowValueThresholdMinor {
			return ShopClosedTimeoutDecision{Resolution: TimeoutReschedule, Reason: "no_profile_low_value"}
		}
		return ShopClosedTimeoutDecision{Resolution: TimeoutReturnToWarehouse, Reason: "no_credit_profile"}
	}

	if tier == ShopClosedRiskBlock {
		return ShopClosedTimeoutDecision{Resolution: TimeoutReturnToWarehouse, Reason: "risk_tier_block"}
	}
	if tier == ShopClosedRiskHigh {
		if in.ForceBypassEnabled {
			return ShopClosedTimeoutDecision{Resolution: TimeoutForceBypass, Reason: "high_risk_force_bypass"}
		}
		return ShopClosedTimeoutDecision{Resolution: TimeoutReturnToWarehouse, Reason: "high_risk"}
	}

	// LOW / MEDIUM: credit leave when available and amount covered.
	if in.CreditAllowed && in.AvailableCreditMinor >= in.OrderTotalMinor && in.OrderTotalMinor >= 0 {
		return ShopClosedTimeoutDecision{Resolution: TimeoutCreditLeave, Reason: "credit_available"}
	}

	// Credit blocked at low/medium: optional supervised bypass, else return/reschedule.
	if in.ForceBypassEnabled {
		return ShopClosedTimeoutDecision{Resolution: TimeoutForceBypass, Reason: "credit_blocked_force_bypass"}
	}
	if in.LowValueThresholdMinor > 0 && in.OrderTotalMinor > 0 && in.OrderTotalMinor <= in.LowValueThresholdMinor {
		return ShopClosedTimeoutDecision{Resolution: TimeoutReschedule, Reason: "credit_blocked_low_value"}
	}
	return ShopClosedTimeoutDecision{Resolution: TimeoutReturnToWarehouse, Reason: "credit_unavailable"}
}

func normalizeShopClosedRisk(t ShopClosedRiskTier) ShopClosedRiskTier {
	switch ShopClosedRiskTier(strings.ToUpper(strings.TrimSpace(string(t)))) {
	case ShopClosedRiskLow:
		return ShopClosedRiskLow
	case ShopClosedRiskHigh:
		return ShopClosedRiskHigh
	case ShopClosedRiskBlock:
		return ShopClosedRiskBlock
	default:
		return ShopClosedRiskMedium
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
	switch strings.ToUpper(strings.TrimSpace(reason)) {
	case ShopClosedReasonNoAnswer, ShopClosedReasonClosed, ShopClosedReasonRefused, ShopClosedReasonOther, "":
		return true
	default:
		return false
	}
}

// NormalizeShopClosedReason returns a canonical reason code (default CLOSED).
func NormalizeShopClosedReason(reason string) string {
	r := strings.ToUpper(strings.TrimSpace(reason))
	if r == "" {
		return ShopClosedReasonClosed
	}
	if ValidShopClosedReason(r) {
		return r
	}
	return ShopClosedReasonOther
}

// ShopClosedResolution codes written to Orders.ShopClosedResolution.
const (
	ShopClosedResRescheduled = "RESCHEDULED"
	ShopClosedResCreditLeave = "CREDIT_LEAVE"
	ShopClosedResCancelled   = "CANCELLED"
	ShopClosedResBypass      = "BYPASS"
	ShopClosedResReturned    = "RETURNED"
)
