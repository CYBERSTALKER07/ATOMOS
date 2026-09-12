package payout

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"

)

// Supplier payout batches: NetPayout = Σcaptured − Σrefunds − commission,
// computed over provider-confirmed payment legs in the period. The batch is
// executed via bank-file export (CSV payment instruction). Prod-bar decision:
// bank-file is the permanent settlement transport — see docs/PAYOUT_RAIL_DECISION.md.
// Live rails remain pluggable via Rail + IsLive fail-closed; ledger rows are
// the source of truth, the file is the bank transport.
//
// Idempotency: one batch per (supplier, period) — unique index — plus a
// caller-supplied idempotency key. Re-generating a period returns the
// existing batch; money never double-pays.

const (
	StatusDraft     = "DRAFT"
	StatusExported  = "EXPORTED"
	StatusSubmitted = "SUBMITTED" // dispatched to a live rail, awaiting settlement webhook
	StatusPaid      = "PAID"
)

var (
	ErrNothingPayable     = errors.New("nothing payable for the period")
	ErrBankDetailsMissing = errors.New("supplier bank details incomplete")
	ErrBatchNotFound      = errors.New("payout batch not found")
)

type Batch struct {
	BatchID            string    `json:"batch_id"`
	SupplierID         string    `json:"supplier_id"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	GrossCapturedMinor int64     `json:"gross_captured_minor"`
	RefundedMinor      int64     `json:"refunded_minor"`
	CommissionMinor    int64     `json:"commission_minor"`
	NetPayoutMinor     int64     `json:"net_payout_minor"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	ExportFileURI      string    `json:"export_file_uri,omitempty"`
	RailReference      string    `json:"rail_reference,omitempty"`
	IdempotencyKey     string    `json:"idempotency_key,omitempty"`
	CreatedBy          string    `json:"created_by,omitempty"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

// CommissionResolver prices the platform commission for a batch. Implemented
// by the billing fee schedule; the zero resolver keeps staging honest (0
// commission, explicitly recorded).

type zeroCommission struct{}

func (zeroCommission) CommissionMinor(context.Context, string, int64, string) (int64, error) {
	return 0, nil
}

type Service struct {
	repo       *Repository
	commission CommissionResolver
	rail       Rail
	cache      *cache.Cache
	now        func() time.Time
	newID      func() string
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:       repo,
		commission: zeroCommission{},
		rail:       BankFileRail{},
		now:        func() time.Time { return time.Now().UTC() },
		newID:      func() string { return fmt.Sprintf("po-%d", time.Now().UnixNano()) },
	}
}

// SetRail swaps the settlement transport (default BankFileRail). Live rails
// are wired here once a bank/payment-rail integration lands.
func (s *Service) SetRail(r Rail) {
	if r != nil {
		s.rail = r
	}
}

func (s *Service) SetCommissionResolver(r CommissionResolver) {
	if r != nil {
		s.commission = r
	}
}

func (s *Service) SetCache(c *cache.Cache) {
	if s != nil {
		s.cache = c
	}
}

// GenerateBatch computes and persists a DRAFT payout batch. Re-generation of
// the same (supplier, period) returns the existing batch (replay-safe).
func (s *Service) GenerateBatch(ctx context.Context, supplierID string, periodStart, periodEnd time.Time, actor, idemKey string) ([]Batch, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("payout service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" || !periodStart.Before(periodEnd) {
		return nil, fmt.Errorf("supplier_id and a valid period required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	idemKey = strings.TrimSpace(idemKey)
	if idemKey == "" {
		idemKey = fmt.Sprintf("payout-%s-%s-%s", supplierID, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	}

	existingList, err := s.repo.ListBySupplierPeriod(ctx, supplierID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	if len(existingList) > 0 {
		return existingList, nil
	}

	batches, err := s.repo.GenerateBatchesInTxn(ctx, supplierID, periodStart, periodEnd, func(sum SumLegsResult) (Batch, error) {
		currency, err := coalescePayoutCurrency(ctx, supplierID, sum.Currency)
		if err != nil {
			return Batch{}, err
		}
		
		return Batch{
			BatchID:            s.newID(),
			SupplierID:         supplierID,
			PeriodStart:        periodStart,
			PeriodEnd:          periodEnd,
			GrossCapturedMinor: sum.Captured,
			RefundedMinor:      sum.Refunded,
			CommissionMinor:    sum.Commission,
			NetPayoutMinor:     sum.NetPayout,
			Currency:           currency,
			Status:             StatusDraft,
			IdempotencyKey:     fmt.Sprintf("%s-%s", idemKey, currency),
			CreatedBy:          actor,
		}, nil
	})
	
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return nil, ErrNothingPayable
	}
	
	return batches, nil
}

// ExportBankFile renders the CSV payment instruction for the batch and marks
// it EXPORTED. Fails closed when supplier bank details are incomplete.
func (s *Service) ExportBankFile(ctx context.Context, supplierID, batchID string) ([]byte, Batch, error) {
	b, err := s.GetBatch(ctx, supplierID, batchID)
	if err != nil {
		return nil, Batch{}, err
	}
	if b.Status == StatusPaid {
		return nil, Batch{}, fmt.Errorf("batch %s already PAID", batchID)
	}
	details, err := s.repo.SupplierBankDetails(ctx, b.SupplierID)
	if err != nil {
		return nil, Batch{}, err
	}
	raw, err := RenderBankFile(b, details)
	if err != nil {
		return nil, Batch{}, err
	}
	if err := s.repo.UpdateStatus(ctx, b.BatchID, StatusExported, ""); err != nil {
		return nil, Batch{}, err
	}
	b.Status = StatusExported
	return raw, b, nil
}

// MarkPaid records the bank's settlement of an exported batch.
func (s *Service) MarkPaid(ctx context.Context, supplierID, batchID string) error {
	b, err := s.GetBatch(ctx, supplierID, batchID)
	if err != nil {
		return err
	}
	if b.Status != StatusExported {
		return fmt.Errorf("batch %s must be EXPORTED before PAID (current %s)", batchID, b.Status)
	}
	return s.repo.UpdateStatus(ctx, batchID, StatusPaid, "")
}

// ListBatches returns supplier-scoped payout batches, newest first.
func (s *Service) ListBatches(ctx context.Context, supplierID string) ([]Batch, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("payout service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required")
	}
	return s.repo.ListBySupplier(ctx, supplierID)
}

// GetBatch returns one batch if it belongs to supplierID.
func (s *Service) GetBatch(ctx context.Context, supplierID, batchID string) (Batch, error) {
	if s == nil || s.repo == nil {
		return Batch{}, fmt.Errorf("payout service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	batchID = strings.TrimSpace(batchID)
	if supplierID == "" || batchID == "" {
		return Batch{}, ErrBatchNotFound
	}
	b, found, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return Batch{}, err
	}
	if !found || b.SupplierID != supplierID {
		return Batch{}, ErrBatchNotFound
	}
	return b, nil
}

// Repository is the Spanner store for payout batches.
type Repository struct {
	client *spanner.Client
}

func NewRepository(client *spanner.Client) *Repository {
	return &Repository{client: client}
}

// SumLegs aggregates provider-confirmed legs for the supplier over the period
// (captured vs refunded reversal legs). Period bounds: [start, end).
type SumLegsResult struct {
	Currency   string
	Captured   int64
	Refunded   int64
	Commission int64
	NetPayout  int64
	SliceIDs   []string
}

// SumLegsResult removed unused SumLegsByCurrency

// coalescePayoutCurrency uses stored ISO code from payment legs, else the shipped pack.
// Planned/unknown packs fail closed — never invent UZS.
func coalescePayoutCurrency(ctx context.Context, supplierID, stored string) (string, error) {
	c, err := auth.CoalesceCurrency(ctx, supplierID, stored)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(c) == "" {
		return "", auth.ErrMarketPackNotShipped
	}
	return c, nil
}
