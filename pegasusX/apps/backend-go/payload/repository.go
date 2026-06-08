package payload

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// PayloadTx provides granular access to payload entities within a transaction.
type PayloadTx interface {
	ListManifests(ctx context.Context) ([]ManifestRow, error)
	SaveManifest(ctx context.Context, m ManifestRow) error
	ListManifestOrders(ctx context.Context, manifestID string) ([]ManifestOrder, error)
	SaveManifestOrder(ctx context.Context, mo ManifestOrder, seq int64) error
	ListExceptions(ctx context.Context) ([]ManifestException, error)
	SaveException(ctx context.Context, e ManifestException) error
}

// Repository is the mutation seam for payload write paths.
type Repository interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error
	Hydrate(ctx context.Context, supplierID string, s *Service) error
}
