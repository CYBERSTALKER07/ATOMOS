package anomalies

import (
	"context"
	"time"
)

// Repository abstracts persistence and live dataset loading for §7.7 Anomaly & Fraud Cases.
type Repository interface {
	LoadDetectorDataset(ctx context.Context, tenantID, entityID string) (DetectorDataset, error)
	SaveDetectedCases(
		ctx context.Context,
		tenantID string,
		candidates []AnomalyCase,
		auditRec AuditRecorder,
		auditIn AuditRecordInput,
	) (created []AnomalyCase, deduplicatedCount int, err error)
	ListCases(ctx context.Context, f ListFilter) ([]AnomalyCase, error)
	GetCase(ctx context.Context, tenantID, caseID string) (AnomalyCase, bool, error)
	ResolveCase(
		ctx context.Context,
		tenantID, caseID string,
		mutate func(cur AnomalyCase) (AnomalyCase, error),
		auditRec AuditRecorder,
		auditIn func(updated AnomalyCase) AuditRecordInput,
		resolvedAt time.Time,
	) (AnomalyCase, error)
}
