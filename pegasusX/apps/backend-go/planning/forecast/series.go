package forecast

import (
	"time"

	"cloud.google.com/go/civil"
)

// DenseDaily builds a contiguous float series for [start,end] inclusive (UTC days).
// Missing days are 0.
func DenseDaily(dayQty map[civil.Date]int64, start, end civil.Date) []float64 {
	if end.Before(start) {
		return nil
	}
	n := 0
	for d := start; !d.After(end); d = d.AddDays(1) {
		n++
	}
	out := make([]float64, n)
	i := 0
	for d := start; !d.After(end); d = d.AddDays(1) {
		out[i] = float64(dayQty[d])
		i++
	}
	return out
}

// DayQtyFromSparse converts a sparse day→qty map into DenseDaily over lookback ending at asOf.
func DayQtyFromSparse(sparse map[string]int64, asOf time.Time, lookbackDays int) ([]float64, civil.Date, civil.Date) {
	if lookbackDays < 1 {
		lookbackDays = 60
	}
	end := civil.DateOf(asOf.UTC())
	start := end.AddDays(-(lookbackDays - 1))
	m := make(map[civil.Date]int64, len(sparse))
	for k, v := range sparse {
		d, err := civil.ParseDate(k)
		if err != nil {
			continue
		}
		m[d] = v
	}
	return DenseDaily(m, start, end), start, end
}

// TrailingMean7 is the naive comparator: sum of last up-to-7 days / 7 (audit baseline).
func TrailingMean7(y []float64) float64 {
	if len(y) == 0 {
		return 0
	}
	n := 7
	if len(y) < n {
		n = len(y)
	}
	var sum float64
	for i := len(y) - n; i < len(y); i++ {
		sum += y[i]
	}
	return sum / 7.0
}
