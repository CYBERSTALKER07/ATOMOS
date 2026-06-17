package payload

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestHandleManifestException_EscalationSeamParity(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	// Prepare manifest/order state to trigger OVERFLOW escalation on this request.
	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	mIdx := svc.findManifestIndexLocked("mf_payload_1")
	svc.manifests[mIdx].State = payloadManifestStateLoading
	svc.overflowCount["ord_payload_1"] = payloadExceptionEscalationThreshold - 1
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payload/manifest-exception", strings.NewReader(`{"manifest_id":"mf_payload_1","order_id":"ord_payload_1","reason":"OVERFLOW","metadata":"retry"}`))
	rr := httptest.NewRecorder()

	svc.HandleManifestException(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}

	types := payloadOutboxEventTypes(repo.events)
	assertPayloadContainsEventType(t, types, events.EventManifestOrderException)
	assertPayloadContainsEventType(t, types, events.EventManifestDLQEscalation)

	assertPayloadCacheDeletedKeys(t, cacheBackend.deletedKeys,
		payloadManifestKey("mf_payload_1"),
		payloadManifestListKey("supplier-test"),
		payloadOrderListKey("supplier-test"),
		payloadExceptionListKey("supplier-test"),
	)

	assertPayloadWSMessageContainsType(t, supplierConn.messages, events.EventManifestOrderException)
	assertPayloadWSMessageContainsType(t, supplierConn.messages, events.EventManifestDLQEscalation)
	assertPayloadWSMessageContainsType(t, payloadConn.messages, events.EventManifestOrderException)
	assertPayloadWSMessageContainsType(t, payloadConn.messages, events.EventManifestDLQEscalation)
}

func TestHandleApplyReassign_SeamParity(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	if svc.findManifestIndexLocked("mf_payload_2") < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo manifest mf_payload_2")
	}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"balance"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}

	types := payloadOutboxEventTypes(repo.events)
	if len(types) != 1 || types[0] != events.EventManifestRebalanced {
		t.Fatalf("unexpected outbox events: %#v", types)
	}

	var payload map[string]any
	if err := json.Unmarshal(repo.events[0].Payload, &payload); err != nil {
		t.Fatalf("failed to decode outbox payload: %v", err)
	}
	if got, _ := payload["order_id"].(string); got != "ord_payload_1" {
		t.Fatalf("expected order_id ord_payload_1, got %q", got)
	}
	if got, _ := payload["from_manifest_id"].(string); got != "mf_payload_1" {
		t.Fatalf("expected from_manifest_id mf_payload_1, got %q", got)
	}
	if got, _ := payload["to_manifest_id"].(string); got != "mf_payload_2" {
		t.Fatalf("expected to_manifest_id mf_payload_2, got %q", got)
	}
	if got, _ := payload["from_route_id"].(string); got != "route_veh_payload_1" {
		t.Fatalf("expected from_route_id route_veh_payload_1, got %q", got)
	}
	if got, _ := payload["to_route_id"].(string); got != "route_veh_payload_2" {
		t.Fatalf("expected to_route_id route_veh_payload_2, got %q", got)
	}

	assertPayloadCacheDeletedKeys(t, cacheBackend.deletedKeys,
		payloadManifestKey("mf_payload_1"),
		payloadManifestListKey("supplier-test"),
		payloadOrderListKey("supplier-test"),
		payloadManifestKey("mf_payload_2"),
	)

	assertPayloadWSMessageContainsType(t, supplierConn.messages, events.EventManifestRebalanced)
	assertPayloadWSMessageContainsType(t, payloadConn.messages, events.EventManifestRebalanced)
}

func TestHandleApplyReassign_TargetManifestCapacityExceeded(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	targetIdx := svc.findManifestIndexLocked("mf_payload_2")
	if targetIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo manifest mf_payload_2")
	}
	svc.manifests[targetIdx].TotalVolumeVU = 9
	svc.manifests[targetIdx].MaxVolumeVU = 10
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}

	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"capacity-test"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "target_manifest_capacity_exceeded" {
		t.Fatalf("expected target_manifest_capacity_exceeded, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected zero outbox events, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation, got %#v", cacheBackend.deletedKeys)
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after conflict")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
}

