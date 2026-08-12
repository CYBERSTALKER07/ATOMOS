package ar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ErrInvoicesDisabled rejects credit leave-behind while AR invoicing is off:
// a credit delivery without an AR open item is uncollectible revenue.
var ErrInvoicesDisabled = errors.New("credit leave-behind rejected: AR_INVOICES_ENABLED is off")

// InvoicesEnabled gates AR invoice persistence (SSMR).
func InvoicesEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AR_INVOICES_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

const (
	StatusOpen    = "OPEN"
	StatusPartial = "PARTIAL"
	StatusPaid    = "PAID"
	StatusVoid    = "VOID"
	AgingCurrent  = "CURRENT"
	Aging1to30    = "1_30"
	Aging31to60   = "31_60"
	Aging61to90   = "61_90"
	Aging90Plus   = "90_PLUS"
)

// Invoice is an AR open item created at credit leave.
type Invoice struct {
	InvoiceID       string    `json:"invoice_id"`
	SupplierID      string    `json:"supplier_id"`
	RetailerID      string    `json:"retailer_id"`
	OrderID         string    `json:"order_id"`
	Status          string    `json:"status"`
	PrincipalMinor  int64     `json:"principal_minor"`
	BalanceMinor    int64     `json:"balance_minor"`
	Currency        string    `json:"currency"`
	CreditLeaveAt   time.Time `json:"credit_leave_at"`
	DueAt           time.Time `json:"due_at"`
	TermsDays       int64     `json:"terms_days"`
	GracePeriodDays int64     `json:"grace_period_days"`
	AgingBucket     string    `json:"aging_bucket,omitempty"`
	LastDunnedAt    time.Time `json:"last_dunned_at,omitempty"`
	DunningStep     int64     `json:"dunning_step"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Repository persists AR invoices.
type Repository interface {
	OpenInvoice(ctx context.Context, inv Invoice) error
	GetByOrder(ctx context.Context, orderID string) (Invoice, bool, error)
	GetByID(ctx context.Context, invoiceID string) (Invoice, error)
	ListByRetailer(ctx context.Context, retailerID string, status string, limit int) ([]Invoice, error)
	ListBySupplier(ctx context.Context, supplierID string, status string, limit int) ([]Invoice, error)
	ListOpenForDunning(ctx context.Context, limit int) ([]Invoice, error)
	UpdateDunning(ctx context.Context, invoiceID string, step int64, agingBucket string, lastDunnedAt time.Time, version int64) error
	ApplyPayment(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error
	ApplyCreditNote(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error
	RecomputeAging(ctx context.Context, now time.Time, limit int) (int, error)
}

// Service owns AR open items + aging.
type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() string
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		now:   func() time.Time { return time.Now().UTC() },
		newID: func() string { return fmt.Sprintf("ari_%d", time.Now().UnixNano()) },
	}
}

func (s *Service) SetNow(fn func() time.Time) { s.now = fn }

// OpenFromCreditLeave creates an OPEN invoice with due date from terms (idempotent per order).
func (s *Service) OpenFromCreditLeave(ctx context.Context, supplierID, retailerID, orderID string, amountMinor, termsDays, graceDays int64, creditLeaveAt, dueAt time.Time) (Invoice, error) {
	if !InvoicesEnabled() || amountMinor <= 0 {
		return Invoice{}, nil
	}
	if existing, found, err := s.repo.GetByOrder(ctx, orderID); err != nil {
		return Invoice{}, err
	} else if found {
		return existing, nil
	}
	if termsDays <= 0 {
		termsDays = 30
	}
	if dueAt.IsZero() {
		dueAt = creditLeaveAt.AddDate(0, 0, int(termsDays))
	}
	inv := Invoice{
		InvoiceID:       s.newID(),
		SupplierID:      supplierID,
		RetailerID:      retailerID,
		OrderID:         orderID,
		Status:          StatusOpen,
		PrincipalMinor:  amountMinor,
		BalanceMinor:    amountMinor,
		Currency:        "UZS",
		CreditLeaveAt:   creditLeaveAt,
		DueAt:           dueAt,
		TermsDays:       termsDays,
		GracePeriodDays: graceDays,
		AgingBucket:     AgingCurrent,
		Version:         1,
		CreatedAt:       s.now(),
		UpdatedAt:       s.now(),
	}
	if err := s.repo.OpenInvoice(ctx, inv); err != nil {
		return Invoice{}, err
	}
	return inv, nil
}

func (s *Service) ListRetailerInvoices(ctx context.Context, retailerID, status string, limit int) ([]Invoice, error) {
	return s.repo.ListByRetailer(ctx, retailerID, status, limit)
}

func (s *Service) ListSupplierInvoices(ctx context.Context, supplierID, status string, limit int) ([]Invoice, error) {
	return s.repo.ListBySupplier(ctx, supplierID, status, limit)
}

// RecordPayment pays down an open AR invoice (cash collection against a credit
// delivery, or a card capture applied to the invoice). Idempotent on
// idempotencyKey; returns the updated invoice. This is the production entry
// point that was previously missing — without it credit invoices could never
// be settled and every one marched to CREDIT_HOLD regardless of payment.
func (s *Service) RecordPayment(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) (Invoice, error) {
	if amountMinor <= 0 {
		return Invoice{}, fmt.Errorf("amount_minor must be positive")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return Invoice{}, fmt.Errorf("idempotency_key required")
	}
	if err := s.repo.ApplyPayment(ctx, invoiceID, amountMinor, idempotencyKey); err != nil {
		return Invoice{}, err
	}
	return s.GetByID(ctx, invoiceID)
}

// RecordPaymentForOrder pays down the AR invoice opened for a given order, if
// one exists. No-op when there is no invoice (cash/card orders) — safe to call
// unconditionally from the cash-collection and capture paths.
func (s *Service) RecordPaymentForOrder(ctx context.Context, orderID string, amountMinor int64, idempotencyKey string) error {
	inv, found, err := s.repo.GetByOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if inv.Status == StatusPaid || inv.Status == StatusVoid {
		return nil
	}
	_, err = s.RecordPayment(ctx, inv.InvoiceID, amountMinor, idempotencyKey)
	return err
}

// GetByID loads an invoice by primary key.
func (s *Service) GetByID(ctx context.Context, invoiceID string) (Invoice, error) {
	return s.repo.GetByID(ctx, invoiceID)
}

// GetByOrder loads the invoice opened for an order, if any.
func (s *Service) GetByOrder(ctx context.Context, orderID string) (Invoice, bool, error) {
	return s.repo.GetByOrder(ctx, orderID)
}

func (s *Service) RunAgingPass(ctx context.Context) (int, error) {
	return s.repo.RecomputeAging(ctx, s.now(), 500)
}

// AgingBucketFor computes bucket from due date.
func AgingBucketFor(dueAt, now time.Time) string {
	if !now.After(dueAt) {
		return AgingCurrent
	}
	days := int(now.Sub(dueAt).Hours() / 24)
	switch {
	case days <= 30:
		return Aging1to30
	case days <= 60:
		return Aging31to60
	case days <= 90:
		return Aging61to90
	default:
		return Aging90Plus
	}
}

// MemoryRepository for tests.
type MemoryRepository struct {
	mu         sync.RWMutex
	byID       map[string]Invoice
	byOrder    map[string]string
	ledgerKeys map[string]bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:       map[string]Invoice{},
		byOrder:    map[string]string{},
		ledgerKeys: map[string]bool{},
	}
}

func (r *MemoryRepository) OpenInvoice(_ context.Context, inv Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byOrder[inv.OrderID]; ok {
		_ = id
		return nil
	}
	r.byID[inv.InvoiceID] = inv
	r.byOrder[inv.OrderID] = inv.InvoiceID
	return nil
}

func (r *MemoryRepository) GetByOrder(_ context.Context, orderID string) (Invoice, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byOrder[orderID]
	if !ok {
		return Invoice{}, false, nil
	}
	inv, ok := r.byID[id]
	return inv, ok, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, invoiceID string) (Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inv, ok := r.byID[invoiceID]
	if !ok {
		return Invoice{}, fmt.Errorf("invoice_not_found")
	}
	return inv, nil
}

func (r *MemoryRepository) ListByRetailer(_ context.Context, retailerID, status string, limit int) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.RetailerID != retailerID {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		out = append(out, inv)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) ListBySupplier(_ context.Context, supplierID, status string, limit int) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.SupplierID != supplierID {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		out = append(out, inv)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) ApplyPayment(_ context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ledgerKeys[idempotencyKey] {
		return nil
	}
	inv, ok := r.byID[invoiceID]
	if !ok {
		return fmt.Errorf("invoice_not_found")
	}
	inv.BalanceMinor -= amountMinor
	if inv.BalanceMinor < 0 {
		inv.BalanceMinor = 0
	}
	if inv.BalanceMinor == 0 {
		inv.Status = StatusPaid
	} else {
		inv.Status = StatusPartial
	}
	r.byID[invoiceID] = inv
	r.ledgerKeys[idempotencyKey] = true
	return nil
}

func (r *MemoryRepository) ApplyCreditNote(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return r.ApplyPayment(ctx, invoiceID, amountMinor, idempotencyKey)
}

func (r *MemoryRepository) ListOpenForDunning(_ context.Context, limit int) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 200
	}
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.Status != StatusOpen && inv.Status != StatusPartial {
			continue
		}
		out = append(out, inv)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) UpdateDunning(_ context.Context, invoiceID string, step int64, agingBucket string, lastDunnedAt time.Time, version int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.byID[invoiceID]
	if !ok {
		return fmt.Errorf("invoice_not_found")
	}
	if inv.Version != version {
		return fmt.Errorf("version_conflict")
	}
	inv.DunningStep = step
	inv.AgingBucket = agingBucket
	inv.LastDunnedAt = lastDunnedAt
	inv.Version++
	inv.UpdatedAt = time.Now().UTC()
	r.byID[invoiceID] = inv
	return nil
}

func (r *MemoryRepository) RecomputeAging(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for id, inv := range r.byID {
		if inv.Status != StatusOpen && inv.Status != StatusPartial {
			continue
		}
		inv.AgingBucket = AgingBucketFor(inv.DueAt, now)
		r.byID[id] = inv
		n++
		if n >= limit {
			break
		}
	}
	return n, nil
}

// SpannerRepository persists ArInvoices.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) OpenInvoice(ctx context.Context, inv Invoice) error {
	// Spanner (and especially the emulator) rejects client wall-clock values
	// that are even slightly ahead of server time. Persist audit stamps via
	// commit timestamp; clamp CreditLeaveAt a second into the past so a host
	// clock that leads the emulator cannot fail the money-path open.
	creditLeaveAt := inv.CreditLeaveAt.UTC()
	if skew := time.Now().UTC().Add(-time.Second); creditLeaveAt.After(skew) {
		creditLeaveAt = skew
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, inv.InvoiceID, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventARInvoiceOpened, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			InvoiceID:      inv.InvoiceID,
			SupplierID:     inv.SupplierID,
			RetailerID:     inv.RetailerID,
			OrderID:        inv.OrderID,
			PrincipalMinor: inv.PrincipalMinor,
			DueAt:          inv.DueAt.UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
				"InvoiceId":       inv.InvoiceID,
				"SupplierId":      inv.SupplierID,
				"RetailerId":      inv.RetailerID,
				"OrderId":         inv.OrderID,
				"Status":          inv.Status,
				"PrincipalMinor":  inv.PrincipalMinor,
				"BalanceMinor":    inv.BalanceMinor,
				"Currency":        inv.Currency,
				"CreditLeaveAt":   creditLeaveAt,
				"DueAt":           inv.DueAt,
				"TermsDays":       inv.TermsDays,
				"GracePeriodDays": inv.GracePeriodDays,
				"AgingBucket":     inv.AgingBucket,
				"DunningStep":     inv.DunningStep,
				"Version":         inv.Version,
				"CreatedAt":       spanner.CommitTimestamp,
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("ArLedgerEntries", map[string]any{
				"EntryId":        inv.InvoiceID + ":OPEN",
				"InvoiceId":      inv.InvoiceID,
				"SupplierId":     inv.SupplierID,
				"RetailerId":     inv.RetailerID,
				"EntryType":      "OPEN",
				"AmountMinor":    inv.PrincipalMinor,
				"IdempotencyKey": "open:" + inv.OrderID,
				"RefOrderId":     inv.OrderID,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
}

func (r *SpannerRepository) GetByOrder(ctx context.Context, orderID string) (Invoice, bool, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, CreditLeaveAt, DueAt, TermsDays, GracePeriodDays, AgingBucket, DunningStep, Version, CreatedAt, UpdatedAt FROM ArInvoices WHERE OrderId = @oid`,
		Params: map[string]any{"oid": orderID},
	}
	row, err := r.client.Single().Query(ctx, stmt).Next()
	if err != nil {
		if err == iterator.Done {
			return Invoice{}, false, nil
		}
		return Invoice{}, false, err
	}
	inv, err := scanInvoice(row)
	return inv, err == nil, err
}

