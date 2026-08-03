package segment

import (
	"sort"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

const (
	manualReasonPrefix = "manual:"
	bootstrapReason    = "bootstrap:rule"
)

// RetailerBootstrapInput aggregates signals for segment assignment.
type RetailerBootstrapInput struct {
	RetailerID   string
	OrderCount   int64
	ClaimCount   int64
	CreditScore  int64
	RiskTier     credit.RiskTier
	InTopVolume  bool
}

// bootstrapSegment assigns A/B/C per O9-1 heuristics.
func bootstrapSegment(in RetailerBootstrapInput) string {
	if in.RiskTier == credit.RiskTierBlock || in.RiskTier == credit.RiskTierHigh {
		return SegmentC
	}
	if in.ClaimCount >= 3 && in.OrderCount < 5 {
		return SegmentC
	}
	if in.InTopVolume && in.CreditScore >= 70 && in.RiskTier == credit.RiskTierLow {
		return SegmentA
	}
	if in.RiskTier == credit.RiskTierLow && in.OrderCount >= 20 {
		return SegmentA
	}
	if in.RiskTier == credit.RiskTierMedium {
		return SegmentB
	}
	return SegmentB
}

// SkuBootstrapInput holds per-SKU signals for velocity class assignment.
type SkuBootstrapInput struct {
	Sku           string
	OrderQty      int64
	VelocityRank  float64 // 0..1 percentile among supplier SKUs
	PriceMinor    int64
	MedianPrice   int64
	HandlingClass string
}

// bootstrapSkuClass maps velocity/margin/strategic signals to velocity class A/B/C.
func bootstrapSkuClass(in SkuBootstrapInput) SkuClass {
	strategic := in.HandlingClass == "PREMIUM" ||
		(in.MedianPrice > 0 && in.PriceMinor >= in.MedianPrice*2)
	velocity := VelocityB
	if in.OrderQty == 0 || in.VelocityRank <= 0.3 {
		velocity = VelocityC
	} else if in.VelocityRank >= 0.7 {
		velocity = VelocityA
	}
	return SkuClass{
		Sku:           in.Sku,
		VelocityClass: velocity,
		StrategicFlag: strategic,
	}
}

func topVolumePercentile(stats []RetailerOrderStats, pct float64) map[string]bool {
	if len(stats) == 0 {
		return map[string]bool{}
	}
	sorted := append([]RetailerOrderStats(nil), stats...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OrderCount > sorted[j].OrderCount
	})
	cutoff := int(float64(len(sorted)) * pct)
	if cutoff < 1 && len(sorted) > 0 {
		cutoff = 1
	}
	out := make(map[string]bool, cutoff)
	for i := 0; i < cutoff && i < len(sorted); i++ {
		out[sorted[i].RetailerID] = true
	}
	return out
}

func velocityRanks(qtys map[string]int64) map[string]float64 {
	if len(qtys) == 0 {
		return map[string]float64{}
	}
	type pair struct {
		sku string
		qty int64
	}
	pairs := make([]pair, 0, len(qtys))
	for sku, qty := range qtys {
		pairs = append(pairs, pair{sku: sku, qty: qty})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].qty > pairs[j].qty
	})
	out := make(map[string]float64, len(pairs))
	n := len(pairs)
	for i, p := range pairs {
		if n <= 1 {
			out[p.sku] = 1.0
			continue
		}
		out[p.sku] = 1.0 - float64(i)/float64(n-1)
	}
	return out
}

func medianPrice(prices []int64) int64 {
	if len(prices) == 0 {
		return 0
	}
	sorted := append([]int64(nil), prices...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func isManualSegment(reason string) bool {
	return len(reason) >= len(manualReasonPrefix) && reason[:len(manualReasonPrefix)] == manualReasonPrefix
}
