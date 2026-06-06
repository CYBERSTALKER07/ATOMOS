package dispatch

import (
	"fmt"
	"sort"
	"strings"
)

const defaultFlexibleWindowClose = "23:59"

// EffectiveWindowClose returns wc or the flexible end-of-day sentinel when unset.
func EffectiveWindowClose(wc string) string {
	if strings.TrimSpace(wc) == "" {
		return defaultFlexibleWindowClose
	}
	return strings.TrimSpace(wc)
}

// ParseTimeMinutes parses "HH:MM" to minutes since midnight. Returns -1 on failure.
func ParseTimeMinutes(value string) int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return -1
	}
	var hour, minute int
	if _, err := fmt.Sscanf(value, "%d:%d", &hour, &minute); err != nil {
		return -1
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return -1
	}
	return hour*60 + minute
}

// HasReceivingWindow reports whether the retailer configured a bounded window.
func HasReceivingWindow(open, close string) bool {
	return strings.TrimSpace(open) != "" || strings.TrimSpace(close) != ""
}

// SortByWindowUrgency orders dispatch candidates by earliest effective close,
// then by total amount descending as a stable tiebreaker.
func SortByWindowUrgency(orders []DispatchableOrder) {
	sort.SliceStable(orders, func(i, j int) bool {
		closeI := ParseTimeMinutes(EffectiveWindowClose(orders[i].ReceivingWindowClose))
		closeJ := ParseTimeMinutes(EffectiveWindowClose(orders[j].ReceivingWindowClose))
		if closeI != closeJ {
			return closeI < closeJ
		}
		return orders[i].TotalMinor > orders[j].TotalMinor
	})
}
