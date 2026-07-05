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
