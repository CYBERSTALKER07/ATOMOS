package payload

import "context"

// WarmManifestCache hydrates Spanner-backed manifests before optional demo seeding.
func (s *Service) WarmManifestCache(ctx context.Context) {
	if s == nil {
		return
	}
	_ = s.hydrateFromRepo(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureManifestStateLocked()
}
