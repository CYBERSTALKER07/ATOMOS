package payload

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
)

// ManifestDetailSnapshotForDriver exposes payloader manifest detail on the driver
// manifest read path using the shared factory read-model shape.
func (s *Service) ManifestDetailSnapshotForDriver(driverID, manifestID, date string) (factory.ManifestDetailSnapshot, bool) {
	if s == nil {
		return factory.ManifestDetailSnapshot{}, false
	}
	driverID = strings.TrimSpace(driverID)
	manifestID = strings.TrimSpace(manifestID)
	date = strings.TrimSpace(date)
	if driverID == "" {
		return factory.ManifestDetailSnapshot{}, false
	}

	_ = s.hydrateFromRepo(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureManifestStateLocked()

	idx := s.findDriverManifestIndexLocked(driverID, manifestID, date)
	if idx < 0 {
		return factory.ManifestDetailSnapshot{}, false
	}
	return s.factoryManifestDetailLocked(s.manifests[idx]), true
}

func (s *Service) factoryManifestDetailLocked(manifest ManifestRow) factory.ManifestDetailSnapshot {
	manifestID := manifest.ManifestID
	orders := append([]ManifestOrder(nil), s.manifestOrders[manifestID]...)
	transfers := make([]factory.TransferRow, 0, len(orders))
	for _, order := range orders {
		transfers = append(transfers, factory.TransferRow{
			TransferID: "xfer_" + order.OrderID,
			OrderID:    order.OrderID,
			ManifestID: manifestID,
			State:      order.State,
			TotalVU:    order.VolumeVU,
			DriverID:   manifest.DriverID,
			VehicleID:  manifest.VehicleID,
			UpdatedAt:  order.UpdatedAt,
			CreatedAt:  order.UpdatedAt,
		})
	}

	exceptions := make([]factory.ManifestException, 0)
	for i := range s.exceptions {
		if s.exceptions[i].ManifestID != manifestID {
			continue
		}
		ex := s.exceptions[i]
		exceptions = append(exceptions, factory.ManifestException{
			ExceptionID:  ex.ExceptionID,
			ManifestID:   ex.ManifestID,
			TransferID:   "xfer_" + ex.OrderID,
			Reason:       ex.Reason,
			Metadata:     ex.Metadata,
			AttemptCount: ex.AttemptCount,
			Escalated:    ex.Escalated,
			CreatedAt:    ex.CreatedAt,
		})
	}

	stopCount := manifest.StopCount
	if stopCount == 0 {
		stopCount = len(transfers)
	}

	factoryManifest := factory.ManifestRow{
		ManifestID:       manifest.ManifestID,
		State:            manifest.State,
		TransferCnt:      stopCount,
		TotalVolumeVU:    manifest.TotalVolumeVU,
		MaxVolumeVU:      manifest.MaxVolumeVU,
		DriverID:         manifest.DriverID,
		VehicleID:        manifest.VehicleID,
		CreatedAt:        manifest.CreatedAt,
		UpdatedAt:        manifest.UpdatedAt,
		LoadingStartedAt: manifest.LoadingStartedAt,
		SealedAt:         manifest.SealedAt,
	}

	transitions := payloadManifestTransitionsLocked(manifest)

	return factory.ManifestDetailSnapshot{
		Manifest:    factoryManifest,
		Transfers:   transfers,
		Transitions: transitions,
		Exceptions:  exceptions,
		RouteID:     routeIDForManifest(manifest),
		StopCount:   stopCount,
		OrderCount:  len(transfers),
	}
}

func payloadManifestTransitionsLocked(manifest ManifestRow) []factory.ManifestTransition {
	transitions := make([]factory.ManifestTransition, 0, 3)
	if manifest.LoadingStartedAt != "" {
		transitions = append(transitions, factory.ManifestTransition{
			Action:    "START_LOADING",
			FromState: payloadManifestStateDraft,
			ToState:   payloadManifestStateLoading,
			At:        manifest.LoadingStartedAt,
		})
	}
	if manifest.SealedAt != "" {
		from := payloadManifestStateLoading
		if manifest.LoadingStartedAt == "" {
			from = payloadManifestStateDraft
		}
		transitions = append(transitions, factory.ManifestTransition{
			Action:    "SEAL",
			FromState: from,
			ToState:   payloadManifestStateSealed,
			At:        manifest.SealedAt,
		})
	}
	return transitions
}

func (s *Service) findDriverManifestIndexLocked(driverID, manifestID, date string) int {
	if manifestID != "" {
		idx := s.findManifestIndexLocked(manifestID)
		if idx < 0 {
			return -1
		}
		manifest := s.manifests[idx]
		if strings.TrimSpace(manifest.DriverID) != driverID {
			return -1
		}
		if !payloadManifestMatchesDate(manifest, date) {
			return -1
		}
		return idx
	}

	bestIdx := -1
	bestRank := -1
	bestUpdatedAt := ""
	for i := range s.manifests {
		manifest := s.manifests[i]
		if strings.TrimSpace(manifest.DriverID) != driverID {
			continue
		}
		if !payloadManifestMatchesDate(manifest, date) {
			continue
		}
		rank := payloadManifestSelectionRank(manifest.State)
		if bestIdx < 0 || rank > bestRank || (rank == bestRank && manifest.UpdatedAt > bestUpdatedAt) {
			bestIdx = i
			bestRank = rank
			bestUpdatedAt = manifest.UpdatedAt
		}
	}
	return bestIdx
}

func payloadManifestMatchesDate(manifest ManifestRow, date string) bool {
	if date == "" {
		return true
	}
	return strings.HasPrefix(manifest.CreatedAt, date) || strings.HasPrefix(manifest.UpdatedAt, date)
}

func payloadManifestSelectionRank(state string) int {
	switch strings.TrimSpace(state) {
	case payloadManifestStateDispatched:
		return 6
	case payloadManifestStateSealed:
		return 5
	case payloadManifestStateLoading:
		return 4
	case payloadManifestStateDraft:
		return 3
	default:
		return 0
	}
}
