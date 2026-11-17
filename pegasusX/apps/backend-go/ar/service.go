package ar

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// ErrInvoicesDisabled rejects credit leave-behind while AR invoicing is off:
// a credit delivery without an AR open item is uncollectible revenue.
var ErrInvoicesDisabled = errors.New("credit leave-behind rejected: AR_INVOICES_ENABLED is off")

// ErrForbidden indicates the actor lacks tenant ownership for the AR operation.
var ErrForbidden = errors.New("ar: forbidden")

// ErrInvalidCursor reports a malformed AR pagination cursor.
var ErrInvalidCursor = errors.New("ar: invalid cursor")

// InvoicePager is an optional Repository extension supporting SQL keyset cursor pagination.
type InvoicePager interface {
	ListByRetailerPage(ctx context.Context, retailerID, status string, limit int, cursor string) ([]Invoice, string, error)
	ListBySupplierPage(ctx context.Context, supplierID, status string, limit int, cursor string) ([]Invoice, string, error)
	ListOpenForDunningPage(ctx context.Context, limit int, cursor string) ([]Invoice, string, error)
}

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

	InvoiceSourceLive      = "LIVE"
	InvoiceSourceMigration = "MIGRATION"
)

// Invoice is an AR open item created at credit leave or imported via migration.
type Invoice struct {
	InvoiceID       string     `json:"invoice_id"`
	SupplierID      string     `json:"supplier_id"`
	RetailerID      string     `json:"retailer_id"`
	OrderID         string     `json:"order_id"`
	Status          string     `json:"status"`
	PrincipalMinor  int64      `json:"principal_minor"`
	BalanceMinor    int64      `json:"balance_minor"`
	Currency        string     `json:"currency"`
	CreditLeaveAt   time.Time  `json:"credit_leave_at"`
	DueAt           time.Time  `json:"due_at"`
	TermsDays       int64      `json:"terms_days"`
	GracePeriodDays int64      `json:"grace_period_days"`
	AgingBucket     string     `json:"aging_bucket,omitempty"`
	LastDunnedAt    time.Time  `json:"last_dunned_at,omitempty"`
	DunningStep     int64      `json:"dunning_step"`
	Source          string     `json:"source,omitempty"`
	ExternalID      string     `json:"external_id,omitempty"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
	DisputedAt      *time.Time `json:"disputed_at,omitempty"`
	DisputeReason   string     `json:"dispute_reason,omitempty"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// IsMigrationDunningFrozen returns true when a migrated opening balance has not been
// acknowledged by the retailer or is actively disputed.
func (inv Invoice) IsMigrationDunningFrozen() bool {
	if !strings.EqualFold(strings.TrimSpace(inv.Source), InvoiceSourceMigration) {
		return false
	}
	return inv.AcknowledgedAt == nil || inv.DisputedAt != nil
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
	VoidInvoice(ctx context.Context, invoiceID string, idempotencyKey string) error
	RecomputeAging(ctx context.Context, now time.Time, limit int) (int, error)
}

// CreditClearerFunc clears order credit reservation balance inside a Spanner RW txn (or sequentially when txn == nil).
type CreditClearerFunc func(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error

// DisputeLookupFunc returns summed OPEN / UNDER_REVIEW claim amounts keyed by OrderId.
type DisputeLookupFunc func(ctx context.Context, orderIDs []string) (map[string]int64, error)

// Service owns AR open items + aging.
type Service struct {
	repo          Repository
	now           func() time.Time
	newID         func() string
	creditClearer CreditClearerFunc
	disputeLookup DisputeLookupFunc
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		now:   func() time.Time { return time.Now().UTC() },
		newID: func() string { return fmt.Sprintf("ari_%s", strings.ReplaceAll(uuid.NewString(), "-", "")) },
	}
}

func (s *Service) SetNow(fn func() time.Time) { s.now = fn }

// SetCreditClearer wires order credit balance clearing on invoice settlement / FIFO payment.
func (s *Service) SetCreditClearer(fn func(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error) {
	s.creditClearer = fn
}

// SetDisputeLookup overrides default Spanner Claims dispute lookup (primarily for tests / decoupled wiring).
func (s *Service) SetDisputeLookup(fn func(ctx context.Context, orderIDs []string) (map[string]int64, error)) {
	s.disputeLookup = fn
}

// MigrationInvoiceRepository is implemented by SpannerRepository and MemoryRepository
// to support migrated opening-balance inspection and retailer acceptance/dispute.
type MigrationInvoiceRepository interface {
	ListMigrationOpeningBalances(ctx context.Context, retailerID string) ([]Invoice, error)
	AcknowledgeMigrationInvoice(ctx context.Context, retailerID, invoiceID string, now time.Time) (Invoice, error)
	DisputeMigrationInvoice(ctx context.Context, retailerID, invoiceID, reason string, now time.Time) (Invoice, error)
}

// ErrRepoUnavailable indicates the AR service was constructed without a repository.
var ErrRepoUnavailable = errors.New("ar repository unavailable")

func (s *Service) migrationRepo() (MigrationInvoiceRepository, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	migRepo, ok := s.repo.(MigrationInvoiceRepository)
	if !ok {
		return nil, fmt.Errorf("ar repository does not support migration opening balances")
	}
	return migRepo, nil
}

// ListMigrationOpeningBalances returns migrated opening-balance AR invoices for retailerID.
func (s *Service) ListMigrationOpeningBalances(ctx context.Context, retailerID string) ([]Invoice, error) {
	migRepo, err := s.migrationRepo()
	if err != nil {
		return nil, err
	}
	return migRepo.ListMigrationOpeningBalances(ctx, retailerID)
}

// AcknowledgeMigrationInvoice records retailer acceptance and unfreezes migration dunning.
func (s *Service) AcknowledgeMigrationInvoice(ctx context.Context, retailerID, invoiceID string) (Invoice, error) {
	migRepo, err := s.migrationRepo()
	if err != nil {
		return Invoice{}, err
	}
	return migRepo.AcknowledgeMigrationInvoice(ctx, retailerID, invoiceID, s.now())
}

// DisputeMigrationInvoice records a retailer balance dispute reason and keeps dunning frozen.
func (s *Service) DisputeMigrationInvoice(ctx context.Context, retailerID, invoiceID, reason string) (Invoice, error) {
	migRepo, err := s.migrationRepo()
	if err != nil {
		return Invoice{}, err
	}
	return migRepo.DisputeMigrationInvoice(ctx, retailerID, invoiceID, reason, s.now())
}

type claimsRowReader interface {
	Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
}

func dedupeNonEmptyIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func queryClaimsDisputeMap(ctx context.Context, reader claimsRowReader, orderIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	cleanIDs := dedupeNonEmptyIDs(orderIDs)
	if reader == nil || len(cleanIDs) == 0 {
		return out, nil
	}
	iter := reader.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, IFNULL(SUM(IFNULL(AmountMinor, 0)), 0) AS DisputedMinor
			FROM Claims
			WHERE OrderId IN UNNEST(@ids) AND Status IN ('OPEN', 'UNDER_REVIEW')
			GROUP BY OrderId`,
		Params: map[string]any{"ids": cleanIDs},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var oid string
		var amt int64
		if err := row.Columns(&oid, &amt); err != nil {
			return nil, err
		}
		if amt > 0 {
			out[oid] = amt
		}
	}
	return out, nil
}

func (s *Service) lookupDisputes(ctx context.Context, orderIDs []string) (map[string]int64, error) {
	cleanIDs := dedupeNonEmptyIDs(orderIDs)
	if len(cleanIDs) == 0 {
		return map[string]int64{}, nil
	}
	if s != nil && s.disputeLookup != nil {
		res, err := s.disputeLookup(ctx, cleanIDs)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return map[string]int64{}, nil
		}
		return res, nil
	}
	if s != nil {
		if sr, ok := s.repo.(*SpannerRepository); ok && sr.client != nil {
			return queryClaimsDisputeMap(ctx, sr.client.Single(), cleanIDs)
		}
	}
	return map[string]int64{}, nil
}

func clampDisputed(balanceMinor, disputedMinor int64) (disputed, undisputed int64) {
	if balanceMinor <= 0 {
		return 0, 0
	}
	if disputedMinor <= 0 {
		return 0, balanceMinor
	}
	disputed = disputedMinor
	if disputed > balanceMinor {
		disputed = balanceMinor
	}
	return disputed, balanceMinor - disputed
}

func parseStatusFilter(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts)*2)
	out := make([]string, 0, len(parts)*2)
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, p := range parts {
		tok := strings.ToUpper(strings.TrimSpace(p))
		if tok == "" {
			continue
		}
		if tok == "UNPAID" {
			add(StatusOpen)
			add(StatusPartial)
			continue
		}
		add(tok)
	}
	return out
}

func matchesStatusFilter(invStatus string, statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}
	upper := strings.ToUpper(strings.TrimSpace(invStatus))
	for _, st := range statuses {
		if upper == st {
			return true
		}
	}
	return false
}

// OpenFromCreditLeaveRequest opens an AR invoice at credit leave (or billing).
type OpenFromCreditLeaveRequest struct {
	SupplierID    string
	RetailerID    string
	OrderID       string
	AmountMinor   int64
	Currency      string // ISO-4217; empty reads the shipped pack, never invents UZS
	TermsDays     int64
	GraceDays     int64
	CreditLeaveAt time.Time
	DueAt         time.Time
}

// buildOpenInvoice validates and constructs an OPEN invoice for credit leave (no I/O).
func (s *Service) buildOpenInvoice(ctx context.Context, req OpenFromCreditLeaveRequest) (Invoice, error) {
	if !InvoicesEnabled() {
		if req.AmountMinor > 0 {
			return Invoice{}, ErrInvoicesDisabled
		}
		return Invoice{}, nil
	}
	if req.AmountMinor <= 0 {
		return Invoice{}, nil
	}
	termsDays := req.TermsDays
	if termsDays <= 0 {
		termsDays = 30
	}
	dueAt := req.DueAt
	if dueAt.IsZero() {
		dueAt = req.CreditLeaveAt.AddDate(0, 0, int(termsDays))
	}
	currency := fxrates.NormalizeCurrency(req.Currency)
	if currency == "" {
		packCur, err := auth.CurrencyFromContext(ctx, req.SupplierID)
		if err != nil {
			return Invoice{}, err
		}
		currency = fxrates.NormalizeCurrency(packCur)
	}
	if len(currency) != 3 {
		return Invoice{}, fmt.Errorf("%w: %q", fxrates.ErrInvalidCurrency, req.Currency)
	}
	leaveAt := req.CreditLeaveAt
	if leaveAt.IsZero() {
		leaveAt = s.now()
	}
	return Invoice{
		InvoiceID:       s.newID(),
		SupplierID:      req.SupplierID,
		RetailerID:      req.RetailerID,
		OrderID:         req.OrderID,
		Status:          StatusOpen,
		PrincipalMinor:  req.AmountMinor,
		BalanceMinor:    req.AmountMinor,
		Currency:        currency,
		CreditLeaveAt:   leaveAt,
		DueAt:           dueAt,
		TermsDays:       termsDays,
		GracePeriodDays: req.GraceDays,
		AgingBucket:     AgingCurrent,
		Version:         1,
		CreatedAt:       s.now(),
		UpdatedAt:       s.now(),
	}, nil
}

// OpenFromCreditLeave creates an OPEN invoice with due date from terms (idempotent per order).
func (s *Service) OpenFromCreditLeave(ctx context.Context, req OpenFromCreditLeaveRequest) (Invoice, error) {
	if existing, found, err := s.repo.GetByOrder(ctx, req.OrderID); err != nil {
		return Invoice{}, err
	} else if found {
		return existing, nil
	}
	inv, err := s.buildOpenInvoice(ctx, req)
	if err != nil {
		return Invoice{}, err
	}
	if inv.InvoiceID == "" {
		return Invoice{}, nil
	}
	if err := s.repo.OpenInvoice(ctx, inv); err != nil {
		return Invoice{}, err
	}
	return inv, nil
}

// OpenFromCreditLeaveInTxn opens the AR invoice on the same Spanner RW txn as
// credit-leave order mutation (B6 money fail-closed). Idempotent per order.
// When the repo is not Spanner-backed, falls back to OpenFromCreditLeave (separate txn).
func (s *Service) OpenFromCreditLeaveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req OpenFromCreditLeaveRequest) error {
	if s == nil {
		return fmt.Errorf("ar service unavailable")
	}
	if txn == nil {
		_, err := s.OpenFromCreditLeave(ctx, req)
		return err
	}
	// Prefer in-txn path when Spanner repo (same database as orders).
	if _, ok := s.repo.(*SpannerRepository); ok {
		inv, err := s.buildOpenInvoice(ctx, req)
		if err != nil {
			return err
		}
		if inv.InvoiceID == "" {
			return nil
		}
		return openInvoiceInTxn(ctx, txn, inv)
	}
	// Memory / test repos: sequential open after order commit is still fail-closed at HTTP layer.
	_, err := s.OpenFromCreditLeave(ctx, req)
	return err
}

func (s *Service) ListRetailerInvoices(ctx context.Context, retailerID, status string, limit int) ([]Invoice, error) {
	return s.repo.ListByRetailer(ctx, retailerID, status, limit)
}

func (s *Service) ListRetailerInvoicesPage(ctx context.Context, retailerID, status string, limit int, cursor string) ([]Invoice, string, error) {
	if pager, ok := s.repo.(InvoicePager); ok {
		return pager.ListByRetailerPage(ctx, retailerID, status, limit, cursor)
	}
	if strings.TrimSpace(cursor) != "" {
		return nil, "", ErrInvalidCursor
	}
	list, err := s.repo.ListByRetailer(ctx, retailerID, status, limit)
	return list, "", err
}

func (s *Service) ListSupplierInvoices(ctx context.Context, supplierID, status string, limit int) ([]Invoice, error) {
	return s.repo.ListBySupplier(ctx, supplierID, status, limit)
}

func (s *Service) ListSupplierInvoicesPage(ctx context.Context, supplierID, status string, limit int, cursor string) ([]Invoice, string, error) {
	if pager, ok := s.repo.(InvoicePager); ok {
		return pager.ListBySupplierPage(ctx, supplierID, status, limit, cursor)
	}
	if strings.TrimSpace(cursor) != "" {
		return nil, "", ErrInvalidCursor
	}
	list, err := s.repo.ListBySupplier(ctx, supplierID, status, limit)
	return list, "", err
}

// RecordPayment pays down an open AR invoice (cash collection against a credit
// delivery, or a card capture applied to the invoice). Idempotent on
// idempotencyKey; returns the updated invoice. This is the production entry
// point that was previously missing — without it credit invoices could never
// be settled and every one marched to CREDIT_HOLD regardless of payment.
// currency, when non-empty, must match the invoice currency (P2-8).
func (s *Service) RecordPayment(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey, currency string) (Invoice, error) {
	if amountMinor <= 0 {
		return Invoice{}, fmt.Errorf("amount_minor must be positive")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return Invoice{}, fmt.Errorf("idempotency_key required")
	}
	if strings.TrimSpace(currency) != "" {
		inv, err := s.repo.GetByID(ctx, invoiceID)
		if err != nil {
			return Invoice{}, err
		}
		if err := fxrates.AssertSameCurrency(inv.Currency, currency); err != nil {
			return Invoice{}, err
		}
	}
	if err := s.repo.ApplyPayment(ctx, invoiceID, amountMinor, idempotencyKey); err != nil {
		return Invoice{}, err
	}
	return s.GetByID(ctx, invoiceID)
}

// RecordPaymentForOrder pays down the AR invoice opened for a given order, if
// one exists. No-op when there is no invoice (cash/card orders) — safe to call
// unconditionally from the cash-collection and capture paths.
// currency, when non-empty, must match the invoice (typically order.Currency).
func (s *Service) RecordPaymentForOrder(ctx context.Context, orderID string, amountMinor int64, idempotencyKey, currency string) error {
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
	_, err = s.RecordPayment(ctx, inv.InvoiceID, amountMinor, idempotencyKey, currency)
	return err
}

// RecordPaymentForOrderInTxn pays down the order's AR invoice on an existing
// Spanner RW txn (G1.A — same commit as CollectCash payment leg). No-op when
// no invoice exists. Fail-closed: returns error on apply failure so the caller
// can abort cash capture.
func (s *Service) RecordPaymentForOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, amountMinor int64, idempotencyKey, currency string) error {
	return s.RecordPaymentForOrderWithModeInTxn(ctx, txn, orderID, amountMinor, "", idempotencyKey, currency)
}

// RecordPaymentForOrderWithModeInTxn pays down the order's AR invoice on an
// existing Spanner RW txn while routing the GL debit leg through explicit
// paymentMode (falling back to key inference when blank).
func (s *Service) RecordPaymentForOrderWithModeInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, amountMinor int64, paymentMode, idempotencyKey, currency string) error {
	if s == nil {
		return fmt.Errorf("ar service unavailable")
	}
	if amountMinor <= 0 {
		return fmt.Errorf("amount_minor must be positive")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return fmt.Errorf("idempotency_key required")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("order_id required")
	}

	resolvedMode := strings.TrimSpace(paymentMode)
	if resolvedMode == "" {
		resolvedMode = inferARPaymentModeFromKey(idempotencyKey)
	}

	// Spanner path: load + apply on the caller's txn.
	if txn != nil {
		if _, ok := s.repo.(*SpannerRepository); ok {
			inv, found, err := getInvoiceByOrderInTxn(ctx, txn, orderID)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			if inv.Status == StatusPaid || inv.Status == StatusVoid {
				return nil
			}
			if strings.TrimSpace(currency) != "" {
				if err := fxrates.AssertSameCurrency(inv.Currency, currency); err != nil {
					return err
				}
			}
			return applyPaymentWithModeInTxn(ctx, txn, inv.InvoiceID, amountMinor, resolvedMode, idempotencyKey)
		}
	}

	// Memory / non-Spanner repos: sequential apply (still fail-closed at HTTP).
	return s.RecordPaymentForOrder(ctx, orderID, amountMinor, idempotencyKey, currency)
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

// MemoryCreditProfileState configures simulated RetailerCreditProfiles flags for unit tests.
type MemoryCreditProfileState struct {
	Status           string
	PrepayOnlyLocked bool
	ActiveQuarterKey string
}

// MemoryRepository for tests.
type MemoryRepository struct {
	mu              sync.RWMutex
	byID            map[string]Invoice
	byOrder         map[string]string
	ledgerKeys      map[string]bool
	ledgerEntries   []LedgerEntry
	advanceBalances map[string]AdvanceBalance
	promises        map[string]PaymentPromise
	creditProfiles  map[string]MemoryCreditProfileState
	fifoResults     map[string]PayFIFOResponse
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:            map[string]Invoice{},
		byOrder:         map[string]string{},
		ledgerKeys:      map[string]bool{},
		advanceBalances: map[string]AdvanceBalance{},
		promises:        map[string]PaymentPromise{},
		creditProfiles:  map[string]MemoryCreditProfileState{},
		fifoResults:     map[string]PayFIFOResponse{},
	}
}

func (r *MemoryRepository) ensureMapsLocked() {
	if r.byID == nil {
		r.byID = map[string]Invoice{}
	}
	if r.byOrder == nil {
		r.byOrder = map[string]string{}
	}
	if r.ledgerKeys == nil {
		r.ledgerKeys = map[string]bool{}
	}
	if r.advanceBalances == nil {
		r.advanceBalances = map[string]AdvanceBalance{}
	}
	if r.promises == nil {
		r.promises = map[string]PaymentPromise{}
	}
	if r.creditProfiles == nil {
		r.creditProfiles = map[string]MemoryCreditProfileState{}
	}
	if r.fifoResults == nil {
		r.fifoResults = map[string]PayFIFOResponse{}
	}
}

func relKey(retailerID, supplierID string) string {
	return retailerID + "|" + supplierID
}

// SetCreditProfileState configures simulated credit profile status/flags for promise tests.
func (r *MemoryRepository) SetCreditProfileState(retailerID, supplierID, status string, prepayOnlyLocked bool, activeQuarterKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	r.creditProfiles[relKey(retailerID, supplierID)] = MemoryCreditProfileState{
		Status:           status,
		PrepayOnlyLocked: prepayOnlyLocked,
		ActiveQuarterKey: activeQuarterKey,
	}
}

// GetCreditProfileState reads simulated credit profile flags for promise tests.
func (r *MemoryRepository) GetCreditProfileState(retailerID, supplierID string) (MemoryCreditProfileState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.creditProfiles == nil {
		return MemoryCreditProfileState{}, false
	}
	st, ok := r.creditProfiles[relKey(retailerID, supplierID)]
	return st, ok
}

func (r *MemoryRepository) OpenInvoice(_ context.Context, inv Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	if id, ok := r.byOrder[inv.OrderID]; ok {
		_ = id
		return nil
	}
	r.byID[inv.InvoiceID] = inv
	r.byOrder[inv.OrderID] = inv.InvoiceID
	openKey := "open:" + inv.OrderID
	if !r.ledgerKeys[openKey] {
		r.ledgerKeys[openKey] = true
		ts := inv.CreatedAt
		if ts.IsZero() {
			ts = time.Now().UTC()
		}
		r.ledgerEntries = append(r.ledgerEntries, LedgerEntry{
			EntryID:        ledgerEntryID("open", inv.OrderID),
			InvoiceID:      inv.InvoiceID,
			SupplierID:     inv.SupplierID,
			RetailerID:     inv.RetailerID,
			EntryType:      "OPEN",
			AmountMinor:    inv.PrincipalMinor,
			IdempotencyKey: openKey,
			RefOrderID:     inv.OrderID,
			CreatedAt:      ts.UTC(),
		})
	}
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

func encodeInvoiceCursor(dueAt time.Time, invoiceID string) string {
	payload := dueAt.UTC().Format(time.RFC3339Nano) + "|" + invoiceID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeInvoiceCursor(cursor string) (time.Time, string, bool, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return time.Time{}, "", false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	return ts.UTC(), parts[1], true, nil
}

func sortInvoicesDueAsc(items []Invoice) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].DueAt.Equal(items[j].DueAt) {
			return items[i].DueAt.Before(items[j].DueAt)
		}
		return items[i].InvoiceID < items[j].InvoiceID
	})
}

func paginateMemoryInvoices(items []Invoice, limit int, cursor string) ([]Invoice, string, error) {
	sortInvoicesDueAsc(items)
	cursorTs, cursorID, hasCursor, err := decodeInvoiceCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	filtered := items
	if hasCursor {
		filtered = make([]Invoice, 0, len(items))
		for _, inv := range items {
			due := inv.DueAt.UTC()
			if due.After(cursorTs) || (due.Equal(cursorTs) && inv.InvoiceID > cursorID) {
				filtered = append(filtered, inv)
			}
		}
	}
	if len(filtered) > limit {
		page := filtered[:limit]
		last := page[len(page)-1]
		return page, encodeInvoiceCursor(last.DueAt, last.InvoiceID), nil
	}
	return filtered, "", nil
}

func (r *MemoryRepository) ListByRetailer(ctx context.Context, retailerID, status string, limit int) ([]Invoice, error) {
	page, _, err := r.ListByRetailerPage(ctx, retailerID, status, limit, "")
	return page, err
}

func (r *MemoryRepository) ListByRetailerPage(_ context.Context, retailerID, status string, limit int, cursor string) ([]Invoice, string, error) {
	if limit <= 0 {
		limit = 50
	}
	statuses := parseStatusFilter(status)
	r.mu.RLock()
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.RetailerID != retailerID {
			continue
		}
		if !matchesStatusFilter(inv.Status, statuses) {
			continue
		}
		out = append(out, inv)
	}
	r.mu.RUnlock()
	return paginateMemoryInvoices(out, limit, cursor)
}

func (r *MemoryRepository) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Invoice, error) {
	page, _, err := r.ListBySupplierPage(ctx, supplierID, status, limit, "")
	return page, err
}

func (r *MemoryRepository) ListBySupplierPage(_ context.Context, supplierID, status string, limit int, cursor string) ([]Invoice, string, error) {
	if limit <= 0 {
		limit = 50
	}
	statuses := parseStatusFilter(status)
	r.mu.RLock()
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.SupplierID != supplierID {
			continue
		}
		if !matchesStatusFilter(inv.Status, statuses) {
			continue
		}
		out = append(out, inv)
	}
	r.mu.RUnlock()
	return paginateMemoryInvoices(out, limit, cursor)
}

func (r *MemoryRepository) ApplyPayment(_ context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	if r.ledgerKeys[idempotencyKey] {
		return nil
	}
	inv, ok := r.byID[invoiceID]
	if !ok {
		return fmt.Errorf("invoice_not_found")
	}
	applied := amountMinor
	if applied > inv.BalanceMinor {
		applied = inv.BalanceMinor
	}
	if applied < 0 {
		applied = 0
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
	inv.Version++
	inv.UpdatedAt = time.Now().UTC()
	r.byID[invoiceID] = inv
	r.ledgerKeys[idempotencyKey] = true
	r.ledgerEntries = append(r.ledgerEntries, LedgerEntry{
		EntryID:        ledgerEntryID("pay", idempotencyKey),
		InvoiceID:      invoiceID,
		SupplierID:     inv.SupplierID,
		RetailerID:     inv.RetailerID,
		EntryType:      "PAYMENT",
		AmountMinor:    -applied,
		IdempotencyKey: idempotencyKey,
		RefOrderID:     inv.OrderID,
		CreatedAt:      inv.UpdatedAt,
	})
	return nil
}

func (r *MemoryRepository) ApplyCreditNote(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return r.ApplyPayment(ctx, invoiceID, amountMinor, idempotencyKey)
}

func (r *MemoryRepository) VoidInvoice(ctx context.Context, invoiceID string, idempotencyKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	if r.ledgerKeys[idempotencyKey] {
		return nil
	}
	inv, ok := r.byID[invoiceID]
	if !ok {
		return fmt.Errorf("invoice_not_found")
	}
	voidedAmt := inv.BalanceMinor
	inv.BalanceMinor = 0
	inv.Status = StatusVoid
	inv.Version++
	inv.UpdatedAt = time.Now().UTC()
	r.byID[invoiceID] = inv
	r.ledgerKeys[idempotencyKey] = true
	r.ledgerEntries = append(r.ledgerEntries, LedgerEntry{
		EntryID:        ledgerEntryID("void", idempotencyKey),
		InvoiceID:      invoiceID,
		SupplierID:     inv.SupplierID,
		RetailerID:     inv.RetailerID,
		EntryType:      "WRITE_OFF",
		AmountMinor:    -voidedAmt,
		IdempotencyKey: idempotencyKey,
		RefOrderID:     inv.OrderID,
		CreatedAt:      inv.UpdatedAt,
	})
	return nil
}

func (r *MemoryRepository) ListOpenForDunning(ctx context.Context, limit int) ([]Invoice, error) {
	page, _, err := r.ListOpenForDunningPage(ctx, limit, "")
	return page, err
}

func (r *MemoryRepository) ListOpenForDunningPage(_ context.Context, limit int, cursor string) ([]Invoice, string, error) {
	if limit <= 0 {
		limit = 200
	}
	r.mu.RLock()
	out := make([]Invoice, 0)
	for _, inv := range r.byID {
		if inv.Status != StatusOpen && inv.Status != StatusPartial {
			continue
		}
		if inv.IsMigrationDunningFrozen() {
			continue
		}
		out = append(out, inv)
	}
	r.mu.RUnlock()
	return paginateMemoryInvoices(out, limit, cursor)
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

// Note: Spanner RecomputeAging is overridden below with outbox.

// SpannerRepository persists ArInvoices.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) OpenInvoice(ctx context.Context, inv Invoice) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return openInvoiceInTxn(ctx, txn, inv)
	})
}

func ledgerEntryID(prefix, key string) string {
	raw := prefix + ":" + key
	if len(raw) <= 36 {
		return raw
	}
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("arl_%s", hex.EncodeToString(h[:]))[:36]
}

// openInvoiceInTxn inserts ArInvoices + ledger OPEN + outbox on an existing RW txn (B6).
func openInvoiceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, inv Invoice) error {
	if txn == nil {
		return fmt.Errorf("nil spanner txn")
	}
	// Idempotent: order already has an invoice in this txn snapshot.
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT InvoiceId FROM ArInvoices WHERE OrderId = @oid LIMIT 1`,
		Params: map[string]any{"oid": inv.OrderID},
	})
	_, qerr := iter.Next()
	iter.Stop()
	if qerr == nil {
		return nil
	}
	if qerr != iterator.Done {
		return qerr
	}
	// Idempotent guard on append-only ArLedgerEntries before InsertMap.
	openKey := "open:" + inv.OrderID
	openIter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
		Params: map[string]any{"k": openKey},
	})
	_, oErr := openIter.Next()
	openIter.Stop()
	if oErr == nil {
		return nil
	}
	if oErr != iterator.Done {
		return oErr
	}
	// Spanner (and especially the emulator) rejects client wall-clock values
	// that are even slightly ahead of server time. Persist audit stamps via
	// commit timestamp; clamp CreditLeaveAt a second into the past so a host
	// clock that leads the emulator cannot fail the money-path open.
	creditLeaveAt := inv.CreditLeaveAt.UTC()
	if skew := time.Now().UTC().Add(-time.Second); creditLeaveAt.After(skew) {
		creditLeaveAt = skew
	}
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
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId":        ledgerEntryID("open", inv.OrderID),
			"InvoiceId":      inv.InvoiceID,
			"SupplierId":     inv.SupplierID,
			"RetailerId":     inv.RetailerID,
			"EntryType":      "OPEN",
			"AmountMinor":    inv.PrincipalMinor,
			"IdempotencyKey": openKey,
			"RefOrderId":     inv.OrderID,
			"CreatedAt":      spanner.CommitTimestamp,
		}),
	}); err != nil {
		return err
	}
	return buf.Flush(ctx)
}

