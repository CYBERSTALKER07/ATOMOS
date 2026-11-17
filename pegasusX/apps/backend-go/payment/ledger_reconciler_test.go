package payment

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type mockLedgerReconcileStore struct {
	mu            sync.Mutex
	legs          []CapturedPaymentLegSummary
	ledgerEntries []LedgerEntryRecord
	legsErr       error
	ledgerErr     error
}

func (m *mockLedgerReconcileStore) ListCapturedPaymentLegsSince(ctx context.Context, since time.Time, limit int) ([]CapturedPaymentLegSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.legsErr != nil {
		return nil, m.legsErr
	}
	return m.legs, nil
}

func (m *mockLedgerReconcileStore) ListLedgerEntriesSince(ctx context.Context, since time.Time, limit int) ([]LedgerEntryRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ledgerErr != nil {
		return nil, m.ledgerErr
	}
	return m.ledgerEntries, nil
}

func TestLedgerReconciler_HappyPath_ZeroAnomalies(t *testing.T) {
	now := time.Now().UTC()
	store := &mockLedgerReconcileStore{
		legs: []CapturedPaymentLegSummary{
			{
				OrderID:        "ord-happy-1",
				LegID:          "leg-1",
				Method:         "UZUM",
				AmountMinor:    100000,
				Status:         "CAPTURED",
				IdempotencyKey: "idem-leg-1",
				CapturedAt:     now,
			},
			{
				OrderID:        "ord-happy-2",
				LegID:          "leg-2a",
				Method:         "CARD",
				AmountMinor:    30000,
				Status:         "CAPTURED",
				IdempotencyKey: "idem-leg-2a",
				CapturedAt:     now,
			},
			{
				OrderID:        "ord-happy-2",
				LegID:          "leg-2b",
				Method:         "LOYALTY",
				AmountMinor:    20000,
				Status:         "CAPTURED",
				IdempotencyKey: "idem-leg-2b",
				CapturedAt:     now,
			},
			// Refund leg should be excluded from positive payment leg matching
			{
				OrderID:        "ord-happy-3",
				LegID:          "leg-3-ref",
				Method:         "REFUND",
				AmountMinor:    15000,
				Status:         "CAPTURED",
				IdempotencyKey: "idem-leg-3-ref",
				CapturedAt:     now,
			},
		},
		ledgerEntries: []LedgerEntryRecord{
			{
				LedgerEntryID: "ple-1",
				OrderID:       "ord-happy-1",
				Gateway:       "UZUM",
				EntryType:     "PAYMENT_CAPTURED",
				AmountMinor:   100000,
				Currency:      "UZS",
				OccurredAt:    now,
			},
			{
				LedgerEntryID: "ple-2",
				OrderID:       "ord-happy-2",
				Gateway:       "SPLIT_TENDER",
				EntryType:     "PAYMENT_CAPTURED",
				AmountMinor:   50000, // Matches 30000 + 20000
				Currency:      "UZS",
				OccurredAt:    now,
			},
		},
	}

	reconciler := NewLedgerReconciler(store, slog.Default())
	report, err := reconciler.ReconcileWindow(context.Background(), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ReconcileWindow failed: %v", err)
	}

	if report.Anomalies != 0 {
		t.Errorf("expected 0 anomalies, got %d (missing: %v, mismatches: %+v, unbalanced: %v)",
			report.Anomalies, report.MissingLedgerOrderIDs, report.AmountMismatches, report.UnbalancedJournals)
	}
	if len(report.MissingLedgerOrderIDs) != 0 {
		t.Errorf("expected 0 missing ledger orders, got %d", len(report.MissingLedgerOrderIDs))
	}
	if len(report.AmountMismatches) != 0 {
		t.Errorf("expected 0 amount mismatches, got %d", len(report.AmountMismatches))
	}
	if report.MatchedOrders != 2 {
		t.Errorf("expected 2 matched orders, got %d", report.MatchedOrders)
	}
	if report.TotalCapturedLegs != 3 { // 3 non-refund legs
		t.Errorf("expected 3 total captured legs, got %d", report.TotalCapturedLegs)
	}
}

func TestLedgerReconciler_DoubleEntryGeneralLedger_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	store := &mockLedgerReconcileStore{
		legs: []CapturedPaymentLegSummary{
			{
				OrderID:     "ord-gl-1",
				LegID:       "leg-gl-1",
				Method:      "CARD",
				AmountMinor: 75000,
				Status:      "CAPTURED",
				CapturedAt:  now,
			},
		},
		ledgerEntries: []LedgerEntryRecord{
			// Double entry postings: DEBIT 75000 and CREDIT 75000
			{
				LedgerEntryID: "ple-gl-debit",
				OrderID:       "ord-gl-1",
				Gateway:       "GENERAL_LEDGER",
				EntryType:     "GL_DEBIT_AR_RETAILER",
				AmountMinor:   75000,
				Currency:      "UZS",
				OccurredAt:    now,
			},
			{
				LedgerEntryID: "ple-gl-credit",
				OrderID:       "ord-gl-1",
				Gateway:       "GENERAL_LEDGER",
				EntryType:     "GL_CREDIT_AP_SUPPLIER",
				AmountMinor:   75000,
				Currency:      "UZS",
				OccurredAt:    now,
			},
		},
	}

	reconciler := NewLedgerReconciler(store, slog.Default())
	report, err := reconciler.ReconcileWindow(context.Background(), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ReconcileWindow failed: %v", err)
	}

	if report.Anomalies != 0 {
		t.Errorf("expected 0 anomalies in GL double-entry match, got %d: %+v", report.Anomalies, report)
	}
	if report.MatchedOrders != 1 {
		t.Errorf("expected 1 matched order, got %d", report.MatchedOrders)
	}
}

