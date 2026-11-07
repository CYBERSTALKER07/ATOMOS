package order

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAssertPaidEqualsDue(t *testing.T) {
	if err := assertPaidEqualsDue(1500, 1500); err != nil {
		t.Fatal(err)
	}
	if err := assertPaidEqualsDue(0, 1500); !errors.Is(err, ErrPaymentAmountRequired) {
		t.Fatalf("zero paid: %v", err)
	}
	if err := assertPaidEqualsDue(1500, 0); !errors.Is(err, ErrPaymentAmountRequired) {
		t.Fatalf("zero due: %v", err)
	}
	if err := assertPaidEqualsDue(1000, 1500); !errors.Is(err, ErrPaymentAmountMismatch) {
		t.Fatalf("short pay: %v", err)
	}
}

func TestSettleExternalPayment_RejectsPaidNotDue(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusAwaitingPayment)
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)
	if err := svc.SettleExternalPayment(context.Background(), o.OrderID, "payme", 1); !errors.Is(err, ErrPaymentAmountMismatch) {
		t.Fatalf("err=%v want mismatch", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("updateCalls=%d want 0", repo.updateCalls)
	}
}

func TestSettleExternalPayment_RejectsMissingPaid(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusAwaitingPayment)
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)
	if err := svc.SettleExternalPayment(context.Background(), o.OrderID, "payme", 0); !errors.Is(err, ErrPaymentAmountRequired) {
		t.Fatalf("err=%v want required", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("updateCalls=%d want 0", repo.updateCalls)
	}
}
