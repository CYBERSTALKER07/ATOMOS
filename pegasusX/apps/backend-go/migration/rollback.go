package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

const (
	// SnapshotOpInsert indicates a newly inserted entity deleted (or superseded for financial tables) on rollback.
	SnapshotOpInsert = "INSERT"
	// SnapshotOpUpdate indicates an updated entity restored to its pre-apply before_json image on rollback.
	SnapshotOpUpdate = "UPDATE"
)

var (
	// ErrRollbackWindowExpired indicates the 72h post-apply instant rollback window has elapsed.
	ErrRollbackWindowExpired = errors.New("rollback_window_expired")
	// ErrRollbackNotAllowed indicates session state (e.g. CUTOVER_COMPLETE) forbids instant rollback.
	ErrRollbackNotAllowed = errors.New("rollback_not_allowed_in_state")
)

// RollbackResult summarizes compensating mutations applied when rolling back a migration session.
type RollbackResult struct {
	Session             Session  `json:"session"`
	RevertedSnapshots   int64    `json:"reverted_snapshots"`
	ReversedInvoices    int64    `json:"reversed_invoices"`
	SupersededOrders    int64    `json:"superseded_orders"`
	ReversedAmountMinor int64    `json:"reversed_amount_minor"`
	ReversalEntryIDs    []string `json:"reversal_entry_ids,omitempty"`
}

