package payload

import (
	"context"
	"reflect"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// apply runs a mutation and optional outbox emission through the repository seam.
// TRK6-002 Fix: Use snapshot/rollback of in-memory state instead of full-table reads in Tx.
func (s *Service) apply(
	ctx context.Context,
	mutate func(tx PayloadTx) error,
	emit func(outbox.TxnBuffer) error,
) error {
	s.mu.RLock()
	snapManifests := make([]ManifestRow, len(s.manifests))
	copy(snapManifests, s.manifests)
	
	snapManifestOrders := make(map[string][]ManifestOrder, len(s.manifestOrders))
	for k, v := range s.manifestOrders {
		arr := make([]ManifestOrder, len(v))
		copy(arr, v)
		snapManifestOrders[k] = arr
	}
	
	snapExceptions := make([]ManifestException, len(s.exceptions))
	copy(snapExceptions, s.exceptions)
	s.mu.RUnlock()

	origManifests := make(map[string]ManifestRow)
	for _, m := range snapManifests {
		origManifests[m.ManifestID] = m
	}
	origExceptions := make(map[string]ManifestException)
	for _, e := range snapExceptions {
		origExceptions[e.ExceptionID] = e
	}

	err := s.repo.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		// Reset state for retry
		s.mu.Lock()
		s.manifests = make([]ManifestRow, len(snapManifests))
		copy(s.manifests, snapManifests)
		
		s.manifestOrders = make(map[string][]ManifestOrder, len(snapManifestOrders))
		for k, v := range snapManifestOrders {
			arr := make([]ManifestOrder, len(v))
			copy(arr, v)
			s.manifestOrders[k] = arr
		}
		
		s.exceptions = make([]ManifestException, len(snapExceptions))
		copy(s.exceptions, snapExceptions)
		s.mu.Unlock()

		if err := mutate(tx); err != nil {
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
		for mID, orders := range s.manifestOrders {
			origOrd := snapManifestOrders[mID]
			for i, o := range orders {
				changed := true
				if i < len(origOrd) && reflect.DeepEqual(origOrd[i], o) {
					changed = false
				}
				if changed {
					if err := tx.SaveManifestOrder(ctx, o, int64(i+1)); err != nil {
						return err
					}
				}
			}
		}
		for _, e := range s.exceptions {
			if orig, ok := origExceptions[e.ExceptionID]; !ok || !reflect.DeepEqual(orig, e) {
				if err := tx.SaveException(ctx, e); err != nil {
					return err
				}
			}
		}
		return nil
	}, emit)

	if err != nil {
		// Rollback memory on Tx failure
		s.mu.Lock()
		s.manifests = snapManifests
		s.manifestOrders = snapManifestOrders
		s.exceptions = snapExceptions
		s.mu.Unlock()
	}
	return err
}
