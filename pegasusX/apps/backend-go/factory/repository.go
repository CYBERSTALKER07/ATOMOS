package factory

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// FactoryTx provides granular data access within a transaction.
type FactoryTx interface {
	ListManifests(ctx context.Context) ([]ManifestRow, error)
	SaveManifest(ctx context.Context, m ManifestRow) error
	ListTransfers(ctx context.Context) ([]TransferRow, error)
	SaveTransfer(ctx context.Context, t TransferRow) error
	SaveStaff(ctx context.Context, row StaffRow) error
	ResolveException(ctx context.Context, row ManifestException, orderID string) error
}

// Repository is the mutation seam for factory write paths.
type Repository interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error
	UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error
	Hydrate(ctx context.Context, factoryID string, s *Service) error
	CreateFactory(ctx context.Context, f Factory, emit func(outbox.TxnBuffer) error) error
	GetFactory(ctx context.Context, factoryID string) (Factory, error)
	UpdateFactory(ctx context.Context, f Factory, emit func(outbox.TxnBuffer) error) error
	ListFactories(ctx context.Context, supplierID string, limit, offset int) ([]Factory, error)
}

// inMemoryRepository is a local/testing fallback. Production/ssmr forbid silent
// no-op commits (B2 M-P0-9).
type inMemoryRepository struct{}

func NewInMemoryRepository() Repository {
	return &inMemoryRepository{}
}

func memoryRepoBlocked(component string) error {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")))
	if env == "production" || env == "prod" || env == "ssmr" {
		return fmt.Errorf("%s: in-memory repository cannot commit mutations under PEGASUSX_ENV=%s; configure Spanner", component, env)
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_INFRA_ADAPTERS")), "true") {
		return fmt.Errorf("%s: in-memory repository blocked when REQUIRE_INFRA_ADAPTERS=true", component)
	}
	return nil
}

type emptyFactoryTx struct{}

func (emptyFactoryTx) ListManifests(context.Context) ([]ManifestRow, error)  { return nil, nil }
func (emptyFactoryTx) SaveManifest(context.Context, ManifestRow) error       { return nil }
func (emptyFactoryTx) ListTransfers(context.Context) ([]TransferRow, error)  { return nil, nil }
func (emptyFactoryTx) SaveTransfer(context.Context, TransferRow) error       { return nil }
func (emptyFactoryTx) SaveStaff(context.Context, StaffRow) error             { return nil }
func (emptyFactoryTx) ResolveException(context.Context, ManifestException, string) error {
	return nil
}

type discardTxnBuffer struct{}

func (discardTxnBuffer) BufferOutbox(context.Context, outbox.Event) error { return nil }

func (m *inMemoryRepository) CreateFactory(ctx context.Context, f Factory, emit func(outbox.TxnBuffer) error) error {
	if err := memoryRepoBlocked("factory"); err != nil {
		return err
	}
	if emit != nil {
		return emit(discardTxnBuffer{})
	}
	return nil
}
func (m *inMemoryRepository) GetFactory(ctx context.Context, factoryID string) (Factory, error) {
	return Factory{}, fmt.Errorf("factory not found (in-memory)")
}
func (m *inMemoryRepository) UpdateFactory(ctx context.Context, f Factory, emit func(outbox.TxnBuffer) error) error {
	if err := memoryRepoBlocked("factory"); err != nil {
		return err
	}
	if emit != nil {
		return emit(discardTxnBuffer{})
	}
	return nil
}
func (m *inMemoryRepository) ListFactories(ctx context.Context, supplierID string, limit, offset int) ([]Factory, error) {
	return nil, nil
}

func (r *inMemoryRepository) RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error {
	if err := memoryRepoBlocked("factory"); err != nil {
		return err
	}
	if fn != nil {
		if err := fn(ctx, emptyFactoryTx{}); err != nil {
			return err
		}
	}
	if emit != nil {
		return emit(discardTxnBuffer{})
	}
	return nil
}

func (r *inMemoryRepository) UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error {
	if err := memoryRepoBlocked("factory"); err != nil {
		return err
	}
	if emit != nil {
		return emit(discardTxnBuffer{})
	}
	return nil
}

func (r *inMemoryRepository) Hydrate(ctx context.Context, factoryID string, s *Service) error {
	return nil
}
