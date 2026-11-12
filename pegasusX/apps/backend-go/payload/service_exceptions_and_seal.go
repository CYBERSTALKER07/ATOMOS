package payload

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	ordersvc "github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

func (s *Service) HandleManifestException(w http.ResponseWriter, r *http.Request) {
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
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req manifestExceptionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Reason = strings.ToUpper(strings.TrimSpace(req.Reason))
	if req.ManifestID == "" || req.OrderID == "" || req.Reason == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_order_id_reason_required"})
		return
	}
	if req.Reason != "OVERFLOW" && req.Reason != "DAMAGED" && req.Reason != "MANUAL" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_reason"})
		return
	}

	var exception ManifestException
	err = s.apply(r.Context(), func(_ PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		mIdx := s.findManifestIndexLocked(req.ManifestID)
		if mIdx < 0 {
			return fmt.Errorf("manifest_not_found")
		}
		manifest := s.manifests[mIdx]
		if manifest.State != payloadManifestStateLoading {
			return fmt.Errorf("manifest_not_loading")
		}

		moIdx := s.findManifestOrderIndexLocked(req.ManifestID, req.OrderID)
		if moIdx < 0 {
			return fmt.Errorf("manifest_order_not_found")
		}

		now := s.now().Format(time.RFC3339Nano)
		manifestOrders := s.manifestOrders[req.ManifestID]
		removedVolume := manifestOrders[moIdx].VolumeVU
		manifestOrders[moIdx].State = "REMOVED_" + req.Reason
		manifestOrders[moIdx].Reason = req.Reason
		manifestOrders[moIdx].UpdatedAt = now
		s.manifestOrders[req.ManifestID] = manifestOrders

		if manifest.TotalVolumeVU >= removedVolume {
			manifest.TotalVolumeVU -= removedVolume
		}
		if manifest.StopCount > 0 {
			manifest.StopCount--
		}
		manifest.UpdatedAt = now
		s.manifests[mIdx] = manifest

		oIdx := s.findOrderIndexLocked(req.OrderID)
		if oIdx >= 0 {
			s.orders[oIdx].ManifestID = ""
			s.orders[oIdx].Status = "PENDING"
			s.orders[oIdx].DispatchPriority = 10
			s.orders[oIdx].UpdatedAt = now
			if req.Reason == "OVERFLOW" {
				s.overflowCount[req.OrderID]++
				s.orders[oIdx].OverflowCount = s.overflowCount[req.OrderID]
			}
		}

		attemptCount := s.overflowCount[req.OrderID]
		if req.Reason != "OVERFLOW" {
			attemptCount = 1
		}
		escalated := req.Reason == "OVERFLOW" && attemptCount >= payloadExceptionEscalationThreshold
		exception = ManifestException{
			ExceptionID:  s.nextIDLocked("mex"),
			ManifestID:   req.ManifestID,
			OrderID:      req.OrderID,
			Reason:       req.Reason,
			Metadata:     strings.TrimSpace(req.Metadata),
			AttemptCount: attemptCount,
			Escalated:    escalated,
			CreatedAt:    now,
		}
		s.exceptions = append(s.exceptions, exception)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			ManifestDomain: events.ManifestDomainSupplier,
			BaseEvent:      events.BaseEvent{Type: events.EventManifestOrderException, Timestamp: exception.CreatedAt},
			ManifestID:     req.ManifestID,
			OrderID:        req.OrderID,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			Reason:         req.Reason,
			AttemptCount:   exception.AttemptCount,
			Escalated:      exception.Escalated,
		}); err != nil {
			return err
		}
		if exception.Escalated {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
				ManifestDomain: events.ManifestDomainSupplier,
				BaseEvent:      events.BaseEvent{Type: events.EventManifestDLQEscalation, Timestamp: exception.CreatedAt},
				ManifestID:     req.ManifestID,
				OrderID:        req.OrderID,
				SupplierID:     s.resolveSupplierScope(r.Context()),
				Reason:         req.Reason,
				AttemptCount:   exception.AttemptCount,
			})
		}
		return nil
	})
	if err != nil {
		switch err.Error() {
		case "manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
			return
		case "manifest_not_loading":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "manifest_not_loading"})
			return
		case "manifest_order_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_order_not_found"})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "manifest_exception_failed"})
			return
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(req.ManifestID), payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context())), payloadExceptionListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestOrderException, map[string]any{
		"manifest_id":   req.ManifestID,
		"order_id":      req.OrderID,
		"reason":        req.Reason,
		"attempt_count": exception.AttemptCount,
		"escalated":     exception.Escalated,
		"exception_id":  exception.ExceptionID,
	})
	if exception.Escalated {
		s.broadcastPayloadEvent(r.Context(), events.EventManifestDLQEscalation, map[string]any{
			"manifest_id":   req.ManifestID,
			"order_id":      req.OrderID,
			"reason":        req.Reason,
			"attempt_count": exception.AttemptCount,
			"exception_id":  exception.ExceptionID,
		})
	}

	resp := map[string]any{
		"exception_id":   exception.ExceptionID,
		"manifest_id":    exception.ManifestID,
		"order_id":       exception.OrderID,
		"reason":         exception.Reason,
		"overflow_count": exception.AttemptCount,
		"escalated":      exception.Escalated,
		"reinjected":     true,
		"new_priority":   10,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_response_failed"})
		return
	}
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleManifestExceptions serves GET /v1/payloader/manifest-exceptions.
func (s *Service) HandleManifestExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	escalatedOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("escalated")), "true")

	s.mu.Lock()
	s.ensureDemoDataLocked()
	rows := append([]ManifestException(nil), s.exceptions...)
	s.mu.Unlock()

	if escalatedOnly {
		filtered := make([]ManifestException, 0, len(rows))
		for i := range rows {
			if rows[i].Escalated {
				filtered = append(filtered, rows[i])
			}
		}
		rows = filtered
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt > rows[j].CreatedAt })
	writeJSON(w, http.StatusOK, map[string]any{"exceptions": rows})
}

