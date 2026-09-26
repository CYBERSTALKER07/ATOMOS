// Package seasonalcore holds shared calendar seasonality builtins used by
// planning (forecast baselines) and replenishment (suggested qty) without
// creating an import cycle between those packages.
package seasonalcore

import (
	"strings"
	"time"
)

const (
	// MinMultiplier is the lower clamp for persisted / estimated multipliers.
	MinMultiplier = 0.5
	// MaxMultiplier is the upper clamp for persisted / estimated multipliers.
	MaxMultiplier = 2.5
	// DefaultOverrideMultiplier is the documented fallback when a custom
	// override has no explicit multiplier and does not match a builtin id.
	DefaultOverrideMultiplier = 1.2
)

// Template is a recurring calendar window with a demand multiplier.
type Template struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	StartMonth      int     `json:"start_month"`
	StartDay        int     `json:"start_day"`
	EndMonth        int     `json:"end_month"`
	EndDay          int     `json:"end_day"`
	Multiplier      float64 `json:"multiplier"`
	ConfidenceFloor float64 `json:"confidence_floor"`
}

// Builtins are the hard-coded annual windows (holiday peak ×1.35, summer ×1.15).
var Builtins = []Template{
	{ID: "holiday_peak", Name: "Holiday Peak", StartMonth: 11, StartDay: 15, EndMonth: 1, EndDay: 5, Multiplier: 1.35, ConfidenceFloor: 0.75},
	{ID: "summer_surge", Name: "Summer Surge", StartMonth: 6, StartDay: 1, EndMonth: 8, EndDay: 31, Multiplier: 1.15, ConfidenceFloor: 0.75},
}

// BuiltinByID returns a builtin template by id.
func BuiltinByID(id string) (Template, bool) {
	id = strings.TrimSpace(id)
	for _, tpl := range Builtins {
		if tpl.ID == id {
			return tpl, true
		}
	}
	return Template{}, false
}

// ActiveOn reports whether tpl is active on the given UTC day.
func ActiveOn(tpl Template, on time.Time) bool {
	on = on.UTC()
	year := on.Year()
	start := time.Date(year, time.Month(tpl.StartMonth), tpl.StartDay, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.Month(tpl.EndMonth), tpl.EndDay, 23, 59, 59, 0, time.UTC)
	if tpl.StartMonth > tpl.EndMonth {
		if on.Month() >= time.Month(tpl.StartMonth) {
			end = time.Date(year+1, time.Month(tpl.EndMonth), tpl.EndDay, 23, 59, 59, 0, time.UTC)
		} else {
			start = time.Date(year-1, time.Month(tpl.StartMonth), tpl.StartDay, 0, 0, 0, 0, time.UTC)
		}
	}
	return !on.Before(start) && !on.After(end)
}

// WindowBounds returns the inclusive UTC start/end for tpl in the given year
// (year-crossing windows end in year+1).
func WindowBounds(tpl Template, year int) (start, end time.Time) {
	start = time.Date(year, time.Month(tpl.StartMonth), tpl.StartDay, 0, 0, 0, 0, time.UTC)
	end = time.Date(year, time.Month(tpl.EndMonth), tpl.EndDay, 23, 59, 59, 0, time.UTC)
	if tpl.StartMonth > tpl.EndMonth {
		end = time.Date(year+1, time.Month(tpl.EndMonth), tpl.EndDay, 23, 59, 59, 0, time.UTC)
	}
	return start, end
}

// BuiltinMultiplierFor returns the first matching builtin multiplier, else 1.0.
func BuiltinMultiplierFor(on time.Time) float64 {
	for _, tpl := range Builtins {
		if ActiveOn(tpl, on) {
			if tpl.Multiplier > 0 {
				return tpl.Multiplier
			}
			return 1.0
		}
	}
	return 1.0
}

// ClampMultiplier clamps m into [MinMultiplier, MaxMultiplier].
func ClampMultiplier(m float64) float64 {
	if m < MinMultiplier {
		return MinMultiplier
	}
	if m > MaxMultiplier {
		return MaxMultiplier
	}
	return m
}

// ResolveOverrideMultiplier picks the persisted multiplier for a new override:
// explicit value (clamped) → inherit builtin by template_id → DefaultOverrideMultiplier.
func ResolveOverrideMultiplier(explicit *float64, templateID string) float64 {
	if explicit != nil {
		return ClampMultiplier(*explicit)
	}
	if tpl, ok := BuiltinByID(templateID); ok && tpl.Multiplier > 0 {
		return ClampMultiplier(tpl.Multiplier)
	}
	return DefaultOverrideMultiplier
}
