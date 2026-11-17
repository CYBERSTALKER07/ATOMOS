package forecast

// FitCrostonSBA fits Croston with Syntetos–Boylan bias correction.
// Updates z (demand size) and p (inter-arrival) only on non-zero periods.
// Point forecast: ŷ = (1 - α/2) · z / p
func FitCrostonSBA(y []float64, alpha float64) (forecast float64, residuals []float64) {
	if len(y) == 0 {
		return 0, nil
	}
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.1
	}
	z := 0.0
	p := 1.0
	init := false
	q := 0 // periods since last demand
	residuals = make([]float64, 0, len(y))

	for t := 0; t < len(y); t++ {
		q++
		var pred float64
		if init && p > 0 {
			pred = (1 - alpha/2) * z / p
		}
		if t > 0 || init {
			residuals = append(residuals, y[t]-pred)
		}
		if y[t] > 0 {
			if !init {
				z = y[t]
				p = float64(q)
				init = true
			} else {
				z = alpha*y[t] + (1-alpha)*z
				p = alpha*float64(q) + (1-alpha)*p
			}
			q = 0
		}
	}
	if !init || p <= 0 {
		return 0, residuals
	}
	return (1 - alpha/2) * z / p, residuals
}
