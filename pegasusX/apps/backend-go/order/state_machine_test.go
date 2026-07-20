package order

import (
	"errors"
	"testing"
)

func TestValidateStatusTransition_pendingToLoaded(t *testing.T) {
	if err := ValidateStatusTransition(StatusPending, StatusLoaded); err != nil {
		t.Fatalf("expected allowed transition, got %v", err)
	}
}

func TestValidateStatusTransition_completedBlocked(t *testing.T) {
	err := ValidateStatusTransition(StatusCompleted, StatusPending)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestValidateStatusTransition_idempotent(t *testing.T) {
	if err := ValidateStatusTransition(StatusInTransit, StatusInTransit); err != nil {
		t.Fatalf("same status should be allowed, got %v", err)
	}
}

func TestValidateStatusTransition_cancelRequestedHasExits(t *testing.T) {
	for _, next := range []Status{StatusCancelled, StatusLoaded, StatusInTransit, StatusArrived} {
		if err := ValidateStatusTransition(StatusCancelRequested, next); err != nil {
			t.Fatalf("CANCEL_REQUESTED -> %s should be allowed, got %v", next, err)
		}
	}
	if err := ValidateStatusTransition(StatusCancelRequested, StatusCompleted); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("CANCEL_REQUESTED -> COMPLETED must stay blocked, got %v", err)
	}
}

func TestValidateStatusTransition_fiscalHardGate(t *testing.T) {
	// Forbidden soft-completes
	for _, from := range []Status{StatusArrived, StatusAwaitingPayment, StatusPendingCashCollection} {
		if err := ValidateStatusTransition(from, StatusCompleted); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("%s -> COMPLETED must be blocked, got %v", from, err)
		}
	}
	// Capture → FISCALIZING
	for _, from := range []Status{StatusAwaitingPayment, StatusPendingCashCollection, StatusDeliveredOnCredit} {
		if err := ValidateStatusTransition(from, StatusFiscalizing); err != nil {
			t.Fatalf("%s -> FISCALIZING should be allowed, got %v", from, err)
		}
	}
	if err := ValidateStatusTransition(StatusFiscalizing, StatusCompleted); err != nil {
		t.Fatalf("FISCALIZING -> COMPLETED should be allowed, got %v", err)
	}
	if err := ValidateStatusTransition(StatusFiscalizing, StatusFiscalFailed); err != nil {
		t.Fatalf("FISCALIZING -> FISCAL_FAILED should be allowed, got %v", err)
	}
	if err := ValidateStatusTransition(StatusFiscalFailed, StatusFiscalizing); err != nil {
		t.Fatalf("FISCAL_FAILED -> FISCALIZING should be allowed, got %v", err)
	}
	if err := ValidateStatusTransition(StatusFiscalFailed, StatusCompleted); err != nil {
		t.Fatalf("FISCAL_FAILED -> COMPLETED (force) should be allowed, got %v", err)
	}
}

func TestValidateStatusTransition_preorderStatuses(t *testing.T) {
	// SCHEDULED → AUTO_ACCEPTED | PENDING | CANCELLED
	for _, next := range []Status{StatusAutoAccepted, StatusPending, StatusCancelled, StatusCancelRequested} {
		if err := ValidateStatusTransition(StatusScheduled, next); err != nil {
			t.Fatalf("SCHEDULED -> %s should be allowed, got %v", next, err)
		}
	}
	if err := ValidateStatusTransition(StatusScheduled, StatusLoaded); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("SCHEDULED -> LOADED must be blocked, got %v", err)
	}
	// AUTO_ACCEPTED → PENDING | CANCELLED
	for _, next := range []Status{StatusPending, StatusCancelled, StatusCancelRequested} {
		if err := ValidateStatusTransition(StatusAutoAccepted, next); err != nil {
			t.Fatalf("AUTO_ACCEPTED -> %s should be allowed, got %v", next, err)
		}
	}
	if err := ValidateStatusTransition(StatusAutoAccepted, StatusInTransit); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("AUTO_ACCEPTED -> IN_TRANSIT must be blocked, got %v", err)
	}
}
