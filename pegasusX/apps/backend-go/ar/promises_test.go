package ar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestQuarterKey_AsiaTashkent(t *testing.T) {
	cases := []struct {
		ts   time.Time
		want string
	}{
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "2026-Q1"},
		{time.Date(2026, 3, 31, 18, 59, 59, 0, time.UTC), "2026-Q1"},
		{time.Date(2026, 3, 31, 19, 0, 0, 0, time.UTC), "2026-Q2"}, // 00:00 Apr 1 Tashkent
		{time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC), "2026-Q4"},
		{time.Date(2026, 12, 31, 19, 0, 0, 0, time.UTC), "2027-Q1"}, // 00:00 Jan 1 2027 Tashkent
	}
	for _, tc := range cases {
		if got := QuarterKey(tc.ts, ""); got != tc.want {
			t.Fatalf("QuarterKey(%v) = %q, want %q", tc.ts, got, tc.want)
		}
	}
}

func TestQuarterlyPromise_SelfServeLimitAndSupplierOverride(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	t.Setenv("AR_MAX_PROMISE_DAYS", "7")

	repo := NewMemoryRepository()
	svc := NewService(repo)
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:       "ret-user",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-1",
	})

	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1",
		Status: StatusOpen, PrincipalMinor: 100_000, BalanceMinor: 100_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -2), GracePeriodDays: 1, Version: 1, CreatedAt: now, UpdatedAt: now,
	})

	// Reject invalid extension days
	if _, err := svc.CreateRetailerPromise(ctx, "ret-1", "sup-1", "inv-1", 8); !errors.Is(err, ErrPromiseInvalidExtension) {
		t.Fatalf("expected ErrPromiseInvalidExtension, got %v", err)
	}

	// Reject prepay-locked relationship
	repo.SetCreditProfileState("ret-1", "sup-locked", "ACTIVE", true, "")
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-locked", SupplierID: "sup-locked", RetailerID: "ret-1", OrderID: "ord-locked",
		Status: StatusOpen, PrincipalMinor: 50_000, BalanceMinor: 50_000, Currency: "UZS",
		DueAt: now, Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if _, err := svc.CreateRetailerPromise(ctx, "ret-1", "sup-locked", "inv-locked", 3); !errors.Is(err, ErrPromisePrepayLocked) {
		t.Fatalf("expected ErrPromisePrepayLocked, got %v", err)
	}

	// 1st self-serve promise succeeds and sets ActiveQuarterKey = 2026-Q4
	p1, err := svc.CreateRetailerPromise(ctx, "ret-1", "sup-1", "inv-1", 5)
	if err != nil {
		t.Fatalf("CreateRetailerPromise: %v", err)
	}
	if p1.QuarterKey != "2026-Q4" || p1.ApprovedBy != PromiseApprovedBySystem || p1.Status != PromiseStatusActive {
		t.Fatalf("unexpected promise: %+v", p1)
	}
	st, ok := repo.GetCreditProfileState("ret-1", "sup-1")
	if !ok || st.ActiveQuarterKey != "2026-Q4" {
		t.Fatalf("expected ActiveQuarterKey=2026-Q4, got %+v (ok=%v)", st, ok)
	}

	// 2nd self-serve promise in same quarter fails with ErrQuarterlyPromiseLimitReached
	_, err = svc.CreateRetailerPromise(ctx, "ret-1", "sup-1", "inv-1", 3)
	if !errors.Is(err, ErrQuarterlyPromiseLimitReached) {
		t.Fatalf("expected ErrQuarterlyPromiseLimitReached, got %v", err)
	}

	// Supplier admin override succeeds in the same quarter
	adminCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "finance-lead-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	})
	pOverride, err := svc.GrantSupplierPromise(adminCtx, "sup-1", "ret-1", "inv-1", 10, "")
	if err != nil {
		t.Fatalf("GrantSupplierPromise: %v", err)
	}
	if pOverride.ApprovedBy != "finance-lead-1" || pOverride.QuarterKey != "2026-Q4" {
		t.Fatalf("unexpected override promise: %+v", pOverride)
	}

	// Next quarter unlocks self-service again
	svc.SetNow(func() time.Time { return time.Date(2027, 1, 5, 10, 0, 0, 0, time.UTC) })
	pNextQ, err := svc.CreateRetailerPromise(ctx, "ret-1", "sup-1", "inv-1", 4)
	if err != nil || pNextQ.QuarterKey != "2027-Q1" {
		t.Fatalf("next quarter promise failed: %+v err=%v", pNextQ, err)
	}
}

