package ar

import (
	"context"
	"testing"
	"time"
)

func TestOpenInvoiceIdempotentAndDueDate(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	svc := NewService(NewMemoryRepository())
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	due := leave.AddDate(0, 0, 14)
	inv1, err := svc.OpenFromCreditLeave(context.Background(), "s", "r", "ord-1", 50_000, 14, 0, leave, due)
	if err != nil {
		t.Fatal(err)
	}
	inv2, err := svc.OpenFromCreditLeave(context.Background(), "s", "r", "ord-1", 50_000, 14, 0, leave, due)
	if err != nil {
		t.Fatal(err)
	}
	if inv1.InvoiceID != inv2.InvoiceID {
		t.Fatalf("idempotent mismatch %s vs %s", inv1.InvoiceID, inv2.InvoiceID)
	}
	if !inv1.DueAt.Equal(due) {
		t.Fatalf("due=%v want %v", inv1.DueAt, due)
	}
}

func TestAgingBuckets(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if AgingBucketFor(now.AddDate(0, 0, 1), now) != AgingCurrent {
		t.Fatal("current")
	}
	if AgingBucketFor(now.AddDate(0, 0, -10), now) != Aging1to30 {
		t.Fatal("1_30")
	}
	if AgingBucketFor(now.AddDate(0, 0, -100), now) != Aging90Plus {
		t.Fatal("90+")
	}
}

func TestGlobalTermsChangeDoesNotMutateOpenInvoice(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	leave := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	due := leave.AddDate(0, 0, 7)
	inv, err := svc.OpenFromCreditLeave(context.Background(), "s", "r", "ord-2", 10_000, 7, 0, leave, due)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate terms change — open invoice DueAt must stay.
	if inv.TermsDays != 7 || !inv.DueAt.Equal(due) {
		t.Fatalf("invoice mutated unexpectedly: %+v", inv)
	}
}
