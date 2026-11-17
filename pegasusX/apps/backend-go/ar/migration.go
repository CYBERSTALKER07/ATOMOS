package ar

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

var (
	// ErrMigrationCurrencyMismatch rejects opening AR rows whose currency differs from the supplier base currency.
	ErrMigrationCurrencyMismatch = errors.New("ar migration: currency does not match supplier currency")
	// ErrMigrationCreditLedgerMismatch rejects opening AR writes when RetailerCreditProfiles disagrees with open AR sum.
	ErrMigrationCreditLedgerMismatch = errors.New("ar migration: credit profile balance does not match AR subledger sum")
	// ErrMigrationInvalidAmount rejects non-positive opening AR amounts.
	ErrMigrationInvalidAmount = errors.New("ar migration: opening balance must be strictly positive")
	// ErrMigrationJournalRequired fails closed when balanced double-entry construction/validation fails.
	ErrMigrationJournalRequired = errors.New("ar migration: balanced journal entry validation failed")
	// ErrMigrationJournalRecorderUnavailable fails closed when no journal recorder is wired into OpenFromMigration.
	ErrMigrationJournalRecorderUnavailable = errors.New("ar migration: journal recorder is required and unavailable")
)

// MigrationJournalInput describes the balanced double-entry journal parameters required for one migrated AR row.
type MigrationJournalInput struct {
	EntryID     string
	SessionID   string
	OrderID     string
	SupplierID  string
	RetailerID  string
	Currency    string
	AmountMinor int64
	Memo        string
	OccurredAt  time.Time
}

// MigrationJournalRecorder builds, validates, and converts an opening AR double-entry journal into Spanner mutations.
// Implementations must enforce sum(debits) == sum(credits) fail-closed.
type MigrationJournalRecorder interface {
	BuildOpeningBalanceJournal(in MigrationJournalInput) (entry any, mutations []*spanner.Mutation, entryID string, err error)
}

// MigrationJournalRecorderFunc adapts a function to MigrationJournalRecorder.
type MigrationJournalRecorderFunc func(in MigrationJournalInput) (entry any, mutations []*spanner.Mutation, entryID string, err error)

// BuildOpeningBalanceJournal delegates to the wrapped function.
func (f MigrationJournalRecorderFunc) BuildOpeningBalanceJournal(in MigrationJournalInput) (any, []*spanner.Mutation, string, error) {
	return f(in)
}

// OpenFromMigrationRequest imports one legacy nasiya/opening AR balance without invoking OpenFromCreditLeave.
type OpenFromMigrationRequest struct {
	SessionID          string
	SupplierID         string
	RetailerID         string
	RetailerExternalID string
	ExternalID         string
	InvoiceID          string
	OrderID            string
	EntryID            string
	SupplierCurrency   string
	Currency           string
	PrincipalMinor     int64
	BalanceMinor       int64
	CreditLeaveAt      time.Time
	DueAt              time.Time
	TermsDays          int64
	GracePeriodDays    int64
	WarehouseID        string
	Notes              string
	Now                time.Time
	JournalRecorder    MigrationJournalRecorder
}

// CreditProfileState tracks a retailer's supplier credit profile during transactional migration writes.
type CreditProfileState struct {
	Exists               bool
	CreditLimitMinor     int64
	CurrentBalanceMinor  int64
	ReservedMinor        int64
	AvailableCreditMinor int64
	Status               string
	Version              int64
}

// MigrationTxnCache holds read-your-writes AR and credit profile state inside a single chunk RW transaction.
type MigrationTxnCache struct {
	SeenExternalIDs    map[string]string // supplierID|externalID -> invoiceID
	SeenInvoices       map[string]Invoice
	CheckedExternalIDs map[string]bool
	SeenLedgerKeys     map[string]bool
	OpenARSums         map[string]int64 // supplierID|retailerID -> cumulative OPEN/PARTIAL BalanceMinor
	CreditProfiles     map[string]CreditProfileState
	WrittenOrderIDs    map[string]bool
	CreditDirtyPairs   map[string]bool
	PendingGLInputs    []gl.JournalEntryInput
}

// NewMigrationTxnCache initializes an empty in-transaction cache for batched nasiya_ar applies.
func NewMigrationTxnCache() *MigrationTxnCache {
	return &MigrationTxnCache{
		SeenExternalIDs:    map[string]string{},
		SeenInvoices:       map[string]Invoice{},
		CheckedExternalIDs: map[string]bool{},
		SeenLedgerKeys:     map[string]bool{},
		OpenARSums:         map[string]int64{},
		CreditProfiles:     map[string]CreditProfileState{},
		WrittenOrderIDs:    map[string]bool{},
		CreditDirtyPairs:   map[string]bool{},
	}
}

// PreparedMigrationOpening is the pure validated payload ready for Spanner or memory persistence.
type PreparedMigrationOpening struct {
	Invoice             Invoice
	LedgerEntry         LedgerEntry
	JournalEntry        any
	JournalEntryID      string
	SyntheticOrderID    string
	OrderMutationMap    map[string]any
	InvoiceRow          map[string]any
	LedgerRow           map[string]any
	ReservationRow      map[string]any
	ReservationMutation *spanner.Mutation
	JournalMutations    []*spanner.Mutation
}

