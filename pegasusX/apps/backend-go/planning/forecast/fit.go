package forecast

import (
	"math"
	"sort"
)

// Result is one series forecast with bands and metadata.
type Result struct {
	Class          DemandClass
	ADI            float64
	CV2            float64
	PointForecast  float64
	LowUnits       int64
	HighUnits      int64
	Alpha          float64
	Beta           float64
	Gamma          float64
	BaselineSource string
	BlockedReason  string
	Residuals      []float64
}

const (
	minCalendarDays = 60
	minNonZeroDays  = 14
	defaultAlpha    = 0.1
	defaultBeta     = 0.05
	defaultGamma    = 0.1
)

// ForecastSeries classifies y and fits the matching model for a 1-step-ahead point forecast.
func ForecastSeries(y []float64) Result {
	if len(y) == 0 {
		return Result{Class: ClassIntermittent, BaselineSource: "croston", BlockedReason: "insufficient_history"}
	}
	class, adi, cv2 := ClassifySeries(y)
	nz := NonZeroCount(y)
	out := Result{Class: class, ADI: adi, CV2: cv2}

	if len(y) < minCalendarDays || nz < minNonZeroDays {
		// Short series: global defaults + SES; mark blocked for confidence consumers.
		out.BlockedReason = "insufficient_history"
		out.Alpha = defaultAlpha
		f, res := FitSES(y, defaultAlpha)
		out.PointForecast = math.Max(0, f)
		out.Residuals = res
		out.BaselineSource = "ses"
		out.LowUnits, out.HighUnits = residualBands(out.PointForecast, res)
		return out
	}

	holdout := len(y) / 5
	if holdout < 7 {
		holdout = 7
	}
	if holdout >= len(y)/2 {
		holdout = len(y) / 3
	}
	if holdout < 1 {
		holdout = 1
	}

	switch class {
	case ClassSmooth:
		alpha, beta, gamma := fitHWParams(y, holdout)
		f, res := FitHoltWinters(y, alpha, beta, gamma, 1)
		out.Alpha, out.Beta, out.Gamma = alpha, beta, gamma
		out.PointForecast = math.Max(0, f)
		out.Residuals = res
		out.BaselineSource = "holt_winters"
	case ClassErratic:
		alpha := fitSESParams(y, holdout)
		f, res := FitSES(y, alpha)
		out.Alpha = alpha
		out.PointForecast = math.Max(0, f)
		out.Residuals = res
		out.BaselineSource = "ses"
	default:
		alpha := fitCrostonParams(y, holdout)
		f, res := FitCrostonSBA(y, alpha)
		out.Alpha = alpha
		out.PointForecast = math.Max(0, f)
		out.Residuals = res
		out.BaselineSource = "croston"
	}
	out.LowUnits, out.HighUnits = residualBands(out.PointForecast, out.Residuals)
	return out
}

func alphaGrid() []float64 {
	return []float64{0.05, 0.1, 0.15, 0.2, 0.3, 0.4}
}

func fitSESParams(y []float64, holdout int) float64 {
	bestA, bestSSE := defaultAlpha, math.Inf(1)
	trainEnd := len(y) - holdout
	if trainEnd < 2 {
		return defaultAlpha
	}
	for _, a := range alphaGrid() {
		sse := 0.0
		level := y[0]
		for t := 1; t < len(y); t++ {
			pred := level
			if t >= trainEnd {
				d := y[t] - pred
				sse += d * d
			}
			level = a*y[t] + (1-a)*level
		}
		if sse < bestSSE {
			bestSSE, bestA = sse, a
		}
	}
	return bestA
}

func fitCrostonParams(y []float64, holdout int) float64 {
	bestA, bestSSE := defaultAlpha, math.Inf(1)
	trainEnd := len(y) - holdout
	if trainEnd < 2 {
		return defaultAlpha
	}
	for _, a := range alphaGrid() {
		// One-step preds via progressive fit on prefix.
		sse := 0.0
		for t := trainEnd; t < len(y); t++ {
			f, _ := FitCrostonSBA(y[:t], a)
			d := y[t] - f
			sse += d * d
		}
		if sse < bestSSE {
			bestSSE, bestA = sse, a
		}
	}
	return bestA
}

func fitHWParams(y []float64, holdout int) (alpha, beta, gamma float64) {
	bestA, bestB, bestG := defaultAlpha, defaultBeta, defaultGamma
	bestSSE := math.Inf(1)
	trainEnd := len(y) - holdout
	if trainEnd < seasonalPeriod*2 {
		return bestA, bestB, bestG
	}
	for _, a := range alphaGrid() {
		for _, b := range []float64{0.05, 0.1, 0.2} {
			for _, g := range []float64{0.05, 0.1, 0.2, 0.3} {
				sse := 0.0
				for t := trainEnd; t < len(y); t++ {
					f, _ := FitHoltWinters(y[:t], a, b, g, 1)
					d := y[t] - f
					sse += d * d
				}
				if sse < bestSSE {
					bestSSE = sse
					bestA, bestB, bestG = a, b, g
				}
			}
		}
	}
	return bestA, bestB, bestG
}

func residualBands(point float64, residuals []float64) (low, high int64) {
	pt := int64(math.Round(point))
	if pt < 0 {
		pt = 0
	}
	if len(residuals) < 3 {
		margin := int64(math.Max(1, math.Round(point*0.1)))
		low = pt - margin
		if low < 0 {
			low = 0
		}
		high = pt + margin
		return low, high
	}
	sorted := append([]float64(nil), residuals...)
	sort.Float64s(sorted)
	p10 := percentile(sorted, 0.10)
	p90 := percentile(sorted, 0.90)
	lowF := point + p10
	highF := point + p90
	if lowF < 0 {
		lowF = 0
	}
	if highF < lowF {
		highF = lowF
	}
	low = int64(math.Floor(lowF))
	high = int64(math.Ceil(highF))
	if high < pt {
		high = pt
	}
	if low > pt {
		low = pt
	}
	return low, high
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := p * float64(len(sorted)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	w := idx - float64(lo)
	return sorted[lo]*(1-w) + sorted[hi]*w
}

// WAPEBias computes Σ|f-a|/Σa and Σ(f-a)/Σa over paired slices.
func WAPEBias(forecast, actual []float64) (wape, bias float64) {
	n := len(forecast)
	if len(actual) < n {
		n = len(actual)
	}
	var sumAbs, sumSigned, sumAct float64
	for i := 0; i < n; i++ {
		sumAbs += math.Abs(forecast[i] - actual[i])
		sumSigned += forecast[i] - actual[i]
		sumAct += actual[i]
	}
	if sumAct <= 0 {
		return 0, 0
	}
	return sumAbs / sumAct, sumSigned / sumAct
}
