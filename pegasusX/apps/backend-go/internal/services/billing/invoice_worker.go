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
