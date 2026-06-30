package factory

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// FactoryTx provides granular data access within a transaction.
type FactoryTx interface {
	ListManifests(ctx context.Context) ([]ManifestRow, error)
	SaveManifest(ctx context.Context, m ManifestRow) error
	ListTransfers(ctx context.Context) ([]TransferRow, error)
	SaveTransfer(ctx context.Context, t TransferRow) error
}

// Repository is the mutation seam for factory write paths.
type Repository interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error
	UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error
	Hydrate(ctx context.Context, factoryID string, s *Service) error
	CreateFactory(ctx context.Context, f Factory) error
	GetFactory(ctx context.Context, factoryID string) (Factory, error)
	UpdateFactory(ctx context.Context, f Factory) error
	ListFactories(ctx context.Context, supplierID string, limit, offset int) ([]Factory, error)
}

// inMemoryRepository is a no-op fallback
type inMemoryRepository struct{}

func (m *inMemoryRepository) CreateFactory(ctx context.Context, f Factory) error { return nil }
func (m *inMemoryRepository) GetFactory(ctx context.Context, factoryID string) (Factory, error) { return Factory{}, nil }
func (m *inMemoryRepository) UpdateFactory(ctx context.Context, f Factory) error { return nil }
func (m *inMemoryRepository) ListFactories(ctx context.Context, supplierID string, limit, offset int) ([]Factory, error) { return nil, nil }


func NewInMemoryRepository() Repository {
	return &inMemoryRepository{}
}

func (r *inMemoryRepository) RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *inMemoryRepository) UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *inMemoryRepository) Hydrate(ctx context.Context, factoryID string, s *Service) error {
	return nil
}
