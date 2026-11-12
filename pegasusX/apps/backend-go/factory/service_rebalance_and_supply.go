package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

func (s *Service) SetPlanning(p *PlanningService) {
	if s != nil {
		s.planning = p
	}
}

func (s *Service) handleBatcherDispatch(w http.ResponseWriter, r *http.Request, body []byte) {
	if s.planning == nil || s.planning.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "batcher_unavailable"})
		return
	}
	factoryID := s.resolveFactoryNode(r.Context())
	supplierID := s.resolveSupplierScope(r.Context())
	result, err := s.planning.RunBatchDispatch(r.Context(), factoryID, supplierID)
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "dispatch_failed", err)
		return
	}
	manifestID := ""
	if len(result.ManifestIDs) > 0 {
		manifestID = result.ManifestIDs[0]
	}
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":                 "dispatch_planned",
		"created_manifest_count": result.CreatedManifestCount,
		"manifests_created":      result.CreatedManifestCount,
		"manifest_id":            manifestID,
		"manifest_ids":           result.ManifestIDs,
		"unassigned":             result.Unassigned,
		"optimizer_class":        result.OptimizerClass,
		"dispatch_algo":          result.DispatchAlgo,
	})
}

// HandleManifestRebalance serves POST /v1/factory/manifests/rebalance.
func (s *Service) HandleManifestRebalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	var req manifestRebalanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	req.ToDriverID = strings.TrimSpace(req.ToDriverID)
	req.ToVehicle = strings.TrimSpace(req.ToVehicle)
	req.SourceManifestID = strings.TrimSpace(req.SourceManifestID)
	req.TargetManifestID = strings.TrimSpace(req.TargetManifestID)
	if req.SourceManifestID == "" {
		req.SourceManifestID = req.ManifestID
	}
	if len(req.TransferIDs) == 0 && req.TransferID != "" {
		req.TransferIDs = []string{req.TransferID}
	}
	if req.TargetManifestID != "" && req.TargetManifestID != req.SourceManifestID {
		s.handleCrossManifestRebalance(w, r, body, req)
		return
	}
	if req.ManifestID == "" || req.TransferID == "" || (req.ToDriverID == "" && req.ToVehicle == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_transfer_id_and_target_required"})
		return
	}

	var manifest ManifestRow
	var reassign ManifestReassignment
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		mIdx := s.findManifestIndexLocked(req.ManifestID)
		if mIdx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest = s.manifests[mIdx]
		if manifest.State != manifestStateDraft && manifest.State != manifestStateLoading {
			return fmt.Errorf("manifest_not_mutable")
		}

		transfers := s.manifestTransfers[req.ManifestID]
		tIdx := s.findTransferIndexLocked(transfers, req.TransferID)
		if tIdx < 0 {
			return fmt.Errorf("transfer_not_found")
		}
		if transfers[tIdx].State != "ASSIGNED" && transfers[tIdx].State != "REASSIGNED" {
			return fmt.Errorf("transfer_not_mutable")
		}
		globalTransferIdx := s.findGlobalTransferIndexLocked(req.TransferID)
		if globalTransferIdx < 0 {
			return fmt.Errorf("transfer_not_found")
		}
		globalTransfer := s.transfers[globalTransferIdx]
		if globalTransfer.ManifestID != "" && globalTransfer.ManifestID != req.ManifestID {
			return fmt.Errorf("transfer_manifest_mismatch")
		}
		if globalTransfer.OrderID != transfers[tIdx].OrderID || globalTransfer.State != transfers[tIdx].State {
			return fmt.Errorf("transfer_ledger_mismatch")
		}
		if globalTransfer.DriverID != "" && transfers[tIdx].DriverID != "" && globalTransfer.DriverID != transfers[tIdx].DriverID {
			return fmt.Errorf("transfer_ledger_mismatch")
		}
		if globalTransfer.VehicleID != "" && transfers[tIdx].VehicleID != "" && globalTransfer.VehicleID != transfers[tIdx].VehicleID {
			return fmt.Errorf("transfer_ledger_mismatch")
		}
		if manifest.VehicleID != "" && transfers[tIdx].VehicleID != "" && transfers[tIdx].VehicleID != manifest.VehicleID {
			return fmt.Errorf("transfer_route_mismatch")
		}

		fromDriver := transfers[tIdx].DriverID
		fromVehicle := transfers[tIdx].VehicleID
		if (req.ToDriverID == "" || req.ToDriverID == fromDriver) && (req.ToVehicle == "" || req.ToVehicle == fromVehicle) {
			return fmt.Errorf("transfer_already_assigned")
		}
		if req.ToDriverID != "" {
			transfers[tIdx].DriverID = req.ToDriverID
		}
		if req.ToVehicle != "" {
			if manifest.VehicleID != "" && req.ToVehicle != manifest.VehicleID {
				return fmt.Errorf("transfer_route_mismatch")
			}
			transfers[tIdx].VehicleID = req.ToVehicle
		}
		if manifest.VehicleID != "" && transfers[tIdx].VehicleID != "" && transfers[tIdx].VehicleID != manifest.VehicleID {
			return fmt.Errorf("transfer_route_mismatch")
		}
		transfers[tIdx].State = "REASSIGNED"
		transfers[tIdx].ReassignDepth++
		transfers[tIdx].UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifestTransfers[req.ManifestID] = transfers

		s.transfers[globalTransferIdx].DriverID = transfers[tIdx].DriverID
		s.transfers[globalTransferIdx].VehicleID = transfers[tIdx].VehicleID
		s.transfers[globalTransferIdx].ReassignDepth = transfers[tIdx].ReassignDepth
		s.transfers[globalTransferIdx].State = transfers[tIdx].State
		s.transfers[globalTransferIdx].UpdatedAt = transfers[tIdx].UpdatedAt

		manifest.ReassignmentDepth++
		manifest.UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifests[mIdx] = manifest

		reassign = ManifestReassignment{
			ManifestID:      req.ManifestID,
			TransferID:      req.TransferID,
			FromDriverID:    fromDriver,
			ToDriverID:      transfers[tIdx].DriverID,
			FromVehicleID:   fromVehicle,
			ToVehicleID:     transfers[tIdx].VehicleID,
			Depth:           transfers[tIdx].ReassignDepth,
			Reason:          strings.TrimSpace(req.Reason),
			ReassignedAt:    s.now().Format(time.RFC3339Nano),
			Recommendation:  "load_balanced",
			AppliedBy:       strings.TrimSpace(req.AppliedBy),
			CorrelationHint: req.ManifestID + ":" + req.TransferID,
		}
		s.manifestReassignments[req.ManifestID] = append(s.manifestReassignments[req.ManifestID], reassign)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:     events.BaseEvent{Type: events.EventManifestRebalanced},
			ManifestID:    req.ManifestID,
			TransferID:    req.TransferID,
			SupplierID:    s.resolveSupplierScope(r.Context()),
			FactoryID:     s.factoryNodeID,
			FromDriverID:  reassign.FromDriverID,
			ToDriverID:    reassign.ToDriverID,
			FromVehicleID: reassign.FromVehicleID,
			ToVehicleID:   reassign.ToVehicleID,
			Depth:         reassign.Depth,
			Reason:        reassign.Reason,
		})
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_mutable"})
			return
		case "transfer_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
			return
		case "transfer_manifest_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_manifest_mismatch"})
			return
		case "transfer_ledger_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_ledger_mismatch"})
			return
		case "transfer_route_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_route_mismatch"})
			return
		case "transfer_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "transfer_not_mutable"})
			return
		case "transfer_already_assigned":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_assigned", "manifest_id": req.ManifestID, "transfer_id": req.TransferID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_rebalance_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestRebalanced, map[string]any{
		"manifest_id":            req.ManifestID,
		"transfer_id":            req.TransferID,
		"from_driver_id":         reassign.FromDriverID,
		"to_driver_id":           reassign.ToDriverID,
		"from_vehicle_id":        reassign.FromVehicleID,
		"to_vehicle_id":          reassign.ToVehicleID,
		"reassignment_depth":     reassign.Depth,
		"manifest_reassignments": manifest.ReassignmentDepth,
	})

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":                 "manifest_rebalanced",
		"manifest_id":            req.ManifestID,
		"transfer_id":            req.TransferID,
		"from_driver_id":         reassign.FromDriverID,
		"to_driver_id":           reassign.ToDriverID,
		"from_vehicle_id":        reassign.FromVehicleID,
		"to_vehicle_id":          reassign.ToVehicleID,
		"reassignment_depth":     reassign.Depth,
		"manifest_reassignments": manifest.ReassignmentDepth,
	})
}

