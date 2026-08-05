package replenishment

import "time"

// Mirrors planning builtin seasonal templates so suggested qty applies Multiplier
// without an import cycle (planning → replenishment already exists).
type seasonalWindow struct {
	startMonth, startDay int
	endMonth, endDay     int
	multiplier           float64
}

var seasonalWindows = []seasonalWindow{
	{11, 15, 1, 5, 1.35}, // holiday_peak
	{6, 1, 8, 31, 1.15},  // summer_surge
}

func seasonalMultiplierFor(on time.Time) float64 {
	for _, w := range seasonalWindows {
		if seasonalActiveOn(w, on) {
			if w.multiplier > 0 {
				return w.multiplier
			}
			return 1.0
		}
	}
	return 1.0
}

func seasonalActiveOn(w seasonalWindow, on time.Time) bool {
	year := on.Year()
	start := time.Date(year, time.Month(w.startMonth), w.startDay, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.Month(w.endMonth), w.endDay, 23, 59, 59, 0, time.UTC)
	if w.startMonth > w.endMonth {
		if on.Month() >= time.Month(w.startMonth) {
			end = time.Date(year+1, time.Month(w.endMonth), w.endDay, 23, 59, 59, 0, time.UTC)
		} else {
			start = time.Date(year-1, time.Month(w.startMonth), w.startDay, 0, 0, 0, 0, time.UTC)
		}
	}
	return !on.Before(start) && !on.After(end)
}
