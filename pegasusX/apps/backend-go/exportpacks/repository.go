package exportpacks

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Repository persists ExportPacks rows and atomic Outbox + AuditEvents emissions.
type Repository interface {
	CreatePack(ctx context.Context, pack *ExportPack, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) error
	GetPack(ctx context.Context, tenantID, packID string) (*ExportPack, error)
	GetPackByID(ctx context.Context, packID string) (*ExportPack, error)
	ListPacks(ctx context.Context, tenantID, entityID string, limit int) ([]*ExportPack, error)
}
