package ar

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestStatusMultiFilter_CommaSeparatedAndUnpaid(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	seed := []struct {
		id     string
		due    time.Time
		status string
		bal    int64
	}{
		{"ari-1", now.AddDate(0, 0, -10), StatusOpen, 100_000},
		{"ari-2", now.AddDate(0, 0, -5), StatusPartial, 50_000},
		{"ari-3", now.AddDate(0, 0, -2), StatusPaid, 0},
		{"ari-4", now.AddDate(0, 0, -1), StatusVoid, 0},
	}
	for i, s := range seed {
		err := repo.OpenInvoice(ctx, Invoice{
			InvoiceID: s.id, SupplierID: "sup-1", RetailerID: "ret-1",
			OrderID: "ord-" + s.id, Status: s.status,
			PrincipalMinor: 100_000, BalanceMinor: s.bal, Currency: "UZS",
			CreditLeaveAt: now.AddDate(0, 0, -30), DueAt: s.due, Version: int64(i + 1),
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Single status still works
	openOnly, _, err := repo.ListByRetailerPage(ctx, "ret-1", StatusOpen, 50, "")
	if err != nil || len(openOnly) != 1 || openOnly[0].InvoiceID != "ari-1" {
		t.Fatalf("OPEN only = %+v err=%v", openOnly, err)
	}

	// Comma-separated OPEN,PARTIAL
	both, _, err := repo.ListByRetailerPage(ctx, "ret-1", "OPEN,PARTIAL", 50, "")
	if err != nil || len(both) != 2 || both[0].InvoiceID != "ari-1" || both[1].InvoiceID != "ari-2" {
		t.Fatalf("OPEN,PARTIAL = %+v err=%v", both, err)
	}

	// UNPAID expands to OPEN,PARTIAL for both retailer and supplier pagination
	unpaidRet, cursor1, err := repo.ListByRetailerPage(ctx, "ret-1", "UNPAID", 1, "")
	if err != nil || len(unpaidRet) != 1 || unpaidRet[0].InvoiceID != "ari-1" || cursor1 == "" {
		t.Fatalf("UNPAID page1 = %+v cursor=%q err=%v", unpaidRet, cursor1, err)
	}
	unpaidRet2, cursor2, err := repo.ListByRetailerPage(ctx, "ret-1", "UNPAID", 1, cursor1)
	if err != nil || len(unpaidRet2) != 1 || unpaidRet2[0].InvoiceID != "ari-2" || cursor2 != "" {
		t.Fatalf("UNPAID page2 = %+v cursor=%q err=%v", unpaidRet2, cursor2, err)
	}
	unpaidSup, _, err := repo.ListBySupplierPage(ctx, "sup-1", "OPEN,PARTIAL", 50, "")
	if err != nil || len(unpaidSup) != 2 {
		t.Fatalf("supplier OPEN,PARTIAL = %+v err=%v", unpaidSup, err)
	}
}

func TestApplyPaymentFIFO_OldestFirstDisputeExclusionOverpayAndIdempotency(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:       "usr-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-1",
	})
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	// inv-1 (oldest, 100k, disputed 100k -> undisputed 0 -> MUST BE SKIPPED!)
	// inv-2 (2nd oldest, 120k, disputed 20k -> undisputed 100k)
	// inv-3 (3rd oldest, 80k, undisputed 80k)
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1",
		Status: StatusOpen, PrincipalMinor: 100_000, BalanceMinor: 100_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -20), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-2", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2",
		Status: StatusPartial, PrincipalMinor: 150_000, BalanceMinor: 120_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -10), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-3", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-3",
		Status: StatusOpen, PrincipalMinor: 80_000, BalanceMinor: 80_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -5), Version: 1, CreatedAt: now, UpdatedAt: now,
	})

	svc.SetDisputeLookup(func(_ context.Context, _ []string) (map[string]int64, error) {
		return map[string]int64{
			"ord-1": 100_000,
			"ord-2": 20_000,
		}, nil
	})

	// Seed an active promise for inv-3 -> should be FULFILLED once inv-3 hits 0
	if _, err := svc.CreateRetailerPromise(ctx, "ret-1", "sup-1", "inv-3", 5); err != nil {
		t.Fatalf("seed promise: %v", err)
	}

	clearedByOrder := map[string]int64{}
	var clearCalls int64
	svc.SetCreditClearer(func(_ context.Context, _ *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
		atomic.AddInt64(&clearCalls, 1)
		clearedByOrder[orderID] += amountMinor
		return nil
	})

	// Pay 230_000: should pay 100_000 onto inv-2 (leaving disputed 20k untouched), 80_000 onto inv-3 (PAID),
	// skip inv-1 completely, and route excess 50_000 into ArAdvanceBalances!
	resp, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:     "sup-1",
		AmountMinor:    230_000,
		Currency:       "UZS",
		PaymentMethod:  "WALLET",
		IdempotencyKey: "fifo-key-1",
	})
	if err != nil {
		t.Fatalf("ApplyPaymentFIFO: %v", err)
	}
	if resp.PaidMinor != 180_000 || resp.AdvanceAddedMinor != 50_000 || resp.AdvanceBalanceMinor != 50_000 {
		t.Fatalf("unexpected totals: %+v", resp)
	}
	if len(resp.Allocations) != 2 || resp.Allocations[0].InvoiceID != "inv-2" || resp.Allocations[0].AppliedMinor != 100_000 ||
		resp.Allocations[1].InvoiceID != "inv-3" || resp.Allocations[1].AppliedMinor != 80_000 {
		t.Fatalf("unexpected allocations: %+v", resp.Allocations)
	}
	if clearedByOrder["ord-1"] != 0 || clearedByOrder["ord-2"] != 100_000 || clearedByOrder["ord-3"] != 80_000 {
		t.Fatalf("unexpected credit clears: %+v", clearedByOrder)
	}

	inv1, _ := repo.GetByID(ctx, "inv-1")
	inv2, _ := repo.GetByID(ctx, "inv-2")
	inv3, _ := repo.GetByID(ctx, "inv-3")
	if inv1.BalanceMinor != 100_000 || inv1.Status != StatusOpen {
		t.Fatalf("disputed inv-1 should remain untouched: %+v", inv1)
	}
	if inv2.BalanceMinor != 20_000 || inv2.Status != StatusPartial {
		t.Fatalf("inv-2 should retain 20k disputed: %+v", inv2)
	}
	if inv3.BalanceMinor != 0 || inv3.Status != StatusPaid {
		t.Fatalf("inv-3 should be PAID: %+v", inv3)
	}

	promises, err := svc.ListPromises(ctx, "ret-1", "sup-1", 10)
	if err != nil || len(promises) != 1 || promises[0].Status != PromiseStatusFulfilled {
		t.Fatalf("expected promise FULFILLED, got %+v err=%v", promises, err)
	}

	// Duplicate retry with same IdempotencyKey MUST return identical response with zero side effects
	retry, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:     "sup-1",
		AmountMinor:    230_000,
		Currency:       "UZS",
		PaymentMethod:  "WALLET",
		IdempotencyKey: "fifo-key-1",
	})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retry.PaidMinor != 180_000 || retry.AdvanceAddedMinor != 50_000 || retry.AdvanceBalanceMinor != 50_000 || len(retry.Allocations) != 2 {
		t.Fatalf("unexpected retry response: %+v", retry)
	}
	if atomic.LoadInt64(&clearCalls) != 2 {
		t.Fatalf("creditClearer called %d times, want 2 (no duplicate calls)", atomic.LoadInt64(&clearCalls))
	}
}

