package recon

import (
	"context"
	"time"
)

// Repository abstracts Spanner and in-memory persistence for four-way reconciliation.
type Repository interface {
	LoadSnapshot(ctx context.Context, entityID, periodID, currency string, asOf time.Time) (Snapshot, error)
	SaveRunAndBreaks(ctx context.Context, run ReconciliationRun, breaks []ReconciliationBreak) (ReconciliationRun, []ReconciliationBreak, error)
	ListRuns(ctx context.Context, filter ListRunsFilter) ([]ReconciliationRun, error)
	ListBreaks(ctx context.Context, filter ListBreaksFilter) ([]ReconciliationBreak, error)
	GetBreak(ctx context.Context, entityID, breakID string) (ReconciliationBreak, error)
	UpdateBreak(ctx context.Context, entityID, breakID string, req PatchBreakRequest, now time.Time) (ReconciliationBreak, error)
	ListActiveSupplierIDs(ctx context.Context, limit int) ([]string, error)
}
