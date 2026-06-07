package factory

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type crossManifestRebalanceResult struct {
	SourceManifest ManifestRow
	TargetManifest ManifestRow
	TransfersMoved int
	VolumeMovedVU  int64
}

func (s *Service) handleCrossManifestRebalance(w http.ResponseWriter, r *http.Request, req manifestRebalanceRequest) {
	if req.SourceManifestID == "" || req.TargetManifestID == "" || len(req.TransferIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source_target_and_transfer_ids_required"})
		return
	}

	var result crossManifestRebalanceResult
	err := s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		moved, err := s.rebalanceCrossManifestLocked(req, &result)
		if err != nil {
			return err
		}
		if moved == 0 {
			return fmt.Errorf("transfer_not_found")
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		for _, transferID := range req.TransferIDs {
			transferID = strings.TrimSpace(transferID)
			if transferID == "" {
				continue
			}
			if err := outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.TargetManifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventManifestOrderInjected},
				ManifestID: req.TargetManifestID,
				// "source_manifest_id":  req.SourceManifestID,
				TransferID: req.TransferID,
				SupplierID: s.supplierID,
				FactoryID:  s.factoryNodeID,
				Reason:     strings.TrimSpace(req.Reason),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		case "manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_mutable"})
		case "transfer_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
		case "transfer_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_not_mutable"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_rebalance_failed"})
		}
		return
	}

	s.invalidateFactoryKeys(
		r.Context(),
		factoryManifestKey(req.SourceManifestID),
		factoryManifestKey(req.TargetManifestID),
		factoryManifestListKey(s.supplierID),
		factoryTransferListKey(s.supplierID),
	)
	s.broadcastFactoryEvent(r.Context(), events.EventManifestRebalanced, map[string]any{
		"source_manifest_id": req.SourceManifestID,
		"target_manifest_id": req.TargetManifestID,
		"transfers_moved":    result.TransfersMoved,
		"volume_moved_vu":    result.VolumeMovedVU,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"source_manifest_id": req.SourceManifestID,
		"target_manifest_id": req.TargetManifestID,
		"transfers_moved":    result.TransfersMoved,
		"volume_moved_vu":    result.VolumeMovedVU,
		"reason":             strings.TrimSpace(req.Reason),
	})
}

func (s *Service) rebalanceCrossManifestLocked(req manifestRebalanceRequest, result *crossManifestRebalanceResult) (int, error) {
	sourceIdx := s.findManifestIndexLocked(req.SourceManifestID)
	targetIdx := s.findManifestIndexLocked(req.TargetManifestID)
	if sourceIdx < 0 || targetIdx < 0 {
		return 0, fmt.Errorf("manifest_not_found")
	}
	source := s.manifests[sourceIdx]
	target := s.manifests[targetIdx]
	if !manifestMutable(source.State) || !manifestMutable(target.State) {
		return 0, fmt.Errorf("manifest_not_mutable")
	}

	sourceTransfers := append([]TransferRow(nil), s.manifestTransfers[req.SourceManifestID]...)
	targetTransfers := append([]TransferRow(nil), s.manifestTransfers[req.TargetManifestID]...)
	now := s.now().Format(time.RFC3339Nano)
	moved := 0
	var volume int64

	for _, transferID := range req.TransferIDs {
		transferID = strings.TrimSpace(transferID)
		if transferID == "" {
			continue
		}
		tIdx := s.findTransferIndexLocked(sourceTransfers, transferID)
		if tIdx < 0 {
			continue
		}
		row := sourceTransfers[tIdx]
		if row.State != "ASSIGNED" && row.State != "REASSIGNED" {
			return moved, fmt.Errorf("transfer_not_mutable")
		}
		sourceTransfers = append(sourceTransfers[:tIdx], sourceTransfers[tIdx+1:]...)
		row.ManifestID = req.TargetManifestID
		row.DriverID = target.DriverID
		row.VehicleID = target.VehicleID
		row.UpdatedAt = now
		targetTransfers = append(targetTransfers, row)
		volume += row.TotalVU
		moved++

		globalIdx := s.findGlobalTransferIndexLocked(transferID)
		if globalIdx >= 0 {
			s.transfers[globalIdx] = row
		}
	}

	if moved == 0 {
		return 0, fmt.Errorf("transfer_not_found")
	}

	recalcManifestTotals(&source, sourceTransfers)
	recalcManifestTotals(&target, targetTransfers)
	source.UpdatedAt = now
	target.UpdatedAt = now
	s.manifests[sourceIdx] = source
	s.manifests[targetIdx] = target
	s.manifestTransfers[req.SourceManifestID] = sourceTransfers
	s.manifestTransfers[req.TargetManifestID] = targetTransfers

	result.SourceManifest = source
	result.TargetManifest = target
	result.TransfersMoved = moved
	result.VolumeMovedVU = volume
	return moved, nil
}

func manifestMutable(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case manifestStateDraft, manifestStateLoading:
		return true
	default:
		return false
	}
}

func recalcManifestTotals(manifest *ManifestRow, transfers []TransferRow) {
	manifest.TransferCnt = len(transfers)
	var total int64
	for i := range transfers {
		total += transfers[i].TotalVU
	}
	manifest.TotalVolumeVU = total
}
