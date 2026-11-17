package recon

import (
	"context"
	"log/slog"
	"time"
)

// NightlyBatchReport summarizes a scheduled multi-supplier reconciliation sweep.
type NightlyBatchReport struct {
	SuppliersScanned int64
	RunsCompleted    int64
	BreaksOpened     int64
	FailedSuppliers  int64
}

// RunNightlyWorker periodically executes RunNightlyBatch across active suppliers until ctx is cancelled.
func (s *Service) RunNightlyWorker(ctx context.Context, interval time.Duration) {
	if s == nil || s.repo == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	s.runNightlySafely(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runNightlySafely(ctx)
		}
	}
}

func (s *Service) runNightlySafely(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	report, err := s.RunNightlyBatch(ctx, time.Now().UTC())
	if err != nil {
		slog.Warn("nightly reconciliation batch error", "err", err)
		return
	}
	slog.Info("nightly reconciliation batch completed",
		"suppliers_scanned", report.SuppliersScanned,
		"runs_completed", report.RunsCompleted,
		"breaks_opened", report.BreaksOpened,
		"failed_suppliers", report.FailedSuppliers,
	)
}

// RunNightlyBatch runs four-way reconciliation once across active suppliers.
func (s *Service) RunNightlyBatch(ctx context.Context, asOf time.Time) (NightlyBatchReport, error) {
	var report NightlyBatchReport
	if s == nil || s.repo == nil {
		return report, nil
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	supplierIDs, err := s.repo.ListActiveSupplierIDs(ctx, 500)
	if err != nil {
		return report, err
	}
	report.SuppliersScanned = int64(len(supplierIDs))

	for _, sid := range supplierIDs {
		if ctx.Err() != nil {
			return report, ctx.Err()
		}
		out, runErr := s.RunReconciliation(ctx, RunRequest{
			EntityID:    sid,
			Mode:        ModeNightly,
			TriggeredBy: "worker:recon_nightly",
		})
		if runErr != nil {
			report.FailedSuppliers++
			slog.Warn("nightly reconciliation failed for supplier", "supplier_id", sid, "err", runErr)
			continue
		}
		report.RunsCompleted++
		report.BreaksOpened += out.Run.BreakCount
	}
	return report, nil
}