func TestPromiseBreachWorker_FulfillAndBreach(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := context.Background()
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return t0 })

	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-paid", SupplierID: "sup-1", RetailerID: "ret-good", OrderID: "ord-good",
		Status: StatusOpen, PrincipalMinor: 50_000, BalanceMinor: 50_000, Currency: "UZS",
		DueAt: t0.AddDate(0, 0, -2), Version: 1, CreatedAt: t0, UpdatedAt: t0,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-unpaid", SupplierID: "sup-1", RetailerID: "ret-bad", OrderID: "ord-bad",
		Status: StatusOpen, PrincipalMinor: 90_000, BalanceMinor: 90_000, Currency: "UZS",
		DueAt: t0.AddDate(0, 0, -2), Version: 1, CreatedAt: t0, UpdatedAt: t0,
	})

	pGood, err := svc.CreateRetailerPromise(ctx, "ret-good", "sup-1", "inv-paid", 3)
	if err != nil {
		t.Fatal(err)
	}
	pBad, err := svc.CreateRetailerPromise(ctx, "ret-bad", "sup-1", "inv-unpaid", 3)
	if err != nil {
		t.Fatal(err)
	}

	// Mark inv-paid settled manually (without hitting promise) to verify worker resolves FULFILLED
	_ = repo.ApplyPayment(ctx, "inv-paid", 50_000, "manual-paid")

	// Advance clock past 3-day DueAt
	tAfter := t0.AddDate(0, 0, 5)
	svc.SetNow(func() time.Time { return tAfter })

	var breachedIDs, outcomes []string
	w := NewPromiseBreachWorker(svc, nil)
	w.SetOnBreach(func(_ context.Context, supplierID, retailerID, promiseID string) error {
		breachedIDs = append(breachedIDs, retailerID+":"+promiseID)
		return nil
	})
	w.SetNotify(func(_ context.Context, promise PaymentPromise, outcome string) error {
		outcomes = append(outcomes, promise.PromiseID+":"+outcome)
		return nil
	})

	fulfilled, breached, err := w.RunPromiseBreachOnce(ctx)
	if err != nil {
		t.Fatalf("RunPromiseBreachOnce: %v", err)
	}
	if fulfilled != 1 || breached != 1 {
		t.Fatalf("fulfilled=%d breached=%d, want 1/1", fulfilled, breached)
	}
	if len(breachedIDs) != 1 || breachedIDs[0] != "ret-bad:"+pBad.PromiseID {
		t.Fatalf("unexpected breached callbacks: %+v", breachedIDs)
	}
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 notify callbacks, got %+v", outcomes)
	}

	goodList, _ := svc.ListPromises(ctx, "ret-good", "sup-1", 10)
	badList, _ := svc.ListPromises(ctx, "ret-bad", "sup-1", 10)
	if goodList[0].PromiseID != pGood.PromiseID || goodList[0].Status != PromiseStatusFulfilled {
		t.Fatalf("expected ret-good FULFILLED, got %+v", goodList)
	}
	if badList[0].PromiseID != pBad.PromiseID || badList[0].Status != PromiseStatusBreached {
		t.Fatalf("expected ret-bad BREACHED, got %+v", badList)
	}
}

func TestDunningAndDelinquency_DisputeExclusionAndActivePromisePause(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	t.Setenv("AR_DUNNING_ENABLED", "true")

	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	// inv-disp: 30 days overdue, 100k balance, 100k disputed -> must be skipped by dunning & NOT locked
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-disp", SupplierID: "sup-1", RetailerID: "ret-disp", OrderID: "ord-disp",
		Status: StatusOpen, PrincipalMinor: 100_000, BalanceMinor: 100_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -30), DunningStep: StepCreditHold, Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	// inv-prom: 25 days overdue, active unexpired promise -> paused for dunning & lock
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-prom", SupplierID: "sup-1", RetailerID: "ret-prom", OrderID: "ord-prom",
		Status: StatusOpen, PrincipalMinor: 100_000, BalanceMinor: 100_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -25), DunningStep: StepOverdue, Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	svc.SetDisputeLookup(func(_ context.Context, _ []string) (map[string]int64, error) {
		return map[string]int64{"ord-disp": 100_000}, nil
	})
	if _, err := svc.CreateRetailerPromise(ctx, "ret-prom", "sup-1", "inv-prom", 5); err != nil {
		t.Fatal(err)
	}

	var held, bumped int
	w := NewDunningWorker(svc, nil)
	w.SetAutoHold(func(_ context.Context, _, _ string) error { held++; return nil })
	w.SetDelinquencyBump(func(_ context.Context, _, _ string) error { bumped++; return nil })

	adv, h, b, err := w.AdvanceDunning(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if adv != 0 || h != 0 || b != 0 || held != 0 || bumped != 0 {
		t.Fatalf("expected dunning paused/skipped for disputed and promised invoices, got adv=%d h=%d b=%d", adv, h, b)
	}

	lockDisp, err := svc.CheckRetailerDelinquencyLock(ctx, "ret-disp", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if lockDisp.IsLocked || lockDisp.DisputedMinor != 100_000 || lockDisp.UndisputedMinor != 0 {
		t.Fatalf("disputed-only retailer must not lock: %+v", lockDisp)
	}

	lockProm, err := svc.CheckRetailerDelinquencyLock(ctx, "ret-prom", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if lockProm.IsLocked || lockProm.ActivePromise == nil {
		t.Fatalf("active promise must pause lock and populate ActivePromise: %+v", lockProm)
	}
}
