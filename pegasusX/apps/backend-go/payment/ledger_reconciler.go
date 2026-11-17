package payment

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// Default table names in Cloud Spanner.
const (
	DefaultSpannerLedgerTable = "PaymentLedgerEntries"
	AltSpannerLedgerTable     = "FinancialLedger"
)

// CapturedPaymentLegSummary summarizes a captured order payment leg.
type CapturedPaymentLegSummary struct {
	OrderID        string    `json:"order_id"`
	LegID          string    `json:"leg_id"`
	Method         string    `json:"method"`
	AmountMinor    int64     `json:"amount_minor"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotency_key"`
	ProviderRef    string    `json:"provider_ref,omitempty"`
	CapturedAt     time.Time `json:"captured_at"`
}

// AmountMismatch records a discrepancy between captured payment legs and financial ledger entries.
type AmountMismatch struct {
	OrderID          string `json:"order_id"`
	LegTotalMinor    int64  `json:"leg_total_minor"`
	LedgerTotalMinor int64  `json:"ledger_total_minor"`
	DifferenceMinor  int64  `json:"difference_minor"`
}

// LedgerReconciliationReport is the result of ReconcileWindow.
type LedgerReconciliationReport struct {
	Since                 time.Time        `json:"since"`
	ReconciledAt          time.Time        `json:"reconciled_at"`
	TotalCapturedLegs     int              `json:"total_captured_legs"`
	TotalLedgerEntries    int              `json:"total_ledger_entries"`
	MatchedOrders         int              `json:"matched_orders"`
	MissingLedgerOrderIDs []string         `json:"missing_ledger_order_ids"`
	AmountMismatches      []AmountMismatch `json:"amount_mismatches"`
	UnbalancedJournals    []string         `json:"unbalanced_journals,omitempty"`
	Anomalies             int              `json:"anomalies"`
}

// LedgerReconcileStore defines the persistence interface required for financial ledger reconciliation.
type LedgerReconcileStore interface {
	ListCapturedPaymentLegsSince(ctx context.Context, since time.Time, limit int) ([]CapturedPaymentLegSummary, error)
	ListLedgerEntriesSince(ctx context.Context, since time.Time, limit int) ([]LedgerEntryRecord, error)
}

// SpannerLedgerReconcileStore implements LedgerReconcileStore using Cloud Spanner.
type SpannerLedgerReconcileStore struct {
	client      *spanner.Client
	ledgerTable string
}

// NewSpannerLedgerReconcileStore creates a new Spanner-backed store.
func NewSpannerLedgerReconcileStore(client *spanner.Client) *SpannerLedgerReconcileStore {
	return &SpannerLedgerReconcileStore{
		client:      client,
		ledgerTable: DefaultSpannerLedgerTable,
	}
}

// WithLedgerTable overrides the ledger table name if a custom table/view is used.
func (s *SpannerLedgerReconcileStore) WithLedgerTable(table string) *SpannerLedgerReconcileStore {
	if strings.TrimSpace(table) != "" {
		s.ledgerTable = strings.TrimSpace(table)
	}
	return s
}