// HandleRecommendReassign serves POST /v1/payloader/recommend-reassign.
func (s *Service) HandleRecommendReassign(w http.ResponseWriter, r *http.Request) {
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
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req recommendReassignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)

	s.mu.Lock()
	s.ensureDemoDataLocked()
	if req.OrderID == "" && len(s.orders) > 0 {
		for i := range s.orders {
			if s.orders[i].Status == "LOADED" || s.orders[i].Status == "PENDING" {
				req.OrderID = s.orders[i].OrderID
				break
			}
		}
		if req.OrderID == "" {
			req.OrderID = s.orders[0].OrderID
		}
	}

	oIdx := s.findOrderIndexLocked(req.OrderID)
	if oIdx < 0 {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	order := s.orders[oIdx]
	var orderRecord *ordersvc.Order
	if s.orderReader != nil {
		if ord, ok, _ := s.orderReader.GetOrder(r.Context(), order.OrderID); ok {
			orderRecord = &ord
			if order.Lat == 0 && order.Lng == 0 {
				order.Lat = ord.Lat
				order.Lng = ord.Lng
				s.orders[oIdx].Lat = ord.Lat
				s.orders[oIdx].Lng = ord.Lng
			}
		}
	}

	recommendations := make([]ReassignRecommendation, 0)
	for i := range s.manifests {
		manifest := s.manifests[i]
		if manifest.State != payloadManifestStateDraft && manifest.State != payloadManifestStateLoading {
			continue
		}
		toRoute := routeIDForManifest(manifest)
		if toRoute == order.RouteID {
			continue
		}
		loadPenalty := 0.0
		if manifest.MaxVolumeVU > 0 {
			loadPenalty = (float64(manifest.TotalVolumeVU) / float64(manifest.MaxVolumeVU)) * 40.0
		}
		score := 100.0 - float64(manifest.StopCount*6) - loadPenalty
		if score < 1 {
			score = 1
		}
		recommendations = append(recommendations, ReassignRecommendation{
			OrderID:    order.OrderID,
			FromRoute:  order.RouteID,
			ToRoute:    toRoute,
			ToDriverID: manifest.DriverID,
			Score:      score,
			Reason:     "lower_route_load",
		})
	}
	truckRecs := buildTruckRecommendationsLocked(s, order, recommendations)

	// Authoritative retailer name resolution
	retailerName := "Retailer"
	if orderRecord != nil && orderRecord.RetailerID != "" {
		retailerName = orderRecord.RetailerID
	}

	// Physical cargo volume derivation (zero financial conflation)
	orderVolume := float64(0)
	if oIdx >= 0 {
		// Priority 1: Volume assigned in active/loading manifest
		if order.ManifestID != "" {
			for _, mo := range s.manifestOrders[order.ManifestID] {
				if mo.OrderID == order.OrderID && mo.VolumeVU > 0 {
					orderVolume = float64(mo.VolumeVU)
					break
				}
			}
		}
		// Priority 2: Volume found across any in-memory manifest orders
		if orderVolume <= 0 {
			for _, mOrders := range s.manifestOrders {
				for _, mo := range mOrders {
					if mo.OrderID == order.OrderID && mo.VolumeVU > 0 {
						orderVolume = float64(mo.VolumeVU)
						break
					}
				}
				if orderVolume > 0 {
					break
				}
			}
		}
		// Priority 3: Compute physical volume directly from line-item dimensions
		if orderVolume <= 0 && orderRecord != nil && len(orderRecord.LineItems) > 0 {
			var physVol float64
			for _, item := range orderRecord.LineItems {
				qty := float64(item.Quantity)
				if qty <= 0 {
					qty = 1
				}
				unitVU := item.UnitVolumeVU
				if unitVU <= 0 {
					unitVU = 1.0 // baseline 1 VU per item unit
				}
				physVol += qty * unitVU
			}
			if physVol > 0 {
				orderVolume = physVol
			}
		}
		// Priority 4: Cached OrderRow VolumeVU
		if orderVolume <= 0 && order.VolumeVU > 0 {
			orderVolume = float64(order.VolumeVU)
		}
	}
	// Floor: Minimum physical cargo volume is 1.0 VU (never 0, never arbitrary 10.0, never money)
	if orderVolume <= 0 {
		orderVolume = 1.0
	}

	currentDriver := order.RouteID
	s.mu.Unlock()

	sort.Slice(truckRecs, func(i, j int) bool { return truckRecs[i].Score > truckRecs[j].Score })
	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"order_id":        req.OrderID,
		"retailer_name":   retailerName,
		"order_volume_vu": orderVolume,
		"current_driver":  currentDriver,
		"recommendations": truckRecs,
	})
}

