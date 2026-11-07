package proximity

import "time"

// TodayStart returns midnight today in the given location.
// If loc is nil, it uses the shipped pack timezone (GS-M4). Planned pack → UTC.
func TodayStart(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = PackLocation()
		if loc == nil {
			loc = time.UTC
		}
	}
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