// ListCapturedPaymentLegsSince queries OrderPaymentLegs for captured legs since a timestamp.
func (s *SpannerLedgerReconcileStore) ListCapturedPaymentLegsSince(ctx context.Context, since time.Time, limit int) ([]CapturedPaymentLegSummary, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5000
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, LegId, Method, AmountMinor, Status, IdempotencyKey, ProviderRef, CapturedAt
		      FROM OrderPaymentLegs
		      WHERE Status = 'CAPTURED' AND CapturedAt >= @since
		      ORDER BY CapturedAt ASC
		      LIMIT @limit`,
		Params: map[string]any{
			"since": since,
			"limit": int64(limit),
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var summaries []CapturedPaymentLegSummary
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("spanner list captured payment legs: %w", err)
		}
		var leg CapturedPaymentLegSummary
		var providerRef spanner.NullString
		var capturedAt spanner.NullTime
		if err := row.Columns(&leg.OrderID, &leg.LegID, &leg.Method, &leg.AmountMinor, &leg.Status, &leg.IdempotencyKey, &providerRef, &capturedAt); err != nil {
			return nil, fmt.Errorf("spanner read captured payment leg: %w", err)
		}
		if providerRef.Valid {
			leg.ProviderRef = providerRef.StringVal
		}
		if capturedAt.Valid {
			leg.CapturedAt = capturedAt.Time
		}
		summaries = append(summaries, leg)
	}
	return summaries, nil
}

// ListLedgerEntriesSince queries the financial ledger (PaymentLedgerEntries / FinancialLedger) since a timestamp.
func (s *SpannerLedgerReconcileStore) ListLedgerEntriesSince(ctx context.Context, since time.Time, limit int) ([]LedgerEntryRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10000
	}

	records, err := s.queryLedgerTable(ctx, s.ledgerTable, since, limit)
	if err != nil && spanner.ErrCode(err) == codes.NotFound && s.ledgerTable == DefaultSpannerLedgerTable {
		// Fallback to alternative table if primary does not exist
		return s.queryLedgerTable(ctx, AltSpannerLedgerTable, since, limit)
	}
	return records, err
}

func (s *SpannerLedgerReconcileStore) queryLedgerTable(ctx context.Context, table string, since time.Time, limit int) ([]LedgerEntryRecord, error) {
	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`SELECT LedgerEntryId, SessionId, OrderId, SupplierId, RetailerId, Gateway, EntryType, AmountMinor, Currency, ReferenceId, Source, OccurredAt, CreatedAt
		      FROM %s
		      WHERE OccurredAt >= @since
		      ORDER BY OccurredAt ASC
		      LIMIT @limit`, table),
		Params: map[string]any{
			"since": since,
			"limit": int64(limit),
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var records []LedgerEntryRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("spanner list ledger entries from %s: %w", table, err)
		}
		var r LedgerEntryRecord
		var sessionID, orderID, supplierID, retailerID, referenceID spanner.NullString
		if err := row.Columns(
			&r.LedgerEntryID,
			&sessionID,
			&orderID,
			&supplierID,
			&retailerID,
			&r.Gateway,
			&r.EntryType,
			&r.AmountMinor,
			&r.Currency,
			&referenceID,
			&r.Source,
			&r.OccurredAt,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("spanner read ledger entry: %w", err)
		}
		if sessionID.Valid {
			r.SessionID = sessionID.StringVal
		}
		if orderID.Valid {
			r.OrderID = orderID.StringVal
		}
		if supplierID.Valid {
			r.SupplierID = supplierID.StringVal
		}
		if retailerID.Valid {
			r.RetailerID = retailerID.StringVal
		}
		if referenceID.Valid {
			r.ReferenceID = referenceID.StringVal
		}
		records = append(records, r)
	}
	return records, nil
}

// LedgerReconciler verifies consistency between captured payment legs and financial ledger entries.
type LedgerReconciler struct {
	store          LedgerReconcileStore
	log            *slog.Logger
	queryLimit     int
	mu             sync.Mutex
	pendingBatches [][]JournalEntry
}

// NewLedgerReconciler creates a LedgerReconciler with the specified store and logger.
func NewLedgerReconciler(store LedgerReconcileStore, log *slog.Logger) *LedgerReconciler {
	if log == nil {
		log = slog.Default()
	}
	return &LedgerReconciler{
		store:      store,
		log:        log,
		queryLimit: 10000,
	}
}

// NewSpannerLedgerReconciler creates a LedgerReconciler backed by Cloud Spanner.
func NewSpannerLedgerReconciler(client *spanner.Client, log *slog.Logger) *LedgerReconciler {
	return NewLedgerReconciler(NewSpannerLedgerReconcileStore(client), log)
}

// AddPendingJournalBatches queues journal entries to be validated during the next ReconcileWindow.
func (r *LedgerReconciler) AddPendingJournalBatches(batches ...[]JournalEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pendingBatches = append(r.pendingBatches, batches...)
}

// ValidateJournalBatch validates that every JournalEntry in the batch satisfies the double-entry invariant.
func ValidateJournalBatch(entries []JournalEntry) ([]string, error) {
	var invalidIDs []string
	for _, je := range entries {
		if err := je.Validate(); err != nil {
			id := strings.TrimSpace(je.EntryID)
			if id == "" {
				id = strings.TrimSpace(je.OrderID)
			}
			if id == "" {
				id = strings.TrimSpace(je.TransactionID)
			}
			if id == "" {
				id = "unspecified_entry"
			}
			invalidIDs = append(invalidIDs, fmt.Sprintf("%s: %v", id, err))
		}
	}
	if len(invalidIDs) > 0 {
		return invalidIDs, fmt.Errorf("found %d unbalanced/invalid journal entries: %s", len(invalidIDs), strings.Join(invalidIDs, "; "))
	}
	return nil, nil
}

// ValidateJournalBatch checks a batch of JournalEntries on the reconciler.
func (r *LedgerReconciler) ValidateJournalBatch(entries []JournalEntry) ([]string, error) {
	return ValidateJournalBatch(entries)
}

// ReconcileWindow checks captured payment legs vs ledger entries occurred since the given time.
func (r *LedgerReconciler) ReconcileWindow(ctx context.Context, since time.Time) (*LedgerReconciliationReport, error) {
	report := &LedgerReconciliationReport{
		Since:                 since,
		ReconciledAt:          time.Now().UTC(),
		MissingLedgerOrderIDs: make([]string, 0),
		AmountMismatches:      make([]AmountMismatch, 0),
		UnbalancedJournals:    make([]string, 0),
	}

	// 1. Validate any queued JournalEntry batches
	r.mu.Lock()
	queued := r.pendingBatches
	r.pendingBatches = nil
	r.mu.Unlock()

	for _, batch := range queued {
		if invalid, err := ValidateJournalBatch(batch); err != nil {
			report.UnbalancedJournals = append(report.UnbalancedJournals, invalid...)
		}
	}

	if r.store == nil {
		report.Anomalies = len(report.UnbalancedJournals)
		return report, nil
	}

	// 2. Fetch captured payment legs and ledger entries
	legs, err := r.store.ListCapturedPaymentLegsSince(ctx, since, r.queryLimit)
	if err != nil {
		return nil, fmt.Errorf("reconciler list captured payment legs: %w", err)
	}

	ledgerEntries, err := r.store.ListLedgerEntriesSince(ctx, since, r.queryLimit)
	if err != nil {
		return nil, fmt.Errorf("reconciler list ledger entries: %w", err)
	}

	report.TotalLedgerEntries = len(ledgerEntries)

	// 3. Index ledger entries by OrderId and ReferenceId
	ledgerByOrder := make(map[string][]LedgerEntryRecord)
	for _, entry := range ledgerEntries {
		oid := strings.TrimSpace(entry.OrderID)
		if oid == "" && entry.ReferenceID != "" {
			ref := strings.TrimSpace(entry.ReferenceID)
			switch {
			case strings.HasPrefix(ref, "jentry_order_"):
				oid = strings.TrimPrefix(ref, "jentry_order_")
			case strings.HasPrefix(ref, "ord-") || strings.HasPrefix(ref, "ord_"):
				oid = ref
			}
		}
		if oid != "" {
			ledgerByOrder[oid] = append(ledgerByOrder[oid], entry)
		}
	}

	// 4. Group captured payment legs by OrderId (excluding REFUND legs)
	legsByOrder := make(map[string][]CapturedPaymentLegSummary)
	for _, leg := range legs {
		if strings.EqualFold(strings.TrimSpace(leg.Method), "REFUND") {
			continue
		}
		oid := strings.TrimSpace(leg.OrderID)
		if oid == "" {
			continue
		}
		legsByOrder[oid] = append(legsByOrder[oid], leg)
		report.TotalCapturedLegs++
	}

	// 5. Compare legs against ledger entries
	for oid, legList := range legsByOrder {
		entries, exists := ledgerByOrder[oid]
		if !exists || len(entries) == 0 {
			report.MissingLedgerOrderIDs = append(report.MissingLedgerOrderIDs, oid)
			continue
		}

		var legTotal int64
		for _, l := range legList {
			legTotal += l.AmountMinor
		}

		// Calculate ledger total
		var ledgerTotal int64
		hasGLDebit := false
		hasGLCredit := false
		var glDebitTotal int64

		for _, entry := range entries {
			if strings.HasPrefix(entry.EntryType, "GL_DEBIT") {
				hasGLDebit = true
				glDebitTotal += entry.AmountMinor
			} else if strings.HasPrefix(entry.EntryType, "GL_CREDIT") {
				hasGLCredit = true
			}
			if entry.AmountMinor > 0 {
				ledgerTotal += entry.AmountMinor
			}
		}

		// In a balanced double-entry General Ledger batch (both GL_DEBIT and GL_CREDIT present),
		// each transaction is posted as debit + credit; the economic transaction value is sum(GL_DEBITs).
		if hasGLDebit && hasGLCredit && glDebitTotal > 0 {
			ledgerTotal = glDebitTotal
		}

		if legTotal != ledgerTotal {
			report.AmountMismatches = append(report.AmountMismatches, AmountMismatch{
				OrderID:          oid,
				LegTotalMinor:    legTotal,
				LedgerTotalMinor: ledgerTotal,
				DifferenceMinor:  legTotal - ledgerTotal,
			})
		} else {
			report.MatchedOrders++
		}
	}

	// Sort report collections deterministically
	sort.Strings(report.MissingLedgerOrderIDs)
	sort.Slice(report.AmountMismatches, func(i, j int) bool {
		return report.AmountMismatches[i].OrderID < report.AmountMismatches[j].OrderID
	})
	sort.Strings(report.UnbalancedJournals)

	report.Anomalies = len(report.MissingLedgerOrderIDs) + len(report.AmountMismatches) + len(report.UnbalancedJournals)
	return report, nil
}

// StartLedgerReconciler runs ReconcileWindow periodically until the context is cancelled.
func StartLedgerReconciler(ctx context.Context, reconciler *LedgerReconciler, interval time.Duration) {
	if reconciler == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial reconciliation sweep on worker startup
	initSince := time.Now().UTC().Add(-24 * time.Hour)
	if report, err := reconciler.ReconcileWindow(ctx, initSince); err != nil {
		reconciler.log.Warn("initial ledger reconciliation sweep failed", "err", err)
	} else if report.Anomalies > 0 {
		reconciler.log.Warn("initial ledger reconciliation detected anomalies",
			"anomalies", report.Anomalies,
			"missing_ledger_orders", len(report.MissingLedgerOrderIDs),
			"amount_mismatches", len(report.AmountMismatches),
			"unbalanced_journals", len(report.UnbalancedJournals),
		)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Lookback window covers twice the interval to handle any transaction commit latency
			lookback := 2 * interval
			if lookback < 15*time.Minute {
				lookback = 15 * time.Minute
			}
			since := time.Now().UTC().Add(-lookback)
			if report, err := reconciler.ReconcileWindow(ctx, since); err != nil {
				reconciler.log.Warn("periodic ledger reconciliation failed", "err", err)
			} else if report.Anomalies > 0 {
				reconciler.log.Warn("ledger reconciliation detected anomalies",
					"anomalies", report.Anomalies,
					"missing_ledger_orders", len(report.MissingLedgerOrderIDs),
					"amount_mismatches", len(report.AmountMismatches),
					"unbalanced_journals", len(report.UnbalancedJournals),
				)
			}
		}
	}
}
