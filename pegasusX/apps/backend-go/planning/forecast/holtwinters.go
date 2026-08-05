package forecast

import "math"

const seasonalPeriod = 7

// FitHoltWinters fits multiplicative-seasonal Holt–Winters (m=7).
// L_t = α(y_t / S_{t-m}) + (1-α)(L_{t-1} + T_{t-1})
// T_t = β(L_t - L_{t-1}) + (1-β)T_{t-1}
// S_t = γ(y_t / L_t) + (1-γ)S_{t-m}
// ŷ_{t+h} = (L_t + h·T_t)·S_{t-m+h}
func FitHoltWinters(y []float64, alpha, beta, gamma float64, horizon int) (forecast float64, residuals []float64) {
	if len(y) < seasonalPeriod*2 {
		// Fall back to SES when too short for seasonality.
		return FitSES(y, alpha)
	}
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.1
	}
	if beta < 0 || beta >= 1 {
		beta = 0.05
	}
	if gamma < 0 || gamma >= 1 {
		gamma = 0.1
	}
	if horizon < 1 {
		horizon = 1
	}

	m := seasonalPeriod
	s := make([]float64, m)
	// Initialize seasonal indices from first cycle averages.
	var firstSum float64
	for i := 0; i < m; i++ {
		firstSum += y[i]
	}
	avg := firstSum / float64(m)
	if avg <= 0 {
		avg = 1
	}
	for i := 0; i < m; i++ {
		if y[i] > 0 {
			s[i] = y[i] / avg
		} else {
			s[i] = 1
		}
		if s[i] <= 0 {
			s[i] = 1
		}
	}
	level := avg
	trend := 0.0
	if len(y) >= 2*m {
		var secondSum float64
		for i := m; i < 2*m; i++ {
			secondSum += y[i]
		}
		trend = (secondSum/float64(m) - avg) / float64(m)
	}

	residuals = make([]float64, 0, len(y)-m)
	for t := m; t < len(y); t++ {
		season := s[t%m]
		if season <= 1e-9 {
			season = 1
		}
		pred := (level + trend) * season
		residuals = append(residuals, y[t]-pred)

		yt := y[t]
		prevLevel := level
		level = alpha*(yt/season) + (1-alpha)*(prevLevel+trend)
		if level <= 0 {
			level = math.Max(1e-6, prevLevel)
		}
		trend = beta*(level-prevLevel) + (1-beta)*trend
		s[t%m] = gamma*(yt/level) + (1-gamma)*season
		if s[t%m] <= 0 {
			s[t%m] = 1
		}
	}
	seasonH := s[(len(y)+horizon-1)%m]
	if seasonH <= 0 {
		seasonH = 1
	}
	forecast = (level + float64(horizon)*trend) * seasonH
	if forecast < 0 {
		forecast = 0
	}
	return forecast, residuals
}