// HandleManifestCancelTransfer serves POST /v1/factory/manifests/cancel-transfer.
func (s *Service) HandleManifestCancelTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	var req manifestCancelTransferRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	if req.ManifestID == "" || req.TransferID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_and_transfer_id_required"})
		return
	}

	var exception ManifestException
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		mIdx := s.findManifestIndexLocked(req.ManifestID)
		if mIdx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest := s.manifests[mIdx]
		if manifest.State == manifestStateCompleted || manifest.State == manifestStateCancelled {
			return fmt.Errorf("manifest_not_mutable")
		}

		transfers := s.manifestTransfers[req.ManifestID]
		tIdx := s.findTransferIndexLocked(transfers, req.TransferID)
		if tIdx < 0 {
			return fmt.Errorf("transfer_not_found")
		}
		if transfers[tIdx].State == "CANCELLED" {
			return fmt.Errorf("transfer_already_cancelled")
		}

		transfers[tIdx].State = "CANCELLED"
		transfers[tIdx].ExceptionCount++
		transfers[tIdx].UpdatedAt = s.now().Format(time.RFC3339Nano)
		s.manifestTransfers[req.ManifestID] = transfers

		for i := range s.transfers {
			if s.transfers[i].TransferID == req.TransferID {
				s.transfers[i].State = "CANCELLED"
				s.transfers[i].ExceptionCount++
				s.transfers[i].UpdatedAt = transfers[tIdx].UpdatedAt
				break
			}
		}

		if manifest.TransferCnt > 0 {
			manifest.TransferCnt--
		}
		if manifest.TotalVolumeVU >= transfers[tIdx].TotalVU {
			manifest.TotalVolumeVU -= transfers[tIdx].TotalVU
		}
		manifest.UpdatedAt = s.now().Format(time.RFC3339Nano)
		manifest.LastExceptionAt = manifest.UpdatedAt
		manifest.EscalatedException = transfers[tIdx].ExceptionCount >= manifestExceptionEscalationThreshold
		s.manifests[mIdx] = manifest

		exception = ManifestException{
			ExceptionID:   s.nextIDLocked("mex"),
			ManifestID:    req.ManifestID,
			TransferID:    req.TransferID,
			Reason:        coalesceReason(strings.TrimSpace(req.Reason), "CANCEL_TRANSFER"),
			Metadata:      strings.TrimSpace(req.Metadata),
			AttemptCount:  transfers[tIdx].ExceptionCount,
			Escalated:     transfers[tIdx].ExceptionCount >= manifestExceptionEscalationThreshold,
			CreatedAt:     s.now().Format(time.RFC3339Nano),
			CorrelationID: req.ManifestID + ":" + req.TransferID,
		}
		s.manifestExceptions = append(s.manifestExceptions, exception)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventManifestOrderException},
			ManifestID:   req.ManifestID,
			TransferID:   req.TransferID,
			SupplierID:   s.resolveSupplierScope(r.Context()),
			FactoryID:    s.factoryNodeID,
			Reason:       exception.Reason,
			AttemptCount: exception.AttemptCount,
			Escalated:    exception.Escalated,
		}); err != nil {
			return err
		}
		if exception.Escalated {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
				BaseEvent:    events.BaseEvent{Type: events.EventManifestDLQEscalation},
				ManifestID:   req.ManifestID,
				TransferID:   req.TransferID,
				SupplierID:   s.resolveSupplierScope(r.Context()),
				FactoryID:    s.factoryNodeID,
				Reason:       exception.Reason,
				AttemptCount: exception.AttemptCount,
			})
		}
		return nil
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_mutable"})
			return
		case "transfer_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer_not_found"})
			return
		case "transfer_already_cancelled":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_cancelled", "manifest_id": req.ManifestID, "transfer_id": req.TransferID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cancel_transfer_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())), factoryExceptionListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestOrderException, map[string]any{
		"manifest_id":   req.ManifestID,
		"transfer_id":   req.TransferID,
		"reason":        exception.Reason,
		"attempt_count": exception.AttemptCount,
		"escalated":     exception.Escalated,
		"exception_id":  exception.ExceptionID,
	})
	if exception.Escalated {
		s.broadcastFactoryEvent(r.Context(), events.EventManifestDLQEscalation, map[string]any{
			"manifest_id":   req.ManifestID,
			"transfer_id":   req.TransferID,
			"reason":        exception.Reason,
			"attempt_count": exception.AttemptCount,
			"exception_id":  exception.ExceptionID,
		})
	}

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":        "transfer_cancelled",
		"manifest_id":   req.ManifestID,
		"transfer_id":   req.TransferID,
		"exception_id":  exception.ExceptionID,
		"attempt_count": exception.AttemptCount,
		"escalated":     exception.Escalated,
	})
}

