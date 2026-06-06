package payload

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Apply runs a mutation and optional outbox emission through the repository seam.
func (s *Service) apply(
	ctx context.Context,
	mutate func() error,
	emit func(outbox.TxnBuffer) error,
) error {
	var snapshotFn func() *PersistenceSnapshot
	if _, ok := s.repo.(*SpannerRepository); ok {
		snapshotFn = func() *PersistenceSnapshot {
			s.mu.RLock()
			defer s.mu.RUnlock()
			return s.buildPersistenceSnapshotLocked()
		}
	}
	return s.repo.Apply(ctx, mutate, snapshotFn, emit)
}