// OpenFromMigrationResult reports the created/reused opening AR artifacts and updated credit headroom.
type OpenFromMigrationResult struct {
	Invoice              Invoice
	SyntheticOrderID     string
	LedgerEntryID        string
	JournalEntry         any
	JournalEntryID       string
	CreditBalanceMinor   int64
	CreditAvailableMinor int64
	AlreadyExisted       bool
}

func migShortID(prefix, key string, hexLen int) string {
	sum := sha256.Sum256([]byte(key))
	hexStr := fmt.Sprintf("%x", sum)
	if hexLen > 0 && hexLen < len(hexStr) {
		hexStr = hexStr[:hexLen]
	}
	out := prefix + hexStr
	if len(out) > 36 {
		return out[:36]
	}
	return out
}

func migRelKey(supplierID, retailerID string) string {
	return strings.TrimSpace(supplierID) + "|" + strings.TrimSpace(retailerID)
}

func nullableMigString(v string) spanner.NullString {
	v = strings.TrimSpace(v)
	if v == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}

// PrepareMigrationOpeningBalance validates request invariants and constructs the synthetic
// MIGRATION order, unacknowledged MIGRATION AR invoice, AR opening ledger entry, and
// balanced Dr AR / Cr EQUITY:OPENING_BALANCE journal payload via the required recorder.
func PrepareMigrationOpeningBalance(req OpenFromMigrationRequest) (*PreparedMigrationOpening, error) {
	if req.JournalRecorder == nil {
		return nil, ErrMigrationJournalRecorderUnavailable
	}
	supplierID := strings.TrimSpace(req.SupplierID)
	retailerID := strings.TrimSpace(req.RetailerID)
	extID := strings.TrimSpace(req.ExternalID)
	sessionID := strings.TrimSpace(req.SessionID)
	if supplierID == "" || retailerID == "" || extID == "" {
		return nil, fmt.Errorf("ar migration: supplier_id, retailer_id, and external_id are required")
	}
	if req.BalanceMinor <= 0 {
		return nil, fmt.Errorf("%w: got %d", ErrMigrationInvalidAmount, req.BalanceMinor)
	}
	principal := req.PrincipalMinor
	if principal <= 0 {
		principal = req.BalanceMinor
	}
	if principal < req.BalanceMinor {
		principal = req.BalanceMinor
	}

	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = strings.ToUpper(strings.TrimSpace(req.SupplierCurrency))
	}
	if cur == "" {
		return nil, fmt.Errorf("%w: empty currency", ErrMigrationCurrencyMismatch)
	}
	if supCur := strings.ToUpper(strings.TrimSpace(req.SupplierCurrency)); supCur != "" && cur != supCur {
		return nil, fmt.Errorf("%w: row currency %s != supplier currency %s", ErrMigrationCurrencyMismatch, cur, supCur)
	}

	now := req.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	creditLeaveAt := req.CreditLeaveAt.UTC()
	if creditLeaveAt.IsZero() {
		creditLeaveAt = now
	}
	termsDays := req.TermsDays
	if termsDays < 0 {
		termsDays = 0
	}
	dueAt := req.DueAt.UTC()
	if dueAt.IsZero() {
		dueAt = creditLeaveAt.Add(time.Duration(termsDays) * 24 * time.Hour)
	} else if termsDays == 0 && dueAt.After(creditLeaveAt) {
		termsDays = int64(dueAt.Sub(creditLeaveAt) / (24 * time.Hour))
	}
	graceDays := req.GracePeriodDays
	if graceDays < 0 {
		graceDays = 0
	}

	keySeed := supplierID + "|" + extID
	invoiceID := strings.TrimSpace(req.InvoiceID)
	if invoiceID == "" {
		invoiceID = migShortID("ari_mig_", keySeed, 28)
	}
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		orderID = migShortID("ord_armig_", keySeed, 26)
	}
	ledgerEntryID := migShortID("arle_m_", keySeed, 29)
	idemKey := "mig_ar:" + supplierID + ":" + extID

	status := StatusOpen
	if req.BalanceMinor < principal {
		status = StatusPartial
	}
	agingBucket := AgingBucketFor(dueAt, now)

	entryID := strings.TrimSpace(req.EntryID)
	if entryID == "" {
		retSlug := strings.TrimSpace(req.RetailerExternalID)
		if retSlug == "" {
			retSlug = retailerID
		}
		sessSlug := sessionID
		if sessSlug == "" {
			sessSlug = "manual"
		}
		entryID = fmt.Sprintf("jentry_migration_%s_%s", sessSlug, retSlug)
		if len(entryID) > 128 {
			entryID = migShortID("jentry_mig_", sessSlug+"|"+retSlug+"|"+extID, 24)
		}
	}

	journalEntry, journalMuts, resolvedEntryID, err := req.JournalRecorder.BuildOpeningBalanceJournal(MigrationJournalInput{
		EntryID:     entryID,
		SessionID:   sessionID,
		OrderID:     orderID,
		SupplierID:  supplierID,
		RetailerID:  retailerID,
		Currency:    cur,
		AmountMinor: req.BalanceMinor,
		Memo:        fmt.Sprintf("Opening migrated nasiya balance (%s)", extID),
		OccurredAt:  now,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMigrationJournalRequired, err)
	}
	if strings.TrimSpace(resolvedEntryID) == "" {
		resolvedEntryID = entryID
	}

	lineItemsRaw, err := json.Marshal([]map[string]any{})
	if err != nil {
		return nil, err
	}
	orderRow := map[string]any{
		"OrderId":              orderID,
		"SupplierId":           supplierID,
		"RetailerId":           retailerID,
		"WarehouseId":          nullableMigString(req.WarehouseID),
		"Status":               "DELIVERED",
		"OrderSource":          "MIGRATION",
		"ConfirmationStatus":   "CONFIRMED",
		"LineItemsJson":        lineItemsRaw,
		"TotalMinor":           principal,
		"OriginalTotalMinor":   principal,
		"Currency":             cur,
		"DeliveryPriority":     "STANDARD",
		"FiscalStatus":         "HISTORICAL_SKIPPED",
		"ExternalOrderNo":      extID,
		"SupersededByRollback": false,
		"WarehouseNotes":       nullableMigString(req.Notes),
		"Version":              int64(1),
		"CreatedAt":            creditLeaveAt,
		"UpdatedAt":            now,
	}

	inv := Invoice{
		InvoiceID:       invoiceID,
		SupplierID:      supplierID,
		RetailerID:      retailerID,
		OrderID:         orderID,
		Status:          status,
		PrincipalMinor:  principal,
		BalanceMinor:    req.BalanceMinor,
		Currency:        cur,
		CreditLeaveAt:   creditLeaveAt,
		DueAt:           dueAt,
		TermsDays:       termsDays,
		GracePeriodDays: graceDays,
		AgingBucket:     agingBucket,
		Source:          InvoiceSourceMigration,
		ExternalID:      extID,
		AcknowledgedAt:  nil,
		Version:         1,
		CreatedAt:       creditLeaveAt,
		UpdatedAt:       now,
	}

	invoiceRow := map[string]any{
		"InvoiceId":       inv.InvoiceID,
		"SupplierId":      inv.SupplierID,
		"RetailerId":      inv.RetailerID,
		"OrderId":         inv.OrderID,
		"Status":          inv.Status,
		"PrincipalMinor":  inv.PrincipalMinor,
		"BalanceMinor":    inv.BalanceMinor,
		"Currency":        inv.Currency,
		"CreditLeaveAt":   inv.CreditLeaveAt,
		"DueAt":           inv.DueAt,
		"TermsDays":       inv.TermsDays,
		"GracePeriodDays": inv.GracePeriodDays,
		"AgingBucket":     inv.AgingBucket,
		"DunningStep":     int64(0),
		"Source":          InvoiceSourceMigration,
		"ExternalId":      inv.ExternalID,
		"AcknowledgedAt":  spanner.NullTime{},
		"DisputedAt":      spanner.NullTime{},
		"DisputeReason":   spanner.NullString{},
		"Version":         inv.Version,
		"CreatedAt":       inv.CreatedAt,
		"UpdatedAt":       inv.UpdatedAt,
	}

	ledgerEntry := LedgerEntry{
		EntryID:        ledgerEntryID,
		InvoiceID:      inv.InvoiceID,
		SupplierID:     inv.SupplierID,
		RetailerID:     inv.RetailerID,
		EntryType:      "OPENING_BALANCE",
		AmountMinor:    inv.BalanceMinor,
		IdempotencyKey: idemKey,
		RefOrderID:     inv.OrderID,
		CreatedAt:      now,
	}
	ledgerRow := map[string]any{
		"EntryId":        ledgerEntry.EntryID,
		"InvoiceId":      ledgerEntry.InvoiceID,
		"SupplierId":     ledgerEntry.SupplierID,
		"RetailerId":     ledgerEntry.RetailerID,
		"EntryType":      ledgerEntry.EntryType,
		"AmountMinor":    ledgerEntry.AmountMinor,
		"IdempotencyKey": ledgerEntry.IdempotencyKey,
		"RefOrderId":     ledgerEntry.RefOrderID,
		"CreatedAt":      ledgerEntry.CreatedAt,
	}

	resRow := map[string]any{
		"OrderId":     orderID,
		"RetailerId":  retailerID,
		"SupplierId":  supplierID,
		"AmountMinor": req.BalanceMinor,
		"Status":      "CONVERTED",
		"CreatedAt":   spanner.CommitTimestamp,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	resMutation := spanner.InsertOrUpdateMap("OrderCreditReservations", resRow)

	return &PreparedMigrationOpening{
		Invoice:             inv,
		LedgerEntry:         ledgerEntry,
		JournalEntry:        journalEntry,
		JournalEntryID:      resolvedEntryID,
		SyntheticOrderID:    orderID,
		OrderMutationMap:    orderRow,
		InvoiceRow:          invoiceRow,
		LedgerRow:           ledgerRow,
		ReservationRow:      resRow,
		ReservationMutation: resMutation,
		JournalMutations:    journalMuts,
	}, nil
}

