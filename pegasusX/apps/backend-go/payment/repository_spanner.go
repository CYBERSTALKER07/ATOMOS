package payment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository persists payment aggregates and emitted outbox events
// atomically inside one Spanner ReadWriteTransaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed payment repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
	audits []outbox.AuditEntry
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *spannerTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
	return nil
}

// CreateSession persists one payment session and any outbox events atomically.
func (r *SpannerRepository) CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentSessions", map[string]any{
		"SessionId":   s.SessionID,
		"OrderId":     s.OrderID,
		"SupplierId":  s.SupplierID,
		"RetailerId":  s.RetailerID,
		"Gateway":     s.Gateway,
		"Currency":    s.Currency,
		"AmountMinor": s.AmountMinor,
		"Mode":        s.Mode,
		"Status":      s.Status,
		"CreatedAt":   s.CreatedAt.UTC(),
		"UpdatedAt":   s.UpdatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base, ledgerMutation(buildSessionLedgerEntry(s)))
}

// CreateSessionWithAttempt persists one payment session and first attempt in a
// single transaction with optional outbox writes.
func (r *SpannerRepository) CreateSessionWithAttempt(ctx context.Context, s SessionRecord, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	sessionMutation := spanner.InsertOrUpdateMap("PaymentSessions", map[string]any{
		"SessionId":   s.SessionID,
		"OrderId":     s.OrderID,
		"SupplierId":  s.SupplierID,
		"RetailerId":  s.RetailerID,
		"Gateway":     s.Gateway,
		"Currency":    s.Currency,
		"AmountMinor": s.AmountMinor,
		"Mode":        s.Mode,
		"Status":      s.Status,
		"CreatedAt":   s.CreatedAt.UTC(),
		"UpdatedAt":   s.UpdatedAt.UTC(),
	})

	attemptMutation := spanner.InsertOrUpdateMap("PaymentAttempts", map[string]any{
		"AttemptId":         a.AttemptID,
		"SessionId":         a.SessionID,
		"Gateway":           a.Gateway,
		"ExecutionAction":   nullIfEmpty(a.ExecutionAction),
		"ExecutionMode":     nullIfEmpty(a.ExecutionMode),
		"ProviderReference": nullIfEmpty(a.ProviderReference),
		"Status":            a.Status,
		"CreatedAt":         a.CreatedAt.UTC(),
		"UpdatedAt":         a.UpdatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, sessionMutation, attemptMutation, ledgerMutation(buildSessionLedgerEntry(s)))
}

// SaveAttempt persists one payment attempt and any outbox events atomically.
func (r *SpannerRepository) SaveAttempt(ctx context.Context, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentAttempts", map[string]any{
		"AttemptId":         a.AttemptID,
		"SessionId":         a.SessionID,
		"Gateway":           a.Gateway,
		"ExecutionAction":   nullIfEmpty(a.ExecutionAction),
		"ExecutionMode":     nullIfEmpty(a.ExecutionMode),
		"ProviderReference": nullIfEmpty(a.ProviderReference),
		"Status":            a.Status,
		"CreatedAt":         a.CreatedAt.UTC(),
		"UpdatedAt":         a.UpdatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base)
}

// SaveChargeback persists one chargeback request and optional outbox events.
func (r *SpannerRepository) SaveChargeback(ctx context.Context, c ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentChargebacks", map[string]any{
		"ChargebackId": c.ChargebackID,
		"OrderId":      c.OrderID,
		"RetailerId":   c.RetailerID,
		"Gateway":      c.Gateway,
		"AmountMinor":  c.AmountMinor,
		"Currency":     c.Currency,
		"CreatedAt":    c.CreatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base, ledgerMutation(buildChargebackLedgerEntry(c)))
}

// SaveReversal persists one chargeback reversal and optional outbox events.
func (r *SpannerRepository) SaveReversal(ctx context.Context, rev ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentReversals", map[string]any{
		"ReversalId": rev.ReversalID,
		"SessionId":  rev.SessionID,
		"CreatedAt":  rev.CreatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base, ledgerMutation(buildReversalLedgerEntry(rev)))
}

