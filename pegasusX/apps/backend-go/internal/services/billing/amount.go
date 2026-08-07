package billing

import "math"

// ResolveMeterAmountMajor converts ORDER_FINALIZED amount fields to major currency units.
// Precedence: amount_minor → nested total.amount (minor) → total_minor → legacy major amount.
func ResolveMeterAmountMajor(amountMinor, totalNestedMinor, totalMinor int64, legacyMajor float64) float64 {
	minor := ResolveMeterAmountMinor(amountMinor, totalNestedMinor, totalMinor, legacyMajor)
	if minor <= 0 {
		return 0
	}
	return float64(minor) / 100.0
}

// ResolveMeterAmountMinor returns the order amount in minor units (2-decimal).
// Precedence matches ResolveMeterAmountMajor.
func ResolveMeterAmountMinor(amountMinor, totalNestedMinor, totalMinor int64, legacyMajor float64) int64 {
	if amountMinor > 0 {
		return amountMinor
	}
	if totalNestedMinor > 0 {
		return totalNestedMinor
	}
	if totalMinor > 0 {
		return totalMinor
	}
	if legacyMajor > 0 {
		return int64(math.Round(legacyMajor * 100.0))
	}
	return 0
}

// MinorToMajor converts 2-decimal minor units to major float for metering.
func MinorToMajor(minor int64) float64 {
	if minor <= 0 {
		return 0
	}
	return float64(minor) / 100.0
}
