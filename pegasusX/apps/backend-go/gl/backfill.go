package gl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

const backfillBatchSize = 25

// BackfillOptions configures BackfillFromPaymentLedgerEntries execution.
type BackfillOptions struct {
	Limit    int    `json:"limit,omitempty"`
	PostedBy string `json:"posted_by,omitempty"`
}

// BackfillReport summarizes idempotent backfill from PaymentLedgerEntries into JournalEntries.
type BackfillReport struct {
	EntityID          string `json:"entity_id"`
	RowsScanned       int    `json:"rows_scanned"`
	GroupsEvaluated   int    `json:"groups_evaluated"`
	Posted            int    `json:"posted"`
	AlreadyPosted     int    `json:"already_posted"`
	SkippedUnbalanced int    `json:"skipped_unbalanced"`
	SkippedNonPosting int    `json:"skipped_non_posting"`
}

type pleRow struct {
	LedgerEntryID string
	SessionID     string
	OrderID       string
	SupplierID    string
	RetailerID    string
	Gateway       string
	EntryType     string
	AmountMinor   int64
	Currency      string
	ReferenceID   string
	Source        string
	OccurredAt    time.Time
}

// BackfillFromPaymentLedgerEntries scans PaymentLedgerEntries for entityID, translates
// GL_* double-entry groups and settled legacy payment facts into balanced JournalEntryInputs
// keyed idempotently by ReferenceId ("backfill:ple:" + ReferenceId), and appends them
// inside batched Spanner RW transactions.
func (r *SpannerRepository) BackfillFromPaymentLedgerEntries(ctx context.Context, entityID string, opts BackfillOptions) (BackfillReport, error) {
	if r == nil || r.client == nil {
		return BackfillReport{}, fmt.Errorf("gl spanner repository: nil client")
	}
	return BackfillFromPaymentLedgerEntries(ctx, r.client, entityID, opts)
}