// SaveWebhook persists one validated webhook and optional outbox events idempotently.
func (r *SpannerRepository) SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PaymentWebhooks", spanner.Key{w.WebhookID}, []string{"WebhookId"})
		if err != nil && spanner.ErrCode(err) != codes.NotFound {
			return fmt.Errorf("read webhook: %w", err)
		}
		if err == nil && row != nil {
			// Webhook already processed, skip emitting events and mutations
			return nil
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		base := spanner.InsertOrUpdateMap("PaymentWebhooks", map[string]any{
			"WebhookId":      w.WebhookID,
			"Gateway":        w.Gateway,
			"TransactionId":  w.TransactionID,
			"SessionId":      nullIfEmpty(w.SessionID),
			"OrderId":        nullIfEmpty(w.OrderID),
			"Status":         w.Status,
			"AmountMinor":    w.AmountMinor,
			"Currency":       w.Currency,
			"SignatureValid": w.SignatureValid,
			"ReceivedAt":     w.ReceivedAt.UTC(),
		})

		mutations := make([]*spanner.Mutation, 0, 2+len(buf.events))
		mutations = append(mutations, base, ledgerMutation(buildWebhookLedgerEntry(w)))
		for _, e := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
}

// FindStuckSessions reads sessions in AWAITING_PAYMENT state older than cutoff.
func (r *SpannerRepository) FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]SessionRecord, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner payment repository: nil client")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, SupplierId, RetailerId, Gateway, Currency, AmountMinor, Mode, Status, CreatedAt, UpdatedAt
		      FROM PaymentSessions
		      WHERE Status = 'AWAITING_PAYMENT' AND UpdatedAt < @cutoff
		      LIMIT @lim`,
		Params: map[string]any{
			"cutoff": cutoff,
			"lim":    int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var sessions []SessionRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var s SessionRecord
		if err := row.Columns(&s.SessionID, &s.OrderID, &s.SupplierID, &s.RetailerID, &s.Gateway, &s.Currency, &s.AmountMinor, &s.Mode, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// GetSession reads a single session by its SessionID.
func (r *SpannerRepository) GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	if r == nil || r.client == nil {
		return SessionRecord{}, false, fmt.Errorf("spanner payment repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, SupplierId, RetailerId, Gateway, Currency, AmountMinor, Mode, Status, CreatedAt, UpdatedAt
		      FROM PaymentSessions
		      WHERE SessionId = @sid`,
		Params: map[string]any{"sid": sessionID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return SessionRecord{}, false, nil
	}
	if err != nil {
		return SessionRecord{}, false, err
	}
	var s SessionRecord
	if err := row.Columns(&s.SessionID, &s.OrderID, &s.SupplierID, &s.RetailerID, &s.Gateway, &s.Currency, &s.AmountMinor, &s.Mode, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return SessionRecord{}, false, err
	}
	return s, true, nil
}

// HasChargebackForOrder checks if a chargeback exists for a given OrderID.
func (r *SpannerRepository) HasChargebackForOrder(ctx context.Context, orderID string) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("spanner payment repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT 1
		      FROM PaymentChargebacks
		      WHERE OrderId = @oid
		      LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetSessionByOrderID retrieves the latest session for a given order ID., if any.
func (r *SpannerRepository) GetSessionByOrderID(ctx context.Context, orderID string) (SessionRecord, bool, error) {
	if r == nil || r.client == nil {
		return SessionRecord{}, false, fmt.Errorf("spanner payment repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, SupplierId, RetailerId, Gateway, Currency, AmountMinor, Mode, Status, CreatedAt, UpdatedAt
		      FROM PaymentSessions
		      WHERE OrderId = @order_id
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"order_id": orderID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return SessionRecord{}, false, nil
	}
	if err != nil {
		return SessionRecord{}, false, err
	}
	var s SessionRecord
	if err := row.Columns(&s.SessionID, &s.OrderID, &s.SupplierID, &s.RetailerID, &s.Gateway, &s.Currency, &s.AmountMinor, &s.Mode, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return SessionRecord{}, false, err
	}
	return s, true, nil
}

// ListLedgerEntries reads bounded payment ledger entries using index-backed
// filters and stale reads for low-contention operational access.
func (r *SpannerRepository) ListLedgerEntries(ctx context.Context, q LedgerQuery) ([]LedgerEntryRecord, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner payment repository: nil client")
	}

	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	where := make([]string, 0, 3)
	params := map[string]any{"limit": int64(limit)}
	if supplierID := strings.TrimSpace(q.SupplierID); supplierID != "" {
		where = append(where, "SupplierId = @supplier_id")
		params["supplier_id"] = supplierID
	}
	if orderID := strings.TrimSpace(q.OrderID); orderID != "" {
		where = append(where, "OrderId = @order_id")
		params["order_id"] = orderID
	}
	if sessionID := strings.TrimSpace(q.SessionID); sessionID != "" {
		where = append(where, "SessionId = @session_id")
		params["session_id"] = sessionID
	}
	if gateway := strings.TrimSpace(q.Gateway); gateway != "" {
		where = append(where, "Gateway = @gateway")
		params["gateway"] = strings.ToUpper(gateway)
	}
	if entryType := strings.TrimSpace(q.EntryType); entryType != "" {
		where = append(where, "EntryType = @entry_type")
		params["entry_type"] = strings.ToUpper(entryType)
	}
	if q.OccurredFrom != nil {
		where = append(where, "OccurredAt >= @occurred_from")
		params["occurred_from"] = q.OccurredFrom.UTC()
	}
	if q.OccurredTo != nil {
		where = append(where, "OccurredAt <= @occurred_to")
		params["occurred_to"] = q.OccurredTo.UTC()
	}

	sql := `SELECT LedgerEntryId, SessionId, OrderId, SupplierId, RetailerId, Gateway, EntryType, AmountMinor, Currency, ReferenceId, Source, OccurredAt, CreatedAt FROM PaymentLedgerEntries`
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " ORDER BY OccurredAt DESC LIMIT @limit"

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	items := make([]LedgerEntryRecord, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("payment ledger query: %w", err)
		}

		var item LedgerEntryRecord
		var sessionID spanner.NullString
		var orderID spanner.NullString
		var supplierID spanner.NullString
		var retailerID spanner.NullString
		var referenceID spanner.NullString
		if err := row.Columns(
			&item.LedgerEntryID,
			&sessionID,
			&orderID,
			&supplierID,
			&retailerID,
			&item.Gateway,
			&item.EntryType,
			&item.AmountMinor,
			&item.Currency,
			&referenceID,
			&item.Source,
			&item.OccurredAt,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("payment ledger scan: %w", err)
		}
		if sessionID.Valid {
			item.SessionID = sessionID.StringVal
		}
		if orderID.Valid {
			item.OrderID = orderID.StringVal
		}
		if supplierID.Valid {
			item.SupplierID = supplierID.StringVal
		}
		if retailerID.Valid {
			item.RetailerID = retailerID.StringVal
		}
		if referenceID.Valid {
			item.ReferenceID = referenceID.StringVal
		}
		items = append(items, item)
	}

	return items, nil
}

