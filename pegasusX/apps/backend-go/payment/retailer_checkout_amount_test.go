package payment

import (
	"errors"
	"testing"
)

func TestResolveCardCheckoutAmount_SnapshotWins(t *testing.T) {
	got, err := resolveCardCheckoutAmount(0, 0, 1500)
	if err != nil || got != 1500 {
		t.Fatalf("got %d %v want 1500", got, err)
	}
}

func TestResolveCardCheckoutAmount_MatchingClientOK(t *testing.T) {
	got, err := resolveCardCheckoutAmount(1500, 0, 1500)
	if err != nil || got != 1500 {
		t.Fatalf("got %d %v", got, err)
	}
}

func TestResolveCardCheckoutAmount_LegacyField(t *testing.T) {
	got, err := resolveCardCheckoutAmount(0, 1500, 1500)
	if err != nil || got != 1500 {
		t.Fatalf("got %d %v", got, err)
	}
}

func TestResolveCardCheckoutAmount_ClientMismatch(t *testing.T) {
	_, err := resolveCardCheckoutAmount(1, 0, 1500)
	if !errors.Is(err, ErrCheckoutAmountMismatch) {
		t.Fatalf("err=%v want amount_mismatch", err)
	}
}

func TestResolveCardCheckoutAmount_NoSnapshot(t *testing.T) {
	_, err := resolveCardCheckoutAmount(1500, 0, 0)
	if !errors.Is(err, ErrOrderSnapshotRequired) {
		t.Fatalf("err=%v want snapshot required", err)
	}
}
