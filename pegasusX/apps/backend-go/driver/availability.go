package driver

import "strings"

// Driver unavailable reason codes (end session / go offline).
const (
	ReasonShiftComplete         = "SHIFT_COMPLETE"
	ReasonReturningToWarehouse  = "RETURNING_TO_WAREHOUSE"
	ReasonTruckDamaged          = "TRUCK_DAMAGED"
	ReasonPersonal              = "PERSONAL"
	ReasonOther                 = "OTHER"
)

var driverUnavailableReasons = map[string]struct{}{
	ReasonShiftComplete:        {},
	ReasonReturningToWarehouse: {},
	ReasonTruckDamaged:         {},
	ReasonPersonal:             {},
	ReasonOther:                {},
}

func normalizeDriverUnavailableReason(reason string) string {
	r := strings.ToUpper(strings.TrimSpace(reason))
	if r == "" {
		return ReasonShiftComplete
	}
	if _, ok := driverUnavailableReasons[r]; ok {
		return r
	}
	return ReasonOther
}

func driverOffShiftTruckStatus(reason string) string {
	if strings.EqualFold(strings.TrimSpace(reason), ReasonReturningToWarehouse) {
		return ReasonReturningToWarehouse
	}
	return "OFF_SHIFT"
}
