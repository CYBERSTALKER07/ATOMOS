package payload

import (
	"context"
	"strings"
)

// ManifestGateSnapshot exposes payloader manifest state for driver ghost-stop checks.
func (s *Service) ManifestGateSnapshot(manifestID string) (state string, stopCount int, totalVolumeVU int64, found bool) {
	if s == nil {
		return "", 0, 0, false
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return "", 0, 0, false
	}
	_ = s.hydrateFromRepo(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureManifestStateLocked()
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		return "", 0, 0, false
	}
	manifest := s.manifests[idx]
	return manifest.State, manifest.StopCount, manifest.TotalVolumeVU, true
}
