package order

import (
	"errors"
	"fmt"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

const (
	DeliveryModeStandard  = "STANDARD"
	DeliveryModeScheduled = "SCHEDULED"

	DeliveryPriorityStandard = "STANDARD"
	DeliveryPriorityExpress  = "EXPRESS"

	PreorderMinScheduledLeadDays = 3
	PreorderMaxScheduledLeadDays = 90
	PreorderEditLockDays         = 2
	PreorderMinStandardLeadDays  = 1
)

// DeliveryPriority is STANDARD or EXPRESS.
type DeliveryPriority string

// ClassifyDelivery decides order source/status from mode and requested dates (Tashkent calendar days).
func ClassifyDelivery(now time.Time, mode string, requestedDelivery *time.Time, deliverBefore *time.Time) (OrderSource, Status, ConfirmationStatus, *time.Time, *time.Time, error) {
	mode = normalizeDeliveryMode(mode)
	today := proximity.TashkentTodayStart(now)

	switch mode {
	case DeliveryModeScheduled:
		if requestedDelivery == nil {
			return "", "", "", nil, nil, errors.New("requested_delivery_date required for SCHEDULED mode")
		}
		deliveryDay := proximity.TashkentTodayStart(*requestedDelivery)
		leadDays := calendarDaysBetween(today, deliveryDay)
		if leadDays < PreorderMinScheduledLeadDays {
			return "", "", "", nil, nil, fmt.Errorf("scheduled pre-order requires delivery at least %d calendar days ahead (got %d)", PreorderMinScheduledLeadDays, leadDays)
		}
		if leadDays > PreorderMaxScheduledLeadDays {
			return "", "", "", nil, nil, fmt.Errorf("scheduled pre-order cannot exceed %d calendar days ahead", PreorderMaxScheduledLeadDays)
		}
		return OrderSourceManualPreorder, StatusScheduled, ConfirmationStatusDraft, requestedDelivery, nil, nil

	default: // STANDARD
		var latest *time.Time
		if deliverBefore != nil {
			latest = deliverBefore
		} else if requestedDelivery != nil {
			// T+1/T+2 picked on calendar → treat as deliver-before Standard, not pre-order.
			latest = requestedDelivery
		}
		if latest != nil {
			latestDay := proximity.TashkentTodayStart(*latest)
			leadDays := calendarDaysBetween(today, latestDay)
			if leadDays < PreorderMinStandardLeadDays {
				return "", "", "", nil, nil, fmt.Errorf("standard delivery earliest is T+%d (no same-day)", PreorderMinStandardLeadDays)
			}
		}
		return OrderSourceManual, StatusPending, ConfirmationStatusConfirmed, nil, latest, nil
	}
}

func normalizeDeliveryMode(mode string) string {
	switch mode {
	case DeliveryModeScheduled, "scheduled", "PREORDER", "preorder":
		return DeliveryModeScheduled
	default:
		return DeliveryModeStandard
	}
}

func normalizeDeliveryPriority(p string) DeliveryPriority {
	if p == string(DeliveryPriorityExpress) || p == "express" || p == "EXPRESS" {
		return DeliveryPriorityExpress
	}
	return DeliveryPriorityStandard
}

func calendarDaysBetween(from, to time.Time) int {
	from = proximity.TashkentTodayStart(from)
	to = proximity.TashkentTodayStart(to)
	return int(to.Sub(from).Hours() / 24)
}

// PreorderLeadDays returns calendar days from now until requested delivery.
func PreorderLeadDays(now time.Time, requestedDelivery *time.Time) int {
	if requestedDelivery == nil {
		return 0
	}
	return calendarDaysBetween(now, *requestedDelivery)
}

// PreorderEditLocked is true when retailer/warehouse edits are blocked (T-2).
func PreorderEditLocked(now time.Time, o Order) bool {
	if o.RequestedDeliveryDate == nil {
		return false
	}
	if o.CancelLockedAt != nil && !o.CancelLockedAt.After(now) {
		return true
	}
	return PreorderLeadDays(now, o.RequestedDeliveryDate) <= PreorderEditLockDays
}

// PreorderCancelLocked is true when retailer cancel is blocked.
func PreorderCancelLocked(now time.Time, o Order) bool {
	return o.CancelLockedAt != nil && !o.CancelLockedAt.After(now)
}

// PreorderGuardPhase returns long (L>=7) or compressed (3<=L<7) guard profile.
func PreorderGuardPhase(leadDays int) string {
	if leadDays >= 7 {
		return "long"
	}
	if leadDays >= PreorderMinScheduledLeadDays {
		return "compressed"
	}
	return ""
}

// IsScheduledPreorder reports whether the order is in the manual pre-order lane.
func IsScheduledPreorder(o Order) bool {
	return o.Source == OrderSourceManualPreorder && (o.Status == StatusScheduled || o.Status == StatusAutoAccepted)
}

func deliveryModeLabel(o Order) string {
	if IsScheduledPreorder(o) || (o.Source == OrderSourceManualPreorder && o.Status == StatusScheduled) {
		return DeliveryModeScheduled
	}
	return DeliveryModeStandard
}

func preorderBadgeLabel(o Order) string {
	if o.Source == OrderSourceManualPreorder && o.Status == StatusScheduled {
		return "Pre-order"
	}
	if o.DeliveryPriority == DeliveryPriorityExpress {
		return "Express"
	}
	if o.DeliverBefore != nil {
		return "Deliver by"
	}
	if o.Status == StatusAutoAccepted {
		return "Confirmed for dispatch"
	}
	return ""
}
