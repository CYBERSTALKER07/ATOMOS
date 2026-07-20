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
		// ADR-009: no soft ARRIVED → COMPLETED (fiscal hard-gate).
		allowed = next == StatusAwaitingPayment || next == StatusPendingCashCollection || next == StatusDeliveredOnCredit || next == StatusCancelRequested
	case StatusArrivedShopClosed:
		allowed = next == StatusAwaitingPayment || next == StatusDeliveredOnCredit
	case StatusDeliveredOnCredit:
		// §9.1: fiscal only when money received — settlement capture enters FISCALIZING.
		// Force-complete (ADMIN/WAREHOUSE_ADMIN) may still land COMPLETED via service gate.
		allowed = next == StatusFiscalizing || next == StatusCompleted
	case StatusAwaitingPayment:
		// Card/cash capture → FISCALIZING; cash choice → PENDING_CASH_COLLECTION.
		allowed = next == StatusFiscalizing || next == StatusPendingCashCollection || next == StatusDeliveredOnCredit
	case StatusPendingCashCollection:
		allowed = next == StatusFiscalizing
	case StatusFiscalizing:
		allowed = next == StatusCompleted || next == StatusFiscalFailed
	case StatusFiscalFailed:
		// Retry → FISCALIZING; force-complete → COMPLETED (role-gated in service).
		allowed = next == StatusFiscalizing || next == StatusCompleted
	case StatusCancelRequested:
		// Approve -> CANCELLED; deny/resume -> back to the operational leg it
		// was requested from. Without exits this status would brick the order.
		allowed = next == StatusCancelled || next == StatusLoaded || next == StatusInTransit || next == StatusArrived
	case StatusCompleted:
		allowed = false
	case StatusCancelled:
		allowed = next == StatusReconciliationRequired
	case StatusReconciliationRequired:
		allowed = next == StatusCompleted || next == StatusCancelled
	case StatusBackordered:
		allowed = next == StatusPending || next == StatusScheduled || next == StatusCancelled
	case StatusScheduled:
		// Preorder: midnight guard may auto-accept; T-1 promote → PENDING;
		// retailer/warehouse may cancel before promote.
		allowed = next == StatusAutoAccepted || next == StatusPending || next == StatusCancelled || next == StatusCancelRequested
	case StatusAutoAccepted:
		// Confirmed preorder waiting for T-1 promote into the operational queue.
		allowed = next == StatusPending || next == StatusCancelled || next == StatusCancelRequested
	default:
		allowed = false
	}

	if !allowed {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, current, next)
	}

	return nil
}
