package billing

// ResolveMeterAmountMajor converts ORDER_FINALIZED amount fields to major currency units.
// Precedence: amount_minor → nested total.amount (minor) → total_minor → legacy amount minor.
func ResolveMeterAmountMajor(amountMinor, totalNestedMinor, totalMinor, legacyAmountMinor int64) float64 {
	minor := ResolveMeterAmountMinor(amountMinor, totalNestedMinor, totalMinor, legacyAmountMinor)
	if minor <= 0 {
		return 0
	}
	return float64(minor) / 100.0
}

// ResolveMeterAmountMinor returns the order amount in minor units (2-decimal).
// Precedence matches ResolveMeterAmountMajor.
func ResolveMeterAmountMinor(amountMinor, totalNestedMinor, totalMinor, legacyAmountMinor int64) int64 {
	if amountMinor > 0 {
		return amountMinor
	}
	if totalNestedMinor > 0 {
		return totalNestedMinor
	}
	if totalMinor > 0 {
		return totalMinor
	}
	if legacyAmountMinor > 0 {
		return legacyAmountMinor
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