// HandleApplyReassign serves POST /v1/payloader/reassign-order.
func (s *Service) HandleApplyReassign(w http.ResponseWriter, r *http.Request) {
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
	var req applyReassignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.ToRouteID = strings.TrimSpace(req.ToRouteID)
	req.ToManifestID = strings.TrimSpace(req.ToManifestID)
	req.ToDriverID = strings.TrimSpace(req.ToDriverID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	var reassignment Reassignment
	err = s.apply(r.Context(), func(tx PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		oIdx := s.findOrderIndexLocked(req.OrderID)
		if oIdx < 0 {
			return fmt.Errorf("order_not_found")
		}
		order := s.orders[oIdx]
		fromRoute := order.RouteID
		fromManifestID := order.ManifestID
		targetManifestID := req.ToManifestID
		targetRouteID := req.ToRouteID
		targetDriverID := req.ToDriverID
		reassignedVolume := int64(10)
		if req.IsPartial && req.PartialVolumeVU > 0 {
			reassignedVolume = req.PartialVolumeVU
		}
		if fromManifestID != "" {
			fromManifestOrders := s.manifestOrders[fromManifestID]
			fromManifestOrderIdx := s.findManifestOrderIndexLocked(fromManifestID, order.OrderID)
			if fromManifestOrderIdx >= 0 && fromManifestOrders[fromManifestOrderIdx].VolumeVU > 0 {
				if req.IsPartial && req.PartialVolumeVU == 0 {
					reassignedVolume = fromManifestOrders[fromManifestOrderIdx].VolumeVU / 2
				} else if !req.IsPartial {
					reassignedVolume = fromManifestOrders[fromManifestOrderIdx].VolumeVU
				}
			}
		}

		if targetManifestID != "" {
			if targetManifestID == order.ManifestID && (targetRouteID == "" || targetRouteID == order.RouteID) {
				return fmt.Errorf("order_already_assigned")
			}
			targetManifestIdx := s.findManifestIndexLocked(targetManifestID)
			if targetManifestIdx < 0 {
				return fmt.Errorf("target_manifest_not_found")
			}
			targetManifest := s.manifests[targetManifestIdx]
			if targetManifest.State != payloadManifestStateDraft && targetManifest.State != payloadManifestStateLoading {
				return fmt.Errorf("target_manifest_not_mutable")
			}
			resolvedTargetRoute := routeIDForManifest(targetManifest)
			if targetRouteID != "" && targetRouteID != resolvedTargetRoute {
				return fmt.Errorf("target_route_mismatch")
			}
			targetRouteID = resolvedTargetRoute
			if targetManifest.MaxVolumeVU > 0 && targetManifest.TotalVolumeVU+reassignedVolume > targetManifest.MaxVolumeVU {
				return fmt.Errorf("target_manifest_capacity_exceeded")
			}
			if targetDriverID != "" && targetManifest.DriverID != targetDriverID {
				return fmt.Errorf("target_driver_manifest_mismatch")
			}
			if targetDriverID == "" {
				targetDriverID = targetManifest.DriverID
			}
		} else {
			sourceManifestMutable := false
			sourceManifestDriverID := ""
			if order.ManifestID != "" {
				sourceManifestIdx := s.findManifestIndexLocked(order.ManifestID)
				if sourceManifestIdx >= 0 {
					sourceManifest := s.manifests[sourceManifestIdx]
					sourceManifestDriverID = sourceManifest.DriverID
					if sourceManifest.State == payloadManifestStateDraft || sourceManifest.State == payloadManifestStateLoading {
						sourceManifestMutable = true
					}
				}
			}
			if targetRouteID != "" && targetRouteID == order.RouteID && sourceManifestMutable {
				if targetDriverID != "" && targetDriverID != sourceManifestDriverID {
					return fmt.Errorf("target_driver_manifest_mismatch")
				}
				return fmt.Errorf("order_already_assigned")
			}
			driverMismatchOnTargetRoute := false
			for i := range s.manifests {
				candidateManifest := s.manifests[i]
				if candidateManifest.State != payloadManifestStateDraft && candidateManifest.State != payloadManifestStateLoading {
					continue
				}
				if candidateManifest.ManifestID == order.ManifestID {
					continue
				}
				candidate := routeIDForManifest(candidateManifest)
				if candidate == order.RouteID {
					if targetRouteID == "" || sourceManifestMutable {
						continue
					}
				}
				if targetRouteID != "" && candidate != targetRouteID {
					continue
				}
				if targetDriverID != "" && candidateManifest.DriverID != targetDriverID {
					if targetRouteID != "" && candidate == targetRouteID {
						driverMismatchOnTargetRoute = true
					}
					continue
				}
				if candidateManifest.MaxVolumeVU > 0 && candidateManifest.TotalVolumeVU+reassignedVolume > candidateManifest.MaxVolumeVU {
					continue
				}
				targetRouteID = candidate
				targetManifestID = candidateManifest.ManifestID
				if targetDriverID == "" {
					targetDriverID = candidateManifest.DriverID
				}
				break
			}
			if targetManifestID == "" {
				if targetRouteID != "" {
					if targetDriverID != "" && driverMismatchOnTargetRoute {
						return fmt.Errorf("target_driver_manifest_mismatch")
					}
					return fmt.Errorf("target_manifest_not_found")
				}
				if sourceManifestMutable {
					return fmt.Errorf("order_already_assigned")
				}
				return fmt.Errorf("reassign_target_unavailable")
			}
		}
		if targetManifestID == order.ManifestID {
			return fmt.Errorf("order_already_assigned")
		}

		sourceManifestIdx := -1
		if order.ManifestID != "" {
			sourceManifestIdx = s.findManifestIndexLocked(order.ManifestID)
			if sourceManifestIdx < 0 {
				return fmt.Errorf("source_manifest_not_found")
			}
			sourceRouteID := routeIDForManifest(s.manifests[sourceManifestIdx])
			if order.RouteID != "" && order.RouteID != sourceRouteID {
				return fmt.Errorf("source_route_manifest_mismatch")
			}
			if s.findManifestOrderIndexLocked(order.ManifestID, order.OrderID) < 0 {
				return fmt.Errorf("source_manifest_order_missing")
			}
		}

		now := s.now().Format(time.RFC3339Nano)
		splitGroupID := order.SplitGroupID
		if req.IsPartial && splitGroupID == "" {
			splitGroupID = uuid.NewString()
			s.orders[oIdx].SplitGroupID = splitGroupID
		}

		if sourceManifestIdx >= 0 {
			fromManifestOrders := s.manifestOrders[order.ManifestID]
			moIdx := s.findManifestOrderIndexLocked(order.ManifestID, order.OrderID)
			removedVol := reassignedVolume
			if req.IsPartial {
				fromManifestOrders[moIdx].VolumeVU -= removedVol
				if fromManifestOrders[moIdx].VolumeVU < 0 {
					fromManifestOrders[moIdx].VolumeVU = 0
				}
				fromManifestOrders[moIdx].SplitGroupID = splitGroupID
			} else {
				fromManifestOrders[moIdx].State = "REMOVED_REASSIGNED"
				fromManifestOrders[moIdx].Reason = "REASSIGNED"
			}
			fromManifestOrders[moIdx].UpdatedAt = now
			s.manifestOrders[order.ManifestID] = fromManifestOrders
			if !req.IsPartial && s.manifests[sourceManifestIdx].StopCount > 0 {
				s.manifests[sourceManifestIdx].StopCount--
			}
			if s.manifests[sourceManifestIdx].TotalVolumeVU >= removedVol {
				s.manifests[sourceManifestIdx].TotalVolumeVU -= removedVol
			}
			s.manifests[sourceManifestIdx].UpdatedAt = now
		}

		if targetManifestID != "" {
			targetManifestIdx := s.findManifestIndexLocked(targetManifestID)
			if targetManifestIdx < 0 {
				return fmt.Errorf("target_manifest_not_found")
			}
			targetManifest := s.manifests[targetManifestIdx]
			if targetManifest.MaxVolumeVU > 0 && targetManifest.TotalVolumeVU+reassignedVolume > targetManifest.MaxVolumeVU {
				return fmt.Errorf("target_manifest_capacity_exceeded")
			}
			targetManifestOrders := s.manifestOrders[targetManifestID]
			targetManifestOrders = append(targetManifestOrders, ManifestOrder{
				ManifestID:   targetManifestID,
				OrderID:      order.OrderID,
				State:        "ASSIGNED",
				VolumeVU:     reassignedVolume,
				SplitGroupID: splitGroupID,
				UpdatedAt:    now,
			})
			s.manifestOrders[targetManifestID] = targetManifestOrders
			targetManifest.StopCount++
			targetManifest.TotalVolumeVU += reassignedVolume
			targetManifest.UpdatedAt = now
			s.manifests[targetManifestIdx] = targetManifest
		}

		s.orders[oIdx].RouteID = targetRouteID
		s.orders[oIdx].ManifestID = targetManifestID
		s.orders[oIdx].Status = "LOADED"
		s.orders[oIdx].ReassignDepth++
		s.orders[oIdx].UpdatedAt = now

		reassignment = Reassignment{
			OrderID:        order.OrderID,
			FromRouteID:    fromRoute,
			ToRouteID:      targetRouteID,
			FromManifestID: fromManifestID,
			ManifestID:     targetManifestID,
			ToDriverID:     targetDriverID,
			Reason:         strings.TrimSpace(req.Reason),
			Depth:          s.orders[oIdx].ReassignDepth,
			AppliedAt:      now,
		}
		s.reassignments = append(s.reassignments, reassignment)
		if err := tx.UpdateOrderAssignment(r.Context(), req.OrderID, targetRouteID, targetDriverID); err != nil {
			return err
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		aggregateID := coalesceString(reassignment.ManifestID, reassignment.FromManifestID, "payload-reassign")
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, aggregateID, events.TopicMain, events.ManifestEvent{
			ManifestDomain: events.ManifestDomainSupplier,
			BaseEvent:      events.BaseEvent{Type: events.EventManifestRebalanced, Timestamp: reassignment.AppliedAt},
			ManifestID:     aggregateID,
			FromManifestID: reassignment.FromManifestID,
			ToManifestID:   reassignment.ManifestID,
			OrderID:        req.OrderID,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			FromRouteID:    reassignment.FromRouteID,
			ToRouteID:      reassignment.ToRouteID,
			ToDriverID:     reassignment.ToDriverID,
			Depth:          reassignment.Depth,
			Reason:         reassignment.Reason,
		})
	})
	if err != nil {
		switch err.Error() {
		case "order_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
			return
		case "target_manifest_not_found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "target_manifest_not_found"})
			return
		case "target_manifest_not_mutable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target_manifest_not_mutable"})
			return
		case "source_manifest_not_found":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "source_manifest_not_found"})
			return
		case "source_route_manifest_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "source_route_manifest_mismatch"})
			return
		case "source_manifest_order_missing":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "source_manifest_order_missing"})
			return
		case "target_route_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target_route_mismatch"})
			return
		case "target_driver_manifest_mismatch":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target_driver_manifest_mismatch"})
			return
		case "target_manifest_capacity_exceeded":
			platform.WriteErrorWithExplain(w, http.StatusConflict, "target_manifest_capacity_exceeded", err)
			return
		case "reassign_target_unavailable":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "reassign_target_unavailable"})
			return
		case "order_already_assigned":
			s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{"status": "already_assigned", "order_id": req.OrderID})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reassign_failed"})
			return
		}
	}

	keys := []string{payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context()))}
	if reassignment.FromManifestID != "" {
		keys = append(keys, payloadManifestKey(reassignment.FromManifestID))
	}
	if reassignment.ManifestID != "" {
		keys = append(keys, payloadManifestKey(reassignment.ManifestID))
	}
	s.invalidatePayloadKeys(r.Context(), keys...)
	s.broadcastPayloadEvent(r.Context(), events.EventManifestRebalanced, map[string]any{
		"order_id":         reassignment.OrderID,
		"from_route_id":    reassignment.FromRouteID,
		"to_route_id":      reassignment.ToRouteID,
		"from_manifest_id": reassignment.FromManifestID,
		"to_manifest_id":   reassignment.ManifestID,
		"manifest_id":      reassignment.ManifestID,
		"to_driver_id":     reassignment.ToDriverID,
		"reassign_depth":   reassignment.Depth,
		"applied_at":       reassignment.AppliedAt,
	})

	if reassignment.ToDriverID != "" {
		s.broadcastDriverEvent(r.Context(), reassignment.ToDriverID, map[string]any{
			"type":        "MANIFEST_AMENDED",
			"driver_id":   reassignment.ToDriverID,
			"manifest_id": reassignment.ManifestID,
			"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
		})
	}

	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":         "order_reassigned",
		"order_id":       reassignment.OrderID,
		"from_route_id":  reassignment.FromRouteID,
		"to_route_id":    reassignment.ToRouteID,
		"to_manifest_id": reassignment.ManifestID,
		"reassign_depth": reassignment.Depth,
		"applied_at":     reassignment.AppliedAt,
	})
}