func TestHandleApplyReassign_OrderAlreadyAssignedNoop(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	originalDepth := svc.orders[orderIdx].ReassignDepth
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_1","reason":"noop"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["status"].(string); got != "already_assigned" {
		t.Fatalf("expected status already_assigned, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events for already_assigned, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation for already_assigned")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events for already_assigned")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events for already_assigned")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after no-op")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth {
		t.Fatalf("expected reassign depth %d to remain unchanged, got %d", originalDepth, got)
	}
}

func TestHandleApplyReassign_TargetUnavailableConflict(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	for i := range svc.manifests {
		svc.manifests[i].State = payloadManifestStateSealed
	}
	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","reason":"auto-no-target"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "reassign_target_unavailable" {
		t.Fatalf("expected reassign_target_unavailable, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events when target unavailable, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation when target unavailable")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events when target unavailable")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events when target unavailable")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after conflict")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
}

func TestHandleApplyReassign_SameManifestCandidateNoop(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	originalDepth := svc.orders[orderIdx].ReassignDepth
	originalManifestOrderCount := len(svc.manifestOrders[originalManifestID])
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_1","reason":"same-manifest"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["status"].(string); got != "already_assigned" {
		t.Fatalf("expected status already_assigned, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events for same-manifest no-op, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation for same-manifest no-op")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events for same-manifest no-op")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events for same-manifest no-op")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after no-op")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth {
		t.Fatalf("expected reassign depth %d to remain unchanged, got %d", originalDepth, got)
	}
	if got := len(svc.manifestOrders[originalManifestID]); got != originalManifestOrderCount {
		t.Fatalf("expected manifest order count %d to remain unchanged, got %d", originalManifestOrderCount, got)
	}
}

func TestHandleApplyReassign_ExplicitSameRouteNoop(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	originalDepth := svc.orders[orderIdx].ReassignDepth
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_route_id":"route_veh_payload_1","reason":"explicit-same-route"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["status"].(string); got != "already_assigned" {
		t.Fatalf("expected status already_assigned, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events for explicit same-route no-op, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation for explicit same-route no-op")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events for explicit same-route no-op")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events for explicit same-route no-op")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after no-op")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth {
		t.Fatalf("expected reassign depth %d to remain unchanged, got %d", originalDepth, got)
	}
}

func TestHandleApplyReassign_ExplicitSameRouteSelectsAlternateManifestWhenSourceNotMutable(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	originalDepth := svc.orders[orderIdx].ReassignDepth

	sourceManifestIdx := svc.findManifestIndexLocked(originalManifestID)
	if sourceManifestIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected source manifest %q", originalManifestID)
	}
	svc.manifests[sourceManifestIdx].State = payloadManifestStateSealed
	svc.manifests[sourceManifestIdx].UpdatedAt = svc.now().Format(time.RFC3339Nano)

	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests, ManifestRow{
		ManifestID:    "mf_payload_alt_same_route",
		VehicleID:     "veh_payload_1",
		DriverID:      "drv_payload_alt",
		State:         payloadManifestStateDraft,
		TotalVolumeVU: 0,
		MaxVolumeVU:   140,
		StopCount:     0,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	svc.manifestOrders["mf_payload_alt_same_route"] = []ManifestOrder{}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_route_id":"route_veh_payload_1","reason":"sealed-source-same-route"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["status"].(string); got != "order_reassigned" {
		t.Fatalf("expected status order_reassigned, got %q", got)
	}
	if got, _ := body["to_manifest_id"].(string); got != "mf_payload_alt_same_route" {
		t.Fatalf("expected to_manifest_id mf_payload_alt_same_route, got %q", got)
	}
	if got, _ := body["to_route_id"].(string); got != originalRouteID {
		t.Fatalf("expected to_route_id %q, got %q", originalRouteID, got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	types := payloadOutboxEventTypes(repo.events)
	if len(types) != 1 || types[0] != events.EventManifestRebalanced {
		t.Fatalf("unexpected outbox events: %#v", types)
	}

	assertPayloadCacheDeletedKeys(t, cacheBackend.deletedKeys,
		payloadManifestKey(originalManifestID),
		payloadManifestListKey("supplier-test"),
		payloadOrderListKey("supplier-test"),
		payloadManifestKey("mf_payload_alt_same_route"),
	)

	assertPayloadWSMessageContainsType(t, supplierConn.messages, events.EventManifestRebalanced)
	assertPayloadWSMessageContainsType(t, payloadConn.messages, events.EventManifestRebalanced)

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after reassignment")
	}
	if got := svc.orders[orderIdx].ManifestID; got != "mf_payload_alt_same_route" {
		t.Fatalf("expected manifest_id mf_payload_alt_same_route, got %q", got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q, got %q", originalRouteID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth+1 {
		t.Fatalf("expected reassign depth %d, got %d", originalDepth+1, got)
	}
}

func TestHandleApplyReassign_TargetDriverManifestMismatch(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests, ManifestRow{
		ManifestID:    "mf_payload_2",
		VehicleID:     "veh_payload_2",
		DriverID:      "drv_payload_2",
		State:         payloadManifestStateDraft,
		TotalVolumeVU: 0,
		MaxVolumeVU:   140,
		StopCount:     0,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}

	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	originalDepth := svc.orders[orderIdx].ReassignDepth
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","to_driver_id":"drv_payload_other","reason":"driver-manifest-mismatch"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "target_driver_manifest_mismatch" {
		t.Fatalf("expected target_driver_manifest_mismatch, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on target_driver_manifest_mismatch")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on target_driver_manifest_mismatch")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on target_driver_manifest_mismatch")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events on target_driver_manifest_mismatch")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after conflict")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth {
		t.Fatalf("expected reassign depth %d to remain unchanged, got %d", originalDepth, got)
	}
}

