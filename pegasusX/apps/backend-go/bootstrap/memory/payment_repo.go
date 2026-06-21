package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// ── Scaffold in-memory payment repository ─────────────────────────────────
// Used when Spanner is unavailable in local/fallback environments.

type inMemoryPaymentRepo struct {
	mu             sync.RWMutex
	sessions       map[string]payment.SessionRecord
	attempts       map[string]payment.PaymentAttemptRecord
	chargebacks    map[string]payment.ChargebackRecord
	reversals      map[string]payment.ReversalRecord
	webhooks       map[string]payment.WebhookRecord
	ledgerEntries  map[string]payment.LedgerEntryRecord
	outboxAppender OutboxAppender
}

func NewPaymentRepo(outboxAppender OutboxAppender) *inMemoryPaymentRepo {
	return &inMemoryPaymentRepo{
		sessions:       make(map[string]payment.SessionRecord),
		attempts:       make(map[string]payment.PaymentAttemptRecord),
		chargebacks:    make(map[string]payment.ChargebackRecord),
		reversals:      make(map[string]payment.ReversalRecord),
		webhooks:       make(map[string]payment.WebhookRecord),
		ledgerEntries:  make(map[string]payment.LedgerEntryRecord),
		outboxAppender: outboxAppender,
	}
}