// RollbackInvoiceBeforeImage captures the minimal pre/post metadata needed when reversing migrated ArInvoices.
type RollbackInvoiceBeforeImage struct {
	InvoiceID    string `json:"invoice_id"`
	SupplierID   string `json:"supplier_id"`
	RetailerID   string `json:"retailer_id"`
	OrderID      string `json:"order_id"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
	BeforeStatus string `json:"before_status,omitempty"`
}

// BuildSnapshotRecord constructs one deterministic MigrationSnapshotRecord with 30-day TTL.
func BuildSnapshotRecord(
	supplierID, sessionID string,
	seq int64,
	targetTable string,
	keyObj any,
	opKind string,
	beforeObj any,
	now time.Time,
) (MigrationSnapshotRecord, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	keyRaw, err := json.Marshal(keyObj)
	if err != nil {
		return MigrationSnapshotRecord{}, fmt.Errorf("marshal snapshot key: %w", err)
	}
	var beforeRaw json.RawMessage
	if beforeObj != nil {
		b, err := json.Marshal(beforeObj)
		if err != nil {
			return MigrationSnapshotRecord{}, fmt.Errorf("marshal snapshot before image: %w", err)
		}
		beforeRaw = b
	}
	return MigrationSnapshotRecord{
		SupplierID:  supplierID,
		SessionID:   sessionID,
		SnapshotSeq: seq,
		TargetTable: strings.TrimSpace(targetTable),
		RowKeyJSON:  string(keyRaw),
		OpKind:      strings.ToUpper(strings.TrimSpace(opKind)),
		BeforeJSON:  beforeRaw,
		ExpiresAt:   now.UTC().Add(SnapshotRetentionTTL),
		CreatedAt:   now.UTC(),
	}, nil
}

// BuildReversalJournalEntry creates a balanced double-entry reversal journal (Dr EQUITY:OPENING_BALANCE / Cr AR:{supplierId})
// for a rolled-back migrated nasiya invoice, verified via payment.JournalEntry.Validate().
func BuildReversalJournalEntry(
	supplierID, sessionID, invoiceID, orderID, retailerID, currency string,
	amountMinor int64,
	now time.Time,
) (*payment.JournalEntry, error) {
	if amountMinor <= 0 {
		return nil, fmt.Errorf("reversal amountMinor must be positive, got %d", amountMinor)
	}
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		cur = "UZS"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	je := &payment.JournalEntry{
		EntryID:       "jentry_migration_reversal_" + invoiceID,
		TransactionID: "rollback_" + sessionID + "_" + invoiceID,
		OrderID:       orderID,
		SessionID:     sessionID,
		SupplierID:    supplierID,
		RetailerID:    retailerID,
		Currency:      cur,
		Memo:          fmt.Sprintf("Migration rollback reversal for AR invoice %s", invoiceID),
		OccurredAt:    now.UTC(),
		Postings: []payment.Posting{
			{
				AccountCode: "EQUITY:OPENING_BALANCE",
				AccountType: payment.AccountClassEquity,
				Direction:   payment.PostingDebit,
				AmountMinor: amountMinor,
				Currency:    cur,
				PartyID:     supplierID,
				Description: "Reverse opening migration AR equity offset",
			},
			{
				AccountCode: fmt.Sprintf("AR:%s", supplierID),
				AccountType: payment.AccountClassAsset,
				Direction:   payment.PostingCredit,
				AmountMinor: amountMinor,
				Currency:    cur,
				PartyID:     retailerID,
				Description: "Reverse opening migration AR asset",
			},
		},
	}
	if err := je.Validate(); err != nil {
		return nil, err
	}
	return je, nil
}

// RollbackSession enforces the 72h post-apply rollback gate, reverses snapshots in reverse sequence order,
// supersedes financial order/AR rows with compensating journal entries (never deleting ledger history), and marks ROLLED_BACK.
func RollbackSession(ctx context.Context, repo Repository, supplierID, sessionID string, now time.Time) (RollbackResult, error) {
	if repo == nil {
		return RollbackResult{}, fmt.Errorf("migration repository is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	sess, err := repo.GetSession(ctx, supplierID, sessionID)
	if err != nil {
		return RollbackResult{}, err
	}
	cur := NormalizeSessionStatus(string(sess.Status))
	if cur == SessionStatusRolledBack {
		return RollbackResult{Session: sess}, nil
	}
	if cur == SessionStatusCutoverComplete {
		return RollbackResult{}, fmt.Errorf("%w: cannot rollback after cutover_complete", ErrRollbackNotAllowed)
	}
	if err := ValidateSessionTransition(cur, SessionStatusRollingBack); err != nil {
		return RollbackResult{}, fmt.Errorf("%w: %v", ErrRollbackNotAllowed, err)
	}

	if cur == SessionStatusApplied {
		if sess.RollbackUntil == nil {
			return RollbackResult{}, fmt.Errorf("%w: missing rollback deadline", ErrRollbackWindowExpired)
		}
		if now.UTC().After(sess.RollbackUntil.UTC()) {
			return RollbackResult{}, fmt.Errorf("%w: deadline %s exceeded", ErrRollbackWindowExpired, sess.RollbackUntil.UTC().Format(time.RFC3339))
		}
	} else if sess.RollbackUntil != nil && now.UTC().After(sess.RollbackUntil.UTC()) {
		return RollbackResult{}, fmt.Errorf("%w: deadline %s exceeded", ErrRollbackWindowExpired, sess.RollbackUntil.UTC().Format(time.RFC3339))
	}

	if _, err := repo.UpdateSessionStatus(ctx, supplierID, sessionID, SessionStatusRollingBack, nil); err != nil {
		return RollbackResult{}, err
	}

	snapshots, err := repo.ListSnapshotsBySession(ctx, supplierID, sessionID)
	if err != nil {
		_, _ = repo.UpdateSessionStatus(ctx, supplierID, sessionID, SessionStatusFailed, mustMarshalRollbackErr(err))
		return RollbackResult{}, err
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].SnapshotSeq > snapshots[j].SnapshotSeq
	})

	res, err := repo.ExecuteSessionRollback(ctx, supplierID, sessionID, snapshots, now.UTC())
	if err != nil {
		_, _ = repo.UpdateSessionStatus(ctx, supplierID, sessionID, SessionStatusFailed, mustMarshalRollbackErr(err))
		return RollbackResult{}, err
	}
	return res, nil
}

func mustMarshalRollbackErr(err error) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"rollback_error": err.Error()})
	return b
}