var invoiceSelectColumnNames = []string{
	"InvoiceId", "SupplierId", "RetailerId", "OrderId", "Status", "PrincipalMinor", "BalanceMinor", "Currency",
	"CreditLeaveAt", "DueAt", "TermsDays", "GracePeriodDays", "AgingBucket", "DunningStep", "Version", "CreatedAt", "UpdatedAt",
	"Source", "ExternalId", "AcknowledgedAt", "DisputedAt", "DisputeReason",
}

const invoiceSelectColumns = `InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, CreditLeaveAt, DueAt, TermsDays, GracePeriodDays, AgingBucket, DunningStep, Version, CreatedAt, UpdatedAt, Source, ExternalId, AcknowledgedAt, DisputedAt, DisputeReason`

func (r *SpannerRepository) GetByOrder(ctx context.Context, orderID string) (Invoice, bool, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT ` + invoiceSelectColumns + ` FROM ArInvoices WHERE OrderId = @oid`,
		Params: map[string]any{"oid": orderID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
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
	row, err := r.client.Single().ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, invoiceSelectColumnNames)
	if err != nil {
		return Invoice{}, err
	}
	return scanInvoice(row)
}

func scanInvoice(row *spanner.Row) (Invoice, error) {
	var inv Invoice
	var aging, source, extID, disputeReason spanner.NullString
	var ackAt, dispAt spanner.NullTime
	if err := row.Columns(&inv.InvoiceID, &inv.SupplierID, &inv.RetailerID, &inv.OrderID, &inv.Status,
		&inv.PrincipalMinor, &inv.BalanceMinor, &inv.Currency, &inv.CreditLeaveAt, &inv.DueAt,
		&inv.TermsDays, &inv.GracePeriodDays, &aging, &inv.DunningStep, &inv.Version, &inv.CreatedAt, &inv.UpdatedAt,
		&source, &extID, &ackAt, &dispAt, &disputeReason); err != nil {
		return inv, err
	}
	inv.AgingBucket = aging.StringVal
	if source.Valid {
		inv.Source = source.StringVal
	}
	if extID.Valid {
		inv.ExternalID = extID.StringVal
	}
	if ackAt.Valid {
		t := ackAt.Time.UTC()
		inv.AcknowledgedAt = &t
	}
	if dispAt.Valid {
		t := dispAt.Time.UTC()
		inv.DisputedAt = &t
	}
	if disputeReason.Valid {
		inv.DisputeReason = disputeReason.StringVal
	}
	return inv, nil
}

func (r *SpannerRepository) ListByRetailer(ctx context.Context, retailerID, status string, limit int) ([]Invoice, error) {
	page, _, err := r.ListByRetailerPage(ctx, retailerID, status, limit, "")
	return page, err
}

func (r *SpannerRepository) ListByRetailerPage(ctx context.Context, retailerID, status string, limit int, cursor string) ([]Invoice, string, error) {
	return r.listPage(ctx, "RetailerId", retailerID, status, limit, cursor)
}

func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Invoice, error) {
	page, _, err := r.ListBySupplierPage(ctx, supplierID, status, limit, "")
	return page, err
}

func (r *SpannerRepository) ListBySupplierPage(ctx context.Context, supplierID, status string, limit int, cursor string) ([]Invoice, string, error) {
	return r.listPage(ctx, "SupplierId", supplierID, status, limit, cursor)
}

func (r *SpannerRepository) listPage(ctx context.Context, col, id, status string, limit int, cursor string) ([]Invoice, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	cursorTs, cursorID, hasCursor, err := decodeInvoiceCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var sql string
	switch col {
	case "SupplierId":
		sql = `SELECT ` + invoiceSelectColumns + ` FROM ArInvoices WHERE SupplierId = @id`
	case "RetailerId":
		sql = `SELECT ` + invoiceSelectColumns + ` FROM ArInvoices WHERE RetailerId = @id`
	default:
		return nil, "", fmt.Errorf("invalid ar column %q", col)
	}
	params := map[string]any{
		"id":    id,
		"limit": int64(limit + 1),
	}
	statuses := parseStatusFilter(status)
	switch len(statuses) {
	case 0:
	case 1:
		sql += " AND Status = @st"
		params["st"] = statuses[0]
	default:
		sql += " AND Status IN UNNEST(@sts)"
		params["sts"] = statuses
	}
	if hasCursor {
		sql += " AND (DueAt > @cursorTs OR (DueAt = @cursorTs AND InvoiceId > @cursorId))"
		params["cursorTs"] = cursorTs
		params["cursorId"] = cursorID
	}
	sql += " ORDER BY DueAt ASC, InvoiceId ASC LIMIT @limit"
	// Strong read on money/AR paths (eliminates ExactStaleness lag after payment/write-off).
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]Invoice, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, "", err
		}
		inv, err := scanInvoice(row)
		if err != nil {
			return nil, "", err
		}
		out = append(out, inv)
	}
	if len(out) > limit {
		page := out[:limit]
		last := page[len(page)-1]
		return page, encodeInvoiceCursor(last.DueAt, last.InvoiceID), nil
	}
	return out, "", nil
}

func (r *SpannerRepository) ApplyPayment(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return applyPaymentInTxn(ctx, txn, invoiceID, amountMinor, idempotencyKey)
	})
}

func (r *SpannerRepository) ApplyPaymentWithMode(ctx context.Context, invoiceID string, amountMinor int64, paymentMode, idempotencyKey string) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return applyPaymentWithModeInTxn(ctx, txn, invoiceID, amountMinor, paymentMode, idempotencyKey)
	})
}

// getInvoiceByOrderInTxn loads ArInvoices for order on an active RW txn (G1.A).
func getInvoiceByOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (Invoice, bool, error) {
	if txn == nil {
		return Invoice{}, false, fmt.Errorf("nil spanner txn")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + invoiceSelectColumns + `
			FROM ArInvoices WHERE OrderId = @oid LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Invoice{}, false, nil
	}
	if err != nil {
		return Invoice{}, false, err
	}
	inv, err := scanInvoice(row)
	if err != nil {
		return Invoice{}, false, err
	}
	return inv, true, nil
}

