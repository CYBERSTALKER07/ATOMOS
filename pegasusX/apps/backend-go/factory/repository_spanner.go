package factory

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository persists factory manifest state and outbox events in one RW transaction.
type SpannerRepository struct {
	client      *spanner.Client
	store       *manifest.Store
	supplierID  string
	factoryNode string
}

// NewSpannerRepository constructs the Spanner backend for factory.
func NewSpannerRepository(client *spanner.Client, supplierID, factoryNodeID string) *SpannerRepository {
	return &SpannerRepository{
		client:      client,
		store:       manifest.NewStore(client),
		supplierID:  supplierID,
		factoryNode: factoryNodeID,
	}
}

// Apply runs the in-memory mutation, then atomically projects the snapshot and outbox rows.
func (r *SpannerRepository) Apply(
	ctx context.Context,
	mutate func() error,
	snapshot func() *PersistenceSnapshot,
	emit func(outbox.TxnBuffer) error,
) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner factory repository: nil client")
	}
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	var snap *PersistenceSnapshot
	if snapshot != nil {
		snap = snapshot()
	}
	if snap == nil {
		snap = &PersistenceSnapshot{}
	}
	return r.store.CommitFactory(ctx, factoryBatchFromSnapshot(r.supplierID, r.factoryNode, snap), emit)
}

// Hydrate loads durable factory manifests into the service when present.
func (r *SpannerRepository) Hydrate(ctx context.Context, s *Service) error {
	if r == nil || r.store == nil || s == nil {
		return fmt.Errorf("hydrate factory: invalid dependencies")
	}
	rows, transferRows, err := r.listFactoryState(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return r.applyHydrateLocked(s, rows, transferRows)
}

// hydrateWhileLocked loads durable manifests; caller must already hold s.mu.
func (r *SpannerRepository) hydrateWhileLocked(ctx context.Context, s *Service) error {
	if r == nil || r.store == nil || s == nil {
		return fmt.Errorf("hydrate factory: invalid dependencies")
	}
	rows, transferRows, err := r.listFactoryState(ctx)
	if err != nil {
		return err
	}
	return r.applyHydrateLocked(s, rows, transferRows)
}

func (r *SpannerRepository) listFactoryState(ctx context.Context) ([]manifest.FactoryTruckRow, []manifest.FactoryTransferRow, error) {
	rows, err := r.store.ListFactoryManifests(ctx, r.factoryNode)
	if err != nil {
		return nil, nil, err
	}
	transferRows, err := r.store.ListFactoryTransfers(ctx, r.factoryNode)
	if err != nil {
		return nil, nil, err
	}
	return rows, transferRows, nil
}

func (r *SpannerRepository) applyHydrateLocked(s *Service, rows []manifest.FactoryTruckRow, transferRows []manifest.FactoryTransferRow) error {
	if len(rows) == 0 && len(transferRows) == 0 {
		return nil
	}
	if len(rows) > 0 {
		s.manifests = make([]ManifestRow, 0, len(rows))
		for _, row := range rows {
			s.manifests = append(s.manifests, manifestRowFromFactoryTruck(row))
		}
	}
	if len(transferRows) > 0 {
		s.transfers = make([]TransferRow, 0, len(transferRows))
		for _, row := range transferRows {
			s.transfers = append(s.transfers, transferRowFromFactoryTransfer(row))
		}
		s.rebuildManifestTransfersLocked()
	}
	s.spannerLoaded = len(s.manifests) > 0 || len(s.transfers) > 0
	return nil
}

// SeedDemoManifests writes demo factory manifests and transfers when missing in Spanner.
func (r *SpannerRepository) SeedDemoManifests(ctx context.Context, snap *PersistenceSnapshot) error {
	if r == nil || r.store == nil || snap == nil {
		return nil
	}
	batch := factoryBatchFromSnapshot(r.supplierID, r.factoryNode, snap)
	if err := r.store.EnsureFactoryDemoManifests(ctx, batch.Manifests); err != nil {
		return err
	}
	return r.store.EnsureFactoryDemoTransfers(ctx, batch.Transfers)
}