func (r *inMemoryPaymentRepo) CreateSession(ctx context.Context, s payment.SessionRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.sessions[s.SessionID] = s
	r.ledgerEntries["pledger_session_"+s.SessionID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_session_" + s.SessionID,
		SessionID:     s.SessionID,
		OrderID:       s.OrderID,
		SupplierID:    s.SupplierID,
		RetailerID:    s.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(s.Gateway)),
		EntryType:     "SESSION_" + strings.ToUpper(strings.TrimSpace(s.Status)),
		AmountMinor:   s.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(s.Currency)),
		ReferenceID:   s.SessionID,
		Source:        "payment.session",
		OccurredAt:    s.UpdatedAt,
		CreatedAt:     s.UpdatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) CreateSessionWithAttempt(ctx context.Context, s payment.SessionRecord, a payment.PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.sessions[s.SessionID] = s
	r.attempts[a.AttemptID] = a
	r.ledgerEntries["pledger_session_"+s.SessionID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_session_" + s.SessionID,
		SessionID:     s.SessionID,
		OrderID:       s.OrderID,
		SupplierID:    s.SupplierID,
		RetailerID:    s.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(s.Gateway)),
		EntryType:     "SESSION_" + strings.ToUpper(strings.TrimSpace(s.Status)),
		AmountMinor:   s.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(s.Currency)),
		ReferenceID:   s.SessionID,
		Source:        "payment.session",
		OccurredAt:    s.UpdatedAt,
		CreatedAt:     s.UpdatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveAttempt(ctx context.Context, a payment.PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.attempts[a.AttemptID] = a
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveChargeback(ctx context.Context, c payment.ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.chargebacks[c.ChargebackID] = c
	r.ledgerEntries["pledger_chargeback_"+c.ChargebackID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_chargeback_" + c.ChargebackID,
		OrderID:       c.OrderID,
		SupplierID:    c.SupplierID,
		RetailerID:    c.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(c.Gateway)),
		EntryType:     "CHARGEBACK_RECORDED",
		AmountMinor:   c.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(c.Currency)),
		ReferenceID:   c.ChargebackID,
		Source:        "payment.chargeback",
		OccurredAt:    c.CreatedAt,
		CreatedAt:     c.CreatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveReversal(ctx context.Context, rev payment.ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.reversals[rev.ReversalID] = rev
	r.ledgerEntries["pledger_reversal_"+rev.ReversalID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_reversal_" + rev.ReversalID,
		SessionID:     rev.SessionID,
		SupplierID:    rev.SupplierID,
		Gateway:       strings.ToUpper(strings.TrimSpace(rev.Gateway)),
		EntryType:     "CHARGEBACK_REVERSAL_RECORDED",
		AmountMinor:   rev.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(rev.Currency)),
		ReferenceID:   rev.ReversalID,
		Source:        "payment.chargeback_reversal",
		OccurredAt:    rev.CreatedAt,
		CreatedAt:     rev.CreatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveWebhook(ctx context.Context, w payment.WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.webhooks[w.WebhookID] = w
	r.ledgerEntries["pledger_webhook_"+w.WebhookID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_webhook_" + w.WebhookID,
		SessionID:     w.SessionID,
		OrderID:       w.OrderID,
		SupplierID:    w.SupplierID,
		RetailerID:    w.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(w.Gateway)),
		EntryType:     "WEBHOOK_" + strings.ToUpper(strings.TrimSpace(w.Status)),
		AmountMinor:   w.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(w.Currency)),
		ReferenceID:   w.TransactionID,
		Source:        "payment.webhook",
		OccurredAt:    w.ReceivedAt,
		CreatedAt:     w.ReceivedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) ListLedgerEntries(_ context.Context, q payment.LedgerQuery) ([]payment.LedgerEntryRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	supplierID := strings.TrimSpace(q.SupplierID)
	orderID := strings.TrimSpace(q.OrderID)
	sessionID := strings.TrimSpace(q.SessionID)
	gateway := strings.ToUpper(strings.TrimSpace(q.Gateway))
	entryType := strings.ToUpper(strings.TrimSpace(q.EntryType))
	items := make([]payment.LedgerEntryRecord, 0, len(r.ledgerEntries))
	for _, entry := range r.ledgerEntries {
		if supplierID != "" && entry.SupplierID != supplierID {
			continue
		}
		if orderID != "" && entry.OrderID != orderID {
			continue
		}
		if sessionID != "" && entry.SessionID != sessionID {
			continue
		}
		if gateway != "" && strings.ToUpper(strings.TrimSpace(entry.Gateway)) != gateway {
			continue
		}
		if entryType != "" && strings.ToUpper(strings.TrimSpace(entry.EntryType)) != entryType {
			continue
		}
		if q.OccurredFrom != nil && entry.OccurredAt.Before(q.OccurredFrom.UTC()) {
			continue
		}
		if q.OccurredTo != nil && entry.OccurredAt.After(q.OccurredTo.UTC()) {
			continue
		}
		items = append(items, entry)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].OccurredAt.Equal(items[j].OccurredAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemoryPaymentRepo) SummarizeLedgerEntries(_ context.Context, q payment.SettlementAuthorityQuery) ([]payment.SettlementAuthorityRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groupLimit := q.GroupLimit
	if groupLimit <= 0 || groupLimit > 1000 {
		groupLimit = 200
	}

	supplierID := strings.TrimSpace(q.SupplierID)
	gateway := strings.ToUpper(strings.TrimSpace(q.Gateway))
	entryType := strings.ToUpper(strings.TrimSpace(q.EntryType))
	groups := make(map[string]payment.SettlementAuthorityRow)

	for _, entry := range r.ledgerEntries {
		if supplierID != "" && entry.SupplierID != supplierID {
			continue
		}
		if gateway != "" && strings.ToUpper(strings.TrimSpace(entry.Gateway)) != gateway {
			continue
		}
		if entryType != "" && strings.ToUpper(strings.TrimSpace(entry.EntryType)) != entryType {
			continue
		}
		if q.OccurredFrom != nil && entry.OccurredAt.Before(q.OccurredFrom.UTC()) {
			continue
		}
		if q.OccurredTo != nil && entry.OccurredAt.After(q.OccurredTo.UTC()) {
			continue
		}

		key := strings.Join([]string{
			strings.ToUpper(strings.TrimSpace(entry.Gateway)),
			strings.ToUpper(strings.TrimSpace(entry.EntryType)),
			strings.ToUpper(strings.TrimSpace(entry.Currency)),
		}, "|")
		row := groups[key]
		if row.Gateway == "" {
			row = payment.SettlementAuthorityRow{
				Gateway:          strings.ToUpper(strings.TrimSpace(entry.Gateway)),
				EntryType:        strings.ToUpper(strings.TrimSpace(entry.EntryType)),
				Currency:         strings.ToUpper(strings.TrimSpace(entry.Currency)),
				FirstOccurredAt:  entry.OccurredAt,
				LastOccurredAt:   entry.OccurredAt,
				EntryCount:       0,
				AmountMinorTotal: 0,
			}
		}
		row.EntryCount++
		row.AmountMinorTotal += entry.AmountMinor
		if entry.OccurredAt.Before(row.FirstOccurredAt) {
			row.FirstOccurredAt = entry.OccurredAt
		}
		if entry.OccurredAt.After(row.LastOccurredAt) {
			row.LastOccurredAt = entry.OccurredAt
		}
		groups[key] = row
	}

	rows := make([]payment.SettlementAuthorityRow, 0, len(groups))
	for _, row := range groups {
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Gateway == rows[j].Gateway {
			if rows[i].EntryType == rows[j].EntryType {
				return rows[i].Currency < rows[j].Currency
			}
			return rows[i].EntryType < rows[j].EntryType
		}
		return rows[i].Gateway < rows[j].Gateway
	})

	if len(rows) > groupLimit {
		rows = rows[:groupLimit]
	}
	return rows, nil
}
