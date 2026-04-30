package proximity

import "time"

// TashkentLocation is the canonical operational timezone for all calendar-day
// boundary logic (pre-order sweepers, daily load resets, SLA monitors).
// Exported so cron.go, factory/, and other packages share one source of truth.
var TashkentLocation = mustLoadLocation("Asia/Tashkent")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Docker minimal images may lack tzdata — fall back to fixed UTC+5.
		return time.FixedZone("UZT", 5*60*60)
	}
	return loc
}

// TashkentNow returns the current wall-clock time in the Tashkent timezone.
func TashkentNow() time.Time {
	return time.Now().In(TashkentLocation)
}

// TashkentMidnight returns 00:00:01 on the given calendar date in Tashkent TZ.
// The 1-second offset avoids ambiguity with the previous day's 00:00:00.
func TashkentMidnight(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 1, 0, TashkentLocation)
}

// TashkentDayStart returns 00:00:00 on the given date in Tashkent TZ (no offset).
func TashkentDayStart(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, TashkentLocation)
}
