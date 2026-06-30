package order

import "fmt"

// ValidateStatusTransition enforces the canonical order lifecycle graph.
func ValidateStatusTransition(current Status, next Status) error {
	if current == next {
		return nil
	}

	allowed := false
	switch current {
	case StatusPending:
		allowed = next == StatusLoaded || next == StatusCancelled || next == StatusDelayed
	case StatusLoaded:
		allowed = next == StatusInTransit || next == StatusCancelled || next == StatusCancelRequested || next == StatusDelayed || next == StatusPending
	case StatusDelayed:
		allowed = next == StatusPending
	case StatusInTransit:
		allowed = next == StatusArrived || next == StatusCancelled || next == StatusCancelRequested || next == StatusPending
	case StatusArrived:
		allowed = next == StatusAwaitingPayment || next == StatusPendingCashCollection || next == StatusCompleted || next == StatusDeliveredOnCredit || next == StatusCancelRequested
	case StatusArrivedShopClosed:
		allowed = next == StatusAwaitingPayment || next == StatusDeliveredOnCredit
	case StatusDeliveredOnCredit:
		allowed = next == StatusCompleted
	case StatusAwaitingPayment:
		allowed = next == StatusCompleted || next == StatusPendingCashCollection
	case StatusPendingCashCollection:
		allowed = next == StatusCompleted
	case StatusCompleted:
		allowed = false
	case StatusCancelled:
		allowed = next == StatusReconciliationRequired
	case StatusReconciliationRequired:
		allowed = next == StatusCompleted || next == StatusCancelled
	case StatusBackordered:
		allowed = next == StatusPending || next == StatusScheduled || next == StatusCancelled
	default:
		allowed = false
	}

	if !allowed {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, current, next)
	}

	return nil
}
