package proximity

import "time"

// TashkentLocation is the canonical operational timezone for calendar-day
// boundary logic (pre-order sweepers, SLA monitors).
var TashkentLocation = mustLoadLocation("Asia/Tashkent")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("UZT", 5*60*60)
	}
	return loc
}

// TashkentNow returns the current wall-clock time in Tashkent.
func TashkentNow() time.Time {
	return time.Now().In(TashkentLocation)
}

// TashkentTodayStart returns midnight today in Tashkent.
func TashkentTodayStart(now time.Time) time.Time {
	t := now.In(TashkentLocation)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, TashkentLocation)
}