func (r *SpannerRepository) GetByID(ctx context.Context, invoiceID string) (Invoice, error) {
	row, err := r.client.Single().ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, []string{
		"InvoiceId", "SupplierId", "RetailerId", "OrderId", "Status", "PrincipalMinor", "BalanceMinor", "Currency", "CreditLeaveAt", "DueAt", "TermsDays", "GracePeriodDays", "AgingBucket", "DunningStep", "Version", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return Invoice{}, err
	}
	return scanInvoice(row)
}

func scanInvoice(row *spanner.Row) (Invoice, error) {
	var inv Invoice
	var aging spanner.NullString
	if err := row.Columns(&inv.InvoiceID, &inv.SupplierID, &inv.RetailerID, &inv.OrderID, &inv.Status,
		&inv.PrincipalMinor, &inv.BalanceMinor, &inv.Currency, &inv.CreditLeaveAt, &inv.DueAt,
		&inv.TermsDays, &inv.GracePeriodDays, &aging, &inv.DunningStep, &inv.Version, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return inv, err
	}
	inv.AgingBucket = aging.StringVal
	return inv, nil
}

func (r *SpannerRepository) ListByRetailer(ctx context.Context, retailerID, status string, limit int) ([]Invoice, error) {
	return r.list(ctx, "RetailerId", retailerID, status, limit)
}

