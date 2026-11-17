package ar

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func testMigrationJournalRecorder(failErr error) MigrationJournalRecorder {
	return MigrationJournalRecorderFunc(func(in MigrationJournalInput) (any, []*spanner.Mutation, string, error) {
		if failErr != nil {
			return nil, nil, "", failErr
		}
		if in.EntryID == "" {
			in.EntryID = "jentry_test_mig_1"
		}
		return map[string]any{
			"entry_id": in.EntryID,
			"amount":   in.AmountMinor,
			"currency": in.Currency,
		}, nil, in.EntryID, nil
	})
}

func TestPrepareMigrationOpeningBalance_GuardsAndCurrencyParity(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)

	if _, err := PrepareMigrationOpeningBalance(OpenFromMigrationRequest{
		SessionID:        "sess-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "INV-1",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		BalanceMinor:     500000,
		Now:              now,
	}); !errors.Is(err, ErrMigrationJournalRecorderUnavailable) {
		t.Fatalf("expected ErrMigrationJournalRecorderUnavailable on nil recorder, got %v", err)
	}

	if _, err := PrepareMigrationOpeningBalance(OpenFromMigrationRequest{
		SessionID:        "sess-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "INV-1",
		SupplierCurrency: "UZS",
		Currency:         "USD",
		BalanceMinor:     500000,
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	}); !errors.Is(err, ErrMigrationCurrencyMismatch) {
		t.Fatalf("expected ErrMigrationCurrencyMismatch on USD vs UZS, got %v", err)
	}

	if _, err := PrepareMigrationOpeningBalance(OpenFromMigrationRequest{
		SessionID:        "sess-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "INV-1",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		BalanceMinor:     0,
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	}); !errors.Is(err, ErrMigrationInvalidAmount) {
		t.Fatalf("expected ErrMigrationInvalidAmount on non-positive balance, got %v", err)
	}

	injected := errors.New("imbalanced journal entry")
	if _, err := PrepareMigrationOpeningBalance(OpenFromMigrationRequest{
		SessionID:        "sess-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "INV-1",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		BalanceMinor:     500000,
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(injected),
	}); !errors.Is(err, ErrMigrationJournalRequired) || !strings.Contains(err.Error(), injected.Error()) {
		t.Fatalf("expected ErrMigrationJournalRequired wrapping %q, got %v", injected, err)
	}
}

