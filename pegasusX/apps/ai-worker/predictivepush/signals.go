package predictivepush

import (
	"context"
	"math"
	"sort"
	"time"

	"cloud.google.com/go/spanner"
)

// DemandPoint represents an observed historical demand event.
type DemandPoint struct {
	Date time.Time
	Qty  int64
}

// CrostonSBAResult contains the calculated forecast and time-series statistics.
type CrostonSBAResult struct {
	DemandRate   float64 // Daily demand rate (units/day)
	SuggestedQty int64   // Forecasted units over horizon
	ADI          float64 // Average Demand Interval
	CV2          float64 // Coefficient of Variation squared
	Category     string  // "SMOOTH", "INTERMITTENT", "ERRATIC", "LUMPY", "NO_DATA", "ZERO_DEMAND", "EARLY_SIGNAL"
	Confidence   float64 // Confidence score [0, 1]
}

// CalculateCrostonSBA computes the Syntetos-Boylan Approximation (SBA) for intermittent demand forecasting.
// Default alpha = 0.15 for demand size and arrival interval smoothing.
func CalculateCrostonSBA(points []DemandPoint, horizonDays int, alpha float64) CrostonSBAResult {
	if horizonDays <= 0 {
		horizonDays = 1
	}
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.15
	}
	if len(points) == 0 {
		return CrostonSBAResult{
			DemandRate:   0,
			SuggestedQty: 1,
			ADI:          0,
			CV2:          0,
			Category:     "NO_DATA",
			Confidence:   0.1,
		}
	}

	sortedPoints := make([]DemandPoint, len(points))
	copy(sortedPoints, points)
	sort.Slice(sortedPoints, func(i, j int) bool {
		return sortedPoints[i].Date.Before(sortedPoints[j].Date)
	})

	var nonZeroQtys []float64
	var nonZeroIntervals []float64

	var lastDate time.Time
	hasPrev := false
	for _, pt := range sortedPoints {
		if pt.Qty <= 0 {
			continue
		}
		if hasPrev {
			interval := math.Max(1.0, math.Round(pt.Date.Sub(lastDate).Hours()/24.0))
			nonZeroIntervals = append(nonZeroIntervals, interval)
		}
		nonZeroQtys = append(nonZeroQtys, float64(pt.Qty))
		lastDate = pt.Date
		hasPrev = true
	}

	if len(nonZeroQtys) == 0 {
		return CrostonSBAResult{
			DemandRate:   0,
			SuggestedQty: 1,
			ADI:          0,
			CV2:          0,
			Category:     "ZERO_DEMAND",
			Confidence:   0.1,
		}
	}

	// Single observation fallback
	if len(nonZeroQtys) == 1 {
		qty := int64(nonZeroQtys[0])
		if qty < 1 {
			qty = 1
		}
		return CrostonSBAResult{
			DemandRate:   nonZeroQtys[0] / float64(horizonDays),
			SuggestedQty: qty,
			ADI:          1.0,
			CV2:          0.0,
			Category:     "EARLY_SIGNAL",
			Confidence:   0.35,
		}
	}

	// Exponential smoothing for demand size (z) and inter-arrival interval (p)
	z := nonZeroQtys[0]
	p := 1.0
	if len(nonZeroIntervals) > 0 {
		p = nonZeroIntervals[0]
	}

	for i := 1; i < len(nonZeroQtys); i++ {
		z = alpha*nonZeroQtys[i] + (1.0-alpha)*z
		if i-1 < len(nonZeroIntervals) {
			p = alpha*nonZeroIntervals[i-1] + (1.0-alpha)*p
		}
	}

	if p <= 0 {
		p = 1.0
	}

	// Syntetos-Boylan bias-corrected rate: (1 - alpha/2) * (z / p)
	sbaMultiplier := 1.0 - (alpha / 2.0)
	dailyRate := sbaMultiplier * (z / p)
	suggestedQty := int64(math.Max(1.0, math.Round(dailyRate*float64(horizonDays))))

	// Statistical quadrant classification (Syntetos & Boylan, 2005)
	// ADI cutoff = 1.32, CV^2 cutoff = 0.49
	var sumZ, sumZSq float64
	for _, v := range nonZeroQtys {
		sumZ += v
		sumZSq += v * v
	}
	n := float64(len(nonZeroQtys))
	meanZ := sumZ / n
	varZ := (sumZSq / n) - (meanZ * meanZ)
	if varZ < 0 {
		varZ = 0
	}
	cv2 := 0.0
	if meanZ > 0 {
		cv2 = varZ / (meanZ * meanZ)
	}

	totalDays := math.Max(1.0, math.Round(sortedPoints[len(sortedPoints)-1].Date.Sub(sortedPoints[0].Date).Hours()/24.0)+1.0)
	adi := totalDays / n

	category := "SMOOTH"
	if adi >= 1.32 && cv2 >= 0.49 {
		category = "LUMPY"
	} else if adi >= 1.32 {
		category = "INTERMITTENT"
	} else if cv2 >= 0.49 {
		category = "ERRATIC"
	}

	confidence := math.Min(0.95, 0.40+0.10*float64(len(nonZeroQtys)))

	return CrostonSBAResult{
		DemandRate:   dailyRate,
		SuggestedQty: suggestedQty,
		ADI:          adi,
		CV2:          cv2,
		Category:     category,
		Confidence:   confidence,
	}
}

// DemandSignal describes an external or internal driver for demand sensing.
type DemandSignal struct {
	SupplierID  string
	WarehouseID string
	ProductID   string
	Qty         int64
	Confidence  float64
	Source      string
	TargetDate  time.Time
}

// DemandSignalProvider aggregates pluggable demand drivers (order history, preorders, stubs).
type DemandSignalProvider interface {
	Collect(ctx context.Context, supplierID string, targetDay time.Time) ([]DemandSignal, error)
}

// CompositeSignalProvider merges built-in and statistical signal providers.
type CompositeSignalProvider struct {
	History *Analyzer
}

// NewCompositeSignalProvider wires the default PX90 signal stack.
func NewCompositeSignalProvider(client *spanner.Client) *CompositeSignalProvider {
	if client == nil {
		return &CompositeSignalProvider{}
	}
	return &CompositeSignalProvider{History: NewAnalyzer(client)}
}

// Collect merges order-history patterns with statistical demand forecasting.
func (p *CompositeSignalProvider) Collect(ctx context.Context, supplierID string, targetDay time.Time) ([]DemandSignal, error) {
	var out []DemandSignal
	if p != nil && p.History != nil {
		events, err := p.History.Analyze(ctx, targetDay)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			if supplierID != "" && ev.SupplierId != supplierID {
				continue
			}
			out = append(out, DemandSignal{
				SupplierID: ev.SupplierId,
				ProductID:  ev.ProductId,
				Qty:        ev.Quantity,
				Confidence: ev.Confidence,
				Source:     "order_history",
				TargetDate: ev.TargetDate,
			})
		}
	}
	if out == nil {
		out = []DemandSignal{}
	}
	return out, nil
}
