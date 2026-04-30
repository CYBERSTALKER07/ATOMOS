package analytics

import (
	"net/http"
	"time"
)

// maxRangeDays caps the maximum date range to prevent runaway queries.
const maxRangeDays = 365

// DateRange holds parsed from/to timestamps with a fallback default window.
type DateRange struct {
	From time.Time
	To   time.Time
}

// ParseDateRange extracts ?from= and ?to= ISO8601 query params.
// Falls back to defaultDays before now → now if not provided.
// Caps the maximum range at 365 days.
func ParseDateRange(r *http.Request, defaultDays int) DateRange {
	now := time.Now().UTC()
	dr := DateRange{
		From: now.AddDate(0, 0, -defaultDays),
		To:   now,
	}

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dr.From = t.UTC()
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			dr.From = t.UTC()
		}
	}

	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dr.To = t.UTC()
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			// End of day
			dr.To = t.UTC().Add(24*time.Hour - time.Nanosecond)
		}
	}

	// Ensure from <= to
	if dr.From.After(dr.To) {
		dr.From, dr.To = dr.To, dr.From
	}

	// Cap at maxRangeDays
	if dr.To.Sub(dr.From) > time.Duration(maxRangeDays)*24*time.Hour {
		dr.From = dr.To.AddDate(0, 0, -maxRangeDays)
	}

	return dr
}