func TestHandleApplyReassign_SourceManifestOrderMissingConflict(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests, ManifestRow{
		ManifestID:    "mf_payload_2",
		VehicleID:     "veh_payload_2",
		DriverID:      "drv_payload_2",
		State:         payloadManifestStateDraft,
		TotalVolumeVU: 0,
		MaxVolumeVU:   140,
		StopCount:     0,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}
	svc.manifestOrders["mf_payload_1"] = []ManifestOrder{{ManifestID: "mf_payload_1", OrderID: "ord_payload_2", State: "ASSIGNED", VolumeVU: 34, UpdatedAt: now}}

	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalRouteID := svc.orders[orderIdx].RouteID
	originalDepth := svc.orders[orderIdx].ReassignDepth
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"source-order-missing"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "source_manifest_order_missing" {
		t.Fatalf("expected source_manifest_order_missing, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on source_manifest_order_missing")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on source_manifest_order_missing")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on source_manifest_order_missing")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events on source_manifest_order_missing")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after conflict")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].RouteID; got != originalRouteID {
		t.Fatalf("expected route_id %q to remain unchanged, got %q", originalRouteID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth {
		t.Fatalf("expected reassign depth %d to remain unchanged, got %d", originalDepth, got)
	}
}

func TestHandleApplyReassign_SourceRouteManifestMismatchConflict(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests, ManifestRow{
		ManifestID:    "mf_payload_2",
		VehicleID:     "veh_payload_2",
		DriverID:      "drv_payload_2",
		State:         payloadManifestStateDraft,
		TotalVolumeVU: 0,
		MaxVolumeVU:   140,
		StopCount:     0,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}

	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalManifestID := svc.orders[orderIdx].ManifestID
	originalDepth := svc.orders[orderIdx].ReassignDepth
	svc.orders[orderIdx].RouteID = "route_veh_payload_drift"
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"source-route-mismatch"}`))
	rr := httptest.NewRecorder()

	svc.HandleApplyReassign(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "source_route_manifest_mismatch" {
		t.Fatalf("expected source_route_manifest_mismatch, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on source_route_manifest_mismatch")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on source_route_manifest_mismatch")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on source_route_manifest_mismatch")
	}
	if len(payloadConn.messages) != 0 {
		t.Fatalf("expected no payload websocket events on source_route_manifest_mismatch")
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after conflict")
	}
	if got := svc.orders[orderIdx].ManifestID; got != originalManifestID {
		t.Fatalf("expected manifest_id %q to remain unchanged, got %q", originalManifestID, got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth {
		t.Fatalf("expected reassign depth %d to remain unchanged, got %d", originalDepth, got)
	}
}

func TestHandleApplyReassign_TargetRouteMismatchAfterSuccess_NoExtraFanout(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests, ManifestRow{
		ManifestID:    "mf_payload_2",
		VehicleID:     "veh_payload_2",
		DriverID:      "drv_payload_2",
		State:         payloadManifestStateDraft,
		TotalVolumeVU: 0,
		MaxVolumeVU:   140,
		StopCount:     0,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}
	svc.mu.Unlock()

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"baseline-success"}`))
	firstRR := httptest.NewRecorder()
	svc.HandleApplyReassign(firstRR, firstReq)
	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}

	outboxTypesAfterFirst := payloadOutboxEventTypes(repo.events)
	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	supplierTypesAfterFirst := payloadWSMessageTypes(supplierConn.messages)
	payloadTypesAfterFirst := payloadWSMessageTypes(payloadConn.messages)

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","to_route_id":"route_veh_payload_1","reason":"force-route-conflict"}`))
	secondRR := httptest.NewRecorder()
	svc.HandleApplyReassign(secondRR, secondReq)

	if secondRR.Code != http.StatusConflict {
		t.Fatalf("expected second status 409, got %d body=%s", secondRR.Code, secondRR.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(secondRR.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if got, _ := body["error"].(string); got != "target_route_mismatch" {
		t.Fatalf("expected target_route_mismatch, got %q", got)
	}

	if repo.applyCalls != 2 {
		t.Fatalf("expected two repo apply calls, got %d", repo.applyCalls)
	}
	if got := payloadOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterFirst) {
		t.Fatalf("expected no extra outbox events on target_route_mismatch, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterFirst[i] {
				t.Fatalf("expected outbox contract sequence unchanged, got %#v want %#v", got, outboxTypesAfterFirst)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterFirst {
		t.Fatalf("expected no extra cache invalidation on target_route_mismatch")
	}
	if got := payloadWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterFirst) {
		t.Fatalf("expected no extra supplier ws events on target_route_mismatch, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterFirst[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged, got %#v want %#v", got, supplierTypesAfterFirst)
			}
		}
	}
	if got := payloadWSMessageTypes(payloadConn.messages); len(got) != len(payloadTypesAfterFirst) {
		t.Fatalf("expected no extra payload ws events on target_route_mismatch, got %#v", got)
	} else {
		for i := range got {
			if got[i] != payloadTypesAfterFirst[i] {
				t.Fatalf("expected payload ws contract sequence unchanged, got %#v want %#v", got, payloadTypesAfterFirst)
			}
		}
	}
}