func normalizeARPaymentMode(raw string) string {
	m := strings.ToUpper(strings.TrimSpace(raw))
	switch m {
	case "", "BANK", "BANK_TRANSFER", "WIRE", "EFT":
		return "BANK"
	case "TRANSFER":
		return "TRANSFER"
	case "CASH_OFFICE", "VAULT", "OFFICE_CASH":
		return "CASH_OFFICE"
	case "CASH":
		return "CASH"
	case "GLOBAL_PAY", "GLOBALPAY", "GP", "CARD":
		return "CARD"
	case "PAYME", "CLICK", "ADYEN", "WALLET":
		return m
	default:
		return "BANK"
	}
}

func inferARPaymentModeFromKey(idempotencyKey string) string {
	k := strings.ToLower(strings.TrimSpace(idempotencyKey))
	switch {
	case strings.Contains(k, "cash_office") || strings.Contains(k, "cash-office"):
		return "CASH_OFFICE"
	case strings.HasPrefix(k, "cash:") ||
		strings.HasPrefix(k, "ar-cash") ||
		strings.Contains(k, "collect_cash") ||
		strings.Contains(k, "collect-cash") ||
		strings.Contains(k, "cash-collect") ||
		strings.Contains(k, "cash_"):
		return "CASH"
	case strings.HasPrefix(k, "card:") ||
		strings.HasPrefix(k, "ar-card") ||
		strings.Contains(k, "card-settle") ||
		strings.Contains(k, "card_settle") ||
		strings.Contains(k, "globalpay") ||
		strings.Contains(k, "global_pay") ||
		strings.Contains(k, "global-pay"):
		return "CARD"
	case strings.Contains(k, "payme"):
		return "PAYME"
	case strings.Contains(k, "click"):
		return "CLICK"
	case strings.Contains(k, "adyen"):
		return "ADYEN"
	case strings.Contains(k, "wallet"):
		return "WALLET"
	default:
		return "BANK"
	}
}

