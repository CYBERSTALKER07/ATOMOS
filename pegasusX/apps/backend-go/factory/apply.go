package factory

import (
	"context"
	"reflect"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// apply runs a mutation and optional outbox emission through the repository seam.
func (s *Service) apply(
	ctx context.Context,
	mutate func() error,
	emit func(outbox.TxnBuffer) error,
) error {
	s.mu.RLock()
	snapManifests := make([]ManifestRow, len(s.manifests))
	copy(snapManifests, s.manifests)
	snapTransfers := make([]TransferRow, len(s.transfers))
	copy(snapTransfers, s.transfers)
	s.mu.RUnlock()

	err := s.repo.RunTx(ctx, func(ctx context.Context, tx FactoryTx) error {
		// Reset memory state from snapshot to safely handle Spanner retries
		// without O(N) database reads.
		s.mu.Lock()
		s.manifests = make([]ManifestRow, len(snapManifests))
		copy(s.manifests, snapManifests)
		s.transfers = make([]TransferRow, len(snapTransfers))
		copy(s.transfers, snapTransfers)
		s.rebuildManifestTransfersLocked()
		s.mu.Unlock()

		origManifests := make(map[string]ManifestRow)
		for _, m := range snapManifests {
			origManifests[m.ManifestID] = m
		}
		origTransfers := make(map[string]TransferRow)
		for _, t := range snapTransfers {
			origTransfers[t.TransferID] = t
		}

		if err := mutate(); err != nil {
			return err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, m := range s.manifests {
			if orig, ok := origManifests[m.ManifestID]; !ok || !reflect.DeepEqual(orig, m) {
				if err := tx.SaveManifest(ctx, m); err != nil {
					return err
				}
			}
		}
		for _, t := range s.transfers {
			if orig, ok := origTransfers[t.TransferID]; !ok || !reflect.DeepEqual(orig, t) {
				if err := tx.SaveTransfer(ctx, t); err != nil {
					return err
				}
			}
		}
		for _, ex := range s.manifestExceptions {
			if err := tx.SaveException(ctx, ex); err != nil {
				return err
			}
		}
		return nil
	}, emit)

	if err != nil {
		// Rollback in-memory state on complete failure
		s.mu.Lock()
		s.manifests = snapManifests
		s.transfers = snapTransfers
		s.rebuildManifestTransfersLocked()
		s.mu.Unlock()
	}

	return err
}