func TestLedgerReconciler_DetectsMissingLedgerEntry(t *testing.T) {
	now := time.Now().UTC()
	store := &mockLedgerReconcileStore{
		legs: []CapturedPaymentLegSummary{
			{
				OrderID:     "ord-missing-ledger-1",
				LegID:       "leg-m1",
				Method:      "PAYME",
				AmountMinor: 45000,
				Status:      "CAPTURED",
				CapturedAt:  now,
			},
		},
		ledgerEntries: []LedgerEntryRecord{}, // completely empty ledger
	}

	reconciler := NewLedgerReconciler(store, slog.Default())
	report, err := reconciler.ReconcileWindow(context.Background(), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ReconcileWindow failed: %v", err)
	}

	if report.Anomalies != 1 {
		t.Fatalf("expected 1 anomaly, got %d", report.Anomalies)
	}
	if len(report.MissingLedgerOrderIDs) != 1 || report.MissingLedgerOrderIDs[0] != "ord-missing-ledger-1" {
		t.Errorf("expected missing ledger order ord-missing-ledger-1, got %v", report.MissingLedgerOrderIDs)
	}
}

func TestLedgerReconciler_DetectsAmountMismatch(t *testing.T) {
	now := time.Now().UTC()
	store := &mockLedgerReconcileStore{
		legs: []CapturedPaymentLegSummary{
			{
				OrderID:     "ord-mismatch-1",
				LegID:       "leg-mm-1",
				Method:      "CLICK",
				AmountMinor: 100000,
				Status:      "CAPTURED",
				CapturedAt:  now,
			},
		},
		ledgerEntries: []LedgerEntryRecord{
			{
				LedgerEntryID: "ple-mm-1",
				OrderID:       "ord-mismatch-1",
				Gateway:       "CLICK",
				EntryType:     "PAYMENT_CAPTURED",
				AmountMinor:   90000, // 10000 shortfall
				Currency:      "UZS",
				OccurredAt:    now,
			},
		},
	}

	reconciler := NewLedgerReconciler(store, slog.Default())
	report, err := reconciler.ReconcileWindow(context.Background(), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ReconcileWindow failed: %v", err)
	}

	if report.Anomalies != 1 {
		t.Fatalf("expected 1 anomaly, got %d", report.Anomalies)
	}
	if len(report.AmountMismatches) != 1 {
		t.Fatalf("expected 1 amount mismatch, got %d", len(report.AmountMismatches))
	}
	mm := report.AmountMismatches[0]
	if mm.OrderID != "ord-mismatch-1" {
		t.Errorf("expected mismatch orderID ord-mismatch-1, got %s", mm.OrderID)
	}
	if mm.LegTotalMinor != 100000 {
		t.Errorf("expected LegTotalMinor 100000, got %d", mm.LegTotalMinor)
	}
	if mm.LedgerTotalMinor != 90000 {
		t.Errorf("expected LedgerTotalMinor 90000, got %d", mm.LedgerTotalMinor)
	}
	if mm.DifferenceMinor != 10000 {
		t.Errorf("expected DifferenceMinor 10000, got %d", mm.DifferenceMinor)
	}
}