func TestHandleApplyReassign_SourceManifestOrderMissingAfterSuccess_NoExtraFanout(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests,
		ManifestRow{
			ManifestID:    "mf_payload_2",
			VehicleID:     "veh_payload_2",
			DriverID:      "drv_payload_2",
			State:         payloadManifestStateDraft,
			TotalVolumeVU: 0,
			MaxVolumeVU:   140,
			StopCount:     0,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		ManifestRow{
			ManifestID:    "mf_payload_3",
			VehicleID:     "veh_payload_3",
			DriverID:      "drv_payload_3",
			State:         payloadManifestStateDraft,
			TotalVolumeVU: 0,
			MaxVolumeVU:   140,
			StopCount:     0,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	)
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}
	svc.manifestOrders["mf_payload_3"] = []ManifestOrder{}
	svc.mu.Unlock()

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"baseline-success"}`))
	firstRR := httptest.NewRecorder()
	svc.HandleApplyReassign(firstRR, firstReq)
	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}

	outboxTypesAfterFirst := payloadOutboxEventTypes(repo.events)
	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	supplierTypesAfterFirst := payloadWSMessageTypes(supplierConn.messages)
	payloadTypesAfterFirst := payloadWSMessageTypes(payloadConn.messages)

	svc.mu.Lock()
	delete(svc.manifestOrders, "mf_payload_2")
	svc.mu.Unlock()

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(`{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_3","reason":"source-order-missing-after-success"}`))
	secondRR := httptest.NewRecorder()
	svc.HandleApplyReassign(secondRR, secondReq)

	if secondRR.Code != http.StatusConflict {
		t.Fatalf("expected second status 409, got %d body=%s", secondRR.Code, secondRR.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(secondRR.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if got, _ := body["error"].(string); got != "source_manifest_order_missing" {
		t.Fatalf("expected source_manifest_order_missing, got %q", got)
	}

	if repo.applyCalls != 2 {
		t.Fatalf("expected two repo apply calls, got %d", repo.applyCalls)
	}
	if got := payloadOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterFirst) {
		t.Fatalf("expected no extra outbox events on source_manifest_order_missing, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterFirst[i] {
				t.Fatalf("expected outbox contract sequence unchanged, got %#v want %#v", got, outboxTypesAfterFirst)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterFirst {
		t.Fatalf("expected no extra cache invalidation on source_manifest_order_missing")
	}
	if got := payloadWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterFirst) {
		t.Fatalf("expected no extra supplier ws events on source_manifest_order_missing, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterFirst[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged, got %#v want %#v", got, supplierTypesAfterFirst)
			}
		}
	}
	if got := payloadWSMessageTypes(payloadConn.messages); len(got) != len(payloadTypesAfterFirst) {
		t.Fatalf("expected no extra payload ws events on source_manifest_order_missing, got %#v", got)
	} else {
		for i := range got {
			if got[i] != payloadTypesAfterFirst[i] {
				t.Fatalf("expected payload ws contract sequence unchanged, got %#v want %#v", got, payloadTypesAfterFirst)
			}
		}
	}
}

func TestHandleApplyReassign_ReplayAfterSuccessIdempotent(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	now := svc.now().Format(time.RFC3339Nano)
	svc.manifests = append(svc.manifests, ManifestRow{
		ManifestID:    "mf_payload_2",
		VehicleID:     "veh_payload_2",
		DriverID:      "drv_payload_2",
		State:         payloadManifestStateDraft,
		TotalVolumeVU: 0,
		MaxVolumeVU:   140,
		StopCount:     0,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	svc.manifestOrders["mf_payload_2"] = []ManifestOrder{}
	orderIdx := svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo order ord_payload_1")
	}
	originalDepth := svc.orders[orderIdx].ReassignDepth
	svc.mu.Unlock()

	body := `{"order_id":"ord_payload_1","to_manifest_id":"mf_payload_2","reason":"replay-idempotency"}`

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(body))
	firstRR := httptest.NewRecorder()
	svc.HandleApplyReassign(firstRR, firstReq)

	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}
	var firstBody map[string]any
	if err := json.Unmarshal(firstRR.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	if got, _ := firstBody["status"].(string); got != "order_reassigned" {
		t.Fatalf("expected first status order_reassigned, got %q", got)
	}

	if len(repo.events) != 1 {
		t.Fatalf("expected one outbox event after first reassignment, got %d", len(repo.events))
	}
	var firstEventPayload map[string]any
	if err := json.Unmarshal(repo.events[0].Payload, &firstEventPayload); err != nil {
		t.Fatalf("failed to decode first outbox payload: %v", err)
	}
	if got, _ := firstEventPayload["type"].(string); got != events.EventManifestRebalanced {
		t.Fatalf("expected outbox type %q, got %q", events.EventManifestRebalanced, got)
	}
	if got, _ := firstEventPayload["order_id"].(string); got != "ord_payload_1" {
		t.Fatalf("expected outbox order_id ord_payload_1, got %q", got)
	}
	if got, _ := firstEventPayload["from_manifest_id"].(string); got != "mf_payload_1" {
		t.Fatalf("expected outbox from_manifest_id mf_payload_1, got %q", got)
	}
	if got, _ := firstEventPayload["to_manifest_id"].(string); got != "mf_payload_2" {
		t.Fatalf("expected outbox to_manifest_id mf_payload_2, got %q", got)
	}
	if got, _ := firstEventPayload["to_route_id"].(string); got != "route_veh_payload_2" {
		t.Fatalf("expected outbox to_route_id route_veh_payload_2, got %q", got)
	}
	if got, _ := firstEventPayload["to_driver_id"].(string); got != "drv_payload_2" {
		t.Fatalf("expected outbox to_driver_id drv_payload_2, got %q", got)
	}
	outboxTypesAfterFirst := payloadOutboxEventTypes(repo.events)
	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	supplierMessagesAfterFirst := len(supplierConn.messages)
	payloadMessagesAfterFirst := len(payloadConn.messages)
	supplierTypesAfterFirst := payloadWSMessageTypes(supplierConn.messages)
	payloadTypesAfterFirst := payloadWSMessageTypes(payloadConn.messages)

	if len(supplierConn.messages) == 0 {
		t.Fatalf("expected supplier ws fanout after first reassignment")
	}
	if envelope := decodePayloadWSEnvelope(t, supplierConn.messages[len(supplierConn.messages)-1]); envelope.Type != events.EventManifestRebalanced {
		t.Fatalf("expected supplier ws type %q, got %q", events.EventManifestRebalanced, envelope.Type)
	} else {
		if got, _ := envelope.Data["order_id"].(string); got != "ord_payload_1" {
			t.Fatalf("expected supplier ws order_id ord_payload_1, got %q", got)
		}
		if got, _ := envelope.Data["from_manifest_id"].(string); got != "mf_payload_1" {
			t.Fatalf("expected supplier ws from_manifest_id mf_payload_1, got %q", got)
		}
		if got, _ := envelope.Data["to_manifest_id"].(string); got != "mf_payload_2" {
			t.Fatalf("expected supplier ws to_manifest_id mf_payload_2, got %q", got)
		}
	}
	if len(payloadConn.messages) == 0 {
		t.Fatalf("expected payload ws fanout after first reassignment")
	}
	if envelope := decodePayloadWSEnvelope(t, payloadConn.messages[len(payloadConn.messages)-1]); envelope.Type != events.EventManifestRebalanced {
		t.Fatalf("expected payload ws type %q, got %q", events.EventManifestRebalanced, envelope.Type)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/payloader/reassign-order", strings.NewReader(body))
	secondRR := httptest.NewRecorder()
	svc.HandleApplyReassign(secondRR, secondReq)

	if secondRR.Code != http.StatusOK {
		t.Fatalf("expected second status 200, got %d body=%s", secondRR.Code, secondRR.Body.String())
	}
	var secondBody map[string]any
	if err := json.Unmarshal(secondRR.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if got, _ := secondBody["status"].(string); got != "already_assigned" {
		t.Fatalf("expected second status already_assigned, got %q", got)
	}

	if repo.applyCalls != 2 {
		t.Fatalf("expected two repo apply calls, got %d", repo.applyCalls)
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected no extra outbox events on replay, got %d", len(repo.events))
	}
	if got := payloadOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterFirst) {
		t.Fatalf("expected outbox contract sequence unchanged on replay, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterFirst[i] {
				t.Fatalf("expected outbox contract sequence unchanged on replay, got %#v want %#v", got, outboxTypesAfterFirst)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterFirst {
		t.Fatalf("expected no extra cache invalidation on replay")
	}
	if len(supplierConn.messages) != supplierMessagesAfterFirst {
		t.Fatalf("expected no extra supplier websocket events on replay")
	}
	if len(payloadConn.messages) != payloadMessagesAfterFirst {
		t.Fatalf("expected no extra payload websocket events on replay")
	}
	if got := payloadWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterFirst) {
		t.Fatalf("expected supplier ws contract sequence unchanged on replay, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterFirst[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged on replay, got %#v want %#v", got, supplierTypesAfterFirst)
			}
		}
	}
	if got := payloadWSMessageTypes(payloadConn.messages); len(got) != len(payloadTypesAfterFirst) {
		t.Fatalf("expected payload ws contract sequence unchanged on replay, got %#v", got)
	} else {
		for i := range got {
			if got[i] != payloadTypesAfterFirst[i] {
				t.Fatalf("expected payload ws contract sequence unchanged on replay, got %#v want %#v", got, payloadTypesAfterFirst)
			}
		}
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	orderIdx = svc.findOrderIndexLocked("ord_payload_1")
	if orderIdx < 0 {
		t.Fatalf("expected demo order ord_payload_1 after replay")
	}
	if got := svc.orders[orderIdx].ManifestID; got != "mf_payload_2" {
		t.Fatalf("expected manifest_id mf_payload_2, got %q", got)
	}
	if got := svc.orders[orderIdx].RouteID; got != "route_veh_payload_2" {
		t.Fatalf("expected route_id route_veh_payload_2, got %q", got)
	}
	if got := svc.orders[orderIdx].ReassignDepth; got != originalDepth+1 {
		t.Fatalf("expected reassign depth %d after replay, got %d", originalDepth+1, got)
	}
}

func newPayloadTestService(repo *payloadRepoSpy, cacheBackend *payloadCacheBackendSpy) *Service {
	_ = os.Setenv("PAYLOAD_PORTAL_SEED", "true")
	now := time.Date(2026, 5, 17, 11, 0, 0, 0, time.UTC)
	cacheClient := cache.New(cacheBackend, nil)
	supplierHub := ws.NewHub("supplier", nil, nil)
	payloadHub := ws.NewHub("payload", nil, nil)

	return NewService(ServiceConfig{
		Repo:        repo,
		Cache:       cacheClient,
		SupplierHub: supplierHub,
		PayloadHub:  payloadHub,
		SupplierID:  "supplier-test",
		Currency:    "UZS",
		Now:         func() time.Time { return now },
	})
}

func payloadOutboxEventTypes(eventsList []outbox.Event) []string {
	types := make([]string, 0, len(eventsList))
	for _, event := range eventsList {
		var payload struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			types = append(types, "")
			continue
		}
		types = append(types, payload.Type)
	}
	return types
}

func payloadWSMessageTypes(messages [][]byte) []string {
	types := make([]string, 0, len(messages))
	for _, raw := range messages {
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			types = append(types, "")
			continue
		}
		types = append(types, envelope.Type)
	}
	return types
}