// ComputeMigrationCreditProfile verifies that the AR subledger open balance sum matches the
// credit profile balance before incrementing, returning the updated credit profile state.
func ComputeMigrationCreditProfile(existing CreditProfileState, preOpenSum int64, addMinor int64) (CreditProfileState, error) {
	if addMinor <= 0 {
		return CreditProfileState{}, fmt.Errorf("%w: delta=%d", ErrMigrationInvalidAmount, addMinor)
	}
	if existing.Exists && existing.CurrentBalanceMinor != preOpenSum {
		return CreditProfileState{}, fmt.Errorf(
			"%w: pre-existing ArInvoices sum %d != RetailerCreditProfiles.CurrentBalanceMinor %d",
			ErrMigrationCreditLedgerMismatch, preOpenSum, existing.CurrentBalanceMinor,
		)
	}
	newBalance := preOpenSum + addMinor
	available := existing.CreditLimitMinor - newBalance - existing.ReservedMinor
	if available < 0 {
		available = 0
	}
	status := strings.TrimSpace(existing.Status)
	if status == "" || strings.EqualFold(status, "INACTIVE") {
		status = "ACTIVE"
	}
	ver := existing.Version + 1
	if !existing.Exists {
		ver = 1
	}
	return CreditProfileState{
		Exists:               existing.Exists,
		CreditLimitMinor:     existing.CreditLimitMinor,
		CurrentBalanceMinor:  newBalance,
		ReservedMinor:        existing.ReservedMinor,
		AvailableCreditMinor: available,
		Status:               status,
		Version:              ver,
	}, nil
}

