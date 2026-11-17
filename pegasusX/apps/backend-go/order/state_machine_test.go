package order

import (
	"errors"
	"testing"
)

func TestValidateStatusTransition_pendingToLoaded(t *testing.T) {
	if err := ValidateStatusTransition(string(StatusPending), string(StatusLoaded), TransitionOpts{}); err != nil {
		t.Fatalf("expected allowed transition, got %v", err)
	}
}

func TestValidateStatusTransition_completedBlocked(t *testing.T) {
	err := ValidateStatusTransition(string(StatusCompleted), string(StatusPending), TransitionOpts{})
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestValidateStatusTransition_idempotent(t *testing.T) {
	if err := ValidateStatusTransition(string(StatusInTransit), string(StatusInTransit), TransitionOpts{}); err != nil {
		t.Fatalf("same status should be allowed, got %v", err)
	}
}

func TestValidateStatusTransition_cancelRequestedHasExits(t *testing.T) {
	for _, next := range []Status{StatusCancelled, StatusLoaded, StatusInTransit, StatusArrived} {
		if err := ValidateStatusTransition(string(StatusCancelRequested), string(next), TransitionOpts{}); err != nil {
			t.Fatalf("CANCEL_REQUESTED -> %s should be allowed, got %v", next, err)
		}
	}
	if err := ValidateStatusTransition(string(StatusCancelRequested), string(StatusCompleted), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("CANCEL_REQUESTED -> COMPLETED must stay blocked, got %v", err)
	}
}

func TestValidateStatusTransition_shopClosedPending(t *testing.T) {
	// ARRIVED → ARRIVED_SHOP_CLOSED (design SHOP_CLOSED_PENDING)
	if err := ValidateStatusTransition(string(StatusArrived), string(StatusShopClosedPending), TransitionOpts{}); err != nil {
		t.Fatalf("ARRIVED -> ARRIVED_SHOP_CLOSED should be allowed: %v", err)
	}
	// Resolutions from shop-closed pending
	for _, next := range []Status{StatusAwaitingPayment, StatusPendingCashCollection, StatusDeliveredOnCredit, StatusCancelled, StatusArrived} {
		if err := ValidateStatusTransition(string(StatusShopClosedPending), string(next), TransitionOpts{}); err != nil {
			t.Fatalf("ARRIVED_SHOP_CLOSED -> %s should be allowed: %v", next, err)
		}
	}
	if err := ValidateStatusTransition(string(StatusShopClosedPending), string(StatusCompleted), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("ARRIVED_SHOP_CLOSED -> COMPLETED must stay blocked")
	}
}

func TestValidateStatusTransition_fiscalHardGate(t *testing.T) {
	// Forbidden soft-completes
	for _, from := range []Status{StatusArrived, StatusAwaitingPayment, StatusPendingCashCollection} {
		if err := ValidateStatusTransition(string(from), string(StatusCompleted), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("%s -> COMPLETED must be blocked, got %v", from, err)
		}
	}
	// Capture → FISCALIZING
	for _, from := range []Status{StatusAwaitingPayment, StatusPendingCashCollection, StatusDeliveredOnCredit} {
		if err := ValidateStatusTransition(string(from), string(StatusFiscalizing), TransitionOpts{}); err != nil {
			t.Fatalf("%s -> FISCALIZING should be allowed, got %v", from, err)
		}
	}
	if err := ValidateStatusTransition(string(StatusFiscalizing), string(StatusCompleted), TransitionOpts{}); err != nil {
		t.Fatalf("FISCALIZING -> COMPLETED should be allowed, got %v", err)
	}
	if err := ValidateStatusTransition(string(StatusFiscalizing), string(StatusFiscalFailed), TransitionOpts{}); err != nil {
		t.Fatalf("FISCALIZING -> FISCAL_FAILED should be allowed, got %v", err)
	}
	if err := ValidateStatusTransition(string(StatusFiscalFailed), string(StatusFiscalizing), TransitionOpts{}); err != nil {
		t.Fatalf("FISCAL_FAILED -> FISCALIZING should be allowed, got %v", err)
	}
	if err := ValidateStatusTransition(string(StatusFiscalFailed), string(StatusCompleted), TransitionOpts{}); err != nil {
		t.Fatalf("FISCAL_FAILED -> COMPLETED (force) should be allowed, got %v", err)
	}
}

func TestValidateStatusTransition_preorderStatuses(t *testing.T) {
	// SCHEDULED → AUTO_ACCEPTED | PENDING | CANCELLED
	for _, next := range []Status{StatusAutoAccepted, StatusPending, StatusCancelled, StatusCancelRequested} {
		if err := ValidateStatusTransition(string(StatusScheduled), string(next), TransitionOpts{}); err != nil {
			t.Fatalf("SCHEDULED -> %s should be allowed, got %v", next, err)
		}
	}
	if err := ValidateStatusTransition(string(StatusScheduled), string(StatusLoaded), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("SCHEDULED -> LOADED must be blocked, got %v", err)
	}
	// AUTO_ACCEPTED → PENDING | CANCELLED
	for _, next := range []Status{StatusPending, StatusCancelled, StatusCancelRequested} {
		if err := ValidateStatusTransition(string(StatusAutoAccepted), string(next), TransitionOpts{}); err != nil {
			t.Fatalf("AUTO_ACCEPTED -> %s should be allowed, got %v", next, err)
		}
	}
	if err := ValidateStatusTransition(string(StatusAutoAccepted), string(StatusInTransit), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("AUTO_ACCEPTED -> IN_TRANSIT must be blocked, got %v", err)
	}
}

func TestValidateStatusTransition_AwaitingPaymentAndPendingCashTransitions(t *testing.T) {
	// StatusAwaitingPayment: allow Fiscalizing, PendingCashCollection, DeliveredOnCredit, CancelRequested, ShopClosedPending
	allowedAwaiting := []Status{
		StatusFiscalizing,
		StatusPendingCashCollection,
		StatusDeliveredOnCredit,
		StatusCancelRequested,
		StatusShopClosedPending,
	}
	for _, next := range allowedAwaiting {
		if err := ValidateStatusTransition(string(StatusAwaitingPayment), string(next), TransitionOpts{}); err != nil {
			t.Errorf("AWAITING_PAYMENT -> %s should be allowed, got: %v", next, err)
		}
		if !CanTransition(StatusAwaitingPayment, next) {
			t.Errorf("CanTransition(AWAITING_PAYMENT, %s) should be true", next)
		}
	}
	// Invalid transitions from AwaitingPayment
	invalidAwaiting := []Status{StatusCompleted, StatusLoaded, StatusInTransit, StatusScheduled}
	for _, next := range invalidAwaiting {
		if err := ValidateStatusTransition(string(StatusAwaitingPayment), string(next), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Errorf("AWAITING_PAYMENT -> %s must be blocked, got: %v", next, err)
		}
		if CanTransition(StatusAwaitingPayment, next) {
			t.Errorf("CanTransition(AWAITING_PAYMENT, %s) should be false", next)
		}
	}

	// StatusPendingCashCollection: allow Fiscalizing, AwaitingPayment, DeliveredOnCredit, CancelRequested
	allowedCash := []Status{
		StatusFiscalizing,
		StatusAwaitingPayment,
		StatusDeliveredOnCredit,
		StatusCancelRequested,
	}
	for _, next := range allowedCash {
		if err := ValidateStatusTransition(string(StatusPendingCashCollection), string(next), TransitionOpts{}); err != nil {
			t.Errorf("PENDING_CASH_COLLECTION -> %s should be allowed, got: %v", next, err)
		}
		if !CanTransition(StatusPendingCashCollection, next) {
			t.Errorf("CanTransition(PENDING_CASH_COLLECTION, %s) should be true", next)
		}
	}
	// Invalid transitions from PendingCashCollection
	invalidCash := []Status{StatusCompleted, StatusLoaded, StatusInTransit, StatusScheduled}
	for _, next := range invalidCash {
		if err := ValidateStatusTransition(string(StatusPendingCashCollection), string(next), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Errorf("PENDING_CASH_COLLECTION -> %s must be blocked, got: %v", next, err)
		}
		if CanTransition(StatusPendingCashCollection, next) {
			t.Errorf("CanTransition(PENDING_CASH_COLLECTION, %s) should be false", next)
		}
	}
}

func TestValidateStatusTransition_CancelRequestedExtendedExits(t *testing.T) {
	// StatusCancelRequested: allow Cancelled, Loaded, InTransit, Arrived, Scheduled, AutoAccepted, ShopClosedPending
	allowed := []Status{
		StatusCancelled,
		StatusLoaded,
		StatusInTransit,
		StatusArrived,
		StatusScheduled,
		StatusAutoAccepted,
		StatusShopClosedPending,
	}
	for _, next := range allowed {
		if err := ValidateStatusTransition(string(StatusCancelRequested), string(next), TransitionOpts{}); err != nil {
			t.Errorf("CANCEL_REQUESTED -> %s should be allowed, got: %v", next, err)
		}
		if !CanTransition(StatusCancelRequested, next) {
			t.Errorf("CanTransition(CANCEL_REQUESTED, %s) should be true", next)
		}
	}
	// Invalid transitions from CancelRequested
	invalid := []Status{StatusCompleted, StatusFiscalizing, StatusFiscalFailed, StatusDeliveredOnCredit}
	for _, next := range invalid {
		if err := ValidateStatusTransition(string(StatusCancelRequested), string(next), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Errorf("CANCEL_REQUESTED -> %s must be blocked, got: %v", next, err)
		}
		if CanTransition(StatusCancelRequested, next) {
			t.Errorf("CanTransition(CANCEL_REQUESTED, %s) should be false", next)
		}
	}
}
