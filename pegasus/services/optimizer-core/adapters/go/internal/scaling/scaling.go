package scaling

import "math"

const Factor int64 = 10_000

func ScaleFloat64(value float64) int64 {
	return int64(math.Round(value * float64(Factor)))
}

func UnscaleInt64(value int64) float64 {
	return float64(value) / float64(Factor)
}

func ScaleSliceFloat64(values []float64) []int64 {
	out := make([]int64, 0, len(values))
	for _, value := range values {
		out = append(out, ScaleFloat64(value))
	}
	return out
}

func ScaleMatrixFloat64(values [][]float64) [][]int64 {
	out := make([][]int64, 0, len(values))
	for _, row := range values {
		scaledRow := make([]int64, 0, len(row))
		for _, value := range row {
			scaledRow = append(scaledRow, ScaleFloat64(value))
		}
		out = append(out, scaledRow)
	}
	return out
}
