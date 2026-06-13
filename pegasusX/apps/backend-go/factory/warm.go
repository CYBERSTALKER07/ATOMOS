package factory

import "context"

// WarmManifestCache hydrates Spanner-backed manifests before optional demo seeding.
func (s *Service) WarmManifestCache(ctx context.Context) {
	if s == nil {
		return
	}
	if r, ok := s.repo.(*SpannerRepository); ok {
		if err := r.Hydrate(ctx, s.factoryNodeID, s); err != nil {
			s.log.WarnContext(ctx, "factory spanner hydrate failed", "err", err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repo.(*SpannerRepository); ok {
		s.spannerLoaded = true
	}
	s.ensureDemoDataLocked()
}
