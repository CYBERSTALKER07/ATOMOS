package multientity

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// Repository defines durable persistence and read-only queries for multi-entity operations.
type Repository interface {
	ListEntities(ctx context.Context, tenantID, statusFilter string) ([]LegalEntity, error)
	GetEntity(ctx context.Context, tenantID, entityID string) (LegalEntity, bool, error)
	IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error)
	CreateEntity(ctx context.Context, ent LegalEntity, seedGL bool, auditRec AuditRecorder, auditIn AuditRecordInput) (LegalEntity, error)
	PatchEntity(ctx context.Context, tenantID, entityID string, mutate func(cur LegalEntity) (LegalEntity, error), auditRec AuditRecorder, auditIn AuditRecordInput) (LegalEntity, LegalEntity, error)

	ListTransfers(ctx context.Context, tenantID, periodID, entityID string, limit int) ([]IntercompanyTransfer, error)
	CreateTransferAtomic(
		ctx context.Context,
		tr IntercompanyTransfer,
		debitJE, creditJE gl.JournalEntryInput,
		auditRec AuditRecorder,
		auditIn AuditRecordInput,
	) (IntercompanyTransfer, bool, error)

	ListFxRevaluationRuns(ctx context.Context, tenantID, periodID, entityID string, limit int) ([]FxRevaluationRun, error)
	CollectOpenForeignCurrencyPositions(
		ctx context.Context,
		tenantID, entityID, baseCurrency string,
		asOf time.Time,
	) ([]FxRevalPosition, error)
	CreateFxRevaluationRunAtomic(
		ctx context.Context,
		run FxRevaluationRun,
		journalIn *gl.JournalEntryInput,
		auditRec AuditRecorder,
		auditIn AuditRecordInput,
	) (FxRevaluationRun, bool, error)

	ReadConsolidationSnapshot(ctx context.Context, tenantID, periodID string, entityIDs []string) (ConsolidationSnapshot, error)
}
