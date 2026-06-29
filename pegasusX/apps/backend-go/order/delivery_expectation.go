package order

import (
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

const (
	ExpectationKindStandard         = "STANDARD"
	ExpectationKindScheduledPreorder = "SCHEDULED_PREORDER"
	ExpectationKindExpress          = "EXPRESS"
	ExpectationKindProposalPending  = "PROPOSAL_PENDING"

	ExpectationUrgencyOnTrack     = "on_track"
	ExpectationUrgencyDueSoon     = "due_soon"
	ExpectationUrgencyOverdue     = "overdue"
	ExpectationUrgencyScheduledFar = "scheduled_far"
)

// DeliveryExpectation is a computed, read-only projection of when and how an
// order should be fulfilled. All timing copy derives from persisted order fields.
type DeliveryExpectation struct {
	Kind                 string  `json:"kind"`
	TargetDate           *string `json:"target_date,omitempty"`
	TargetLabel          string  `json:"target_label"`
	ModeLabel            string  `json:"mode_label,omitempty"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
	Delayed              bool    `json:"delayed"`
	DelayReason          string  `json:"delay_reason,omitempty"`
	Urgency              string  `json:"urgency"`
	BadgeLabel           string  `json:"badge_label,omitempty"`
}

// ComputeDeliveryExpectation derives human-readable delivery timing from the order row.
func ComputeDeliveryExpectation(now time.Time, o Order) DeliveryExpectation {
	out := DeliveryExpectation{
		ModeLabel:            deliveryModeLabel(o),
		ReceivingWindowOpen:  strings.TrimSpace(o.ReceivingWindowOpen),
		ReceivingWindowClose: strings.TrimSpace(o.ReceivingWindowClose),
		BadgeLabel:           preorderBadgeLabel(o),
		Urgency:              ExpectationUrgencyOnTrack,
	}

	if o.ConfirmationStatus == ConfirmationStatusPendingWarehouse && o.ProposedDeliveryDate != nil {
		out.Kind = ExpectationKindProposalPending
		out.TargetDate = isoDatePtr(o.ProposedDeliveryDate)
		out.TargetLabel = fmt.Sprintf("Warehouse proposed %s — awaiting retailer", formatDateLabel(*o.ProposedDeliveryDate))
		out.Urgency = urgencyForTarget(now, *o.ProposedDeliveryDate, o)
		out.Delayed, out.DelayReason = delayedState(now, o, o.ProposedDeliveryDate)
		return out
	}

	if o.DeliveryPriority == DeliveryPriorityExpress {
		out.Kind = ExpectationKindExpress
		if o.DeliverBefore != nil {
			out.TargetDate = isoDatePtr(o.DeliverBefore)
			out.TargetLabel = fmt.Sprintf("Express — deliver by %s", formatDateLabel(*o.DeliverBefore))
			out.Urgency = urgencyForTarget(now, *o.DeliverBefore, o)
			out.Delayed, out.DelayReason = delayedState(now, o, o.DeliverBefore)
		} else {
			out.TargetLabel = "Express delivery"
		}
		return out
	}

	if IsScheduledPreorder(o) || (o.Source == OrderSourceManualPreorder && (o.Status == StatusScheduled || o.Status == StatusAutoAccepted)) {
		out.Kind = ExpectationKindScheduledPreorder
		if o.RequestedDeliveryDate != nil {
			out.TargetDate = isoDatePtr(o.RequestedDeliveryDate)
			out.TargetLabel = fmt.Sprintf("Scheduled for %s", formatDateLabel(*o.RequestedDeliveryDate))
			lead := PreorderLeadDays(now, o.RequestedDeliveryDate)
			if lead >= 7 {
				out.Urgency = ExpectationUrgencyScheduledFar
			} else {
				out.Urgency = urgencyForTarget(now, *o.RequestedDeliveryDate, o)
			}
			out.Delayed, out.DelayReason = delayedState(now, o, o.RequestedDeliveryDate)
		} else {
			out.TargetLabel = "Pre-order"
		}
		return out
	}

	out.Kind = ExpectationKindStandard
	if o.DeliverBefore != nil {
		out.TargetDate = isoDatePtr(o.DeliverBefore)
		out.TargetLabel = fmt.Sprintf("Deliver by %s", formatDateLabel(*o.DeliverBefore))
		out.Urgency = urgencyForTarget(now, *o.DeliverBefore, o)
		out.Delayed, out.DelayReason = delayedState(now, o, o.DeliverBefore)
	} else {
		out.TargetLabel = "Standard delivery"
	}
	return out
}

func urgencyForTarget(now time.Time, target time.Time, o Order) string {
	if isTerminalStatus(o.Status) {
		return ExpectationUrgencyOnTrack
	}
	today := proximity.TashkentTodayStart(now)
	targetDay := proximity.TashkentTodayStart(target)
	days := calendarDaysBetween(today, targetDay)
	if days < 0 {
		return ExpectationUrgencyOverdue
	}
	if days <= 1 {
		return ExpectationUrgencyDueSoon
	}
	return ExpectationUrgencyOnTrack
}

func delayedState(now time.Time, o Order, target *time.Time) (bool, string) {
	if isTerminalStatus(o.Status) {
		return false, ""
	}
	if o.Status == StatusDelayed {
		return true, "Marked delayed by warehouse"
	}
	if o.ConfirmationStatus == ConfirmationStatusPendingWarehouse {
		return true, "Awaiting delivery date confirmation"
	}
	if target == nil {
		return false, ""
	}
	targetDay := proximity.TashkentTodayStart(*target)
	today := proximity.TashkentTodayStart(now)
	if today.After(targetDay) && isPreDeliveryStatus(o.Status) {
		switch {
		case IsScheduledPreorder(o):
			return true, "Pre-order date passed without dispatch"
		case o.Status == StatusPending || o.Status == StatusScheduled || o.Status == StatusAutoAccepted:
			return true, "Warehouse processing behind schedule"
		default:
			return true, "Delivery target passed"
		}
	}
	return false, ""
}

func isPreDeliveryStatus(s Status) bool {
	switch s {
	case StatusPending, StatusScheduled, StatusAutoAccepted, StatusDelayed, StatusBackordered,
		StatusLoaded, StatusCancelRequested:
		return true
	default:
		return false
	}
}

func isoDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := proximity.TashkentTodayStart(*t).Format("2006-01-02")
	return &s
}

func formatDateLabel(t time.Time) string {
	return proximity.TashkentTodayStart(t).Format("Jan 2, 2006")
}