func TestApplyPaymentFIFO_EarmarkAndRetailerPayInvoiceCreditClear(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-old", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-old",
		Status: StatusOpen, PrincipalMinor: 100_000, BalanceMinor: 100_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -20), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-target", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-target",
		Status: StatusOpen, PrincipalMinor: 60_000, BalanceMinor: 60_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -2), Version: 1, CreatedAt: now, UpdatedAt: now,
	})

	cleared := map[string]int64{}
	svc.SetCreditClearer(func(_ context.Context, _ *spanner.ReadWriteTransaction, _, _, orderID string, amountMinor int64) error {
		cleared[orderID] += amountMinor
		return nil
	})

	// Earmarked payment of 90_000 onto inv-target (60_000): pays ONLY inv-target, puts 30_000 to advance!
	resp, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		AmountMinor:      90_000,
		Currency:         "UZS",
		EarmarkInvoiceID: "inv-target",
		IdempotencyKey:   "earmark-1",
	})
	if err != nil {
		t.Fatalf("earmark fifo: %v", err)
	}
	if resp.PaidMinor != 60_000 || resp.AdvanceAddedMinor != 30_000 || resp.AdvanceBalanceMinor != 30_000 {
		t.Fatalf("unexpected earmark response: %+v", resp)
	}
	if cleared["ord-target"] != 60_000 || cleared["ord-old"] != 0 {
		t.Fatalf("unexpected earmarked credit clears: %+v", cleared)
	}

	// Paying an already PAID earmarked invoice returns ErrInvoiceAlreadySettled
	if _, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		AmountMinor:      10_000,
		EarmarkInvoiceID: "inv-target",
		IdempotencyKey:   "earmark-2",
	}); !errors.Is(err, ErrInvoiceAlreadySettled) {
		t.Fatalf("expected ErrInvoiceAlreadySettled, got %v", err)
	}

	// Single-invoice RetailerPayInvoice on inv-old triggers creditClearer AND fulfills promises
	if _, err := svc.CreateRetailerPromise(ctx, "ret-1", "sup-1", "inv-old", 3); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.RetailerPayInvoice(ctx, "ret-1", "inv-old", RetailerPayInvoiceRequest{
		AmountMinor:   100_000,
		PaymentMethod: "CARD",
	}, "single-pay-1")
	if err != nil {
		t.Fatalf("RetailerPayInvoice: %v", err)
	}
	if updated.Status != StatusPaid || updated.BalanceMinor != 0 {
		t.Fatalf("expected inv-old PAID, got %+v", updated)
	}
	if cleared["ord-old"] != 100_000 {
		t.Fatalf("expected ord-old credit clear 100000, got %d", cleared["ord-old"])
	}
	promises, _ := svc.ListPromises(ctx, "ret-1", "sup-1", 10)
	if len(promises) != 1 || promises[0].Status != PromiseStatusFulfilled {
		t.Fatalf("expected promise fulfilled after RetailerPayInvoice, got %+v", promises)
	}
}

