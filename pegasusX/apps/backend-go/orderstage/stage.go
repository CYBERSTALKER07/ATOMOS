// Package orderstage projects raw order status strings onto the simplified
// customer-facing stage. It is a dependency-free leaf so that the order,
// events, and outbox packages can all use it without import cycles.
//
// The mapping is pinned by contracts/order-customer-stage.json; every
// platform (Go, TS, Kotlin, Swift) tests against that fixture.
package orderstage

import "strings"

// Stage is the customer-facing order stage.
type Stage string

const (
	Placed                  Stage = "PLACED"
	Processing              Stage = "PROCESSING"
	OutForDelivery          Stage = "OUT_FOR_DELIVERY"
	Delivered               Stage = "DELIVERED"
	Cancelled               Stage = "CANCELLED"
	ExceptionActionRequired Stage = "EXCEPTION_ACTION_REQUIRED"
)

// Lookup maps a raw status (canonical or legacy alias, any case) to a Stage and
// reports whether the status was recognised.
func Lookup(raw string) (Stage, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SCHEDULED", "AUTO_ACCEPTED", "BACKORDERED", "PENDING_REVIEW":
		return Placed, true
	case "PENDING", "DELAYED", "LOADED", "DISPATCHED":
		return Processing, true
	case "IN_TRANSIT", "EN_ROUTE", "ARRIVING", "ARRIVED",
		"AWAITING_PAYMENT", "PENDING_CASH_COLLECTION":
		return OutForDelivery, true
	// Goods are in the retailer's hands; fiscalization is back-office.
	case "DELIVERED_ON_CREDIT", "FISCALIZING", "COMPLETED":
		return Delivered, true
	case "CANCELLED", "NO_CAPACITY":
		return Cancelled, true
	case "SHOP_CLOSED_PENDING", "ARRIVED_SHOP_CLOSED", "CANCEL_REQUESTED",
		"FISCAL_FAILED", "RECONCILIATION_REQUIRED", "QUARANTINE", "SYNC_CONFLICT":
		return ExceptionActionRequired, true
	default:
		return Processing, false
	}
}

// FromString is Lookup with unknown statuses falling back to Processing.
func FromString(raw string) Stage {
	s, _ := Lookup(raw)
	return s
}

// ProgressPercent returns the UI progress bar value (0-100) for a stage.
func ProgressPercent(s Stage) int {
	switch s {
	case Placed:
		return 20
	case Processing:
		return 50
	case OutForDelivery:
		return 80
	case Delivered:
		return 100
	case ExceptionActionRequired:
		return 75
	default:
		return 0
	}
}