// OpenFromMigration atomically imports a migrated opening AR invoice on Spanner.
// It NEVER invokes OpenFromCreditLeave and ALWAYS fails closed without a validated journal recorder.
func OpenFromMigration(ctx context.Context, client *spanner.Client, req OpenFromMigrationRequest) (*OpenFromMigrationResult, error) {
	if client == nil {
		return nil, fmt.Errorf("ar migration: spanner client is required")
	}
	var res *OpenFromMigrationResult
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cache := NewMigrationTxnCache()
		out, err := OpenFromMigrationInTxn(ctx, txn, cache, req)
		if err != nil {
			return err
		}
		if err := FlushMigrationCreditProfilesAndGLInTxn(ctx, txn, nil, cache, req.Now); err != nil {
			return err
		}
		res = out
		return nil
	})
	return res, err
}

// PreloadMigrationTxnCacheInTxn batch-loads ArInvoices, open AR sums, and credit profiles
// into MigrationTxnCache once per chunk transaction to eliminate per-row N+1 Spanner RPCs.
func PreloadMigrationTxnCacheInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	cache *MigrationTxnCache,
	supplierID string,
	externalIDs []string,
	retailerIDs []string,
) error {
	if txn == nil || cache == nil {
		return nil
	}
	supID := strings.TrimSpace(supplierID)
	if supID == "" {
		return nil
	}

	uniqExt := make([]string, 0, len(externalIDs))
	seenExt := make(map[string]bool, len(externalIDs))
	for _, raw := range externalIDs {
		eid := strings.TrimSpace(raw)
		if eid == "" || seenExt[eid] {
			continue
		}
		seenExt[eid] = true
		extKey := supID + "|" + eid
		if cache.CheckedExternalIDs != nil && cache.CheckedExternalIDs[extKey] {
			continue
		}
		uniqExt = append(uniqExt, eid)
	}
	if len(uniqExt) > 0 {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT ` + invoiceSelectColumns + ` FROM ArInvoices WHERE SupplierId = @sid AND ExternalId IN UNNEST(@eids)`,
			Params: map[string]any{"sid": supID, "eids": uniqExt},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			inv, err := scanInvoice(row)
			if err != nil {
				iter.Stop()
				return err
			}
			extKey := supID + "|" + strings.TrimSpace(inv.ExternalID)
			cache.SeenExternalIDs[extKey] = inv.InvoiceID
			if cache.SeenInvoices != nil {
				cache.SeenInvoices[extKey] = inv
			}
		}
		iter.Stop()
		if cache.CheckedExternalIDs != nil {
			for _, eid := range uniqExt {
				cache.CheckedExternalIDs[supID+"|"+eid] = true
			}
		}
	}

	uniqRet := make([]string, 0, len(retailerIDs))
	seenRet := make(map[string]bool, len(retailerIDs))
	for _, raw := range retailerIDs {
		rid := strings.TrimSpace(raw)
		if rid == "" || seenRet[rid] {
			continue
		}
		seenRet[rid] = true
		relKey := migRelKey(supID, rid)
		if _, hasSum := cache.OpenARSums[relKey]; hasSum {
			if _, hasProf := cache.CreditProfiles[relKey]; hasProf {
				continue
			}
		}
		uniqRet = append(uniqRet, rid)
		if _, ok := cache.OpenARSums[relKey]; !ok {
			cache.OpenARSums[relKey] = 0
		}
		if _, ok := cache.CreditProfiles[relKey]; !ok {
			cache.CreditProfiles[relKey] = CreditProfileState{Exists: false, Status: "ACTIVE"}
		}
	}
	if len(uniqRet) > 0 {
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, COALESCE(SUM(BalanceMinor), 0) FROM ArInvoices
			      WHERE SupplierId = @sid AND RetailerId IN UNNEST(@rids) AND Status IN ('OPEN', 'PARTIAL')
			      GROUP BY RetailerId`,
			Params: map[string]any{"sid": supID, "rids": uniqRet},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var rid string
			var sum int64
			if err := row.Columns(&rid, &sum); err != nil {
				iter.Stop()
				return err
			}
			cache.OpenARSums[migRelKey(supID, rid)] = sum
		}
		iter.Stop()

		keys := make([]spanner.Key, 0, len(uniqRet))
		for _, rid := range uniqRet {
			keys = append(keys, spanner.Key{rid, supID})
		}
		profIter := txn.Read(ctx, "RetailerCreditProfiles", spanner.KeySetFromKeys(keys...),
			[]string{"RetailerId", "CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "AvailableCreditMinor", "Status", "Version"})
		for {
			row, err := profIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				profIter.Stop()
				return err
			}
			var rid string
			st := CreditProfileState{Exists: true}
			if err := row.Columns(&rid, &st.CreditLimitMinor, &st.CurrentBalanceMinor, &st.ReservedMinor, &st.AvailableCreditMinor, &st.Status, &st.Version); err != nil {
				profIter.Stop()
				return err
			}
			cache.CreditProfiles[migRelKey(supID, rid)] = st
		}
		profIter.Stop()
	}
	return nil
}

