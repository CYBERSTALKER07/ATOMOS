package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

func buildPaymentMigrationJournalRecorder(overrideFn func(in ar.MigrationJournalInput) (*payment.JournalEntry, error)) ar.MigrationJournalRecorder {
	return ar.MigrationJournalRecorderFunc(func(in ar.MigrationJournalInput) (any, []*spanner.Mutation, string, error) {
		var (
			je  *payment.JournalEntry
			err error
		)
		if overrideFn != nil {
			je, err = overrideFn(in)
		} else {
			je, err = payment.BuildMigrationOpeningBalanceJournalEntry(
				in.EntryID,
				in.SessionID,
				in.OrderID,
				in.SupplierID,
				in.RetailerID,
				in.Currency,
				in.AmountMinor,
				in.Memo,
				in.OccurredAt,
			)
		}
		if err != nil {
			return nil, nil, "", err
		}
		if je == nil {
			return nil, nil, "", fmt.Errorf("%w: nil journal entry", payment.ErrInsufficientPostings)
		}
		muts, err := payment.JournalEntryMutations(je)
		if err != nil {
			return nil, nil, "", err
		}
		return je, muts, je.EntryID, nil
	})
}

func applyNasiyaRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	balMinor, err := ParseMoneyMinor(fields["balance_due"])
	if err != nil || balMinor <= 0 {
		return fmt.Errorf("%w: nasiya balance_due must be strictly positive, got %q", ErrInvalidInput, fields["balance_due"])
	}

	principalMinor := balMinor
	if rawOrig := strings.TrimSpace(fields["original_amount"]); rawOrig != "" {
		orig, err := ParseMoneyMinor(rawOrig)
		if err != nil || orig <= 0 {
			return fmt.Errorf("%w: invalid original_amount %q", ErrInvalidInput, rawOrig)
		}
		if orig >= balMinor {
			principalMinor = orig
		}
	}

	_, supCur, err := tx.ReadSupplierMeta(ctx)
	if err != nil {
		return err
	}
	supCur = strings.ToUpper(strings.TrimSpace(firstNonEmpty(rctx.SupplierCurrency, supCur, "UZS")))
	rowCur := strings.ToUpper(strings.TrimSpace(firstNonEmpty(fields["currency"], supCur)))
	if rowCur != supCur {
		return fmt.Errorf("%w: nasiya currency %s differs from supplier currency %s", ar.ErrMigrationCurrencyMismatch, rowCur, supCur)
	}

	inn := NormalizeINN(fields["customer_inn"])
	phone := NormalizePhoneUZ(fields["customer_phone"])
	custName := strings.TrimSpace(fields["customer_name"])
	cust, ok, err := tx.ResolveCustomerForFinancials(ctx, rctx.HomeCell, "", inn, phone, custName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: debtor %q (inn=%q phone=%q) not found in linked retailers", ErrInvalidInput, custName, inn, phone)
	}

	now := tx.Now()
	creditLeaveAt := now
	if rawIssued := strings.TrimSpace(fields["issued_date"]); rawIssued != "" {
		parsedIssued, pErr := ParseFlexibleDate(rawIssued)
		if pErr != nil {
			return fmt.Errorf("%w: invalid issued_date %q", ErrInvalidInput, rawIssued)
		}
		creditLeaveAt = parsedIssued.UTC()
	}

	var dueAt time.Time
	termsDays := int64(0)
	if rawDue := strings.TrimSpace(fields["due_date"]); rawDue != "" {
		parsedDue, pErr := ParseFlexibleDate(rawDue)
		if pErr != nil {
			return fmt.Errorf("%w: invalid due_date %q", ErrInvalidInput, rawDue)
		}
		dueAt = parsedDue.UTC()
		if dueAt.After(creditLeaveAt) {
			termsDays = int64(dueAt.Sub(creditLeaveAt) / (24 * time.Hour))
		}
	} else {
		dueAt = creditLeaveAt
	}

	extID := firstNonEmpty(strings.TrimSpace(fields["external_invoice_id"]), fmt.Sprintf("nasiya_%s_r%d", rctx.Session.SessionID, rowIdx))
	retExternal := firstNonEmpty(inn, phone, cust.RetailerID)

	if rctx.NasiyaRetailerCounts == nil {
		rctx.NasiyaRetailerCounts = make(map[string]int)
	}
	rctx.NasiyaRetailerCounts[cust.RetailerID]++
	entryID := fmt.Sprintf("jentry_migration_%s_%s", rctx.Session.SessionID, retExternal)
	if rctx.NasiyaRetailerCounts[cust.RetailerID] > 1 {
		entryID = fmt.Sprintf("%s_%s", entryID, extID)
	}

	res, err := tx.OpenMigrationNasiyaAR(ctx, ar.OpenFromMigrationRequest{
		SessionID:          rctx.Session.SessionID,
		SupplierID:         rctx.Session.SupplierID,
		RetailerID:         cust.RetailerID,
		RetailerExternalID: retExternal,
		ExternalID:         extID,
		EntryID:            entryID,
		SupplierCurrency:   supCur,
		Currency:           rowCur,
		PrincipalMinor:     principalMinor,
		BalanceMinor:       balMinor,
		CreditLeaveAt:      creditLeaveAt,
		DueAt:              dueAt,
		TermsDays:          termsDays,
		WarehouseID:        rctx.DefaultWarehouseID,
		Now:                now,
		JournalRecorder:    buildPaymentMigrationJournalRecorder(rctx.JournalOverrideFn),
	})
	if err != nil {
		return err
	}
	if err := tx.PutIdentity(ctx, PackNasiyaAR, extID, res.Invoice.InvoiceID); err != nil {
		return err
	}
	if res.AlreadyExisted {
		rctx.Result.Skipped++
		return nil
	}
	rctx.Result.Created++
	if je, ok := res.JournalEntry.(*payment.JournalEntry); ok && je != nil {
		rctx.ChunkJournalEntries = append(rctx.ChunkJournalEntries, je)
	}
	return nil
}