// BackfillFromPaymentLedgerEntries is the standalone Spanner backfill entrypoint.
func BackfillFromPaymentLedgerEntries(ctx context.Context, client *spanner.Client, entityID string, opts BackfillOptions) (BackfillReport, error) {
	if client == nil {
		return BackfillReport{}, fmt.Errorf("gl backfill: nil spanner client")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return BackfillReport{}, fmt.Errorf("%w: entity_id is required", ErrInvalidJournalInput)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}
	postedBy := strings.TrimSpace(opts.PostedBy)
	if postedBy == "" {
		postedBy = "gl_backfill"
	}

	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT LedgerEntryId, SessionId, OrderId, SupplierId, RetailerId, Gateway,
		             EntryType, AmountMinor, Currency, ReferenceId, Source, OccurredAt
		      FROM PaymentLedgerEntries@{FORCE_INDEX=Idx_PaymentLedger_BySupplierOccurred}
		      WHERE SupplierId = @entityId
		      ORDER BY OccurredAt ASC, LedgerEntryId ASC
		      LIMIT @limit`,
		Params: map[string]any{
			"entityId": entityID,
			"limit":    int64(limit),
		},
	})
	defer iter.Stop()

	var rows []pleRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return BackfillReport{}, fmt.Errorf("query PaymentLedgerEntries for backfill: %w", err)
		}
		var pr pleRow
		var sessID, ordID, supID, retID, refID spanner.NullString
		if err := row.Columns(
			&pr.LedgerEntryID, &sessID, &ordID, &supID, &retID, &pr.Gateway,
			&pr.EntryType, &pr.AmountMinor, &pr.Currency, &refID, &pr.Source, &pr.OccurredAt,
		); err != nil {
			return BackfillReport{}, fmt.Errorf("scan PaymentLedgerEntries row: %w", err)
		}
		pr.SessionID = nullStrVal(sessID)
		pr.OrderID = nullStrVal(ordID)
		pr.SupplierID = nullStrVal(supID)
		pr.RetailerID = nullStrVal(retID)
		pr.ReferenceID = nullStrVal(refID)
		rows = append(rows, pr)
	}

	report := BackfillReport{
		EntityID:    entityID,
		RowsScanned: len(rows),
	}

	inputs, stats := buildBackfillInputs(entityID, postedBy, rows)
	report.GroupsEvaluated = stats.GroupsEvaluated
	report.SkippedUnbalanced = stats.SkippedUnbalanced
	report.SkippedNonPosting = stats.SkippedNonPosting

	for start := 0; start < len(inputs); start += backfillBatchSize {
		end := start + backfillBatchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		batch := inputs[start:end]
		var results []JournalEntryResult
		if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			res, err := BufferJournalEntriesBatchInTxn(ctx, txn, nil, batch)
			if err != nil {
				return err
			}
			results = res
			return nil
		}); err != nil {
			return report, fmt.Errorf("backfill batch [%d:%d]: %w", start, end, err)
		}
		for _, r := range results {
			if r.AlreadyPosted {
				report.AlreadyPosted++
			} else {
				report.Posted++
			}
		}
	}

	return report, nil
}

func buildBackfillInputs(entityID, postedBy string, rows []pleRow) ([]JournalEntryInput, BackfillReport) {
	var stats BackfillReport
	glGroups := make(map[string][]pleRow)
	glOrder := make([]string, 0)
	legacyRows := make([]pleRow, 0)

	for _, r := range rows {
		upperType := strings.ToUpper(strings.TrimSpace(r.EntryType))
		if strings.HasPrefix(upperType, "GL_DEBIT_") || strings.HasPrefix(upperType, "GL_CREDIT_") {
			refKey := coalesceNonEmpty(r.ReferenceID, r.LedgerEntryID)
			if _, exists := glGroups[refKey]; !exists {
				glOrder = append(glOrder, refKey)
			}
			glGroups[refKey] = append(glGroups[refKey], r)
			continue
		}
		legacyRows = append(legacyRows, r)
	}

	inputs := make([]JournalEntryInput, 0, len(glOrder)+len(legacyRows))
	seenRef := make(map[string]struct{}, len(glOrder)+len(legacyRows))

	// 1. Convert multi-leg GL_* groups.
	for _, refKey := range glOrder {
		stats.GroupsEvaluated++
		grp := glGroups[refKey]
		sort.Slice(grp, func(i, j int) bool {
			return grp[i].LedgerEntryID < grp[j].LedgerEntryID
		})
		if len(grp) < 2 {
			stats.SkippedUnbalanced++
			continue
		}
		first := grp[0]
		currency := strings.ToUpper(strings.TrimSpace(first.Currency))
		if currency == "" {
			currency = "UZS"
		}
		var sumDebit, sumCredit int64
		postings := make([]PostingInput, 0, len(grp))
		valid := true
		for _, row := range grp {
			if row.AmountMinor <= 0 || !strings.EqualFold(coalesceNonEmpty(row.Currency, currency), currency) {
				valid = false
				break
			}
			dir, acct, pt, pid, ok := parseLegacyGLEntryType(row.EntryType, row)
			if !ok {
				valid = false
				break
			}
			if dir == DirectionDebit {
				sumDebit += row.AmountMinor
			} else {
				sumCredit += row.AmountMinor
			}
			postings = append(postings, PostingInput{
				AccountCode:           acct,
				Direction:             dir,
				AmountMinor:           row.AmountMinor,
				Currency:              currency,
				FunctionalAmountMinor: row.AmountMinor,
				PartyType:             pt,
				PartyID:               pid,
				Description:           fmt.Sprintf("Backfilled from %s (%s)", row.Source, row.LedgerEntryID),
			})
		}
		if !valid || sumDebit != sumCredit || sumDebit <= 0 {
			stats.SkippedUnbalanced++
			continue
		}
		seenRef[refKey] = struct{}{}
		inputs = append(inputs, JournalEntryInput{
			EntityID:       entityID,
			SourceType:     "BACKFILL_PLE",
			SourceID:       refKey,
			OrderID:        first.OrderID,
			Memo:           fmt.Sprintf("Backfill PaymentLedgerEntries ref=%s (%s)", refKey, first.Source),
			Currency:       currency,
			IdempotencyKey: "backfill:ple:" + refKey,
			PostedBy:       postedBy,
			OccurredAt:     first.OccurredAt,
			Postings:       postings,
		})
	}

	// 2. Convert legacy single-leg financial PaymentLedgerEntries not already covered by a GL_* group.
	for _, r := range legacyRows {
		refKey := coalesceNonEmpty(r.ReferenceID, r.LedgerEntryID)
		if _, covered := seenRef[refKey]; covered {
			continue
		}
		stats.GroupsEvaluated++
		if r.AmountMinor <= 0 {
			stats.SkippedNonPosting++
			continue
		}
		drAcct, crAcct, ok := mapSingleLegLegacyAccounts(r.EntryType, r.Gateway)
		if !ok {
			stats.SkippedNonPosting++
			continue
		}
		seenRef[refKey] = struct{}{}
		currency := strings.ToUpper(strings.TrimSpace(r.Currency))
		if currency == "" {
			currency = "UZS"
		}
		drPT, drPID := inferLegacyParty(drAcct, r)
		crPT, crPID := inferLegacyParty(crAcct, r)
		inputs = append(inputs, JournalEntryInput{
			EntityID:       entityID,
			SourceType:     "BACKFILL_PLE_LEGACY",
			SourceID:       refKey,
			OrderID:        r.OrderID,
			Memo:           fmt.Sprintf("Backfill legacy PaymentLedgerEntries %s ref=%s", r.EntryType, refKey),
			Currency:       currency,
			IdempotencyKey: "backfill:ple:" + refKey,
			PostedBy:       postedBy,
			OccurredAt:     r.OccurredAt,
			Postings: []PostingInput{
				{
					AccountCode:           drAcct,
					Direction:             DirectionDebit,
					AmountMinor:           r.AmountMinor,
					Currency:              currency,
					FunctionalAmountMinor: r.AmountMinor,
					PartyType:             drPT,
					PartyID:               drPID,
					Description:           fmt.Sprintf("Backfill debit (%s)", r.EntryType),
				},
				{
					AccountCode:           crAcct,
					Direction:             DirectionCredit,
					AmountMinor:           r.AmountMinor,
					Currency:              currency,
					FunctionalAmountMinor: r.AmountMinor,
					PartyType:             crPT,
					PartyID:               crPID,
					Description:           fmt.Sprintf("Backfill credit (%s)", r.EntryType),
				},
			},
		})
	}

	return inputs, stats
}

func parseLegacyGLEntryType(entryType string, row pleRow) (Direction, string, string, string, bool) {
	upper := strings.ToUpper(strings.TrimSpace(entryType))
	raw := strings.TrimSpace(entryType)
	var dir Direction
	var restUpper, restRaw string
	switch {
	case strings.HasPrefix(upper, "GL_DEBIT_"):
		dir = DirectionDebit
		restUpper = upper[len("GL_DEBIT_"):]
		restRaw = raw[len("GL_DEBIT_"):]
	case strings.HasPrefix(upper, "GL_CREDIT_"):
		dir = DirectionCredit
		restUpper = upper[len("GL_CREDIT_"):]
		restRaw = raw[len("GL_CREDIT_"):]
	default:
		return "", "", "", "", false
	}

	type legacyMap struct {
		prefix    string
		account   string
		partyType string
	}
	ordered := []legacyMap{
		{"AR_DRIVER_VARIANCE", AccountDriverReceivableVariance, "DRIVER"},
		{"AP_DRIVER_VARIANCE", AccountDriverPayableVariance, "DRIVER"},
		{"AR_SUPPLIER_FEES", AccountSupplierFeesReceivable, "SUPPLIER"},
		{"AR_RETAILER", AccountAccountsReceivable, "RETAILER"},
		{"CASH_DRIVER", AccountCashInTransit, "DRIVER"},
		{"CASH_OFFICE", AccountCashOffice, "SUPPLIER"},
		{"PSP_GATEWAY", AccountCardPSPTransit, "GATEWAY"},
		{"BANK_OPERATING", AccountBankOperating, "SUPPLIER"},
		{"AP_SUPPLIER", AccountSupplierPayable, "SUPPLIER"},
		{"ESCROW_ORDER", AccountPlatformEscrow, "ORDER"},
		{"WALLET_RETAILER", AccountRetailerCredit, "RETAILER"},
		{"EQUITY_OPENING_BALANCE", AccountOpeningBalanceEquity, "SUPPLIER"},
		{"REV_PLATFORM_FEE", AccountPlatformFeeRevenue, "SUPPLIER"},
		{"EXP_PROMO_DISCOUNT", AccountDiscountsAndAllowances, "SUPPLIER"},
		{"EXP_CASH_WRITEOFF", AccountCashWriteOffExpense, "SUPPLIER"},
		{"REV_CASH_OVERAGE", AccountCashOverShortRevenue, "SUPPLIER"},
		{"EXP_CLAIMS_SETTLEMENT", AccountClaimsSettlementExpense, "SUPPLIER"},
		{"ADJ_CREDITNOTE", AccountCreditNoteAdjustment, "SUPPLIER"},
		{"REV_SALES", AccountSalesRevenue, ""},
		{"TAX_VAT_OUTPUT", AccountVatOutput, ""},
		{"TAX_VAT_INPUT", AccountVatInput, ""},
		{"COGS", AccountCOGS, ""},
		{"INVENTORY", AccountInventory, ""},
		{"AP_VENDOR", AccountVendorPayable, "VENDOR"},
		{"GRNI", AccountGRNI, ""},
		{"EXP_SHRINK", AccountShrinkExpense, ""},
		{"FX_ROUNDING", AccountFxRounding, ""},
	}
	for _, m := range ordered {
		if restUpper == m.prefix {
			return dir, m.account, m.partyType, fallbackPartyFromRow(m.partyType, row), true
		}
		if strings.HasPrefix(restUpper, m.prefix+"_") {
			suffix := strings.TrimSpace(restRaw[len(m.prefix)+1:])
			if suffix == "" {
				suffix = fallbackPartyFromRow(m.partyType, row)
			}
			return dir, m.account, m.partyType, suffix, true
		}
	}
	// Legacy BuildMigrationOpeningBalanceJournalEntry used "AR:<supplierId>" -> "GL_DEBIT_AR_<supplierId>"
	if strings.HasPrefix(restUpper, "AR_") && len(restRaw) > 3 {
		return dir, AccountAccountsReceivable, "RETAILER", coalesceNonEmpty(row.RetailerID, restRaw[3:]), true
	}
	return "", "", "", "", false
}

func fallbackPartyFromRow(partyType string, r pleRow) string {
	switch partyType {
	case "RETAILER":
		return r.RetailerID
	case "SUPPLIER":
		return r.SupplierID
	case "ORDER":
		return r.OrderID
	case "GATEWAY":
		return r.Gateway
	default:
		return ""
	}
}

func mapSingleLegLegacyAccounts(entryType, gateway string) (debitAcct, creditAcct string, ok bool) {
	t := strings.ToUpper(strings.TrimSpace(entryType))
	if t == "" ||
		strings.Contains(t, "FAILED") ||
		strings.Contains(t, "CANCEL") ||
		strings.Contains(t, "VOID") ||
		strings.Contains(t, "PENDING") ||
		strings.Contains(t, "REQUIRED") ||
		strings.Contains(t, "AWAITING") ||
		strings.Contains(t, "INITIATED") ||
		strings.Contains(t, "UNKNOWN") {
		return "", "", false
	}
	if strings.Contains(t, "CHARGEBACK_REVERSAL") {
		return AccountCardPSPTransit, AccountSupplierPayable, true
	}
	if strings.Contains(t, "CHARGEBACK") {
		return AccountSupplierPayable, AccountAccountsReceivable, true
	}
	if strings.Contains(t, "REFUND") {
		return AccountSupplierPayable, AccountCardPSPTransit, true
	}
	if strings.Contains(t, "PAID") ||
		strings.Contains(t, "CAPTURED") ||
		strings.Contains(t, "CLEARED") ||
		strings.Contains(t, "SETTLED") ||
		strings.Contains(t, "SUCCESS") ||
		strings.Contains(t, "SUCCEEDED") ||
		strings.Contains(t, "COMPLETED") {
		if strings.EqualFold(strings.TrimSpace(gateway), "CASH") {
			return AccountCashInTransit, AccountPlatformEscrow, true
		}
		return AccountCardPSPTransit, AccountPlatformEscrow, true
	}
	return "", "", false
}

func inferLegacyParty(acct string, r pleRow) (string, string) {
	switch acct {
	case AccountAccountsReceivable, AccountRetailerCredit:
		return "RETAILER", r.RetailerID
	case AccountCardPSPTransit:
		return "GATEWAY", r.Gateway
	case AccountSupplierPayable, AccountPlatformEscrow:
		return "SUPPLIER", r.SupplierID
	default:
		return "", ""
	}
}
