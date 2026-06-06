package payload

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository persists payload manifest state and outbox events in one RW transaction.
type SpannerRepository struct {
	client     *spanner.Client
	store      *manifest.Store
	supplierID string
}

// NewSpannerRepository constructs the Spanner backend for payload.
func NewSpannerRepository(client *spanner.Client, supplierID string) *SpannerRepository {
	return &SpannerRepository{
		client:     client,
		store:      manifest.NewStore(client),
		supplierID: supplierID,
	}
}

// Store exposes the underlying manifest store for hydration and seeding.
func (r *SpannerRepository) Store() *manifest.Store {
	if r == nil {
		return nil
	}
	return r.store
}

// Apply runs the in-memory mutation, then atomically projects the snapshot and outbox rows.
func (r *SpannerRepository) Apply(
	ctx context.Context,
	mutate func() error,
	snapshot func() *PersistenceSnapshot,
	emit func(outbox.TxnBuffer) error,
) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payload repository: nil client")
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
	return r.store.CommitSupplier(ctx, supplierBatchFromSnapshot(r.supplierID, snap), emit)
}

// Hydrate loads durable manifests into the service when Spanner already has rows.
func (r *SpannerRepository) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	if r == nil || r.store == nil || s == nil {
		return fmt.Errorf("hydrate payload: invalid dependencies")
	}
	rows, err := r.store.ListSupplierManifests(ctx, supplierID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return r.applyHydrateLocked(ctx, supplierID, s, rows)
}

// hydrateWhileLocked loads durable manifests; caller must already hold s.mu.
func (r *SpannerRepository) hydrateWhileLocked(ctx context.Context, supplierID string, s *Service) error {
	if r == nil || r.store == nil || s == nil {
		return fmt.Errorf("hydrate payload: invalid dependencies")
	}
	rows, err := r.store.ListSupplierManifests(ctx, supplierID)
	if err != nil {
		return err
	}
	return r.applyHydrateLocked(ctx, supplierID, s, rows)
}

func (r *SpannerRepository) applyHydrateLocked(ctx context.Context, supplierID string, s *Service, rows []manifest.SupplierTruckRow) error {
	if len(rows) == 0 {
		return nil
	}
	s.manifests = make([]ManifestRow, 0, len(rows))
	s.manifestOrders = make(map[string][]ManifestOrder, len(rows))
	for _, row := range rows {
		s.manifests = append(s.manifests, manifestRowFromSupplierTruck(row))
		orders, err := r.store.ListSupplierManifestOrders(ctx, row.ManifestID)
		if err != nil {
			return err
		}
		mapped := make([]ManifestOrder, 0, len(orders))
		for _, o := range orders {
			mapped = append(mapped, manifestOrderFromSupplierRow(o))
		}
		s.manifestOrders[row.ManifestID] = mapped
	}
	s.spannerLoaded = true
	return nil
}

// SeedDemoManifests writes demo manifests when the supplier has none in Spanner.
func (r *SpannerRepository) SeedDemoManifests(ctx context.Context, supplierID string, snap *PersistenceSnapshot) error {
	if r == nil || r.store == nil || snap == nil {
		return nil
	}
	batch := supplierBatchFromSnapshot(supplierID, snap)
	rows := batch.Manifests
	orders := make(map[string][]manifest.SupplierManifestOrderRow, len(snap.ManifestOrders))
	for id, list := range snap.ManifestOrders {
		for i, mo := range list {
			orders[id] = append(orders[id], supplierOrderFromManifestOrder(id, mo, int64(i+1)))
		}
	}
	return r.store.EnsureSupplierDemoManifests(ctx, supplierID, rows, orders)
}
