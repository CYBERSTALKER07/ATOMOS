package connectors

import "context"

// Repository persists ExternalCredentials, ConnectorSyncRuns, ConnectorEntityMaps,
// and loads tenant Spanner snapshots for ShadowCompare.
type Repository interface {
	UpsertCredential(ctx context.Context, cred ExternalCredential) (ExternalCredential, error)
	GetCredential(ctx context.Context, tenantID, provider, accountLabel, credentialID string) (ExternalCredential, bool, error)
	ListCredentials(ctx context.Context, tenantID, provider string) ([]ExternalCredential, error)

	FindRunByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (ConnectorSyncRun, bool, error)
	SaveSyncRunAndMaps(ctx context.Context, run ConnectorSyncRun, maps []ConnectorEntityMap, emissions []OutboxEmission) error
	GetSyncRun(ctx context.Context, tenantID, runID string) (ConnectorSyncRun, bool, error)
	ListSyncRuns(ctx context.Context, tenantID, provider string, limit int) ([]ConnectorSyncRun, error)
	ListLatestRunsByProvider(ctx context.Context, tenantID string) ([]ConnectorSyncRun, error)

	LoadInternalSnapshot(ctx context.Context, tenantID, provider string) (InternalSnapshot, error)
}
