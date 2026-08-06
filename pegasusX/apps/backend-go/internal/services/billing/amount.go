package billing

// ResolveMeterAmountMajor converts ORDER_FINALIZED amount fields to major currency units.
// Precedence: amount_minor → nested total.amount (minor) → total_minor → legacy major amount.
func ResolveMeterAmountMajor(amountMinor, totalNestedMinor, totalMinor int64, legacyMajor float64) float64 {
	if amountMinor > 0 {
		return float64(amountMinor) / 100.0
	}
	if totalNestedMinor > 0 {
		return float64(totalNestedMinor) / 100.0
	}
	if totalMinor > 0 {
		return float64(totalMinor) / 100.0
	}
	if legacyMajor > 0 {
		return legacyMajor
	}
	return 0
}