type payloadWSEnvelope struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func decodePayloadWSEnvelope(t *testing.T, raw []byte) payloadWSEnvelope {
	t.Helper()
	var envelope payloadWSEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("failed to decode payload ws envelope: %v", err)
	}
	return envelope
}

func assertPayloadContainsEventType(t *testing.T, got []string, want string) {
	t.Helper()
	for _, g := range got {
		if g == want {
			return
		}
	}
	t.Fatalf("expected outbox event %q in %#v", want, got)
}

func assertPayloadCacheDeletedKeys(t *testing.T, deleted [][]string, expected ...string) {
	t.Helper()
	flat := make(map[string]struct{})
	for _, call := range deleted {
		for _, key := range call {
			flat[key] = struct{}{}
		}
	}
	for _, key := range expected {
		if _, ok := flat[key]; !ok {
			t.Fatalf("expected cache invalidation key %q not found in %#v", key, deleted)
		}
	}
}

func assertPayloadWSMessageContainsType(t *testing.T, messages [][]byte, wantType string) {
	t.Helper()
	for _, raw := range messages {
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			continue
		}
		if envelope.Type == wantType {
			return
		}
	}
	t.Fatalf("expected websocket event type %q in messages: %q", wantType, messages)
}

type payloadRepoSpy struct {
	svc *Service
	applyCalls int
	events     []outbox.Event
}