func buildWalletARPaymentGLInput(sid, rid, oid, invoiceID, cur, glIdemKey, memo string, amountMinor int64, occurredAt time.Time) gl.JournalEntryInput {
	return gl.JournalEntryInput{
		EntityID:       sid,
		SourceType:     "AR_PAYMENT",
		SourceID:       invoiceID,
		OrderID:        oid,
		Memo:           memo,
		Currency:       cur,
		IdempotencyKey: glIdemKey,
		OccurredAt:     occurredAt,
		Postings: []gl.PostingInput{
			{
				AccountCode: gl.AccountRetailerCredit,
				Direction:   gl.DirectionDebit,
				AmountMinor: amountMinor,
				Currency:    cur,
				PartyType:   "RETAILER",
				PartyID:     rid,
				Description: "AR invoice paid from retailer wallet",
			},
			{
				AccountCode: gl.AccountAccountsReceivable,
				Direction:   gl.DirectionCredit,
				AmountMinor: amountMinor,
				Currency:    cur,
				PartyType:   "RETAILER",
				PartyID:     rid,
				Description: "AR receivable settled via wallet",
			},
		},
	}
}

// applyPaymentInTxn applies a PAYMENT ledger entry + balance update + outbox on
// an existing Spanner RW txn (G1.A co-atomic with CollectCash).
func applyPaymentInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return applyPaymentWithModeInTxn(ctx, txn, invoiceID, amountMinor, inferARPaymentModeFromKey(idempotencyKey), idempotencyKey)
}

func applyPaymentWithModeInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, invoiceID string, amountMinor int64, paymentMode, idempotencyKey string) error {
	if txn == nil {
		return fmt.Errorf("nil spanner txn")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return fmt.Errorf("idempotency_key required")
	}
	// Idempotency by column (EntryId is a separate synthetic key).
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
		Params: map[string]any{"k": idempotencyKey},
	})
	_, qerr := iter.Next()
	iter.Stop()
	if qerr == nil {
		return nil // already applied
	}
	if qerr != iterator.Done {
		return qerr
	}

	row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID},
		[]string{"SupplierId", "RetailerId", "OrderId", "Currency", "BalanceMinor", "Status", "Version"})
	if err != nil {
		return err
	}
	var sid, rid, status string
	var oidNull, curNull spanner.NullString
	var bal, ver int64
	if err := row.Columns(&sid, &rid, &oidNull, &curNull, &bal, &status, &ver); err != nil {
		return err
	}
	oid := ""
	if oidNull.Valid {
		oid = strings.TrimSpace(oidNull.StringVal)
	}
	cur := "UZS"
	if curNull.Valid && strings.TrimSpace(curNull.StringVal) != "" {
		cur = strings.ToUpper(strings.TrimSpace(curNull.StringVal))
	}
	appliedMinor := amountMinor
	if bal <= 0 {
		appliedMinor = 0
	} else if appliedMinor > bal {
		appliedMinor = bal
	}
	excessMinor := amountMinor - appliedMinor
	newBal := bal - appliedMinor
	if newBal < 0 {
		newBal = 0
	}
	newStatus := StatusPartial
	if newBal == 0 {
		newStatus = StatusPaid
	}
	now := time.Now().UTC()
	buf := outbox.NewSpannerTxnBuffer(txn)
	normMode := normalizeARPaymentMode(paymentMode)
	if strings.TrimSpace(paymentMode) == "" {
		normMode = inferARPaymentModeFromKey(idempotencyKey)
	}

	var muts []*spanner.Mutation
	if appliedMinor > 0 {
		entryID := ledgerEntryID("pay", idempotencyKey)
		eventType := events.EventARInvoicePayment
		if newStatus == StatusPaid {
			eventType = events.EventARInvoiceSettled
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, invoiceID, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent:    events.BaseEvent{Type: eventType, Timestamp: now.Format(time.RFC3339Nano)},
			InvoiceID:    invoiceID,
			SupplierID:   sid,
			RetailerID:   rid,
			OrderID:      oid,
			AmountMinor:  appliedMinor,
			BalanceMinor: newBal,
			Status:       newStatus,
		}); err != nil {
			return err
		}
		muts = append(muts,
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
				"AmountMinor":    -appliedMinor,
				"IdempotencyKey": idempotencyKey,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		)
		glIdemKey := "gl:ar:pay:" + idempotencyKey
		if normMode == "WALLET" {
			if _, err := gl.BufferJournalEntryInTxn(ctx, txn, buf, buildWalletARPaymentGLInput(
				sid, rid, oid, invoiceID, cur, glIdemKey, "AR invoice payment (wallet) "+invoiceID, appliedMinor, now,
			)); err != nil {
				return err
			}
		} else {
			if _, err := gl.PostEventInTxn(ctx, txn, buf, gl.RuleEventARPaymentReceived, gl.RuleEvalContext{
				EntityID:       sid,
				SupplierID:     sid,
				RetailerID:     rid,
				OrderID:        oid,
				SourceID:       invoiceID,
				Currency:       cur,
				PaymentMode:    normMode,
				GrossMinor:     appliedMinor,
				IdempotencyKey: glIdemKey,
				Memo:           "AR invoice payment " + invoiceID,
				OccurredAt:     now,
			}); err != nil {
				return err
			}
		}
	}

	if excessMinor > 0 {
		curAdv, curAdvCurrency, err := readAdvanceBalanceInTxn(ctx, txn, rid, sid)
		if err != nil {
			return err
		}
		advCur := cur
		if advCur == "" {
			advCur = curAdvCurrency
		}
		if advCur == "" {
			advCur = "UZS"
		}
		newAdv := curAdv + excessMinor
		advKey := "adv:" + idempotencyKey
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, advanceLedgerInvoiceSentinel+":"+rid+":"+sid, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent:    events.BaseEvent{Type: EventARAdvanceCredited, Timestamp: now.Format(time.RFC3339Nano)},
			InvoiceID:    advanceLedgerInvoiceSentinel,
			SupplierID:   sid,
			RetailerID:   rid,
			OrderID:      oid,
			AmountMinor:  excessMinor,
			BalanceMinor: newAdv,
			Status:       LedgerEntryAdvanceCredit,
		}); err != nil {
			return err
		}
		muts = append(muts,
			spanner.InsertOrUpdateMap("ArAdvanceBalances", map[string]any{
				"RetailerId":   rid,
				"SupplierId":   sid,
				"BalanceMinor": newAdv,
				"Currency":     advCur,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
			spanner.InsertMap("ArLedgerEntries", map[string]any{
				"EntryId":        ledgerEntryID("adv", idempotencyKey),
				"InvoiceId":      advanceLedgerInvoiceSentinel,
				"SupplierId":     sid,
				"RetailerId":     rid,
				"EntryType":      LedgerEntryAdvanceCredit,
				"AmountMinor":    excessMinor,
				"IdempotencyKey": advKey,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		)
		if _, err := gl.PostEventInTxn(ctx, txn, buf, gl.RuleEventARAdvanceCredited, gl.RuleEvalContext{
			EntityID:       sid,
			SupplierID:     sid,
			RetailerID:     rid,
			OrderID:        oid,
			SourceID:       advanceLedgerInvoiceSentinel + ":" + rid + ":" + sid,
			Currency:       advCur,
			PaymentMode:    normMode,
			GrossMinor:     excessMinor,
			IdempotencyKey: "gl:ar:adv:" + idempotencyKey,
			Memo:           "AR overpayment advance credit " + invoiceID,
			OccurredAt:     now,
		}); err != nil {
			return err
		}
	}

	if len(muts) > 0 {
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
	}
	return buf.Flush(ctx)
}

func (r *SpannerRepository) ApplyCreditNote(ctx context.Context, invoiceID string, amountMinor int64, idempotencyKey string) error {
	return r.ApplyPayment(ctx, invoiceID, amountMinor, idempotencyKey)
}

func (r *SpannerRepository) VoidInvoice(ctx context.Context, invoiceID string, idempotencyKey string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT 1 FROM ArLedgerEntries WHERE IdempotencyKey = @key LIMIT 1`,
			Params: map[string]any{"key": idempotencyKey},
		})
		_, qerr := iter.Next()
		iter.Stop()
		if qerr == nil {
			return nil // already applied
		}
		if qerr != iterator.Done {
			return qerr
		}

		row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID},
			[]string{"SupplierId", "RetailerId", "OrderId", "Currency", "BalanceMinor", "Version"})
		if err != nil {
			return err
		}
		var sid, rid string
		var oidNull, curNull spanner.NullString
		var bal, ver int64
		if err := row.Columns(&sid, &rid, &oidNull, &curNull, &bal, &ver); err != nil {
			return err
		}
		oid := ""
		if oidNull.Valid {
			oid = strings.TrimSpace(oidNull.StringVal)
		}
		cur := "UZS"
		if curNull.Valid && strings.TrimSpace(curNull.StringVal) != "" {
			cur = strings.ToUpper(strings.TrimSpace(curNull.StringVal))
		}

		entryID := ledgerEntryID("void", idempotencyKey)
		now := time.Now().UTC()
		buf := outbox.NewSpannerTxnBuffer(txn)

		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, invoiceID, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent:    events.BaseEvent{Type: "AR_INVOICE_VOIDED", Timestamp: now.Format(time.RFC3339Nano)},
			InvoiceID:    invoiceID,
			SupplierID:   sid,
			RetailerID:   rid,
			OrderID:      oid,
			AmountMinor:  0,
			BalanceMinor: 0,
			Status:       StatusVoid,
		}); err != nil {
			return err
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ArInvoices", map[string]any{
				"InvoiceId":    invoiceID,
				"BalanceMinor": 0,
				"Status":       StatusVoid,
				"Version":      ver + 1,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
			spanner.InsertMap("ArLedgerEntries", map[string]any{
				"EntryId":        entryID,
				"InvoiceId":      invoiceID,
				"SupplierId":     sid,
				"RetailerId":     rid,
				"EntryType":      "VOID",
				"AmountMinor":    -bal, // Adjust balance to zero in ledger
				"IdempotencyKey": idempotencyKey,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		if bal > 0 {
			if _, err := gl.PostEventInTxn(ctx, txn, buf, gl.RuleEventARAdjustWaive, gl.RuleEvalContext{
				EntityID:       sid,
				SupplierID:     sid,
				RetailerID:     rid,
				OrderID:        oid,
				SourceID:       invoiceID,
				Currency:       cur,
				GrossMinor:     bal,
				Action:         "VOID",
				IdempotencyKey: "gl:ar:void:" + idempotencyKey,
				Memo:           "AR invoice void " + invoiceID,
				OccurredAt:     now,
			}); err != nil {
				return err
			}
		}
		return buf.Flush(ctx)
	})
	return err
}

func (r *SpannerRepository) ListOpenForDunning(ctx context.Context, limit int) ([]Invoice, error) {
	page, _, err := r.ListOpenForDunningPage(ctx, limit, "")
	return page, err
}

func (r *SpannerRepository) ListOpenForDunningPage(ctx context.Context, limit int, cursor string) ([]Invoice, string, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	cursorTs, cursorID, hasCursor, err := decodeInvoiceCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	sql := `SELECT ` + invoiceSelectColumns + `
FROM ArInvoices WHERE Status IN ('OPEN','PARTIAL') AND (COALESCE(Source, '') != 'MIGRATION' OR (AcknowledgedAt IS NOT NULL AND DisputedAt IS NULL))`
	params := map[string]any{"limit": int64(limit + 1)}
	if hasCursor {
		sql += ` AND (DueAt > @cursorTs OR (DueAt = @cursorTs AND InvoiceId > @cursorId))`
		params["cursorTs"] = cursorTs
		params["cursorId"] = cursorID
	}
	sql += ` ORDER BY DueAt ASC, InvoiceId ASC LIMIT @limit`
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]Invoice, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, "", err
		}
		inv, err := scanInvoice(row)
		if err != nil {
			return nil, "", err
		}
		out = append(out, inv)
	}
	if len(out) > limit {
		page := out[:limit]
		last := page[len(page)-1]
		return page, encodeInvoiceCursor(last.DueAt, last.InvoiceID), nil
	}
	return out, "", nil
}

func (r *SpannerRepository) UpdateDunning(ctx context.Context, invoiceID string, step int64, agingBucket string, lastDunnedAt time.Time, version int64) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var sid, rid, status string
		var currentVersion int64
		row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, []string{"SupplierId", "RetailerId", "Version", "Status"})
		if err != nil {
			return err
		}
		if err := row.Columns(&sid, &rid, &currentVersion, &status); err != nil {
			return err
		}
		if currentVersion != version || status == StatusPaid || status == StatusVoid {
			// Version mismatch or invoice no longer open; silently abort this dunning step
			return nil
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
	pageSize := limit
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 500
	}
	type agingRow struct {
		id, sid, rid, oid, prevBucket string
		due                           time.Time
		ver                           int64
		bucket                        string
	}
	totalUpdated := 0
	cursorID := ""
	for {
		sql := `SELECT InvoiceId, SupplierId, RetailerId, OrderId, DueAt, AgingBucket, Version
		        FROM ArInvoices WHERE Status IN ('OPEN','PARTIAL')`
		params := map[string]any{"limit": int64(pageSize)}
		if cursorID != "" {
			sql += ` AND InvoiceId > @cursorId`
			params["cursorId"] = cursorID
		}
		sql += ` ORDER BY InvoiceId ASC LIMIT @limit`

		var rows []agingRow
		lastScannedID := ""
		scannedCount := 0
		iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return totalUpdated, err
			}
			var rec agingRow
			var aging spanner.NullString
			if err := row.Columns(&rec.id, &rec.sid, &rec.rid, &rec.oid, &rec.due, &aging, &rec.ver); err != nil {
				iter.Stop()
				return totalUpdated, err
			}
			scannedCount++
			lastScannedID = rec.id
			if aging.Valid {
				rec.prevBucket = aging.StringVal
			}
			rec.bucket = AgingBucketFor(rec.due, now)
			if rec.bucket != rec.prevBucket {
				rows = append(rows, rec)
			}
		}
		iter.Stop()

		// B6 M-P1-6: aging bucket changes leave the bus (one event per invoice).
		for _, rec := range rows {
			_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				var currentVersion int64
				var status string
				row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{rec.id}, []string{"Version", "Status"})
				if err != nil {
					return err
				}
				if err := row.Columns(&currentVersion, &status); err != nil {
					return err
				}
				if currentVersion != rec.ver || status == StatusPaid || status == StatusVoid {
					// Abort silently: concurrent payment modified it since the snapshot read
					return nil
				}

				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.UpdateMap("ArInvoices", map[string]any{
						"InvoiceId":   rec.id,
						"AgingBucket": rec.bucket,
						"Version":     rec.ver + 1,
						"UpdatedAt":   spanner.CommitTimestamp,
					}),
				}); err != nil {
					return err
				}
				buf := outbox.NewSpannerTxnBuffer(txn)
				if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, rec.id, events.TopicMain, events.ARInvoiceEvent{
					BaseEvent:   events.BaseEvent{Type: events.EventARInvoiceAgingUpdated, Timestamp: now.UTC().Format(time.RFC3339Nano)},
					InvoiceID:   rec.id,
					SupplierID:  rec.sid,
					RetailerID:  rec.rid,
					OrderID:     rec.oid,
					AgingBucket: rec.bucket,
				}); err != nil {
					return err
				}
				return buf.Flush(ctx)
			})
			if err != nil {
				return totalUpdated, err
			}
			totalUpdated++
		}

		if scannedCount < pageSize || lastScannedID == "" || lastScannedID == cursorID {
			break
		}
		cursorID = lastScannedID
	}
	return totalUpdated, nil
}
