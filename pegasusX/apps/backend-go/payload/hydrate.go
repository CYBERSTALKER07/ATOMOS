package payload

import "context"

// hydrateFromRepo reloads manifest state from the repository seam. Callers must
// not hold s.mu; Hydrate acquires it while reading Spanner.
func (s *Service) hydrateFromRepo(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.Hydrate(ctx, s.resolveSupplierScope(ctx), s)
}

// ensureManifestStateLocked seeds demo manifests only when the in-memory cache is
// still empty after a repository hydrate (local/dev scaffold).
func (s *Service) ensureManifestStateLocked() {
	if len(s.manifests) > 0 {
		return
	}
	s.ensureDemoDataLocked()
}
