package roi

import (
	"context"
	"time"
)

// AuditRecordInput carries hash-chained audit metadata recorded for ROI mutations.
type AuditRecordInput struct {
	TenantID   string
	ActorID    string
	ActorType  string
	Role       string
	Action     string
	ObjectType string
	ObjectID   string
	BeforeJSON string
	AfterJSON  string
	ReasonCode string
	TraceID    string
	IP         string
	DeviceID   string
}

// AuditRecorder persists immutable AuditEvents.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
}

// PlatformFeeReader computes platform fees billed/metered for a tenant over [start, end].
type PlatformFeeReader interface {
	ReadPlatformFeesMinor(ctx context.Context, tenantID string, start, end time.Time) (feesMinor int64, currency string, err error)
}

// HistoricalBaselineReader computes historical 90d AR/bad-debt/expired-stock exposure.
type HistoricalBaselineReader interface {
	ReadHistoricalMetrics(ctx context.Context, tenantID string, start, end time.Time) (HistoricalMetrics, error)
}

// Repository defines durable storage for RoiBaselines and RoiEvents.
type Repository interface {
	UpsertBaseline(ctx context.Context, b Baseline) error
	GetLatestBaseline(ctx context.Context, tenantID string) (Baseline, bool, error)

	FindEventByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (RoiEvent, bool, error)
	CreateEvent(ctx context.Context, ev RoiEvent) error
	ListEvents(ctx context.Context, f EventsFilter) ([]RoiEvent, error)
}