func (r *payloadRepoSpy) RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if fn != nil {
		if err := fn(ctx, &dummyPayloadTx{svc: r.svc}); err != nil {
			return err
		}
	}
	if emit != nil {
		buf := &payloadTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	_ = ctx
	return nil
}

type payloadTxnBufferSpy struct {
	events []outbox.Event
}

func (b *payloadTxnBufferSpy) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

type payloadCacheBackendSpy struct {
	deletedKeys [][]string
}

func (b *payloadCacheBackendSpy) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (b *payloadCacheBackendSpy) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (b *payloadCacheBackendSpy) Delete(_ context.Context, keys ...string) error {
	copyKeys := append([]string(nil), keys...)
	b.deletedKeys = append(b.deletedKeys, copyKeys)
	return nil
}

func (b *payloadCacheBackendSpy) Publish(context.Context, string, []byte) error {
	return nil
}

func (b *payloadCacheBackendSpy) Subscribe(context.Context, string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	close(ch)
	return ch, func() {}, nil
}

type payloadWSConnSpy struct {
	id       string
	messages [][]byte
}

func (c *payloadWSConnSpy) ID() string {
	return c.id
}

func (c *payloadWSConnSpy) Identity() auth.Claims {
	return auth.Claims{}
}

func (c *payloadWSConnSpy) Send(_ context.Context, payload []byte) error {
	copyPayload := append([]byte(nil), payload...)
	c.messages = append(c.messages, copyPayload)
	return nil
}

func (r *payloadRepoSpy) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}

type dummyPayloadTx struct{ svc *Service }
func (d *dummyPayloadTx) ListManifests(ctx context.Context) ([]ManifestRow, error) { return append([]ManifestRow(nil), d.svc.manifests...), nil }
func (d *dummyPayloadTx) SaveManifest(ctx context.Context, m ManifestRow) error { return nil }
func (d *dummyPayloadTx) ListManifestOrders(ctx context.Context, mid string) ([]ManifestOrder, error) { return d.svc.manifestOrders[mid], nil }
func (d *dummyPayloadTx) SaveManifestOrder(ctx context.Context, mo ManifestOrder, seq int64) error { return nil }
func (d *dummyPayloadTx) ListExceptions(ctx context.Context) ([]ManifestException, error) { return append([]ManifestException(nil), d.svc.exceptions...), nil }
func (d *dummyPayloadTx) SaveException(ctx context.Context, e ManifestException) error { return nil }
