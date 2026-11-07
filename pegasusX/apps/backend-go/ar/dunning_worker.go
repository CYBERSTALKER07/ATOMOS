package ar

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// DunningEnabled gates collections escalation worker (SSMR after AR_INVOICES_ENABLED).
func DunningEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AR_DUNNING_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// DunningNotifyFunc fans out inbox/FCM for a dunning step transition.
type DunningNotifyFunc func(ctx context.Context, inv Invoice, prevStep, nextStep int64) error

// DunningDelinquencyFunc increments RetailerCreditProfiles.DelinquencyCount once per first overdue.
type DunningDelinquencyFunc func(ctx context.Context, supplierID, retailerID string) error

// DunningWorker recomputes aging buckets and advances dunning steps on overdue OPEN invoices.
type DunningWorker struct {
	svc         *Service
	log         *slog.Logger
	hold        func(ctx context.Context, supplierID, retailerID string) error
	notify      DunningNotifyFunc
	delinquency DunningDelinquencyFunc
	now         func() time.Time
}

func NewDunningWorker(svc *Service, log *slog.Logger) *DunningWorker {
	if log == nil {
		log = slog.Default()
	}
	return &DunningWorker{svc: svc, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// SetAutoHold wires CREDIT_HOLD without clearing CreditEnabled.
func (w *DunningWorker) SetAutoHold(fn func(ctx context.Context, supplierID, retailerID string) error) {
	w.hold = fn
}

// SetNotify wires retailer/supplier notification fanout.
func (w *DunningWorker) SetNotify(fn DunningNotifyFunc) {
	w.notify = fn
}

// SetDelinquencyBump wires DelinquencyCount increment on first OVERDUE.
func (w *DunningWorker) SetDelinquencyBump(fn DunningDelinquencyFunc) {
	w.delinquency = fn
}

// RunOnce recomputes aging and advances dunning steps when AR_DUNNING_ENABLED.
func (w *DunningWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.svc == nil || w.svc.repo == nil {
		return nil
	}
	n, err := w.svc.RunAgingPass(ctx)
	if err != nil {
		return err
	}
	w.log.Info("ar aging pass", "updated", n)
	if !DunningEnabled() || !InvoicesEnabled() {
		return nil
	}
	advanced, held, bumped, err := w.AdvanceDunning(ctx, 500)
	if err != nil {
		return err
	}
	w.log.Info("ar dunning pass", "advanced", advanced, "holds", held, "delinquency_bumps", bumped)
	return nil
}

// AdvanceDunning applies DesiredDunningStep to open invoices (monotonic step increases only).
func (w *DunningWorker) AdvanceDunning(ctx context.Context, limit int) (advanced, held, bumped int, err error) {
	now := w.now()
	if w.svc.now != nil {
		now = w.svc.now()
	}
	invoices, err := w.svc.repo.ListOpenForDunning(ctx, limit)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, inv := range invoices {
		next := DesiredDunningStep(inv.DueAt, inv.GracePeriodDays, now)
		if next <= inv.DunningStep {
			continue
		}
		prev := inv.DunningStep
		bucket := AgingBucketFor(inv.DueAt, now)
		if err := w.svc.repo.UpdateDunning(ctx, inv.InvoiceID, next, bucket, now, inv.Version); err != nil {
			w.log.Warn("dunning update failed", "invoice_id", inv.InvoiceID, "err", err)
			continue
		}
		advanced++
		inv.DunningStep = next
		inv.AgingBucket = bucket
		inv.LastDunnedAt = now

		if ShouldBumpDelinquency(prev, next) && w.delinquency != nil {
			if err := w.delinquency(ctx, inv.SupplierID, inv.RetailerID); err != nil {
				w.log.Warn("delinquency bump failed", "retailer_id", inv.RetailerID, "err", err)
			} else {
				bumped++
			}
		}
		if ShouldAutoHold(prev, next) && w.hold != nil {
			if err := w.hold(ctx, inv.SupplierID, inv.RetailerID); err != nil {
				w.log.Warn("auto-hold failed", "retailer_id", inv.RetailerID, "err", err)
			} else {
				held++
			}
		}
		if w.notify != nil {
			if err := w.notify(ctx, inv, prev, next); err != nil {
				w.log.Warn("dunning notify failed", "invoice_id", inv.InvoiceID, "err", err)
			}
		}
	}
	return advanced, held, bumped, nil
}

// Start runs until context cancel.
func (w *DunningWorker) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				w.log.Warn("dunning worker pass failed", "err", err)
			}
		}
	}
}

// NotifyMessage builds inbox title/body for a step.
func NotifyMessage(inv Invoice, nextStep int64) (eventType, title, body string) {
	name := StepName(nextStep)
	eventType = "AR_DUNNING_" + name
	title = fmt.Sprintf("Invoice %s: %s", inv.InvoiceID, name)
	body = fmt.Sprintf("Order %s balance %d %s — dunning step %s (due %s)",
		inv.OrderID, inv.BalanceMinor, inv.Currency, name, inv.DueAt.Format("2006-01-02"))
	return eventType, title, body
}