func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Invoice, error) {
	return r.list(ctx, "SupplierId", supplierID, status, limit)
}

func (r *SpannerRepository) list(ctx context.Context, col, id, status string, limit int) ([]Invoice, error) {
	if limit <= 0 {
		limit = 50
	}
	sql := fmt.Sprintf(`SELECT InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, CreditLeaveAt, DueAt, TermsDays, GracePeriodDays, AgingBucket, DunningStep, Version, CreatedAt, UpdatedAt FROM ArInvoices WHERE %s = @id`, col)
	params := map[string]any{"id": id}
	if status != "" {
		sql += " AND Status = @st"
		params["st"] = status
	}
	sql += fmt.Sprintf(" ORDER BY DueAt ASC LIMIT %d", limit)
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]Invoice, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
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
	return out, nil
}

func (r *SpannerRepository) ApplyPayment(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// idempotency
		_, err := txn.ReadRow(ctx, "ArLedgerEntries", spanner.Key{idempotencyKey}, []string{"EntryId"})
		if err == nil {
			return nil
		}
		if spanner.ErrCode(err) != codes.NotFound {
			// try by unique index via query
		}
		row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID},
			[]string{"SupplierId", "RetailerId", "BalanceMinor", "Status", "Version"})
		if err != nil {
			return err
		}
		var sid, rid, status string
		var bal, ver int64
		if err := row.Columns(&sid, &rid, &bal, &status, &ver); err != nil {
			return err
		}
		newBal := bal - amountMinor
		if newBal < 0 {
			newBal = 0
		}
		newStatus := StatusPartial
		if newBal == 0 {
			newStatus = StatusPaid
		}
		entryID := fmt.Sprintf("arl_%d", time.Now().UnixNano())
		buf := outbox.NewSpannerTxnBuffer(txn)
		eventType := events.EventARInvoicePayment
		if newStatus == StatusPaid {
			eventType = events.EventARInvoiceSettled
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, invoiceID, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent:    events.BaseEvent{Type: eventType, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			InvoiceID:    invoiceID,
			SupplierID:   sid,
			RetailerID:   rid,
			AmountMinor:  amountMinor,
			BalanceMinor: newBal,
			Status:       newStatus,
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ArInvoices", map[string]any{
				"InvoiceId":    invoiceID,
				"BalanceMinor": newBal,
				"Status":       newStatus,
				"Version":      ver + 1,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
			spanner.InsertMap("ArLedgerEntries", map[string]any{
				"EntryId":        entryID,
				"InvoiceId":      invoiceID,
				"SupplierId":     sid,
				"RetailerId":     rid,
				"EntryType":      "PAYMENT",
				"AmountMinor":    -amountMinor,
				"IdempotencyKey": idempotencyKey,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
}

func (r *SpannerRepository) ApplyCreditNote(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return r.ApplyPayment(ctx, invoiceID, amountMinor, idempotencyKey)
}

func (r *SpannerRepository) ListOpenForDunning(ctx context.Context, limit int) ([]Invoice, error) {
	if limit <= 0 {
		limit = 200
	}
	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`SELECT InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, CreditLeaveAt, DueAt, TermsDays, GracePeriodDays, AgingBucket, DunningStep, Version, CreatedAt, UpdatedAt
FROM ArInvoices WHERE Status IN ('OPEN','PARTIAL') ORDER BY DueAt ASC LIMIT %d`, limit),
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]Invoice, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
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
	return out, nil
}

func (r *SpannerRepository) UpdateDunning(ctx context.Context, invoiceID string, step int64, agingBucket string, lastDunnedAt time.Time, version int64) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var sid, rid string
		if row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, []string{"SupplierId", "RetailerId"}); err == nil {
			_ = row.Columns(&sid, &rid)
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, invoiceID, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventARInvoiceDunned, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			InvoiceID:    invoiceID,
			SupplierID:   sid,
			RetailerID:   rid,
			DunningStep:  step,
			AgingBucket:  agingBucket,
			LastDunnedAt: lastDunnedAt.UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ArInvoices", map[string]any{
				"InvoiceId":    invoiceID,
				"DunningStep":  step,
				"AgingBucket":  agingBucket,
				"LastDunnedAt": lastDunnedAt,
				"Version":      version + 1,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
}

func (r *SpannerRepository) RecomputeAging(ctx context.Context, now time.Time, limit int) (int, error) {
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf(`SELECT InvoiceId, DueAt, Version FROM ArInvoices WHERE Status IN ('OPEN','PARTIAL') LIMIT %d`, limit),
		Params: map[string]any{},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	mutations := make([]*spanner.Mutation, 0)
	n := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var id string
		var due time.Time
		var ver int64
		if err := row.Columns(&id, &due, &ver); err != nil {
			return 0, err
		}
		mutations = append(mutations, spanner.UpdateMap("ArInvoices", map[string]any{
			"InvoiceId":   id,
			"AgingBucket": AgingBucketFor(due, now),
			"Version":     ver + 1,
			"UpdatedAt":   spanner.CommitTimestamp,
		}))
		n++
	}
	if len(mutations) == 0 {
		return 0, nil
	}
	_, err := r.client.Apply(ctx, mutations)
	return n, err
}
