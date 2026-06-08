package factory

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// apply runs a mutation and optional outbox emission through the repository seam.
func (s *Service) apply(
	ctx context.Context,
	mutate func() error,
	emit func(outbox.TxnBuffer) error,
) error {
	return s.repo.RunTx(ctx, func(ctx context.Context, tx FactoryTx) error {
		var err error
		var manifests []ManifestRow
		var transfers []TransferRow
		
		manifests, err = tx.ListManifests(ctx)
		if err != nil {
			return err
		}
		transfers, err = tx.ListTransfers(ctx)
		if err != nil {
			return err
		}
		
		s.mu.Lock()
		s.manifests = manifests
		s.transfers = transfers
		s.rebuildManifestTransfersLocked()
		s.mu.Unlock()

		if err := mutate(); err != nil {
			return err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, m := range s.manifests {
			if err := tx.SaveManifest(ctx, m); err != nil {
				return err
			}
		}
		for _, t := range s.transfers {
			if err := tx.SaveTransfer(ctx, t); err != nil {
				return err
			}
		}
		return nil
	}, emit)
}
