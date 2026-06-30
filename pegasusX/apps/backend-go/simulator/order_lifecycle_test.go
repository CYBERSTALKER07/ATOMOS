// Package simulation_test provides end-to-end order lifecycle simulation tests.
// Tests the full state machine transitions, pricing calculation, checkout flow,
// and advanced notification scenarios without external dependencies.
package simulator_test

import (
	"fmt"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1. Order State Machine - Full transition matrix
// ─────────────────────────────────────────────────────────────────────────────

func TestSimOrderLifecycle_HappyPath(t *testing.T) {
	// Simulate: Pending → Loaded → InTransit → Arrived → AwaitingPayment → Completed
	transitions := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusPending, order.StatusLoaded},
		{order.StatusLoaded, order.StatusInTransit},
		{order.StatusInTransit, order.StatusArrived},
		{order.StatusArrived, order.StatusAwaitingPayment},
		{order.StatusAwaitingPayment, order.StatusCompleted},
	}

	for _, tr := range transitions {
		if err := order.ValidateStatusTransition(tr.from, tr.to); err != nil {
			t.Fatalf("happy path %s → %s: %v", tr.from, tr.to, err)
		}
	}
	t.Log("happy path lifecycle: Pending → Loaded → InTransit → Arrived → AwaitingPayment → Completed ✓")
}

func TestSimOrderLifecycle_CashPath(t *testing.T) {
	// Simulate: Pending → Loaded → InTransit → Arrived → PendingCashCollection → Completed
	transitions := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusPending, order.StatusLoaded},
		{order.StatusLoaded, order.StatusInTransit},
		{order.StatusInTransit, order.StatusArrived},
		{order.StatusArrived, order.StatusPendingCashCollection},
		{order.StatusPendingCashCollection, order.StatusCompleted},
	}

	for _, tr := range transitions {
		if err := order.ValidateStatusTransition(tr.from, tr.to); err != nil {
			t.Fatalf("cash path %s → %s: %v", tr.from, tr.to, err)
		}
	}
	t.Log("cash path lifecycle: Pending → Loaded → InTransit → Arrived → PendingCashCollection → Completed ✓")
}

func TestSimOrderLifecycle_CreditPath(t *testing.T) {
	// Simulate: Arrived → DeliveredOnCredit → Completed
	transitions := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusPending, order.StatusLoaded},
		{order.StatusLoaded, order.StatusInTransit},
		{order.StatusInTransit, order.StatusArrived},
		{order.StatusArrived, order.StatusDeliveredOnCredit},
		{order.StatusDeliveredOnCredit, order.StatusCompleted},
	}

	for _, tr := range transitions {
		if err := order.ValidateStatusTransition(tr.from, tr.to); err != nil {
			t.Fatalf("credit path %s → %s: %v", tr.from, tr.to, err)
		}
	}
	t.Log("credit path lifecycle: Pending → ... → Arrived → DeliveredOnCredit → Completed ✓")
}

func TestSimOrderLifecycle_CancellationPaths(t *testing.T) {
	cancellableFrom := []order.Status{
		order.StatusPending,
		order.StatusLoaded,
		order.StatusInTransit,
	}

	for _, from := range cancellableFrom {
		t.Run(fmt.Sprintf("%s_to_cancelled", from), func(t *testing.T) {
			if err := order.ValidateStatusTransition(from, order.StatusCancelled); err != nil {
				t.Errorf("should allow %s → Cancelled: %v", from, err)
			}
		})
	}
}

func TestSimOrderLifecycle_CancelRequestPaths(t *testing.T) {
	cancelRequestFrom := []order.Status{
		order.StatusLoaded,
		order.StatusInTransit,
		order.StatusArrived,
	}

	for _, from := range cancelRequestFrom {
		t.Run(fmt.Sprintf("%s_to_cancel_requested", from), func(t *testing.T) {
			if err := order.ValidateStatusTransition(from, order.StatusCancelRequested); err != nil {
				t.Errorf("should allow %s → CancelRequested: %v", from, err)
			}
		})
	}
}

func TestSimOrderLifecycle_DelayedRecovery(t *testing.T) {
	// Pending → Delayed → Pending → Loaded (delay recovery flow)
	transitions := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusPending, order.StatusDelayed},
		{order.StatusDelayed, order.StatusPending},
		{order.StatusPending, order.StatusLoaded},
	}

	for _, tr := range transitions {
		if err := order.ValidateStatusTransition(tr.from, tr.to); err != nil {
			t.Fatalf("delay recovery %s → %s: %v", tr.from, tr.to, err)
		}
	}
	t.Log("delay recovery: Pending → Delayed → Pending → Loaded ✓")
}