// OpenFromMigrationInTxn executes a single opening AR migration inside an existing Spanner RW transaction.
// Callers batching multiple invoices in one RW transaction must call FlushMigrationCreditProfilesInTxn once before commit.
func OpenFromMigrationInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	cache *MigrationTxnCache,
	req OpenFromMigrationRequest,
) (*OpenFromMigrationResult, error) {
	if txn == nil {
		return nil, fmt.Errorf("ar migration: nil spanner transaction")
	}
	if cache == nil {
		cache = NewMigrationTxnCache()
	}
	prep, err := PrepareMigrationOpeningBalance(req)
	if err != nil {
		return nil, err
	}

	extKey := strings.TrimSpace(req.SupplierID) + "|" + strings.TrimSpace(req.ExternalID)
	if existingInv, ok := cache.SeenInvoices[extKey]; ok {
		return &OpenFromMigrationResult{
			Invoice:          existingInv,
			SyntheticOrderID: existingInv.OrderID,
			AlreadyExisted:   true,
		}, nil
	}
	if existingID, ok := cache.SeenExternalIDs[extKey]; ok && existingID != "" {
		return &OpenFromMigrationResult{
			Invoice:        Invoice{InvoiceID: existingID, SupplierID: req.SupplierID, RetailerID: req.RetailerID, ExternalID: req.ExternalID, Source: InvoiceSourceMigration},
			AlreadyExisted: true,
		}, nil
	}

	if cache.CheckedExternalIDs == nil || !cache.CheckedExternalIDs[extKey] {
		existingInv, found, err := lookupInvoiceByExternalIDInTxn(ctx, txn, req.SupplierID, req.ExternalID)
		if err != nil {
			return nil, err
		}
		if cache.CheckedExternalIDs != nil {
			cache.CheckedExternalIDs[extKey] = true
		}
		if found {
			cache.SeenExternalIDs[extKey] = existingInv.InvoiceID
			if cache.SeenInvoices != nil {
				cache.SeenInvoices[extKey] = existingInv
			}
			return &OpenFromMigrationResult{
				Invoice:          existingInv,
				SyntheticOrderID: existingInv.OrderID,
				AlreadyExisted:   true,
			}, nil
		}
	}

	// Append-only ArLedgerEntries idempotency pre-read guard before InsertMap.
	if cache.SeenLedgerKeys == nil {
		cache.SeenLedgerKeys = map[string]bool{}
	}
	ledgerKey := strings.TrimSpace(prep.LedgerEntry.IdempotencyKey)
	if ledgerKey != "" {
		if cache.SeenLedgerKeys[ledgerKey] {
			return &OpenFromMigrationResult{
				Invoice:          prep.Invoice,
				SyntheticOrderID: prep.SyntheticOrderID,
				LedgerEntryID:    prep.LedgerEntry.EntryID,
				AlreadyExisted:   true,
			}, nil
		}
		lIter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
			Params: map[string]any{"k": ledgerKey},
		})
		_, lErr := lIter.Next()
		lIter.Stop()
		if lErr == nil {
			cache.SeenLedgerKeys[ledgerKey] = true
			return &OpenFromMigrationResult{
				Invoice:          prep.Invoice,
				SyntheticOrderID: prep.SyntheticOrderID,
				LedgerEntryID:    prep.LedgerEntry.EntryID,
				AlreadyExisted:   true,
			}, nil
		}
		if lErr != iterator.Done {
			return nil, lErr
		}
	}

	relKey := migRelKey(req.SupplierID, req.RetailerID)
	preOpenSum, hasCachedSum := cache.OpenARSums[relKey]
	if !hasCachedSum {
		sumVal, sumErr := sumOpenBalanceForRelInTxn(ctx, txn, req.SupplierID, req.RetailerID)
		if sumErr != nil {
			return nil, sumErr
		}
		preOpenSum = sumVal
		cache.OpenARSums[relKey] = preOpenSum
	}

	profile, hasCachedProfile := cache.CreditProfiles[relKey]
	if !hasCachedProfile {
		loaded, loadErr := loadCreditProfileInTxn(ctx, txn, req.SupplierID, req.RetailerID)
		if loadErr != nil {
			return nil, loadErr
		}
		profile = loaded
	}

	updatedProfile, err := ComputeMigrationCreditProfile(profile, preOpenSum, prep.Invoice.BalanceMinor)
	if err != nil {
		return nil, err
	}

	muts := make([]*spanner.Mutation, 0, 5+len(prep.JournalMutations))
	if !cache.WrittenOrderIDs[prep.SyntheticOrderID] {
		muts = append(muts, spanner.InsertOrUpdateMap("Orders", prep.OrderMutationMap))
		cache.WrittenOrderIDs[prep.SyntheticOrderID] = true
	}
	muts = append(muts,
		spanner.InsertOrUpdateMap("ArInvoices", prep.InvoiceRow),
		spanner.InsertMap("ArLedgerEntries", prep.LedgerRow),
	)
	if prep.ReservationMutation != nil {
		muts = append(muts, prep.ReservationMutation)
	} else if prep.ReservationRow != nil {
		muts = append(muts, spanner.InsertOrUpdateMap("OrderCreditReservations", prep.ReservationRow))
	}
	muts = append(muts, prep.JournalMutations...)

	if err := txn.BufferWrite(muts); err != nil {
		return nil, err
	}

	glIn, err := gl.BuildMigrationOpeningBalanceInput(gl.RuleEvalContext{
		EntityID:       prep.Invoice.SupplierID,
		SupplierID:     prep.Invoice.SupplierID,
		RetailerID:     prep.Invoice.RetailerID,
		SourceID:       prep.Invoice.InvoiceID,
		OrderID:        prep.SyntheticOrderID,
		Currency:       prep.Invoice.Currency,
		GrossMinor:     prep.Invoice.BalanceMinor,
		IdempotencyKey: "gl:mig_open:" + prep.JournalEntryID,
		Memo:           fmt.Sprintf("Opening migrated nasiya balance (%s)", prep.Invoice.ExternalID),
		OccurredAt:     prep.Invoice.UpdatedAt,
	})
	if err != nil {
		return nil, err
	}
	cache.PendingGLInputs = append(cache.PendingGLInputs, glIn)
	if ledgerKey != "" {
		cache.SeenLedgerKeys[ledgerKey] = true
	}

	cache.SeenExternalIDs[extKey] = prep.Invoice.InvoiceID
	cache.OpenARSums[relKey] = preOpenSum + prep.Invoice.BalanceMinor
	cache.CreditProfiles[relKey] = updatedProfile
	cache.CreditDirtyPairs[relKey] = true

	return &OpenFromMigrationResult{
		Invoice:              prep.Invoice,
		SyntheticOrderID:     prep.SyntheticOrderID,
		LedgerEntryID:        prep.LedgerEntry.EntryID,
		JournalEntry:         prep.JournalEntry,
		JournalEntryID:       prep.JournalEntryID,
		CreditBalanceMinor:   updatedProfile.CurrentBalanceMinor,
		CreditAvailableMinor: updatedProfile.AvailableCreditMinor,
		AlreadyExisted:       false,
	}, nil
}

