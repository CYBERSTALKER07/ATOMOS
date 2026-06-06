package factory

import "context"

// WarmManifestCache hydrates or seeds demo manifests when Spanner adapters are active.
func (s *Service) WarmManifestCache(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureDemoDataLocked()
	_ = ctx
}
