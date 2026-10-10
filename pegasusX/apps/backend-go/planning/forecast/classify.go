package forecast

import "math"

// DemandClass is the Syntetos–Boylan–Croston quadrant.
type DemandClass int

const (
	ClassSmooth DemandClass = iota
	ClassErratic
	ClassIntermittent // ADI >= 1.32 (includes lumpy)
)

func (c DemandClass) String() string {
	switch c {
	case ClassSmooth:
		return "smooth"
	case ClassErratic:
		return "erratic"
	case ClassIntermittent:
		return "intermittent"
	default:
		return "unknown"
	}
}

const (
	adiThreshold = 1.32
	cv2Threshold = 0.49
)

// ClassifySeries computes ADI and CV² of non-zero demand, then returns the SBC class.
func ClassifySeries(y []float64) (class DemandClass, adi, cv2 float64) {
	nonzero := make([]float64, 0, len(y))
	gaps := make([]int, 0)
	since := 0
	for _, v := range y {
		since++
		if v > 0 {
			nonzero = append(nonzero, v)
			gaps = append(gaps, since)
			since = 0
		}
	}
	n := len(y)
	if n == 0 || len(nonzero) == 0 {
		return ClassIntermittent, math.Inf(1), 0
	}
	sumGap := 0
	for _, g := range gaps {
		sumGap += g
	}
	adi = float64(sumGap) / float64(len(gaps))
	mean := 0.0
	for _, v := range nonzero {
		mean += v
	}
	mean /= float64(len(nonzero))
	if mean <= 0 {
		return ClassIntermittent, adi, 0
	}
	var varSum float64
	for _, v := range nonzero {
		d := v - mean
		varSum += d * d
	}
	cv2 = (varSum / float64(len(nonzero))) / (mean * mean)

	if adi >= adiThreshold {
		return ClassIntermittent, adi, cv2
	}
	if cv2 >= cv2Threshold {
		return ClassErratic, adi, cv2
	}
	return ClassSmooth, adi, cv2
}

// NonZeroCount returns how many periods have demand > 0.
func NonZeroCount(y []float64) int {
	n := 0
	for _, v := range y {
		if v > 0 {
			n++
		}
	}
	return n
}
