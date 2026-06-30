package proximity

import "time"

// TodayStart returns midnight today in the given location.
// If loc is nil, it defaults to TashkentLocation.
func TodayStart(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = TashkentLocation
	}
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
