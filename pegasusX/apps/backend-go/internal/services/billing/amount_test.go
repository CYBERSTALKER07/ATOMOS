package billing

import (
	"context"
	"testing"
)

func TestResolveMeterAmountMajor(t *testing.T) {
	t.Parallel()
	if got := ResolveMeterAmountMajor(12500, 0, 0, 0); got != 125.0 {
		t.Fatalf("got %v", got)
	}
}

func TestProcessOrderFinalized_RejectsNonPositive(t *testing.T) {
	t.Parallel()
	w := NewMeterWorker(nil)
	// Nil client would error on positive amount; non-positive must no-op without Spanner.
	if err := w.ProcessOrderFinalized(context.Background(), "ord-1", 0, "sup-1"); err != nil {
		t.Fatalf("zero amount: %v", err)
	}
	if err := w.ProcessOrderFinalized(context.Background(), "ord-1", -1, "sup-1"); err != nil {
		t.Fatalf("negative amount: %v", err)
	}
	if err := w.ProcessOrderFinalized(context.Background(), "", 10, "sup-1"); err != nil {
		t.Fatalf("empty order: %v", err)
	}
}

func TestProcessOrderFinalized_NilClientOnPositive(t *testing.T) {
	t.Parallel()
	w := NewMeterWorker(nil)
	err := w.ProcessOrderFinalized(context.Background(), "ord-1", 10, "sup-1")
	if err == nil {
		t.Fatal("expected error for nil client with positive amount")
	}
}
