package payload

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
	return s.repo.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		var err error
		var manifests []ManifestRow
		var manifestOrders = make(map[string][]ManifestOrder)
		var exceptions []ManifestException

		manifests, err = tx.ListManifests(ctx)
		if err != nil {
			return err
		}
		
		for _, m := range manifests {
			orders, err := tx.ListManifestOrders(ctx, m.ManifestID)
			if err != nil {
				return err
			}
			if len(orders) > 0 {
				manifestOrders[m.ManifestID] = orders
			}
		}

		exceptions, err = tx.ListExceptions(ctx)
		if err != nil {
			return err
		}

		s.mu.Lock()
		if len(manifests) > 0 {
			s.manifests = manifests
			s.exceptions = exceptions
			for _, m := range manifests {
				if orders, ok := manifestOrders[m.ManifestID]; ok && len(orders) > 0 {
					s.manifestOrders[m.ManifestID] = orders
				}
			}
		} else if len(s.manifests) == 0 {
			s.manifestOrders = manifestOrders
			s.exceptions = exceptions
		}
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
		for _, orders := range s.manifestOrders {
			for i, o := range orders {
				if err := tx.SaveManifestOrder(ctx, o, int64(i+1)); err != nil {
					return err
				}
			}
		}
		for _, e := range s.exceptions {
			if err := tx.SaveException(ctx, e); err != nil {
				return err
			}
		}
		return nil
	}, emit)
}
