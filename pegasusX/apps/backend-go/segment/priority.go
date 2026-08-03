package segment

// ComputePriorityScore combines policy weight and strategic SKU flag.
// Credit risk tier boosts were removed with the credit-score product.
func ComputePriorityScore(policy ServicePolicy, _ any, strategic bool) int64 {
	score := policy.PriorityWeight
	if strategic {
		score += 10
	}
	if score < 0 {
		return 0
	}
	return score
}

// NormalizeRetailerSegment returns a known segment or default C.
func NormalizeRetailerSegment(segment string) string {
	switch segment {
	case SegmentA, "STRATEGIC":
		return SegmentA
	case SegmentB, "STANDARD":
		return SegmentB
	case SegmentC, "OPPORTUNISTIC":
		return SegmentC
	default:
		return SegmentC
	}
}

// NormalizeVelocityClass returns a known class or default B.
func NormalizeVelocityClass(class string) string {
	switch class {
	case VelocityA:
		return VelocityA
	case VelocityB:
		return VelocityB
	case VelocityC:
		return VelocityC
	default:
		return VelocityB
	}
}

// DefaultPolicy returns fallback policy when no row matches.
func DefaultPolicy(supplierID, retailerSegment, skuClass string) ServicePolicy {
	weight := int64(40)
	switch NormalizeRetailerSegment(retailerSegment) {
	case SegmentA:
		weight = 100
	case SegmentB:
		weight = 70
	case SegmentC:
		weight = 30
	}
	if NormalizeVelocityClass(skuClass) == VelocityA {
		weight += 5
	}
	return ServicePolicy{
		SupplierID:            supplierID,
		RetailerSegment:       NormalizeRetailerSegment(retailerSegment),
		SkuClass:              NormalizeVelocityClass(skuClass),
		PriorityWeight:        weight,
		TargetServiceLevelBps: 9500,
		MaxFairShareBps:       4000,
		MinFairShareBps:       500,
		CreditRiskBoost:       0, // scoring product removed
		Enabled:               true,
	}
}
