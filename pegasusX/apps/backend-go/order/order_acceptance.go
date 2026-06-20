package order

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OperatingSchedule is warehouse order-acceptance hours (optional enforcement).
type OperatingSchedule struct {
	Is24h                  bool                  `json:"is_24h,omitempty"`
	EnforceOrderAcceptance bool                  `json:"enforce_order_acceptance,omitempty"`
	Timezone               string                `json:"timezone,omitempty"`
	Schedules              map[string]DayWindow  `json:"schedules,omitempty"`
}

// DayWindow is one day's open/close window (HH:MM, 24h).
type DayWindow struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

var weekdayKeys = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// ParseOperatingSchedule unmarshals warehouse OperatingSchedule JSON.
func ParseOperatingSchedule(raw json.RawMessage) OperatingSchedule {
	if len(raw) == 0 {
		return OperatingSchedule{}
	}
	var sched OperatingSchedule
	_ = json.Unmarshal(raw, &sched)
	return sched
}

// EvaluateOrderAcceptance returns whether orders are accepted now and a human label for the active window.
func EvaluateOrderAcceptance(sched OperatingSchedule, now time.Time) (open bool, label string, nextOpen *time.Time) {
	if !sched.EnforceOrderAcceptance || sched.Is24h {
		return true, "", nil
	}
	tzName := strings.TrimSpace(sched.Timezone)
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	if len(sched.Schedules) == 0 {
		return true, "", nil
	}
	dayKey := weekdayKeys[int(local.Weekday())]
	window, ok := sched.Schedules[dayKey]
	if !ok {
		// No window for this day — treat as closed.
		next := findNextOpen(sched, local, loc)
		return false, "", next
	}
	openMin, closeMin, err := parseDayWindow(window)
	if err != nil {
		return true, "", nil
	}
	nowMin := local.Hour()*60 + local.Minute()
	label = fmt.Sprintf("%s to %s", window.Open, window.Close)
	if tzName != "UTC" {
		label += " (" + tzName + ")"
	}
	if openMin <= closeMin {
		if nowMin >= openMin && nowMin < closeMin {
			return true, label, nil
		}
	} else {
		// Overnight window (e.g. 22:00–06:00).
		if nowMin >= openMin || nowMin < closeMin {
			return true, label, nil
		}
	}
	next := findNextOpen(sched, local, loc)
	return false, label, next
}

func parseDayWindow(w DayWindow) (openMin, closeMin int, err error) {
	openMin, err = parseHHMM(w.Open)
	if err != nil {
		return 0, 0, err
	}
	closeMin, err = parseHHMM(w.Close)
	if err != nil {
		return 0, 0, err
	}
	return openMin, closeMin, nil
}

func parseHHMM(s string) (int, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time %q", s)
	}
	var h, m int
	if _, err := fmt.Sscanf(parts[0], "%d", &h); err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour in %q", s)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid minute in %q", s)
	}
	return h*60 + m, nil
}

func findNextOpen(sched OperatingSchedule, from time.Time, loc *time.Location) *time.Time {
	for offset := 0; offset < 8; offset++ {
		day := from.AddDate(0, 0, offset)
		dayKey := weekdayKeys[int(day.Weekday())]
		window, ok := sched.Schedules[dayKey]
		if !ok {
			continue
		}
		openMin, _, err := parseDayWindow(window)
		if err != nil {
			continue
		}
		candidate := time.Date(day.Year(), day.Month(), day.Day(), openMin/60, openMin%60, 0, 0, loc)
		if offset == 0 && !candidate.After(from) {
			continue
		}
		utc := candidate.UTC()
		return &utc
	}
	return nil
}

// AcceptanceWindowHash fingerprints the schedule for checkout policy tokens.
func AcceptanceWindowHash(sched OperatingSchedule) string {
	raw, _ := json.Marshal(sched)
	return fmt.Sprintf("%x", raw)
}