// FlushMigrationCreditProfilesInTxn buffers one final upsert per dirty (SupplierId, RetailerId) credit profile
// and flushes any pending migrated GL journal entries in a single chained batch.
func FlushMigrationCreditProfilesInTxn(txn *spanner.ReadWriteTransaction, cache *MigrationTxnCache, now time.Time) error {
	return FlushMigrationCreditProfilesAndGLInTxn(context.Background(), txn, nil, cache, now)
}

// FlushMigrationCreditProfilesAndGLInTxn buffers dirty RetailerCreditProfiles updates and batches any
// accumulated migration opening-balance GL entries cleanly inside txn.
func FlushMigrationCreditProfilesAndGLInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, cache *MigrationTxnCache, now time.Time) error {
	if txn == nil || cache == nil {
		return nil
	}
	if len(cache.CreditDirtyPairs) > 0 {
		if now.IsZero() {
			now = time.Now().UTC()
		}
		muts := make([]*spanner.Mutation, 0, len(cache.CreditDirtyPairs))
		for relKey := range cache.CreditDirtyPairs {
			parts := strings.SplitN(relKey, "|", 2)
			if len(parts) != 2 {
				continue
			}
			supplierID, retailerID := parts[0], parts[1]
			st, ok := cache.CreditProfiles[relKey]
			if !ok {
				continue
			}
			row := map[string]any{
				"RetailerId":           retailerID,
				"SupplierId":           supplierID,
				"CreditLimitMinor":     st.CreditLimitMinor,
				"CurrentBalanceMinor":  st.CurrentBalanceMinor,
				"ReservedMinor":        st.ReservedMinor,
				"AvailableCreditMinor": st.AvailableCreditMinor,
				"Status":               st.Status,
				"LastEvaluatedAt":      now.UTC(),
				"Version":              st.Version,
				"UpdatedAt":            spanner.CommitTimestamp,
			}
			if st.Exists {
				muts = append(muts, spanner.UpdateMap("RetailerCreditProfiles", row))
			} else {
				row["RiskScore"] = int64(50)
				row["DelinquencyCount"] = int64(0)
				row["CreatedAt"] = spanner.CommitTimestamp
				muts = append(muts, spanner.InsertOrUpdateMap("RetailerCreditProfiles", row))
			}
		}
		if len(muts) > 0 {
			if err := txn.BufferWrite(muts); err != nil {
				return err
			}
		}
	}
	if len(cache.PendingGLInputs) > 0 {
		if _, err := gl.BufferJournalEntriesBatchInTxn(ctx, txn, outboxBuf, cache.PendingGLInputs); err != nil {
			return err
		}
		cache.PendingGLInputs = nil
	}
	return nil
}

func lookupInvoiceByExternalIDInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, externalID string) (Invoice, bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT ` + invoiceSelectColumns + ` FROM ArInvoices WHERE SupplierId = @sid AND ExternalId = @eid LIMIT 1`,
		Params: map[string]any{"sid": strings.TrimSpace(supplierID), "eid": strings.TrimSpace(externalID)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Invoice{}, false, nil
	}
	if err != nil {
		return Invoice{}, false, err
	}
	inv, err := scanInvoice(row)
	return inv, err == nil, err
}

func sumOpenBalanceForRelInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, retailerID string) (int64, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(SUM(BalanceMinor), 0) FROM ArInvoices
			WHERE SupplierId = @sid AND RetailerId = @rid AND Status IN ('OPEN', 'PARTIAL')`,
		Params: map[string]any{"sid": strings.TrimSpace(supplierID), "rid": strings.TrimSpace(retailerID)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var sum int64
	if err := row.Columns(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

func loadCreditProfileInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, retailerID string) (CreditProfileState, error) {
	row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{strings.TrimSpace(retailerID), strings.TrimSpace(supplierID)},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "AvailableCreditMinor", "Status", "Version"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return CreditProfileState{Exists: false, Status: "ACTIVE"}, nil
		}
		return CreditProfileState{}, err
	}
	var st CreditProfileState
	st.Exists = true
	if err := row.Columns(&st.CreditLimitMinor, &st.CurrentBalanceMinor, &st.ReservedMinor, &st.AvailableCreditMinor, &st.Status, &st.Version); err != nil {
		return CreditProfileState{}, err
	}
	return st, nil
}

// OpenFromMigration executes an in-memory migration AR opening write for unit testing and MemoryApplyStore.
func (r *MemoryRepository) OpenFromMigration(
	req OpenFromMigrationRequest,
	creditProfiles map[string]CreditProfileState,
) (*OpenFromMigrationResult, error) {
	if r == nil {
		return nil, fmt.Errorf("ar migration: nil memory repository")
	}
	prep, err := PrepareMigrationOpeningBalance(req)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()

	for _, existing := range r.byID {
		if existing.SupplierID == prep.Invoice.SupplierID &&
			existing.ExternalID != "" &&
			strings.EqualFold(existing.ExternalID, prep.Invoice.ExternalID) {
			return &OpenFromMigrationResult{
				Invoice:          existing,
				SyntheticOrderID: existing.OrderID,
				AlreadyExisted:   true,
			}, nil
		}
	}

	var preOpenSum int64
	for _, inv := range r.byID {
		if inv.SupplierID == prep.Invoice.SupplierID &&
			inv.RetailerID == prep.Invoice.RetailerID &&
			(inv.Status == StatusOpen || inv.Status == StatusPartial) {
			preOpenSum += inv.BalanceMinor
		}
	}

	key := migRelKey(prep.Invoice.SupplierID, prep.Invoice.RetailerID)
	var existingProfile CreditProfileState
	if creditProfiles != nil {
		existingProfile = creditProfiles[key]
	}
	updatedProfile, err := ComputeMigrationCreditProfile(existingProfile, preOpenSum, prep.Invoice.BalanceMinor)
	if err != nil {
		return nil, err
	}

	r.byID[prep.Invoice.InvoiceID] = prep.Invoice
	r.byOrder[prep.Invoice.OrderID] = prep.Invoice.InvoiceID
	r.ledgerKeys[prep.LedgerEntry.IdempotencyKey] = true
	r.ledgerEntries = append(r.ledgerEntries, prep.LedgerEntry)
	if creditProfiles != nil {
		updatedProfile.Exists = true
		creditProfiles[key] = updatedProfile
	}

	return &OpenFromMigrationResult{
		Invoice:              prep.Invoice,
		SyntheticOrderID:     prep.SyntheticOrderID,
		LedgerEntryID:        prep.LedgerEntry.EntryID,
		JournalEntry:         prep.JournalEntry,
		JournalEntryID:       prep.JournalEntryID,
		CreditBalanceMinor:   updatedProfile.CurrentBalanceMinor,
		CreditAvailableMinor: updatedProfile.AvailableCreditMinor,
		AlreadyExisted:       false,
	}, nil
}

// ExportInvoices returns a deep copy of invoices in MemoryRepository for transactional snapshotting.
func (r *MemoryRepository) ExportInvoices() map[string]Invoice {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Invoice, len(r.byID))
	for k, v := range r.byID {
		out[k] = v
	}
	return out
}

// ImportInvoices replaces the MemoryRepository invoice map atomically from a committed snapshot.
func (r *MemoryRepository) ImportInvoices(m map[string]Invoice) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	r.byID = make(map[string]Invoice, len(m))
	r.byOrder = make(map[string]string, len(m))
	for id, inv := range m {
		r.byID[id] = inv
		if inv.OrderID != "" {
			r.byOrder[inv.OrderID] = id
		}
	}
}

// ExportLedger returns copies of the AR ledger entries and idempotency keys.
func (r *MemoryRepository) ExportLedger() ([]LedgerEntry, map[string]bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := append([]LedgerEntry(nil), r.ledgerEntries...)
	keys := make(map[string]bool, len(r.ledgerKeys))
	for k, v := range r.ledgerKeys {
		keys[k] = v
	}
	return entries, keys
}