// HandleManifestCancel serves POST /v1/factory/manifests/cancel.
func (s *Service) HandleManifestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	var req manifestCancelRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	if req.ManifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	var manifest ManifestRow
	var now string
	err = s.apply(r.Context(), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		idx := s.findManifestIndexLocked(req.ManifestID)
		if idx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest = s.manifests[idx]
		if manifest.State == manifestStateCompleted {
			return fmt.Errorf("manifest_already_completed")
		}
		if manifest.State == manifestStateCancelled {
			return fmt.Errorf("manifest_already_cancelled")
		}

		now = s.now().Format(time.RFC3339Nano)
		fromState := manifest.State
		manifest.State = manifestStateCancelled
		manifest.CancelledAt = now
		manifest.UpdatedAt = now
		s.manifests[idx] = manifest

		transfers := s.manifestTransfers[req.ManifestID]
		for i := range transfers {
			transfers[i].State = "CANCELLED"
			transfers[i].UpdatedAt = now
		}
		s.manifestTransfers[req.ManifestID] = transfers

		for i := range s.transfers {
			if s.transfers[i].ManifestID == req.ManifestID {
				s.transfers[i].State = "CANCELLED"
				s.transfers[i].UpdatedAt = now
			}
		}

		s.appendTransitionLocked(req.ManifestID, "CANCEL", fromState, manifestStateCancelled, req.Reason)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventManifestCancelled},
			ManifestID: req.ManifestID,
			SupplierID: s.resolveSupplierScope(r.Context()),
			FactoryID:  s.factoryNodeID,
			Reason:     strings.TrimSpace(req.Reason),
		})
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_already_completed":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_already_completed"})
			return
		case "manifest_already_cancelled":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_cancelled", "manifest_id": req.ManifestID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_cancel_failed"})
			return
		}
	}

	s.invalidateFactoryKeys(r.Context(), factoryManifestKey(req.ManifestID), factoryManifestListKey(s.resolveSupplierScope(r.Context())), factoryTransferListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestCancelled, map[string]any{
		"manifest_id": req.ManifestID,
		"reason":      strings.TrimSpace(req.Reason),
		"updated_at":  now,
	})

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":      "manifest_cancelled",
		"manifest_id": req.ManifestID,
		"reason":      strings.TrimSpace(req.Reason),
		"updated_at":  now,
	})
}