// SummarizeLedgerEntries reads grouped settlement authority rows from
// immutable PaymentLedgerEntries using bounded stale reads.
func (r *SpannerRepository) SummarizeLedgerEntries(ctx context.Context, q SettlementAuthorityQuery) ([]SettlementAuthorityRow, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner payment repository: nil client")
	}

	groupLimit := q.GroupLimit
	if groupLimit <= 0 || groupLimit > 1000 {
		groupLimit = 200
	}

	where := make([]string, 0, 5)
	params := map[string]any{"group_limit": int64(groupLimit)}
	if supplierID := strings.TrimSpace(q.SupplierID); supplierID != "" {
		where = append(where, "SupplierId = @supplier_id")
		params["supplier_id"] = supplierID
	}
	if gateway := strings.TrimSpace(q.Gateway); gateway != "" {
		where = append(where, "Gateway = @gateway")
		params["gateway"] = strings.ToUpper(gateway)
	}
	if entryType := strings.TrimSpace(q.EntryType); entryType != "" {
		where = append(where, "EntryType = @entry_type")
		params["entry_type"] = strings.ToUpper(entryType)
	}
	if q.OccurredFrom != nil {
		where = append(where, "OccurredAt >= @occurred_from")
		params["occurred_from"] = q.OccurredFrom.UTC()
	}
	if q.OccurredTo != nil {
		where = append(where, "OccurredAt <= @occurred_to")
		params["occurred_to"] = q.OccurredTo.UTC()
	}

	sql := `SELECT Gateway, EntryType, Currency, COUNT(1) AS EntryCount, SUM(AmountMinor) AS AmountMinorTotal, MIN(OccurredAt) AS FirstOccurredAt, MAX(OccurredAt) AS LastOccurredAt FROM PaymentLedgerEntries`
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " GROUP BY Gateway, EntryType, Currency ORDER BY Gateway ASC, EntryType ASC, Currency ASC LIMIT @group_limit"

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	items := make([]SettlementAuthorityRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("payment settlement summary query: %w", err)
		}

		var item SettlementAuthorityRow
		if err := row.Columns(
			&item.Gateway,
			&item.EntryType,
			&item.Currency,
			&item.EntryCount,
			&item.AmountMinorTotal,
			&item.FirstOccurredAt,
			&item.LastOccurredAt,
		); err != nil {
			return nil, fmt.Errorf("payment settlement summary scan: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *SpannerRepository) writeWithOutbox(ctx context.Context, emit func(outbox.TxnBuffer) error, bases ...*spanner.Mutation) error {
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := make([]*spanner.Mutation, 0, len(bases)+len(buf.events))
		mutations = append(mutations, bases...)
		for _, e := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("payment write transaction: %w", err)
	}
	return nil
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func buildSessionLedgerEntry(s SessionRecord) LedgerEntryRecord {
	occurredAt := s.UpdatedAt
	if occurredAt.IsZero() {
		occurredAt = s.CreatedAt
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return LedgerEntryRecord{
		LedgerEntryID: "pledger_session_" + s.SessionID,
		SessionID:     s.SessionID,
		OrderID:       s.OrderID,
		SupplierID:    s.SupplierID,
		RetailerID:    s.RetailerID,
		Gateway:       normalizedUpper(s.Gateway, "UNKNOWN"),
		EntryType:     "SESSION_" + normalizedUpper(s.Status, "UNKNOWN"),
		AmountMinor:   s.AmountMinor,
		Currency:      normalizedUpper(s.Currency, "UZS"),
		ReferenceID:   s.SessionID,
		Source:        "payment.session",
		OccurredAt:    occurredAt,
		CreatedAt:     occurredAt,
	}
}

func buildChargebackLedgerEntry(c ChargebackRecord) LedgerEntryRecord {
	occurredAt := c.CreatedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	source := strings.TrimSpace(c.Source)
	if source == "" {
		source = "payment.chargeback"
	}
	return LedgerEntryRecord{
		// Deterministic ledger id (matches chargeback id) so InsertOrUpdate is idempotent.
		LedgerEntryID: "pledger_chargeback_" + c.ChargebackID,
		OrderID:       c.OrderID,
		SupplierID:    c.SupplierID,
		RetailerID:    c.RetailerID,
		Gateway:       normalizedUpper(c.Gateway, "UNKNOWN"),
		EntryType:     "CHARGEBACK_RECORDED",
		AmountMinor:   c.AmountMinor,
		Currency:      normalizedUpper(c.Currency, "UZS"),
		ReferenceID:   c.ChargebackID,
		Source:        source,
		OccurredAt:    occurredAt,
		CreatedAt:     occurredAt,
	}
}

func buildReversalLedgerEntry(rev ReversalRecord) LedgerEntryRecord {
	occurredAt := rev.CreatedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return LedgerEntryRecord{
		LedgerEntryID: "pledger_reversal_" + rev.ReversalID,
		SessionID:     rev.SessionID,
		SupplierID:    rev.SupplierID,
		Gateway:       normalizedUpper(rev.Gateway, "UNKNOWN"),
		EntryType:     "CHARGEBACK_REVERSAL_RECORDED",
		AmountMinor:   rev.AmountMinor,
		Currency:      normalizedUpper(rev.Currency, "UZS"),
		ReferenceID:   rev.ReversalID,
		Source:        "payment.chargeback_reversal",
		OccurredAt:    occurredAt,
		CreatedAt:     occurredAt,
	}
}

func buildWebhookLedgerEntry(w WebhookRecord) LedgerEntryRecord {
	occurredAt := w.ReceivedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return LedgerEntryRecord{
		LedgerEntryID: "pledger_webhook_" + w.WebhookID,
		SessionID:     w.SessionID,
		OrderID:       w.OrderID,
		SupplierID:    w.SupplierID,
		RetailerID:    w.RetailerID,
		Gateway:       normalizedUpper(w.Gateway, "UNKNOWN"),
		EntryType:     "WEBHOOK_" + normalizedUpper(w.Status, "UNKNOWN"),
		AmountMinor:   w.AmountMinor,
		Currency:      normalizedUpper(w.Currency, "UZS"),
		ReferenceID:   w.TransactionID,
		Source:        "payment.webhook",
		OccurredAt:    occurredAt,
		CreatedAt:     occurredAt,
	}
}

func ledgerMutation(entry LedgerEntryRecord) *spanner.Mutation {
	occurredAt := entry.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = occurredAt
	}

	return spanner.InsertOrUpdateMap("PaymentLedgerEntries", map[string]any{
		"LedgerEntryId": entry.LedgerEntryID,
		"SessionId":     nullIfEmpty(entry.SessionID),
		"OrderId":       nullIfEmpty(entry.OrderID),
		"SupplierId":    nullIfEmpty(entry.SupplierID),
		"RetailerId":    nullIfEmpty(entry.RetailerID),
		"Gateway":       normalizedUpper(entry.Gateway, "UNKNOWN"),
		"EntryType":     normalizedUpper(entry.EntryType, "UNKNOWN"),
		"AmountMinor":   entry.AmountMinor,
		"Currency":      normalizedUpper(entry.Currency, "UZS"),
		"ReferenceId":   nullIfEmpty(entry.ReferenceID),
		"Source":        strings.TrimSpace(entry.Source),
		"OccurredAt":    occurredAt.UTC(),
		"CreatedAt":     createdAt.UTC(),
	})
}

func normalizedUpper(v string, fallback string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return fallback
	}
	return v
}
