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
