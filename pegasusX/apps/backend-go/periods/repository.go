package periods

import "context"

// PeriodTransition describes an atomic state change on a Periods row accompanied by
// an outbox event and immutable AuditEvents row.
type PeriodTransition struct {
	NextPeriod   Period
	EventType    string
	EventReason  string
	ApprovalID   string
	AutoInitOpen bool
}

// Repository defines durable storage for M1.1 accounting periods, live checklists, and close snapshots.
type Repository interface {
	CreatePeriod(ctx context.Context, p Period, auditRec AuditRecorder, auditIn AuditRecordInput) (Period, error)
	GetPeriod(ctx context.Context, entityID, periodID string) (Period, bool, error)
	ListPeriods(ctx context.Context, entityID, statusFilter string) ([]Period, error)
	TransitionPeriod(
		ctx context.Context,
		entityID, periodID string,
		mutateFn func(existing *Period, found bool) (PeriodTransition, error),
		auditRec AuditRecorder,
		auditIn AuditRecordInput,
	) (Period, error)

	EvaluateChecklist(ctx context.Context, entityID, periodID, tzName string) (ChecklistReport, error)
	BuildCloseSnapshotData(ctx context.Context, entityID, periodID, tzName, closedBy string) (CloseSnapshot, error)
}
