package ar

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDesiredDunningStep(t *testing.T) {
	due := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		now   time.Time
		grace int64
		want  int64
	}{
		{"far_before", due.Add(-10 * 24 * time.Hour), 0, StepNone},
		{"due_soon", due.Add(-48 * time.Hour), 0, StepDueSoon},
		{"grace", due.Add(24 * time.Hour), 3, StepDueSoon},
		{"overdue", due.Add(2 * 24 * time.Hour), 0, StepOverdue},
		{"esc1", due.Add(8 * 24 * time.Hour), 0, StepEscalated1},
		{"esc2", due.Add(15 * 24 * time.Hour), 0, StepEscalated2},
		{"hold", due.Add(22 * 24 * time.Hour), 0, StepCreditHold},
		{"collections", due.Add(35 * 24 * time.Hour), 0, StepCollections},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DesiredDunningStep(due, tc.grace, tc.now)
			if got != tc.want {
				t.Fatalf("got %d (%s) want %d (%s)", got, StepName(got), tc.want, StepName(tc.want))
			}
		})
	}
}

func TestAdvanceDunningHoldAndDelinquency(t *testing.T) {
	_ = os.Setenv("AR_INVOICES_ENABLED", "1")
	_ = os.Setenv("AR_DUNNING_ENABLED", "1")
	t.Cleanup(func() {
		_ = os.Unsetenv("AR_INVOICES_ENABLED")
		_ = os.Unsetenv("AR_DUNNING_ENABLED")
	})

	repo := NewMemoryRepository()
	svc := NewService(repo)
	fixed := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return fixed })

	due := fixed.Add(-22 * 24 * time.Hour) // CREDIT_HOLD
	inv := Invoice{
		InvoiceID: "ari-1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1",
		Status: StatusOpen, PrincipalMinor: 1000, BalanceMinor: 1000, Currency: "UZS",
		CreditLeaveAt: due.Add(-30 * 24 * time.Hour), DueAt: due, TermsDays: 30,
		Version: 1, CreatedAt: fixed, UpdatedAt: fixed,
	}
	if err := repo.OpenInvoice(context.Background(), inv); err != nil {
		t.Fatal(err)
	}

	var held, bumped int
	var notified int64
	w := NewDunningWorker(svc, nil)
	w.now = func() time.Time { return fixed }
	w.SetAutoHold(func(ctx context.Context, supplierID, retailerID string) error {
		held++
		return nil
	})
	w.SetDelinquencyBump(func(ctx context.Context, supplierID, retailerID string) error {
		bumped++
		return nil
	})
	w.SetNotify(func(ctx context.Context, inv Invoice, prev, next int64) error {
		notified = next
		return nil
	})

	adv, h, b, err := w.AdvanceDunning(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if adv != 1 || h != 1 || b != 1 {
		t.Fatalf("adv=%d held=%d bumped=%d", adv, h, b)
	}
	if notified != StepCreditHold {
		t.Fatalf("notified step=%d", notified)
	}
	got, ok, err := repo.GetByOrder(context.Background(), "ord-1")
	if err != nil || !ok {
		t.Fatal(err)
	}
	if got.DunningStep != StepCreditHold {
		t.Fatalf("step=%d", got.DunningStep)
	}

	// Monotonic: second pass no re-advance
	adv2, h2, b2, err := w.AdvanceDunning(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if adv2 != 0 || h2 != 0 || b2 != 0 {
		t.Fatalf("second pass adv=%d h=%d b=%d", adv2, h2, b2)
	}
}

func TestShouldBumpAndHold(t *testing.T) {
	if !ShouldBumpDelinquency(StepDueSoon, StepOverdue) {
		t.Fatal("expected bump")
	}
	if ShouldBumpDelinquency(StepOverdue, StepEscalated1) {
		t.Fatal("no second bump")
	}
	if !ShouldAutoHold(StepEscalated2, StepCreditHold) {
		t.Fatal("expected hold")
	}
	if ShouldAutoHold(StepCreditHold, StepCollections) {
		t.Fatal("hold only on first entry")
	}
}
