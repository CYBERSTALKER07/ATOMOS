package payout

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Supplier payout batches: NetPayout = Σcaptured − Σrefunds − commission,
// computed over provider-confirmed payment legs in the period. The batch is
// executed via bank-file export (CSV payment instruction) until a payout rail
// exists — the ledger rows are the source of truth, the file is just the
// transport to the bank.
//
// Idempotency: one batch per (supplier, period) — unique index — plus a
// caller-supplied idempotency key. Re-generating a period returns the
// existing batch; money never double-pays.

const (
	StatusDraft    = "DRAFT"
	StatusExported = "EXPORTED"
	StatusPaid     = "PAID"
)

var (
	ErrNothingPayable  = errors.New("nothing payable for the period")
	ErrBankDetailsMissing = errors.New("supplier bank details incomplete")
	ErrBatchNotFound   = errors.New("payout batch not found")
)

type Batch struct {
	BatchID            string
	SupplierID         string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	GrossCapturedMinor int64
	RefundedMinor      int64
	CommissionMinor    int64
	NetPayoutMinor     int64
	Currency           string
	Status             string
	ExportFileURI      string
	IdempotencyKey     string
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CommissionResolver prices the platform commission for a batch. Implemented
// by the billing fee schedule; the zero resolver keeps staging honest (0
// commission, explicitly recorded).
type CommissionResolver interface {
	CommissionMinor(ctx context.Context, supplierID string, grossCapturedMinor int64, currency string) (int64, error)
}

type zeroCommission struct{}

func (zeroCommission) CommissionMinor(context.Context, string, int64, string) (int64, error) {
	return 0, nil
}

type Service struct {
	repo       *Repository
	commission CommissionResolver
	now        func() time.Time
	newID      func() string
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:       repo,
		commission: zeroCommission{},
		now:        func() time.Time { return time.Now().UTC() },
		newID:      func() string { return fmt.Sprintf("po-%d", time.Now().UnixNano()) },
	}
}

func (s *Service) SetCommissionResolver(r CommissionResolver) {
	if r != nil {
		s.commission = r
	}
}

// GenerateBatch computes and persists a DRAFT payout batch. Re-generation of
// the same (supplier, period) returns the existing batch (replay-safe).
func (s *Service) GenerateBatch(ctx context.Context, supplierID string, periodStart, periodEnd time.Time, actor, idemKey string) (Batch, error) {
	if s == nil || s.repo == nil {
		return Batch{}, fmt.Errorf("payout service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" || !periodStart.Before(periodEnd) {
		return Batch{}, fmt.Errorf("supplier_id and a valid period required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	idemKey = strings.TrimSpace(idemKey)
	if idemKey == "" {
		idemKey = fmt.Sprintf("payout-%s-%s-%s", supplierID, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	}

	if existing, found, err := s.repo.GetBySupplierPeriod(ctx, supplierID, periodStart, periodEnd); err != nil {
		return Batch{}, err
	} else if found {
		return existing, nil
	}
	if existing, found, err := s.repo.GetByIdempotencyKey(ctx, idemKey); err != nil {
		return Batch{}, err
	} else if found {
		return existing, nil
	}

	captured, refunded, currency, err := s.repo.SumLegs(ctx, supplierID, periodStart, periodEnd)
	if err != nil {
		return Batch{}, err
	}
	commission, err := s.commission.CommissionMinor(ctx, supplierID, captured, currency)
	if err != nil {
		return Batch{}, fmt.Errorf("commission: %w", err)
	}
	net := captured - refunded - commission
	if net <= 0 {
		return Batch{}, fmt.Errorf("%w: captured=%d refunded=%d commission=%d", ErrNothingPayable, captured, refunded, commission)
	}
	if currency == "" {
		currency = "UZS"
	}

	b := Batch{
		BatchID:            s.newID(),
		SupplierID:         supplierID,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		GrossCapturedMinor: captured,
		RefundedMinor:      refunded,
		CommissionMinor:    commission,
		NetPayoutMinor:     net,
		Currency:           currency,
		Status:             StatusDraft,
		IdempotencyKey:     idemKey,
		CreatedBy:          actor,
	}
	if err := s.repo.Insert(ctx, b); err != nil {
		return Batch{}, err
	}
	return b, nil
}

// ExportBankFile renders the CSV payment instruction for the batch and marks
// it EXPORTED. Fails closed when supplier bank details are incomplete.
func (s *Service) ExportBankFile(ctx context.Context, batchID string) ([]byte, Batch, error) {
	b, found, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return nil, Batch{}, err
	}
	if !found {
		return nil, Batch{}, ErrBatchNotFound
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
func (s *Service) MarkPaid(ctx context.Context, batchID string) error {
	b, found, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return err
	}
	if !found {
		return ErrBatchNotFound
	}
	if b.Status != StatusExported {
		return fmt.Errorf("batch %s must be EXPORTED before PAID (current %s)", batchID, b.Status)
	}
	return s.repo.UpdateStatus(ctx, batchID, StatusPaid, "")
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
func (r *Repository) SumLegs(ctx context.Context, supplierID string, start, end time.Time) (captured, refunded int64, currency string, err error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT l.Method, l.AmountMinor, o.Currency
		      FROM OrderPaymentLegs l
		      JOIN Orders o ON o.OrderId = l.OrderId
		      WHERE o.SupplierId = @sid AND l.Status = 'CAPTURED'
		        AND l.CapturedAt >= @start AND l.CapturedAt < @end`,
		Params: map[string]any{"sid": supplierID, "start": start, "end": end},
	})
	defer iter.Stop()
	for {
		row, e := iter.Next()
		if e == iterator.Done {
			return captured, refunded, currency, nil
		}
		if e != nil {
			return 0, 0, "", e
		}
		var method, cur string
		var amount int64
		if e := row.Columns(&method, &amount, &cur); e != nil {
			return 0, 0, "", e
		}
		if currency == "" {
			currency = cur
		}
		if method == "REFUND" {
			refunded += amount
		} else {
			captured += amount
		}
	}
}