func TestPrepareMigrationOpeningBalance_OrderCreditReservation(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	prep, err := PrepareMigrationOpeningBalance(OpenFromMigrationRequest{
		SessionID:        "sess-ar-res",
		SupplierID:       "sup-res-1",
		RetailerID:       "ret-res-1",
		ExternalID:       "INV-RES-1",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		PrincipalMinor:   750000,
		BalanceMinor:     500000,
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prep.ReservationRow == nil {
		t.Fatal("expected ReservationRow to be populated")
	}
	if prep.ReservationMutation == nil {
		t.Fatal("expected ReservationMutation to be populated")
	}

	if got := prep.ReservationRow["OrderId"]; got != prep.SyntheticOrderID {
		t.Fatalf("reservation OrderId = %v, want %v", got, prep.SyntheticOrderID)
	}
	if got := prep.ReservationRow["RetailerId"]; got != "ret-res-1" {
		t.Fatalf("reservation RetailerId = %v, want ret-res-1", got)
	}
	if got := prep.ReservationRow["SupplierId"]; got != "sup-res-1" {
		t.Fatalf("reservation SupplierId = %v, want sup-res-1", got)
	}
	if got := prep.ReservationRow["AmountMinor"]; got != int64(500000) {
		t.Fatalf("reservation AmountMinor = %v, want 500000", got)
	}
	if got := prep.ReservationRow["Status"]; got != "CONVERTED" {
		t.Fatalf("reservation Status = %v, want CONVERTED", got)
	}
}

func TestMemoryOpenFromMigration_DunningFreeze_AcknowledgeAndDispute(t *testing.T) {
	_ = os.Setenv("AR_INVOICES_ENABLED", "true")
	_ = os.Setenv("AR_DUNNING_ENABLED", "true")
	t.Cleanup(func() {
		_ = os.Unsetenv("AR_INVOICES_ENABLED")
		_ = os.Unsetenv("AR_DUNNING_ENABLED")
	})

	repo := NewMemoryRepository()
	svc := NewService(repo)
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	svc.SetNow(func() time.Time { return now })

	profiles := map[string]CreditProfileState{
		"sup-1|ret-1": {
			Exists:               true,
			CreditLimitMinor:     10000000,
			CurrentBalanceMinor:  0,
			AvailableCreditMinor: 10000000,
			Status:               "ACTIVE",
			Version:              1,
		},
	}

	// Open overdue migrated invoice (due 25 days ago) -> Dunning must stay FROZEN until retail acknowledgement.
	res, err := repo.OpenFromMigration(OpenFromMigrationRequest{
		SessionID:        "sess-ar-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "1C-DEBT-001",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		PrincipalMinor:   2500000,
		BalanceMinor:     1500000,
		CreditLeaveAt:    now.Add(-45 * 24 * time.Hour),
		DueAt:            now.Add(-25 * 24 * time.Hour),
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	}, profiles)
	if err != nil {
		t.Fatalf("OpenFromMigration: %v", err)
	}
	if res.Invoice.Source != InvoiceSourceMigration {
		t.Fatalf("Source got %q want MIGRATION", res.Invoice.Source)
	}
	if res.Invoice.AcknowledgedAt != nil || !res.Invoice.IsMigrationDunningFrozen() {
		t.Fatalf("expected newly migrated AR invoice to have nil AcknowledgedAt and frozen dunning")
	}
	if res.SyntheticOrderID == "" {
		t.Fatal("expected deterministic synthetic order ID")
	}
	cp := profiles["sup-1|ret-1"]
	if cp.CurrentBalanceMinor != 1500000 || cp.AvailableCreditMinor != 8500000 {
		t.Fatalf("unexpected credit profile after migration opening: %+v", cp)
	}

	// Idempotent re-run must not double-add balance or mutate existing invoice.
	dupRes, err := repo.OpenFromMigration(OpenFromMigrationRequest{
		SessionID:        "sess-ar-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "1C-DEBT-001",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		PrincipalMinor:   2500000,
		BalanceMinor:     1500000,
		CreditLeaveAt:    now.Add(-45 * 24 * time.Hour),
		DueAt:            now.Add(-25 * 24 * time.Hour),
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	}, profiles)
	if err != nil || !dupRes.AlreadyExisted {
		t.Fatalf("expected AlreadyExisted=true on duplicate external_id, got %+v err=%v", dupRes, err)
	}
	if profiles["sup-1|ret-1"].CurrentBalanceMinor != 1500000 {
		t.Fatalf("duplicate open mutated balance: %+v", profiles["sup-1|ret-1"])
	}

	// Mismatch between sum(ArInvoices) and CreditProfile fails closed.
	profiles["sup-1|ret-999"] = CreditProfileState{Exists: true, CurrentBalanceMinor: 999999}
	if _, err := repo.OpenFromMigration(OpenFromMigrationRequest{
		SessionID:        "sess-ar-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-999",
		ExternalID:       "1C-DEBT-MISMATCH",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		BalanceMinor:     500000,
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	}, profiles); !errors.Is(err, ErrMigrationCreditLedgerMismatch) {
		t.Fatalf("expected ErrMigrationCreditLedgerMismatch, got %v", err)
	}

	// Seed second invoice that is disputed, verify DunningWorker skips BOTH while unacknowledged/disputed.
	resDispute, err := repo.OpenFromMigration(OpenFromMigrationRequest{
		SessionID:        "sess-ar-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ExternalID:       "1C-DEBT-DISPUTE",
		SupplierCurrency: "UZS",
		Currency:         "UZS",
		BalanceMinor:     500000,
		CreditLeaveAt:    now.Add(-40 * 24 * time.Hour),
		DueAt:            now.Add(-25 * 24 * time.Hour),
		Now:              now,
		JournalRecorder:  testMigrationJournalRecorder(nil),
	}, profiles)
	if err != nil {
		t.Fatalf("second OpenFromMigration: %v", err)
	}

	openForDunning, err := repo.ListOpenForDunning(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(openForDunning) != 0 {
		t.Fatalf("expected 0 unacknowledged MIGRATION invoices in ListOpenForDunning, got %d", len(openForDunning))
	}

	w := NewDunningWorker(svc, nil)
	w.now = func() time.Time { return now }
	heldCount := 0
	w.SetAutoHold(func(context.Context, string, string) error {
		heldCount++
		return nil
	})
	adv, held, _, err := w.AdvanceDunning(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if adv != 0 || held != 0 || heldCount != 0 {
		t.Fatalf("dunning must be frozen on unacknowledged migration AR: adv=%d held=%d", adv, held)
	}

	// Dispute keeps dunning frozen even if acknowledged later.
	disputedInv, err := repo.DisputeMigrationInvoice(context.Background(), "ret-1", resDispute.Invoice.InvoiceID, "amount overstated", now)
	if err != nil {
		t.Fatalf("DisputeMigrationInvoice: %v", err)
	}
	if !disputedInv.IsMigrationDunningFrozen() {
		t.Fatal("disputed MIGRATION invoice must remain dunning-frozen")
	}

	// Acknowledge first invoice -> unfreezes dunning only for acknowledged undisputed invoice.
	ackInv, err := repo.AcknowledgeMigrationInvoice(context.Background(), "ret-1", res.Invoice.InvoiceID, now)
	if err != nil {
		t.Fatalf("AcknowledgeMigrationInvoice: %v", err)
	}
	if ackInv.IsMigrationDunningFrozen() {
		t.Fatal("acknowledged undisputed MIGRATION invoice should unfreeze dunning")
	}

	openForDunning, err = repo.ListOpenForDunning(context.Background(), 50)
	if err != nil || len(openForDunning) != 1 || openForDunning[0].InvoiceID != res.Invoice.InvoiceID {
		t.Fatalf("expected only acknowledged invoice in ListOpenForDunning, got %+v err=%v", openForDunning, err)
	}
	adv, held, _, err = w.AdvanceDunning(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if adv != 1 || held != 1 {
		t.Fatalf("expected 1 advanced + 1 auto-hold after retailer acknowledgment, got adv=%d held=%d", adv, held)
	}

	// Disputing after acknowledgment freezes dunning again.
	reDisputed, err := repo.DisputeMigrationInvoice(context.Background(), "ret-1", res.Invoice.InvoiceID, "late credit note pending", now)
	if err != nil || !reDisputed.IsMigrationDunningFrozen() {
		t.Fatalf("disputing acknowledged invoice must re-freeze dunning, got %+v err=%v", reDisputed, err)
	}
}
