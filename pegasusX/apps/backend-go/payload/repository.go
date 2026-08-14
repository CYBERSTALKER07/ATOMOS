package payload

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	UpdateOrderAssignment(ctx context.Context, orderID, routeID, driverID string) error
}

// Repository is the mutation seam for payload write paths.
type Repository interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error
	Hydrate(ctx context.Context, supplierID string, s *Service) error
}

type inMemoryRepository struct{}

// NewInMemoryRepository is a fallback for local/testing bootstrap paths.
// Production/ssmr with REQUIRE_INFRA_ADAPTERS forbid silent no-op commits (B2 M-P0-9).
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

// emptyPayloadTx lets local mutate paths run against process memory (Save* no-ops;
// List* returns empty so Service.apply keeps in-process overlays).
type emptyPayloadTx struct{}

func (emptyPayloadTx) ListManifests(context.Context) ([]ManifestRow, error)      { return nil, nil }
func (emptyPayloadTx) SaveManifest(context.Context, ManifestRow) error           { return nil }
func (emptyPayloadTx) ListManifestOrders(context.Context, string) ([]ManifestOrder, error) {
	return nil, nil
}
func (emptyPayloadTx) SaveManifestOrder(context.Context, ManifestOrder, int64) error { return nil }
func (emptyPayloadTx) ListExceptions(context.Context) ([]ManifestException, error) { return nil, nil }
func (emptyPayloadTx) SaveException(context.Context, ManifestException) error     { return nil }
func (emptyPayloadTx) UpdateOrderAssignment(context.Context, string, string, string) error {
	return nil
}

// discardTxnBuffer accepts outbox events without durability (local only).
type discardTxnBuffer struct{}

func (discardTxnBuffer) BufferOutbox(context.Context, outbox.Event) error { return nil }

func (r *inMemoryRepository) RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error {
	if err := memoryRepoBlocked("payload"); err != nil {
		return err
	}
	// Local/dev: execute domain mutate; emit into discard buffer so callers
	// that only check error still run logic without faking durable outbox.
	if fn != nil {
		if err := fn(ctx, emptyPayloadTx{}); err != nil {
			return err
		}
	}
	if emit != nil {
		return emit(discardTxnBuffer{})
	}
	return nil
}

func (r *inMemoryRepository) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}
