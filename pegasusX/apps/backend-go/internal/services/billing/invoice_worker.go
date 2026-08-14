package billing

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
)

// InvoiceWorker turns the wired meter + fee schedule into monthly AR open
// items (SupplierId="PLATFORM"), reusing the AR aging/dunning engine. One
// invoice per supplier per month — idempotent via the billing pseudo-order id.
type InvoiceWorker struct {
	client   *spanner.Client
	ar       *ar.Service
	resolver *FeeScheduleResolver
	log      *slog.Logger
	now      func() time.Time
}

func NewInvoiceWorker(client *spanner.Client, arSvc *ar.Service, resolver *FeeScheduleResolver, log *slog.Logger) *InvoiceWorker {
	return &InvoiceWorker{
		client: client, ar: arSvc, resolver: resolver, log: log,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// GenerateMonthlyInvoices bills every supplier with metered activity in the
// given month. month is truncated to its first day.
func (w *InvoiceWorker) GenerateMonthlyInvoices(ctx context.Context, month time.Time) (int, error) {
	if w == nil || w.client == nil {
		return 0, fmt.Errorf("billing invoice worker unavailable")
	}
	if w.ar == nil {
		return 0, fmt.Errorf("billing requires AR service (AR_INVOICES_ENABLED)")
	}
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	next := first.AddDate(0, 1, 0)

	suppliers, err := w.activeSuppliers(ctx, first, next)
	if err != nil {
		return 0, err
	}
	billed := 0
	var errs []string
	for _, supplierID := range suppliers {
		invoiceID, err := w.GenerateMonthlyInvoice(ctx, supplierID, first)
		if err != nil {
			errs = append(errs, supplierID+": "+err.Error())
			w.log.ErrorContext(ctx, "monthly billing failed", "supplier_id", supplierID, "month", first.Format("2006-01"), "err", err)
			continue
		}
		if invoiceID != "" {
			billed++
		}
	}
	if len(errs) > 0 {
		return billed, fmt.Errorf("billing errors: %s", strings.Join(errs, "; "))
	}
	return billed, nil
}

// GenerateMonthlyInvoice computes the fee and opens the AR open item.
// Returns "" when the supplier's schedule prices zero (billing off / free).
func (w *InvoiceWorker) GenerateMonthlyInvoice(ctx context.Context, supplierID string, month time.Time) (string, error) {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	next := first.AddDate(0, 1, 0)
	// ArInvoices.OrderId is STRING(36): deterministic short key per supplier+month.
	sum := sha1.Sum([]byte(supplierID))
	billingKey := fmt.Sprintf("bill-%x-%s", sum[:6], first.Format("200601"))

	sched, err := w.resolver.Resolve(ctx, supplierID)
	if err != nil {
		return "", err
	}
	orderCount, gmvMinor, err := w.monthTotals(ctx, supplierID, first, next)
	if err != nil {
		return "", err
	}
	fee := sched.MonthlyFee(orderCount, gmvMinor)
	if fee <= 0 {
		return "", nil
	}

	monthEnd := next.Add(-time.Second)
	inv, err := w.ar.OpenFromCreditLeave(ctx, ar.OpenFromCreditLeaveRequest{
		SupplierID:    "PLATFORM",
		RetailerID:    supplierID,
		OrderID:       billingKey,
		AmountMinor:   fee,
		Currency:      sched.Currency,
		TermsDays:     14,
		GraceDays:     3,
		CreditLeaveAt: monthEnd,
		DueAt:         monthEnd.AddDate(0, 0, 14),
	})
	if err != nil {
		return "", fmt.Errorf("open billing AR item: %w", err)
	}
	if inv.InvoiceID == "" {
		return "", fmt.Errorf("AR_INVOICES_ENABLED off: billing invoice for %s not opened", supplierID)
	}
	w.log.InfoContext(ctx, "monthly billing invoice opened",
		"supplier_id", supplierID, "month", first.Format("2006-01"),
		"invoice_id", inv.InvoiceID, "fee_minor", fee,
		"orders", orderCount, "gmv_minor", gmvMinor,
		"tier", sched.Tier, "schedule_id", sched.FeeScheduleID)
	return inv.InvoiceID, nil
}

func (w *InvoiceWorker) activeSuppliers(ctx context.Context, start, end time.Time) ([]string, error) {
	iter := w.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DISTINCT SupplierId FROM BillingMeterEvents
		      WHERE ProcessedAt >= @s AND ProcessedAt < @e`,
		Params: map[string]any{"s": start, "e": end},
	})
	defer iter.Stop()
	var out []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var id string
		if err := row.Column(0, &id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
}

// monthTotals aggregates metered order count and GMV. The meter stores float
// major units (legacy); convert to minor by the currency's 100x scale — the
// only currency in production (UZS) is 2-decimal, and the fee math is exact
// after rounding to minor units at the source boundary.
func (w *InvoiceWorker) monthTotals(ctx context.Context, supplierID string, start, end time.Time) (orderCount, gmvMinor int64, err error) {
	iter := w.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*), COALESCE(SUM(Amount), 0) FROM BillingMeterEvents
		      WHERE SupplierId = @sid AND ProcessedAt >= @s AND ProcessedAt < @e`,
		Params: map[string]any{"sid": supplierID, "s": start, "e": end},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, 0, err
	}
	var gmvMajor float64
	if err := row.Columns(&orderCount, &gmvMajor); err != nil {
		return 0, 0, err
	}
	return orderCount, int64(gmvMajor*100 + 0.5), nil
}

type PlatformInvoice struct {
	InvoiceID      string    `json:"invoice_id"`
	BilledSupplier string    `json:"billed_supplier_id"`
	OrderID        string    `json:"order_id"`
	Status         string    `json:"status"`
	PrincipalMinor int64     `json:"principal_minor"`
	BalanceMinor   int64     `json:"balance_minor"`
	Currency       string    `json:"currency"`
	DueAt          time.Time `json:"due_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// ListPlatformInvoices returns PLATFORM AR invoices (monthly billing). Empty slice when none.
func (w *InvoiceWorker) ListPlatformInvoices(ctx context.Context, limit int) ([]PlatformInvoice, error) {
	if w == nil || w.client == nil {
		return nil, fmt.Errorf("billing unavailable")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	iter := w.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT InvoiceId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, DueAt, CreatedAt
		      FROM ArInvoices
		      WHERE SupplierId = 'PLATFORM'
		      ORDER BY CreatedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{"limit": int64(limit)},
	})
	defer iter.Stop()
	out := make([]PlatformInvoice, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var inv PlatformInvoice
		if err := row.Columns(&inv.InvoiceID, &inv.BilledSupplier, &inv.OrderID, &inv.Status,
			&inv.PrincipalMinor, &inv.BalanceMinor, &inv.Currency, &inv.DueAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
}

type FeeScheduleRow struct {
	FeeScheduleID            string     `json:"fee_schedule_id"`
	SupplierID               string     `json:"supplier_id"`
	Tier                     string     `json:"tier"`
	PerOrderMinor            int64      `json:"per_order_minor"`
	GmvBps                   int64      `json:"gmv_bps"`
	MonthlySubscriptionMinor int64      `json:"monthly_subscription_minor"`
	Currency                 string     `json:"currency"`
	EffectiveFrom            time.Time  `json:"effective_from"`
	EffectiveTo              *time.Time `json:"effective_to,omitempty"`
}

// ListFeeSchedules returns billing schedules. Empty slice when none (honest skip, not invented charges).
func (w *InvoiceWorker) ListFeeSchedules(ctx context.Context, limit int) ([]FeeScheduleRow, error) {
	if w == nil || w.client == nil {
		return nil, fmt.Errorf("billing unavailable")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	iter := w.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FeeScheduleId, SupplierId, Tier, PerOrderMinor, GmvBps, MonthlySubscriptionMinor, Currency, EffectiveFrom, EffectiveTo
		      FROM BillingFeeSchedules
		      ORDER BY EffectiveFrom DESC
		      LIMIT @limit`,
		Params: map[string]any{"limit": int64(limit)},
	})
	defer iter.Stop()
	out := make([]FeeScheduleRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var s FeeScheduleRow
		var effTo spanner.NullTime
		if err := row.Columns(&s.FeeScheduleID, &s.SupplierID, &s.Tier, &s.PerOrderMinor, &s.GmvBps,
			&s.MonthlySubscriptionMinor, &s.Currency, &s.EffectiveFrom, &effTo); err != nil {
			return nil, err
		}
		if effTo.Valid {
			t := effTo.Time
			s.EffectiveTo = &t
		}
		out = append(out, s)
	}
}