// HandleManifestExceptions serves GET /v1/factory/manifest-exceptions.
func (s *Service) HandleManifestExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	escalatedOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("escalated")), "true")

	backend := s.exceptionBackend()
	if backend != nil {
		rows, err := backend.List(r.Context(), s.resolveSupplierScope(r.Context()), strings.TrimSpace(s.resolveFactoryNode(r.Context())))
		if err != nil {
			s.log.WarnContext(r.Context(), "factory exception list failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_list_failed"})
			return
		}
		if len(rows) > 0 || !s.portalSeedEnabled() {
			writeFactoryExceptionList(w, rows, escalatedOnly)
			return
		}
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestException(nil), s.manifestExceptions...)
	s.mu.Unlock()
	if !s.portalSeedEnabled() && s.spannerClient == nil && backend == nil {
		rows = nil
	}
	writeFactoryExceptionList(w, rows, escalatedOnly)
}

// HandleResolveManifestException serves POST /v1/factory/manifest-exceptions/{exceptionID}/resolve.
func (s *Service) HandleResolveManifestException(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	exceptionID := strings.TrimSpace(chi.URLParam(r, "exceptionID"))
	if exceptionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "exception_id_required"})
		return
	}
	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	var req struct {
		Resolution string `json:"resolution"`
		Note       string `json:"note"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	req.Resolution = strings.TrimSpace(req.Resolution)
	if req.Resolution == "" {
		req.Resolution = "RESOLVED"
	}

	row, fromMemory, found, lookupErr := s.lookupManifestException(r.Context(), exceptionID)
	if lookupErr != nil {
		s.log.ErrorContext(r.Context(), "factory exception lookup failed", "err", lookupErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_resolve_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "exception_not_found"})
		return
	}

	var orderID string
	err = s.repo.RunTx(r.Context(), func(ctx context.Context, tx FactoryTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		if fromMemory {
			s.manifestExceptions = removeMemoryExceptionLocked(s.manifestExceptions, exceptionID)
		}
		for i := range s.transfers {
			if s.transfers[i].TransferID == row.TransferID {
				orderID = s.transfers[i].OrderID
				break
			}
		}
		if strings.TrimSpace(orderID) == "" {
			orderID = row.TransferID
		}
		return tx.ResolveException(ctx, row, orderID)
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, row.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventManifestExceptionResolved},
			ManifestID:     row.ManifestID,
			ManifestDomain: events.ManifestDomainFactory,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			FactoryID:      s.resolveFactoryNode(r.Context()),
			TransferID:     row.TransferID,
			Reason:         req.Resolution,
		})
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "factory exception resolve failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_resolve_failed"})
		return
	}
	s.invalidateFactoryKeys(r.Context(), factoryExceptionListKey(s.resolveSupplierScope(r.Context())), factoryManifestKey(row.ManifestID))
	s.broadcastFactoryEvent(r.Context(), events.EventManifestExceptionResolved, map[string]any{
		"exception_id": row.ExceptionID,
		"manifest_id":  row.ManifestID,
		"resolution":   req.Resolution,
	})
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"exception_id": row.ExceptionID,
		"manifest_id":  row.ManifestID,
		"resolution":   req.Resolution,
		"note":         strings.TrimSpace(req.Note),
		"status":       "RESOLVED",
	})
}

// HandleSupplyRequests serves GET /v1/factory/supply-requests.
func (s *Service) HandleSupplyRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rows := s.loadFactorySupplyRequests(r.Context())
	now := s.now()
	mapped := make([]map[string]any, len(rows))
	for i := range rows {
		mapped[i] = supplyRequestToMap(rows[i], s.factoryNodeID, s.resolveSupplierScope(r.Context()), now)
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": mapped})
}

// HandleSLABoard GET /v1/factory/sla-board — open requests sorted breach → at_risk → on_time (G7.1).
func (s *Service) HandleSLABoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rows := s.loadFactorySupplyRequests(r.Context())
	now := s.now()
	sid := s.resolveSupplierScope(r.Context())
	var summary SLABoardSummary
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := supplyRequestToMap(row, s.factoryNodeID, sid, now)
		status, _ := m["sla_status"].(string)
		switch status {
		case SLAStatusBreached:
			summary.Breached++
			summary.TotalOpen++
			items = append(items, m)
		case SLAStatusAtRisk:
			summary.AtRisk++
			summary.TotalOpen++
			items = append(items, m)
		case SLAStatusOnTime:
			summary.OnTime++
			summary.TotalOpen++
			items = append(items, m)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		si, _ := items[i]["sla_status"].(string)
		sj, _ := items[j]["sla_status"].(string)
		if slaStatusRank(si) != slaStatusRank(sj) {
			return slaStatusRank(si) < slaStatusRank(sj)
		}
		di, _ := items[i]["sla_due_at"].(string)
		dj, _ := items[j]["sla_due_at"].(string)
		return di < dj
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": summary,
		"items":   items,
		"as_of":   now.UTC().Format(time.RFC3339Nano),
		"config": map[string]any{
			"default_hours": FactorySLADefaultHours(),
			"at_risk_hours": FactorySLAAtRiskHours(),
		},
	})
}

func (s *Service) loadFactorySupplyRequests(ctx context.Context) []SupplyRequest {
	var rows []SupplyRequest
	if s.spannerClient != nil {
		spRows, err := s.listSupplyRequestsFromSpanner(ctx)
		if err != nil {
			s.log.Warn("factory supply list spanner failed; falling back to memory", "err", err)
		} else {
			rows = spRows
		}
	}
	if len(rows) == 0 {
		s.mu.Lock()
		s.ensureDemoDataLocked()
		rows = append([]SupplyRequest(nil), s.supplyRequests...)
		s.mu.Unlock()
	}
	return rows
}

func supplyRequestToMap(row SupplyRequest, factoryID, supplierID string, now time.Time) map[string]any {
	items := make([]map[string]any, 0, len(row.Items))
	for _, item := range row.Items {
		items = append(items, map[string]any{
			"item_id":            item.ItemID,
			"product_id":         item.ProductID,
			"requested_quantity": item.RequestedQuantity,
			"recommended_qty":    item.RecommendedQty,
			"unit_volume_vu":     item.UnitVolumeVU,
		})
	}
	m := map[string]any{
		"request_id":              row.RequestID,
		"warehouse_id":            row.WarehouseID,
		"factory_id":              factoryID,
		"supplier_id":             supplierID,
		"state":                   row.Status,
		"priority":                row.Priority,
		"notes":                   row.Notes,
		"region_id":               row.RegionID,
		"requested_delivery_date": row.RequestedDeliveryDate,
		"total_volume_vu":         row.TotalVolumeVU,
		"item_count":              len(row.Items),
		"items":                   items,
		"transfer_order_id":       row.LinkedTransferID,
		"created_by":              "",
		"created_at":              row.CreatedAt,
		"updated_at":              row.UpdatedAt,
	}
	EnrichSupplyRequestSLA(m, row.Status, row.CreatedAt, row.RequestedDeliveryDate, now)
	return m
}

type acceptSupplyRequest struct {
	Reason string `json:"reason"`
}

// HandleAcceptSupplyRequest serves POST /v1/factory/supply-requests/{requestID}/accept.
func (s *Service) HandleAcceptSupplyRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	requestID := chi.URLParam(r, "requestID")
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_request_id"})
		return
	}

	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var reqBody acceptSupplyRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	nextState := "ACKNOWLEDGED"
	nowTS := s.now().UTC().Format(time.RFC3339Nano)

	if s.spannerClient != nil {
		rec, err := s.getSupplyRequestFromSpanner(r.Context(), requestID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}
		if strings.ToUpper(rec.State) != "SUBMITTED" && strings.ToUpper(rec.State) != "OPEN" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_state_transition"})
			return
		}
		if err := s.requireSupplyQCPass(r.Context(), requestID); err != nil {
			if errors.Is(err, errQCPassRequired) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "qc_pass_required"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
			return
		}
		err = s.transitionSupplyRequestSpanner(r.Context(), requestID, nextState, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, requestID, events.TopicMain, events.WarehouseEvent{
				BaseEvent:   events.BaseEvent{Type: events.EventSupplyRequestAccepted, Timestamp: nowTS},
				RequestID:   requestID,
				WarehouseID: rec.WarehouseID,
				SupplierID:  rec.SupplierID,
				FactoryID:   s.factoryNodeID,
				Status:      nextState,
			})
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		s.broadcastFactorySupplyEvent(r.Context(), map[string]any{
			"type": events.EventFactorySupplyRequestUpdate,
			"data": map[string]any{"request_id": requestID, "state": nextState, "warehouse_id": rec.WarehouseID},
		})
		idemCommitted = true
		s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"state": nextState, "updated_at": nowTS})
		return
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	var req *SupplyRequest
	for i := range s.supplyRequests {
		if s.supplyRequests[i].RequestID == requestID {
			req = &s.supplyRequests[i]
			break
		}
	}
	if req == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
		return
	}
	if req.Status != "SUBMITTED" {
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_state_transition"})
		return
	}
	s.mu.Unlock()
	if err := s.requireSupplyQCPass(r.Context(), requestID); err != nil {
		if errors.Is(err, errQCPassRequired) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "qc_pass_required"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qc_read_failed"})
		return
	}
	s.mu.Lock()
	req.Status = nextState
	req.UpdatedAt = nowTS
	s.mu.Unlock()

	err = s.repo.UpdateSupplyRequestState(r.Context(), requestID, nextState, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, requestID, events.TopicMain, events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventSupplyRequestAccepted, Timestamp: nowTS},
			RequestID:   requestID,
			WarehouseID: req.WarehouseID,
			SupplierID:  s.resolveSupplierScope(r.Context()),
			FactoryID:   s.factoryNodeID,
			Status:      nextState,
		})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error", "detail": err.Error()})
		return
	}
	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"state": nextState, "updated_at": nowTS})
}

func coalesceReason(reason, fallback string) string {
	if reason != "" {
		return reason
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