func TestSimOrderLifecycle_BackorderFlow(t *testing.T) {
	// Backordered → Pending → Loaded (backorder reactivation)
	transitions := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusBackordered, order.StatusPending},
		{order.StatusPending, order.StatusLoaded},
	}

	for _, tr := range transitions {
		if err := order.ValidateStatusTransition(tr.from, tr.to); err != nil {
			t.Fatalf("backorder flow %s → %s: %v", tr.from, tr.to, err)
		}
	}

	// Also test backorder → scheduled
	if err := order.ValidateStatusTransition(order.StatusBackordered, order.StatusScheduled); err != nil {
		t.Fatalf("backorder → scheduled: %v", err)
	}

	// And backorder → cancelled
	if err := order.ValidateStatusTransition(order.StatusBackordered, order.StatusCancelled); err != nil {
		t.Fatalf("backorder → cancelled: %v", err)
	}

	t.Log("backorder flow: Backordered → {Pending, Scheduled, Cancelled} ✓")
}

func TestSimOrderLifecycle_ReconciliationPath(t *testing.T) {
	// Cancelled → ReconciliationRequired → {Completed, Cancelled}
	transitions := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusCancelled, order.StatusReconciliationRequired},
		{order.StatusReconciliationRequired, order.StatusCompleted},
	}

	for _, tr := range transitions {
		if err := order.ValidateStatusTransition(tr.from, tr.to); err != nil {
			t.Fatalf("reconciliation path %s → %s: %v", tr.from, tr.to, err)
		}
	}

	// ReconciliationRequired → Cancelled is also valid
	if err := order.ValidateStatusTransition(order.StatusReconciliationRequired, order.StatusCancelled); err != nil {
		t.Fatalf("reconciliation → cancelled: %v", err)
	}

	t.Log("reconciliation path: Cancelled → ReconciliationRequired → {Completed, Cancelled} ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Order State Machine - Forbidden transitions
// ─────────────────────────────────────────────────────────────────────────────

func TestSimOrderLifecycle_ForbiddenTransitions(t *testing.T) {
	forbidden := []struct {
		from order.Status
		to   order.Status
	}{
		{order.StatusCompleted, order.StatusPending},
		{order.StatusCompleted, order.StatusCancelled},
		{order.StatusCompleted, order.StatusInTransit},
		{order.StatusDelayed, order.StatusCompleted},
		{order.StatusDelayed, order.StatusInTransit},
		{order.StatusPending, order.StatusCompleted},
		{order.StatusPending, order.StatusArrived},
		{order.StatusPending, order.StatusInTransit},
		{order.StatusLoaded, order.StatusCompleted},
		{order.StatusLoaded, order.StatusArrived},
		{order.StatusInTransit, order.StatusLoaded},
		{order.StatusInTransit, order.StatusCompleted},
		{order.StatusArrived, order.StatusInTransit},
		{order.StatusArrived, order.StatusLoaded},
	}

	for _, tr := range forbidden {
		t.Run(fmt.Sprintf("%s_to_%s", tr.from, tr.to), func(t *testing.T) {
			err := order.ValidateStatusTransition(tr.from, tr.to)
			if err == nil {
				t.Errorf("should forbid %s → %s but allowed it", tr.from, tr.to)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Order State Machine - Idempotent transitions
// ─────────────────────────────────────────────────────────────────────────────

func TestSimOrderLifecycle_IdempotentTransitions(t *testing.T) {
	statuses := []order.Status{
		order.StatusPending,
		order.StatusLoaded,
		order.StatusInTransit,
		order.StatusArrived,
		order.StatusAwaitingPayment,
		order.StatusCompleted,
		order.StatusCancelled,
	}

	for _, s := range statuses {
		t.Run(fmt.Sprintf("idempotent_%s", s), func(t *testing.T) {
			if err := order.ValidateStatusTransition(s, s); err != nil {
				t.Errorf("same-state transition %s → %s should be idempotent: %v", s, s, err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Rollback simulation — Loaded → Pending
// ─────────────────────────────────────────────────────────────────────────────

func TestSimOrderLifecycle_RollbackLoadedToPending(t *testing.T) {
	if err := order.ValidateStatusTransition(order.StatusLoaded, order.StatusPending); err != nil {
		t.Fatalf("rollback Loaded → Pending should be allowed: %v", err)
	}
	t.Log("rollback Loaded → Pending ✓")
}

func TestSimOrderLifecycle_RollbackInTransitToPending(t *testing.T) {
	if err := order.ValidateStatusTransition(order.StatusInTransit, order.StatusPending); err != nil {
		t.Fatalf("rollback InTransit → Pending should be allowed: %v", err)
	}
	t.Log("rollback InTransit → Pending ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. Full end-to-end lifecycle stress: 100 orders through happy path
// ─────────────────────────────────────────────────────────────────────────────

func TestSimOrderLifecycle_StressHappyPath(t *testing.T) {
	happyPath := []order.Status{
		order.StatusPending,
		order.StatusLoaded,
		order.StatusInTransit,
		order.StatusArrived,
		order.StatusAwaitingPayment,
		order.StatusCompleted,
	}

	const numOrders = 100
	for i := 0; i < numOrders; i++ {
		for j := 1; j < len(happyPath); j++ {
			if err := order.ValidateStatusTransition(happyPath[j-1], happyPath[j]); err != nil {
				t.Fatalf("order %d: transition %s → %s: %v", i, happyPath[j-1], happyPath[j], err)
			}
		}
	}
	t.Logf("stress: %d orders completed happy path lifecycle ✓", numOrders)
}