func TestApplyPaymentFIFO_MultiCurrencyIsolation(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:       "usr-multi",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-multi",
	})
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	// Seed 2 UZS invoices and 2 USD invoices
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-uzs-1", SupplierID: "sup-multi", RetailerID: "ret-multi", OrderID: "ord-u1",
		Status: StatusOpen, PrincipalMinor: 50_000, BalanceMinor: 50_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -10), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-usd-1", SupplierID: "sup-multi", RetailerID: "ret-multi", OrderID: "ord-d1",
		Status: StatusOpen, PrincipalMinor: 100, BalanceMinor: 100, Currency: "USD",
		DueAt: now.AddDate(0, 0, -9), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-uzs-2", SupplierID: "sup-multi", RetailerID: "ret-multi", OrderID: "ord-u2",
		Status: StatusOpen, PrincipalMinor: 70_000, BalanceMinor: 70_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -8), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.OpenInvoice(ctx, Invoice{
		InvoiceID: "inv-usd-2", SupplierID: "sup-multi", RetailerID: "ret-multi", OrderID: "ord-d2",
		Status: StatusOpen, PrincipalMinor: 200, BalanceMinor: 200, Currency: "USD",
		DueAt: now.AddDate(0, 0, -7), Version: 1, CreatedAt: now, UpdatedAt: now,
	})

	// 1. Pay 60,000 UZS via FIFO: must allocate strictly to UZS invoices, ignoring USD invoices
	respUZS, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:     "sup-multi",
		RetailerID:     "ret-multi",
		AmountMinor:    60_000,
		Currency:       "UZS",
		IdempotencyKey: "pay-fifo-uzs-1",
	})
	if err != nil {
		t.Fatalf("ApplyPaymentFIFO UZS: %v", err)
	}
	if len(respUZS.Allocations) != 2 {
		t.Fatalf("expected 2 allocations, got %d", len(respUZS.Allocations))
	}
	if respUZS.Allocations[0].InvoiceID != "inv-uzs-1" || respUZS.Allocations[0].AppliedMinor != 50_000 {
		t.Fatalf("alloc[0] = %+v, want inv-uzs-1 for 50000", respUZS.Allocations[0])
	}
	if respUZS.Allocations[1].InvoiceID != "inv-uzs-2" || respUZS.Allocations[1].AppliedMinor != 10_000 {
		t.Fatalf("alloc[1] = %+v, want inv-uzs-2 for 10000", respUZS.Allocations[1])
	}

	// Verify USD invoices were untouched
	invUSD1, err := repo.GetByID(ctx, "inv-usd-1")
	if err != nil || invUSD1.BalanceMinor != 100 {
		t.Fatalf("inv-usd-1 balance modified by UZS FIFO payment: %d err=%v", invUSD1.BalanceMinor, err)
	}

	// 2. Pay 150 USD via FIFO: must allocate strictly to USD invoices, ignoring UZS invoices
	respUSD, err := svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:     "sup-multi",
		RetailerID:     "ret-multi",
		AmountMinor:    150,
		Currency:       "USD",
		IdempotencyKey: "pay-fifo-usd-1",
	})
	if err != nil {
		t.Fatalf("ApplyPaymentFIFO USD: %v", err)
	}
	if len(respUSD.Allocations) != 2 {
		t.Fatalf("expected 2 allocations for USD, got %d", len(respUSD.Allocations))
	}
	if respUSD.Allocations[0].InvoiceID != "inv-usd-1" || respUSD.Allocations[0].AppliedMinor != 100 {
		t.Fatalf("alloc[0] = %+v, want inv-usd-1 for 100", respUSD.Allocations[0])
	}
	if respUSD.Allocations[1].InvoiceID != "inv-usd-2" || respUSD.Allocations[1].AppliedMinor != 50 {
		t.Fatalf("alloc[1] = %+v, want inv-usd-2 for 50", respUSD.Allocations[1])
	}

	// 3. Earmark with currency mismatch must fail with ErrInvalidFIFORequest
	_, err = svc.ApplyPaymentFIFO(ctx, PayFIFORequest{
		SupplierID:       "sup-multi",
		RetailerID:       "ret-multi",
		AmountMinor:      50,
		Currency:         "UZS",
		EarmarkInvoiceID: "inv-usd-2",
		IdempotencyKey:   "mismatch-earmark",
	})
	if !errors.Is(err, ErrInvalidFIFORequest) {
		t.Fatalf("expected ErrInvalidFIFORequest on currency mismatch, got %v", err)
	}
}
