package warehouse

import "strings"

// Out-of-stock checkout policies — warehouse default with optional per-SKU override.
const (
	OutOfStockPolicyInherit         = "INHERIT"
	OutOfStockPolicyReject          = "REJECT"
	OutOfStockPolicyAcceptBackorder = "ACCEPT_BACKORDER"
)

// ResolveOutOfStockPolicy picks the effective policy for one SKU at a warehouse.
func ResolveOutOfStockPolicy(warehouseDefault, productOverride string) string {
	override := strings.ToUpper(strings.TrimSpace(productOverride))
	switch override {
	case OutOfStockPolicyReject, OutOfStockPolicyAcceptBackorder:
		return override
	}
	def := strings.ToUpper(strings.TrimSpace(warehouseDefault))
	switch def {
	case OutOfStockPolicyAcceptBackorder:
		return OutOfStockPolicyAcceptBackorder
	default:
		return OutOfStockPolicyReject
	}
}

// OperatingSchedule is warehouse display hours (informational — does not block ops dispatch).
type OperatingSchedule struct {
	Is24h     bool                       `json:"is_24h,omitempty"`
	Schedules map[string]DayWindow       `json:"schedules,omitempty"`
	Timezone  string                     `json:"timezone,omitempty"`
}

// DayWindow is one day's open/close window (HH:MM, 24h).
type DayWindow struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}