func TestLedgerReconciler_UnbalancedJournalEntryValidation(t *testing.T) {
	// Unbalanced journal entry (debit 50000 vs credit 40000)
	unbalanced := JournalEntry{
		EntryID:    "je-unbalanced-1",
		OrderID:    "ord-unbal-1",
		SupplierID: "sup-1",
		Currency:   "UZS",
		OccurredAt: time.Now().UTC(),
		Postings: []Posting{
			{AccountCode: AccountAccountsReceivable, AccountType: AccountClassAsset, Direction: PostingDebit, AmountMinor: 50000, Currency: "UZS"},
			{AccountCode: AccountSupplierPayable, AccountType: AccountClassLiability, Direction: PostingCredit, AmountMinor: 40000, Currency: "UZS"},
		},
	}

	// Balanced journal entry
	balanced := JournalEntry{
		EntryID:    "je-balanced-1",
		OrderID:    "ord-bal-1",
		SupplierID: "sup-1",
		Currency:   "UZS",
		OccurredAt: time.Now().UTC(),
		Postings: []Posting{
			{AccountCode: AccountAccountsReceivable, AccountType: AccountClassAsset, Direction: PostingDebit, AmountMinor: 50000, Currency: "UZS"},
			{AccountCode: AccountSupplierPayable, AccountType: AccountClassLiability, Direction: PostingCredit, AmountMinor: 50000, Currency: "UZS"},
		},
	}

	invalidIDs, err := ValidateJournalBatch([]JournalEntry{unbalanced, balanced})
	if err == nil {
		t.Fatalf("expected error from ValidateJournalBatch with unbalanced entry, got nil")
	}
	if len(invalidIDs) != 1 {
		t.Fatalf("expected 1 invalid entry ID, got %d: %v", len(invalidIDs), invalidIDs)
	}

	// Now verify that adding unbalanced batches to reconciler records anomaly
	store := &mockLedgerReconcileStore{}
	reconciler := NewLedgerReconciler(store, slog.Default())
	reconciler.AddPendingJournalBatches([]JournalEntry{unbalanced})

	report, err := reconciler.ReconcileWindow(context.Background(), time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ReconcileWindow failed: %v", err)
	}
	if len(report.UnbalancedJournals) != 1 {
		t.Errorf("expected 1 unbalanced journal in report, got %d", len(report.UnbalancedJournals))
	}
	if report.Anomalies != 1 {
		t.Errorf("expected 1 anomaly, got %d", report.Anomalies)
	}
}

func TestLedgerReconciler_ReferenceIDOrderMatching(t *testing.T) {
	now := time.Now().UTC()
	store := &mockLedgerReconcileStore{
		legs: []CapturedPaymentLegSummary{
			{
				OrderID:     "ord-ref-match",
				LegID:       "leg-ref-1",
				Method:      "CARD",
				AmountMinor: 60000,
				Status:      "CAPTURED",
				CapturedAt:  now,
			},
		},
		ledgerEntries: []LedgerEntryRecord{
			{
				LedgerEntryID: "ple-ref-1",
				OrderID:       "", // Empty OrderID, but ReferenceID contains order ID
				ReferenceID:   "jentry_order_ord-ref-match",
				Gateway:       "GENERAL_LEDGER",
				EntryType:     "GL_DEBIT_AR_RETAILER",
				AmountMinor:   60000,
				Currency:      "UZS",
				OccurredAt:    now,
			},
			{
				LedgerEntryID: "ple-ref-2",
				OrderID:       "",
				ReferenceID:   "jentry_order_ord-ref-match",
				Gateway:       "GENERAL_LEDGER",
				EntryType:     "GL_CREDIT_AP_SUPPLIER",
				AmountMinor:   60000,
				Currency:      "UZS",
				OccurredAt:    now,
			},
		},
	}

	reconciler := NewLedgerReconciler(store, slog.Default())
	report, err := reconciler.ReconcileWindow(context.Background(), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ReconcileWindow failed: %v", err)
	}

	if report.Anomalies != 0 {
		t.Errorf("expected 0 anomalies with reference ID matching, got %d: %+v", report.Anomalies, report)
	}
	if report.MatchedOrders != 1 {
		t.Errorf("expected 1 matched order, got %d", report.MatchedOrders)
	}
}

func TestLedgerReconciler_StoreErrors(t *testing.T) {
	storeLegsErr := &mockLedgerReconcileStore{legsErr: errors.New("spanner connection failure")}
	reconciler := NewLedgerReconciler(storeLegsErr, slog.Default())
	if _, err := reconciler.ReconcileWindow(context.Background(), time.Now()); err == nil {
		t.Fatalf("expected error on legs store failure, got nil")
	}

	storeLedgerErr := &mockLedgerReconcileStore{ledgerErr: errors.New("ledger table failure")}
	reconciler = NewLedgerReconciler(storeLedgerErr, slog.Default())
	if _, err := reconciler.ReconcileWindow(context.Background(), time.Now()); err == nil {
		t.Fatalf("expected error on ledger store failure, got nil")
	}
}

func TestSpannerLedgerReconcileStore_NilClient(t *testing.T) {
	store := NewSpannerLedgerReconcileStore(nil)
	ctx := context.Background()
	now := time.Now()

	legs, err := store.ListCapturedPaymentLegsSince(ctx, now, 10)
	if err != nil || len(legs) != 0 {
		t.Errorf("expected empty legs without error, got %v, err: %v", legs, err)
	}

	entries, err := store.ListLedgerEntriesSince(ctx, now, 10)
	if err != nil || len(entries) != 0 {
		t.Errorf("expected empty entries without error, got %v, err: %v", entries, err)
	}
}

func TestStartLedgerReconciler_ContextShutdown(t *testing.T) {
	store := &mockLedgerReconcileStore{}
	reconciler := NewLedgerReconciler(store, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		StartLedgerReconciler(ctx, reconciler, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// success: worker shut down cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("StartLedgerReconciler did not shut down on context cancellation")
	}
}