// ImportLedger restores the AR ledger entries and idempotency keys from a snapshot.
func (r *MemoryRepository) ImportLedger(entries []LedgerEntry, keys map[string]bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	r.ledgerEntries = append([]LedgerEntry(nil), entries...)
	r.ledgerKeys = make(map[string]bool, len(keys))
	for k, v := range keys {
		r.ledgerKeys[k] = v
	}
}

// ListMigrationOpeningBalances returns migrated AR invoices belonging to retailerID.
func (r *MemoryRepository) ListMigrationOpeningBalances(_ context.Context, retailerID string) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.RetailerID == retailerID && strings.EqualFold(inv.Source, InvoiceSourceMigration) {
			out = append(out, inv)
		}
	}
	sortInvoicesDueAsc(out)
	return out, nil
}

// AcknowledgeMigrationInvoice clears the migration dunning hold on an opening AR invoice in memory.
func (r *MemoryRepository) AcknowledgeMigrationInvoice(_ context.Context, retailerID, invoiceID string, now time.Time) (Invoice, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.byID[invoiceID]
	if !ok {
		return Invoice{}, fmt.Errorf("invoice_not_found")
	}
	if strings.TrimSpace(retailerID) != "" && inv.RetailerID != retailerID {
		return Invoice{}, fmt.Errorf("forbidden: invoice belongs to another retailer")
	}
	ts := now.UTC()
	inv.AcknowledgedAt = &ts
	inv.Version++
	inv.UpdatedAt = ts
	r.byID[invoiceID] = inv
	return inv, nil
}

// DisputeMigrationInvoice marks an opening AR invoice disputed and keeps dunning frozen in memory.
func (r *MemoryRepository) DisputeMigrationInvoice(_ context.Context, retailerID, invoiceID, reason string, now time.Time) (Invoice, error) {
	if strings.TrimSpace(reason) == "" {
		return Invoice{}, fmt.Errorf("dispute reason is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.byID[invoiceID]
	if !ok {
		return Invoice{}, fmt.Errorf("invoice_not_found")
	}
	if strings.TrimSpace(retailerID) != "" && inv.RetailerID != retailerID {
		return Invoice{}, fmt.Errorf("forbidden: invoice belongs to another retailer")
	}
	ts := now.UTC()
	inv.DisputedAt = &ts
	inv.DisputeReason = strings.TrimSpace(reason)
	inv.Version++
	inv.UpdatedAt = ts
	r.byID[invoiceID] = inv
	return inv, nil
}

// ListMigrationOpeningBalances loads migrated opening AR invoices for a retailer from Spanner.
func (r *SpannerRepository) ListMigrationOpeningBalances(ctx context.Context, retailerID string) ([]Invoice, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("ar migration: nil spanner client")
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ` + invoiceSelectColumns + ` FROM ArInvoices
			WHERE RetailerId = @rid AND Source = 'MIGRATION'
			ORDER BY DueAt ASC, InvoiceId ASC`,
		Params: map[string]any{"rid": strings.TrimSpace(retailerID)},
	})
	defer iter.Stop()
	out := make([]Invoice, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		inv, err := scanInvoice(row)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
}

// AcknowledgeMigrationInvoice stamps AcknowledgedAt on a retailer's migrated AR invoice, releasing dunning freeze.
func (r *SpannerRepository) AcknowledgeMigrationInvoice(ctx context.Context, retailerID, invoiceID string, now time.Time) (Invoice, error) {
	if r == nil || r.client == nil {
		return Invoice{}, fmt.Errorf("ar migration: nil spanner client")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var updated Invoice
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, invoiceSelectColumnNames)
		if err != nil {
			return err
		}
		inv, err := scanInvoice(row)
		if err != nil {
			return err
		}
		if strings.TrimSpace(retailerID) != "" && inv.RetailerID != retailerID {
			return fmt.Errorf("forbidden: invoice belongs to another retailer")
		}
		ts := now.UTC()
		inv.AcknowledgedAt = &ts
		inv.Version++
		inv.UpdatedAt = ts
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ArInvoices", map[string]any{
				"InvoiceId":      invoiceID,
				"AcknowledgedAt": ts,
				"Version":        inv.Version,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		updated = inv
		return nil
	})
	return updated, err
}

// DisputeMigrationInvoice records a retailer's balance dispute reason on a migrated AR invoice and keeps dunning frozen.
func (r *SpannerRepository) DisputeMigrationInvoice(ctx context.Context, retailerID, invoiceID, reason string, now time.Time) (Invoice, error) {
	if r == nil || r.client == nil {
		return Invoice{}, fmt.Errorf("ar migration: nil spanner client")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Invoice{}, fmt.Errorf("dispute reason is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var updated Invoice
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, invoiceSelectColumnNames)
		if err != nil {
			return err
		}
		inv, err := scanInvoice(row)
		if err != nil {
			return err
		}
		if strings.TrimSpace(retailerID) != "" && inv.RetailerID != retailerID {
			return fmt.Errorf("forbidden: invoice belongs to another retailer")
		}
		ts := now.UTC()
		inv.DisputedAt = &ts
		inv.DisputeReason = reason
		inv.Version++
		inv.UpdatedAt = ts
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ArInvoices", map[string]any{
				"InvoiceId":     invoiceID,
				"DisputedAt":    ts,
				"DisputeReason": reason,
				"Version":       inv.Version,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		updated = inv
		return nil
	})
	return updated, err
}