// HandleSealCompletedManifests serves POST /v1/payloader/manifests/seal-completed.
func (s *Service) HandleSealCompletedManifests(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		ManifestIDs []string `json:"manifest_ids"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	results := make([]map[string]any, 0, len(req.ManifestIDs))
	sealedCount := 0
	for _, manifestID := range req.ManifestIDs {
		manifestID = strings.TrimSpace(manifestID)
		if manifestID == "" {
			continue
		}
		_ = s.hydrateFromRepo(r.Context())
		s.mu.Lock()
		idx := s.findManifestIndexLocked(manifestID)
		alreadySealed := idx >= 0 && s.manifests[idx].State == payloadManifestStateSealed
		s.mu.Unlock()
		if alreadySealed {
			sealedCount++
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "already_sealed",
				"state":       payloadManifestStateSealed,
			})
			continue
		}
		if _, gateErr := s.assertPhysicalSealGates(r.Context(), manifestID); gateErr != nil {
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "pick_wave_blocked",
				"error":       gateErr.Error(),
			})
			continue
		}
		var manifest ManifestRow
		err := s.apply(r.Context(), func(_ PayloadTx) error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.ensureDemoDataLocked()
			var sealErr error
			manifest, sealErr = s.sealManifestLocked(manifestID)
			return sealErr
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
				ManifestDomain: events.ManifestDomainSupplier,
				BaseEvent:      events.BaseEvent{Type: events.EventManifestSealed, Timestamp: manifest.UpdatedAt},
				ManifestID:     manifestID,
				SupplierID:     s.resolveSupplierScope(r.Context()),
				WarehouseID:    s.resolveWarehouseScope(r.Context()),
				State:          payloadManifestStateSealed,
				RouteID:        routeIDForManifest(manifest),
				DriverID:       manifest.DriverID,
				VehicleID:      manifest.VehicleID,
				OrderCount:     manifest.StopCount,
			})
		})
		switch {
		case err == http.ErrMissingFile:
			row := map[string]any{"manifest_id": manifestID, "status": "not_found"}
			platform.AttachExplainToMap(row, "manifest_not_found")
			results = append(results, row)
		case err == http.ErrBodyNotAllowed:
			row := map[string]any{"manifest_id": manifestID, "status": "not_sealable"}
			platform.AttachExplainToMap(row, "manifest_not_sealable")
			results = append(results, row)
		case err != nil:
			row := map[string]any{"manifest_id": manifestID, "status": "seal_failed"}
			platform.AttachExplainToMap(row, "manifest_seal_failed")
			results = append(results, row)
		default:
			sealedCount++
			if s.manifestStore != nil {
				if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), manifestID, "manifest_sealed"); geomErr != nil {
					s.log.WarnContext(r.Context(), "manifest route geometry persist failed",
						"manifest_id", manifestID, "err", geomErr)
				}
			}
			s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context())))
			s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
				"manifest_id": manifestID,
				"state":       payloadManifestStateSealed,
				"route_id":    routeIDForManifest(manifest),
				"driver_id":   manifest.DriverID,
				"vehicle_id":  manifest.VehicleID,
				"order_count": manifest.StopCount,
				"updated_at":  manifest.UpdatedAt,
			})
			if manifest.DriverID != "" {
				s.broadcastDriverEvent(r.Context(), manifest.DriverID, map[string]any{
					"type":        "MANIFEST_DISPATCHED",
					"driver_id":   manifest.DriverID,
					"manifest_id": manifestID,
					"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
				})
			}
			s.afterSealAssignSSCC(r.Context(), manifestID)
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "sealed",
				"state":       manifest.State,
				"sealed_at":   manifest.SealedAt,
				"driver_id":   manifest.DriverID,
				"vehicle_id":  manifest.VehicleID,
			})
		}
	}

	resp := map[string]any{
		"status":       "batch_seal_complete",
		"sealed_count": sealedCount,
		"results":      results,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleSealManifest serves POST /v1/payloader/manifests/{manifestID}/seal.
func (s *Service) HandleSealManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	sealPickWarn, gateErr := s.assertPhysicalSealGates(r.Context(), manifestID)
	if gateErr != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": gateErr.Error()})
		return
	}

	var manifest ManifestRow
	err = s.apply(r.Context(), func(_ PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		var sealErr error
		manifest, sealErr = s.sealManifestLocked(manifestID)
		return sealErr
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
			ManifestDomain: events.ManifestDomainSupplier,
			BaseEvent:      events.BaseEvent{Type: events.EventManifestSealed, Timestamp: manifest.UpdatedAt},
			ManifestID:     manifestID,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			State:          payloadManifestStateSealed,
			RouteID:        routeIDForManifest(manifest),
			DriverID:       manifest.DriverID,
			VehicleID:      manifest.VehicleID,
			OrderCount:     manifest.StopCount,
		})
	})
	if err == http.ErrMissingFile {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		return
	}
	if err == http.ErrBodyNotAllowed {
		platform.WriteErrorWithExplain(w, http.StatusConflict, "manifest_not_sealable", err)
		return
	}
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "manifest_seal_failed", err)
		return
	}

	if s.manifestStore != nil {
		if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), manifestID, "manifest_sealed"); geomErr != nil {
			s.log.WarnContext(r.Context(), "manifest route geometry persist failed",
				"manifest_id", manifestID, "err", geomErr)
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(manifestID), payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
		"manifest_id": manifestID,
		"state":       payloadManifestStateSealed,
		"route_id":    routeIDForManifest(manifest),
		"driver_id":   manifest.DriverID,
		"vehicle_id":  manifest.VehicleID,
		"order_count": manifest.StopCount,
		"updated_at":  manifest.UpdatedAt,
	})

	if manifest.DriverID != "" {
		s.broadcastDriverEvent(r.Context(), manifest.DriverID, map[string]any{
			"type":        "MANIFEST_DISPATCHED",
			"driver_id":   manifest.DriverID,
			"manifest_id": manifestID,
			"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
		})
	}

	resp := map[string]any{
		"status":      "PAYLOAD_MANIFEST_SEALED",
		"manifest_id": manifest.ManifestID,
		"state":       manifest.State,
		"sealed_at":   manifest.SealedAt,
		"stop_count":  manifest.StopCount,
		"volume_vu":   float64(manifest.TotalVolumeVU),
		"max_vu":      float64(manifest.MaxVolumeVU),
	}
	if sealPickWarn != "" {
		resp["pick_wave_warning"] = sealPickWarn
	}
	s.afterSealAssignSSCC(r.Context(), manifestID)

	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleSeal serves POST /v1/payload/seal.
func (s *Service) HandleSeal(w http.ResponseWriter, r *http.Request) {
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
	var req sealRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.ManifestID = strings.TrimSpace(req.ManifestID)
	if req.OrderID == "" && req.ManifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}
	// B2 M-P0-7: order-only seal is a silent memory mutation — require manifest_id
	// and go through apply (outbox). Never hold s.mu across apply (deadlock).
	if req.ManifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "manifest_id_required",
			"message": "POST /v1/payload/seal requires manifest_id; order-only seal is forbidden (use /v1/payloader/manifests/{id}/seal)",
		})
		return
	}
	if _, gateErr := s.assertPhysicalSealGates(r.Context(), req.ManifestID); gateErr != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": gateErr.Error()})
		return
	}

	// B7 PL-P0-6: seal only manifests in JWT warehouse when both sides are set.
	_ = s.hydrateFromRepo(r.Context())
	s.mu.Lock()
	s.ensureManifestStateLocked()
	preIdx := s.findManifestIndexLocked(req.ManifestID)
	var preRow ManifestRow
	if preIdx >= 0 {
		preRow = s.manifests[preIdx]
	}
	s.mu.Unlock()
	if preIdx >= 0 {
		if scopeErr := s.assertManifestWarehouseScope(r.Context(), preRow); scopeErr != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_forbidden"})
			return
		}
	}

	var manifest ManifestRow
	err = s.apply(r.Context(), func(_ PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureDemoDataLocked()
		var sealErr error
		manifest, sealErr = s.sealManifestLocked(req.ManifestID)
		return sealErr
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, req.ManifestID, events.TopicMain, events.ManifestEvent{
			ManifestDomain: events.ManifestDomainSupplier,
			BaseEvent:      events.BaseEvent{Type: events.EventManifestSealed, Timestamp: manifest.UpdatedAt},
			ManifestID:     req.ManifestID,
			SupplierID:     s.resolveSupplierScope(r.Context()),
			WarehouseID:    s.resolveWarehouseScope(r.Context()),
			State:          payloadManifestStateSealed,
			RouteID:        routeIDForManifest(manifest),
			DriverID:       manifest.DriverID,
			VehicleID:      manifest.VehicleID,
			OrderCount:     manifest.StopCount,
		})
	})
	if err == http.ErrMissingFile {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		return
	}
	if err == http.ErrBodyNotAllowed {
		platform.WriteErrorWithExplain(w, http.StatusConflict, "manifest_not_sealable", err)
		return
	}
	if err != nil {
		platform.WriteErrorWithExplain(w, http.StatusInternalServerError, "manifest_seal_failed", err)
		return
	}

	if s.manifestStore != nil {
		if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), req.ManifestID, "manifest_sealed"); geomErr != nil {
			s.log.WarnContext(r.Context(), "manifest route geometry persist failed",
				"manifest_id", req.ManifestID, "err", geomErr)
		}
	}

	s.invalidatePayloadKeys(r.Context(), payloadManifestKey(req.ManifestID), payloadManifestListKey(s.resolveSupplierScope(r.Context())), payloadOrderListKey(s.resolveSupplierScope(r.Context())))
	s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
		"manifest_id":  req.ManifestID,
		"state":        payloadManifestStateSealed,
		"route_id":     routeIDForManifest(manifest),
		"driver_id":    manifest.DriverID,
		"vehicle_id":   manifest.VehicleID,
		"warehouse_id": s.resolveWarehouseScope(r.Context()),
		"order_count":  manifest.StopCount,
		"updated_at":   manifest.UpdatedAt,
	})
	s.afterSealAssignSSCC(r.Context(), req.ManifestID)
	resp := map[string]any{
		"status":      "PAYLOAD_MANIFEST_SEALED",
		"manifest_id": manifest.ManifestID,
		"state":       manifest.State,
		"sealed_at":   manifest.SealedAt,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func coalesceString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func (s *Service) HandleSealAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := readLimitedBody(r, 4*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}

	var req struct {
		// StateFilter, if set, limits sealing to manifests in that state only.
		// Valid values: "DRAFT", "LOADING". Empty means both are sealed.
		StateFilter string `json:"state_filter"`
	}
	// body may be empty for a no-args call — treat as "seal everything".
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	stateFilter := strings.ToUpper(strings.TrimSpace(req.StateFilter))

	// Collect candidate manifest IDs under the read lock.
	s.mu.RLock()
	var candidateIDs []string
	for _, m := range s.manifests {
		state := strings.ToUpper(m.State)
		if stateFilter != "" && state != stateFilter {
			continue
		}
		if state == payloadManifestStateDraft || state == payloadManifestStateLoading {
			candidateIDs = append(candidateIDs, m.ManifestID)
		}
	}
	s.mu.RUnlock()

	if len(candidateIDs) == 0 {
		resp := map[string]any{
			"status":       "no_sealable_manifests",
			"sealed_count": 0,
			"results":      []any{},
		}
		respBytes, _ := json.Marshal(resp)
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}

	results := make([]map[string]any, 0, len(candidateIDs))
	sealedCount := 0

	for _, manifestID := range candidateIDs {
		if _, gateErr := s.assertPhysicalSealGates(r.Context(), manifestID); gateErr != nil {
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "pick_wave_blocked",
				"error":       gateErr.Error(),
			})
			continue
		}
		var sealed ManifestRow
		sealErr := s.apply(r.Context(), func(_ PayloadTx) error {
			s.mu.Lock()
			defer s.mu.Unlock()
			var e error
			sealed, e = s.sealManifestLocked(manifestID)
			return e
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateManifest, manifestID, events.TopicMain, events.ManifestEvent{
				ManifestDomain: events.ManifestDomainSupplier,
				BaseEvent:      events.BaseEvent{Type: events.EventManifestSealed, Timestamp: sealed.UpdatedAt},
				ManifestID:     manifestID,
				SupplierID:     s.resolveSupplierScope(r.Context()),
				WarehouseID:    s.resolveWarehouseScope(r.Context()),
				State:          payloadManifestStateSealed,
				RouteID:        routeIDForManifest(sealed),
				DriverID:       sealed.DriverID,
				VehicleID:      sealed.VehicleID,
				OrderCount:     sealed.StopCount,
			})
		})
		switch {
		case sealErr == http.ErrMissingFile:
			row := map[string]any{"manifest_id": manifestID, "status": "not_found"}
			platform.AttachExplainToMap(row, "manifest_not_found")
			results = append(results, row)
		case sealErr == http.ErrBodyNotAllowed:
			// Already sealed or dispatched — skip silently in seal-all context.
			row := map[string]any{"manifest_id": manifestID, "status": "already_sealed"}
			results = append(results, row)
		case sealErr != nil:
			row := map[string]any{"manifest_id": manifestID, "status": "seal_failed", "error": sealErr.Error()}
			platform.AttachExplainToMap(row, "manifest_seal_failed")
			results = append(results, row)
		default:
			sealedCount++
			if s.manifestStore != nil {
				if geomErr := s.manifestStore.PersistRouteGeometryForManifest(r.Context(), manifestID, "manifest_sealed"); geomErr != nil {
					s.log.WarnContext(r.Context(), "seal-all: route geometry persist failed",
						"manifest_id", manifestID, "err", geomErr)
				}
			}
			s.invalidatePayloadKeys(r.Context(),
				payloadManifestKey(manifestID),
				payloadManifestListKey(s.resolveSupplierScope(r.Context())),
				payloadOrderListKey(s.resolveSupplierScope(r.Context())),
			)
			s.broadcastPayloadEvent(r.Context(), events.EventManifestSealed, map[string]any{
				"manifest_id": manifestID,
				"state":       payloadManifestStateSealed,
				"route_id":    routeIDForManifest(sealed),
				"driver_id":   sealed.DriverID,
				"vehicle_id":  sealed.VehicleID,
				"order_count": sealed.StopCount,
				"updated_at":  sealed.UpdatedAt,
			})
			if sealed.DriverID != "" {
				s.broadcastDriverEvent(r.Context(), sealed.DriverID, map[string]any{
					"type":        "MANIFEST_DISPATCHED",
					"driver_id":   sealed.DriverID,
					"manifest_id": manifestID,
					"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
				})
			}
			s.afterSealAssignSSCC(r.Context(), manifestID)
			results = append(results, map[string]any{
				"manifest_id": manifestID,
				"status":      "sealed",
				"state":       sealed.State,
				"sealed_at":   sealed.SealedAt,
				"driver_id":   sealed.DriverID,
				"vehicle_id":  sealed.VehicleID,
			})
		}
	}

	resp := map[string]any{
		"status":          "seal_all_complete",
		"sealed_count":    sealedCount,
		"candidate_count": len(candidateIDs),
		"results":         results,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}
