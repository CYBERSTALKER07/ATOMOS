package factory

import (
	"context"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Repository is the mutation seam for factory write paths.
type Repository interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error
	UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error
}

// FactoryTx represents a granular ReadWriteTransaction for the factory domain.
type FactoryTx interface {
	// Manifests
	GetManifest(ctx context.Context, manifestID string) (ManifestRow, error)
	ListManifests(ctx context.Context) ([]ManifestRow, error)
	SaveManifest(ctx context.Context, m ManifestRow) error

	// Transfers
	GetTransfer(ctx context.Context, transferID string) (TransferRow, error)
	ListTransfers(ctx context.Context) ([]TransferRow, error)
	ListManifestTransfers(ctx context.Context, manifestID string) ([]TransferRow, error)
	GetUnassignedTransfers(ctx context.Context) ([]TransferRow, error)
	SaveTransfer(ctx context.Context, t TransferRow) error

	// Transitions & Exceptions
	SaveManifestTransition(ctx context.Context, manifestID string, t ManifestTransition) error
	SaveManifestReassignment(ctx context.Context, r ManifestReassignment) error
	SaveManifestException(ctx context.Context, e ManifestException) error
}
