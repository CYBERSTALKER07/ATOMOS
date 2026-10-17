package forecast

// FitSES fits simple exponential smoothing and returns one-step-ahead forecast + residuals.
func FitSES(y []float64, alpha float64) (forecast float64, residuals []float64) {
	if len(y) == 0 {
		return 0, nil
	}
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.1
	}
	level := y[0]
	residuals = make([]float64, 0, len(y)-1)
	for t := 1; t < len(y); t++ {
		pred := level
		residuals = append(residuals, y[t]-pred)
		level = alpha*y[t] + (1-alpha)*level
	}
	return level, residuals
}

// SESOneStepForecast returns the level after fitting on y (forecast for next period).
func SESOneStepForecast(y []float64, alpha float64) float64 {
	f, _ := FitSES(y, alpha)
	return f
}
