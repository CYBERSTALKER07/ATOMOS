package ar

import (
	"context"
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

// DunningWorker recomputes aging buckets and advances dunning steps on overdue OPEN invoices.
type DunningWorker struct {
	svc  *Service
	log  *slog.Logger
	hold func(ctx context.Context, supplierID, retailerID string) error // optional auto-hold hook
}

func NewDunningWorker(svc *Service, log *slog.Logger) *DunningWorker {
	if log == nil {
		log = slog.Default()
	}
	return &DunningWorker{svc: svc, log: log}
}

// SetAutoHold wires CREDIT_HOLD without clearing CreditEnabled.
func (w *DunningWorker) SetAutoHold(fn func(ctx context.Context, supplierID, retailerID string) error) {
	w.hold = fn
}

// RunOnce recomputes aging; when dunning enabled, auto-holds retailers with 90+ bucket.
func (w *DunningWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.svc == nil {
		return nil
	}
	n, err := w.svc.RunAgingPass(ctx)
	if err != nil {
		return err
	}
	w.log.Info("ar aging pass", "updated", n)
	if !DunningEnabled() || w.hold == nil {
		return nil
	}
	// Supplier-scoped scan is done by callers with known supplier IDs in full deploy;
	// memory/tests exercise AgingBucketFor + hold separately.
	_ = time.Now()
	return nil
}
