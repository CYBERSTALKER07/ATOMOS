package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestHandleTransferTransition_UsesRepositoryApply(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/transfers/tr_bay_1/transition", strings.NewReader(`{"target_state":"LOADING"}`))
	req = withFactoryRouteParam(req, "transferID", "tr_bay_1")
	rr := httptest.NewRecorder()

	svc.HandleTransferTransition(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}
	assertFactoryCacheDeletedKeys(t, cacheBackend.deletedKeys, factoryTransferListKey("supplier-test"))

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["state"].(string); got != "LOADING" {
		t.Fatalf("expected state LOADING, got %q", got)
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()
	idx := -1
	for i := range svc.transfers {
		if svc.transfers[i].TransferID == "tr_bay_1" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("expected transfer tr_bay_1 in service state")
	}
	if svc.transfers[idx].State != "LOADING" {
		t.Fatalf("expected persisted transfer state LOADING, got %q", svc.transfers[idx].State)
	}
}

func TestHandleTransfers_FiltersByState(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	repo.svc = svc

	req := httptest.NewRequest(http.MethodGet, "/v1/factory/transfers?states=APPROVED,LOADING&limit=10", nil)
	rr := httptest.NewRecorder()
	svc.HandleTransfers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Transfers []map[string]any `json:"transfers"`
		Total     int              `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total < 2 {
		t.Fatalf("expected at least 2 bay transfers, got total=%d body=%s", body.Total, rr.Body.String())
	}
	for _, row := range body.Transfers {
		state, _ := row["state"].(string)
		switch strings.ToUpper(state) {
		case "APPROVED", "LOADING":
		default:
			t.Fatalf("unexpected filtered state %q in %v", state, row)
		}
	}
}

func TestHandleDispatch_ExplicitLoadingTransferIDs(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/dispatch", strings.NewReader(`{"transfer_ids":["tr_bay_2"],"reason":"bay"}`))
	rr := httptest.NewRecorder()
	svc.HandleDispatch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["manifests_created"].(float64); got < 1 {
		t.Fatalf("expected manifests_created >= 1, got %v", body)
	}
}

func TestHandleManifestStartLoading_SeamParity(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/mf_factory_1/start-loading", strings.NewReader(`{"reason":"ops"}`))
	req = withFactoryRouteParam(req, "manifestID", "mf_factory_1")
	rr := httptest.NewRecorder()

	svc.HandleManifestStartLoading(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != 1 || got[0] != events.EventManifestLoadingStarted {
		t.Fatalf("unexpected outbox events: %#v", got)
	}

	assertFactoryCacheDeletedKeys(t, cacheBackend.deletedKeys,
		factoryManifestKey("mf_factory_1"),
		factoryManifestListKey("supplier-test"),
		factoryTransferListKey("supplier-test"),
	)

	assertFactoryWSMessageContainsType(t, supplierConn.messages, events.EventManifestLoadingStarted)
	assertFactoryWSMessageContainsType(t, factoryConn.messages, events.EventManifestLoadingStarted)
}

func TestHandleDispatch_SeamParity(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/dispatch", strings.NewReader(`{"reason":"wave"}`))
	rr := httptest.NewRecorder()

	svc.HandleDispatch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != 1 || got[0] != events.EventManifestDraftCreated {
		t.Fatalf("unexpected outbox events: %#v", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	manifestID, _ := body["manifest_id"].(string)
	if strings.TrimSpace(manifestID) == "" {
		t.Fatalf("expected manifest_id in response, got %v", body)
	}

	assertFactoryCacheDeletedKeys(t, cacheBackend.deletedKeys,
		factoryManifestKey(manifestID),
		factoryManifestListKey("supplier-test"),
		factoryTransferListKey("supplier-test"),
	)

	assertFactoryWSMessageContainsType(t, supplierConn.messages, events.EventManifestDraftCreated)
	assertFactoryWSMessageContainsType(t, factoryConn.messages, events.EventManifestDraftCreated)
}

func TestHandleManifestSeal_EmitsRouteMetadata(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	idx := svc.findManifestIndexLocked("mf_factory_1")
	if idx < 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo manifest")
	}
	svc.manifests[idx].State = manifestStateLoading
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/mf_factory_1/seal", strings.NewReader(`{"reason":"seal"}`))
	req = withFactoryRouteParam(req, "manifestID", "mf_factory_1")
	rr := httptest.NewRecorder()

	svc.HandleManifestSeal(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(repo.events))
	}
	var payload map[string]any
	if err := json.Unmarshal(repo.events[0].Payload, &payload); err != nil {
		t.Fatalf("failed to decode outbox payload: %v", err)
	}
	if got, _ := payload["type"].(string); got != events.EventManifestSealed {
		t.Fatalf("unexpected event type %q", got)
	}
	if got, _ := payload["route_id"].(string); got != "route_veh_factory_1" {
		t.Fatalf("expected route_id route_veh_factory_1, got %q", got)
	}
	if got, _ := payload["driver_id"].(string); got != "drv_factory_1" {
		t.Fatalf("expected driver_id drv_factory_1, got %q", got)
	}
	if got, _ := payload["vehicle_id"].(string); got != "veh_factory_1" {
		t.Fatalf("expected vehicle_id veh_factory_1, got %q", got)
	}
}

func TestManifestDetailSnapshotForDriver_SelectsCurrentDriverManifest(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc

	snapshot, ok := svc.ManifestDetailSnapshotForDriver("drv_factory_1", "", "2026-05-17")
	if !ok {
		t.Fatalf("expected snapshot for driver")
	}
	if snapshot.Manifest.ManifestID != "mf_factory_1" {
		t.Fatalf("expected manifest mf_factory_1, got %#v", snapshot.Manifest)
	}
	if snapshot.RouteID != "route_veh_factory_1" {
		t.Fatalf("expected route_veh_factory_1, got %#v", snapshot)
	}
	if snapshot.StopCount != 2 || snapshot.OrderCount != 2 {
		t.Fatalf("expected stop/order count 2, got %#v", snapshot)
	}
}

func TestManifestDetailSnapshotForDriver_RejectsDriverMismatch(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc

	if _, ok := svc.ManifestDetailSnapshotForDriver("drv_factory_2", "mf_factory_1", ""); ok {
		t.Fatalf("expected driver mismatch lookup to fail")
	}
}

func TestHandleManifestRebalance_IdempotentAlreadyAssigned(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_1","to_vehicle":"veh_factory_1","reason":"noop"}`))
	rr := httptest.NewRecorder()

	svc.HandleManifestRebalance(rr, req)

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
		t.Fatalf("expected zero outbox events on no-op rebalance, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on no-op rebalance")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on no-op rebalance")
	}
	if len(factoryConn.messages) != 0 {
		t.Fatalf("expected no factory websocket events on no-op rebalance")
	}
}

func TestHandleManifestRebalance_TransferNotMutable(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/cancel-transfer", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","reason":"ops"}`))
	cancelRR := httptest.NewRecorder()
	svc.HandleManifestCancelTransfer(cancelRR, cancelReq)
	if cancelRR.Code != http.StatusOK {
		t.Fatalf("expected cancel status 200, got %d body=%s", cancelRR.Code, cancelRR.Body.String())
	}

	eventsAfterCancel := len(repo.events)
	outboxTypesAfterCancel := factoryOutboxEventTypes(repo.events)
	cacheCallsAfterCancel := len(cacheBackend.deletedKeys)
	supplierMessagesAfterCancel := len(supplierConn.messages)
	factoryMessagesAfterCancel := len(factoryConn.messages)
	supplierTypesAfterCancel := factoryWSMessageTypes(supplierConn.messages)
	factoryTypesAfterCancel := factoryWSMessageTypes(factoryConn.messages)

	rebalanceReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","reason":"retry"}`))
	rebalanceRR := httptest.NewRecorder()
	svc.HandleManifestRebalance(rebalanceRR, rebalanceReq)

	if rebalanceRR.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rebalanceRR.Code, rebalanceRR.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rebalanceRR.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "transfer_not_mutable" {
		t.Fatalf("expected transfer_not_mutable, got %q", got)
	}

	if repo.applyCalls != 2 {
		t.Fatalf("expected two repo apply calls (cancel + rebalance), got %d", repo.applyCalls)
	}
	if len(repo.events) != eventsAfterCancel {
		t.Fatalf("expected no extra outbox events on transfer_not_mutable conflict")
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterCancel) {
		t.Fatalf("expected outbox contract sequence unchanged, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterCancel[i] {
				t.Fatalf("expected outbox contract sequence unchanged, got %#v want %#v", got, outboxTypesAfterCancel)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterCancel {
		t.Fatalf("expected no extra cache invalidation on transfer_not_mutable conflict")
	}
	if len(supplierConn.messages) != supplierMessagesAfterCancel {
		t.Fatalf("expected no extra supplier websocket events on transfer_not_mutable conflict")
	}
	if len(factoryConn.messages) != factoryMessagesAfterCancel {
		t.Fatalf("expected no extra factory websocket events on transfer_not_mutable conflict")
	}
	if got := factoryWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterCancel) {
		t.Fatalf("expected supplier ws contract sequence unchanged, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterCancel[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged, got %#v want %#v", got, supplierTypesAfterCancel)
			}
		}
	}
	if got := factoryWSMessageTypes(factoryConn.messages); len(got) != len(factoryTypesAfterCancel) {
		t.Fatalf("expected factory ws contract sequence unchanged, got %#v", got)
	} else {
		for i := range got {
			if got[i] != factoryTypesAfterCancel[i] {
				t.Fatalf("expected factory ws contract sequence unchanged, got %#v want %#v", got, factoryTypesAfterCancel)
			}
		}
	}
}

func TestHandleManifestRebalance_TransferStateNotMutable(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	transfers := svc.manifestTransfers["mf_factory_1"]
	if len(transfers) == 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo manifest transfers")
	}
	transfers[0].State = "DISPATCHED"
	svc.manifestTransfers["mf_factory_1"] = transfers
	for i := range svc.transfers {
		if svc.transfers[i].TransferID == transfers[0].TransferID {
			svc.transfers[i].State = "DISPATCHED"
		}
	}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","reason":"state-check"}`))
	rr := httptest.NewRecorder()

	svc.HandleManifestRebalance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "transfer_not_mutable" {
		t.Fatalf("expected transfer_not_mutable, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on transfer state conflict")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on transfer state conflict")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on transfer state conflict")
	}
	if len(factoryConn.messages) != 0 {
		t.Fatalf("expected no factory websocket events on transfer state conflict")
	}
}

func TestHandleManifestRebalance_GlobalTransferMissingConflict(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	filtered := make([]TransferRow, 0, len(svc.transfers))
	for i := range svc.transfers {
		if svc.transfers[i].TransferID != "tr_factory_1" {
			filtered = append(filtered, svc.transfers[i])
		}
	}
	svc.transfers = filtered
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","reason":"global-drift"}`))
	rr := httptest.NewRecorder()

	svc.HandleManifestRebalance(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "transfer_not_found" {
		t.Fatalf("expected transfer_not_found, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on transfer_not_found conflict")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on transfer_not_found conflict")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on transfer_not_found conflict")
	}
	if len(factoryConn.messages) != 0 {
		t.Fatalf("expected no factory websocket events on transfer_not_found conflict")
	}
}

func TestHandleManifestRebalance_TransferRouteMismatch(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	transfers := svc.manifestTransfers["mf_factory_1"]
	if len(transfers) == 0 {
		svc.mu.Unlock()
		t.Fatalf("expected demo manifest transfers")
	}
	transfers[0].VehicleID = "veh_factory_drift"
	svc.manifestTransfers["mf_factory_1"] = transfers
	for i := range svc.transfers {
		if svc.transfers[i].TransferID == transfers[0].TransferID {
			svc.transfers[i].VehicleID = "veh_factory_drift"
		}
	}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","reason":"route-consistency"}`))
	rr := httptest.NewRecorder()

	svc.HandleManifestRebalance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "transfer_route_mismatch" {
		t.Fatalf("expected transfer_route_mismatch, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on transfer_route_mismatch conflict")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on transfer_route_mismatch conflict")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on transfer_route_mismatch conflict")
	}
	if len(factoryConn.messages) != 0 {
		t.Fatalf("expected no factory websocket events on transfer_route_mismatch conflict")
	}
}

func TestHandleManifestRebalance_TransferLedgerMismatch(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	for i := range svc.transfers {
		if svc.transfers[i].TransferID == "tr_factory_1" {
			svc.transfers[i].State = "CANCELLED"
			svc.transfers[i].UpdatedAt = svc.now().Format(time.RFC3339Nano)
			break
		}
	}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(`{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","reason":"ledger-parity"}`))
	rr := httptest.NewRecorder()

	svc.HandleManifestRebalance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "transfer_not_mutable" {
		t.Fatalf("expected transfer_not_mutable, got %q", got)
	}

	if repo.applyCalls != 1 {
		t.Fatalf("expected one repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox events on transfer_ledger_mismatch conflict")
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected no cache invalidation on transfer_ledger_mismatch conflict")
	}
	if len(supplierConn.messages) != 0 {
		t.Fatalf("expected no supplier websocket events on transfer_ledger_mismatch conflict")
	}
	if len(factoryConn.messages) != 0 {
		t.Fatalf("expected no factory websocket events on transfer_ledger_mismatch conflict")
	}
}

func TestHandleManifestRebalance_ReplayAfterSuccessIdempotent(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	body := `{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","to_vehicle":"veh_factory_1","reason":"replay-idempotency"}`

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(body))
	firstRR := httptest.NewRecorder()
	svc.HandleManifestRebalance(firstRR, firstReq)

	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}
	var firstBody map[string]any
	if err := json.Unmarshal(firstRR.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	if got, _ := firstBody["status"].(string); got != "manifest_rebalanced" {
		t.Fatalf("expected first status manifest_rebalanced, got %q", got)
	}

	if len(repo.events) != 1 {
		t.Fatalf("expected one outbox event after first rebalance, got %d", len(repo.events))
	}
	var firstEventPayload map[string]any
	if err := json.Unmarshal(repo.events[0].Payload, &firstEventPayload); err != nil {
		t.Fatalf("failed to decode first outbox payload: %v", err)
	}
	if got, _ := firstEventPayload["type"].(string); got != events.EventManifestRebalanced {
		t.Fatalf("expected outbox type %q, got %q", events.EventManifestRebalanced, got)
	}
	if got, _ := firstEventPayload["transfer_id"].(string); got != "tr_factory_1" {
		t.Fatalf("expected outbox transfer_id tr_factory_1, got %q", got)
	}
	if got, _ := firstEventPayload["to_driver_id"].(string); got != "drv_factory_2" {
		t.Fatalf("expected outbox to_driver_id drv_factory_2, got %q", got)
	}
	if got, _ := firstEventPayload["to_vehicle_id"].(string); got != "veh_factory_1" {
		t.Fatalf("expected outbox to_vehicle_id veh_factory_1, got %q", got)
	}
	outboxTypesAfterFirst := factoryOutboxEventTypes(repo.events)
	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	supplierMessagesAfterFirst := len(supplierConn.messages)
	factoryMessagesAfterFirst := len(factoryConn.messages)
	supplierTypesAfterFirst := factoryWSMessageTypes(supplierConn.messages)
	factoryTypesAfterFirst := factoryWSMessageTypes(factoryConn.messages)
	if len(supplierTypesAfterFirst) == 0 || supplierTypesAfterFirst[len(supplierTypesAfterFirst)-1] != events.EventManifestRebalanced {
		t.Fatalf("expected supplier ws event %q after first rebalance, got %#v", events.EventManifestRebalanced, supplierTypesAfterFirst)
	}
	if len(factoryTypesAfterFirst) == 0 || factoryTypesAfterFirst[len(factoryTypesAfterFirst)-1] != events.EventManifestRebalanced {
		t.Fatalf("expected factory ws event %q after first rebalance, got %#v", events.EventManifestRebalanced, factoryTypesAfterFirst)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(body))
	secondRR := httptest.NewRecorder()
	svc.HandleManifestRebalance(secondRR, secondReq)

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
	if got := factoryOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterFirst) {
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
	if len(factoryConn.messages) != factoryMessagesAfterFirst {
		t.Fatalf("expected no extra factory websocket events on replay")
	}
	if got := factoryWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterFirst) {
		t.Fatalf("expected supplier ws contract sequence unchanged on replay, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterFirst[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged on replay, got %#v want %#v", got, supplierTypesAfterFirst)
			}
		}
	}
	if got := factoryWSMessageTypes(factoryConn.messages); len(got) != len(factoryTypesAfterFirst) {
		t.Fatalf("expected factory ws contract sequence unchanged on replay, got %#v", got)
	} else {
		for i := range got {
			if got[i] != factoryTypesAfterFirst[i] {
				t.Fatalf("expected factory ws contract sequence unchanged on replay, got %#v want %#v", got, factoryTypesAfterFirst)
			}
		}
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	transfers := svc.manifestTransfers["mf_factory_1"]
	tIdx := svc.findTransferIndexLocked(transfers, "tr_factory_1")
	if tIdx < 0 {
		t.Fatalf("expected transfer tr_factory_1")
	}
	if got := transfers[tIdx].DriverID; got != "drv_factory_2" {
		t.Fatalf("expected transfer driver drv_factory_2, got %q", got)
	}
	if got := transfers[tIdx].VehicleID; got != "veh_factory_1" {
		t.Fatalf("expected transfer vehicle veh_factory_1, got %q", got)
	}
	if got := transfers[tIdx].ReassignDepth; got != 1 {
		t.Fatalf("expected transfer reassign depth 1 after replay, got %d", got)
	}
}

func TestHandleManifestRebalance_TransferRouteMismatchAfterSuccess_NoExtraFanout(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	firstBody := `{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","to_vehicle_id":"veh_factory_1","reason":"baseline-success"}`
	firstReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(firstBody))
	firstRR := httptest.NewRecorder()
	svc.HandleManifestRebalance(firstRR, firstReq)

	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}

	eventsAfterFirst := len(repo.events)
	outboxTypesAfterFirst := factoryOutboxEventTypes(repo.events)
	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	supplierMessagesAfterFirst := len(supplierConn.messages)
	factoryMessagesAfterFirst := len(factoryConn.messages)
	supplierTypesAfterFirst := factoryWSMessageTypes(supplierConn.messages)
	factoryTypesAfterFirst := factoryWSMessageTypes(factoryConn.messages)

	if eventsAfterFirst == 0 {
		t.Fatalf("expected at least one outbox event after first rebalance")
	}
	if len(supplierTypesAfterFirst) == 0 || supplierTypesAfterFirst[len(supplierTypesAfterFirst)-1] != events.EventManifestRebalanced {
		t.Fatalf("expected supplier ws event %q after first rebalance, got %#v", events.EventManifestRebalanced, supplierTypesAfterFirst)
	}
	if len(factoryTypesAfterFirst) == 0 || factoryTypesAfterFirst[len(factoryTypesAfterFirst)-1] != events.EventManifestRebalanced {
		t.Fatalf("expected factory ws event %q after first rebalance, got %#v", events.EventManifestRebalanced, factoryTypesAfterFirst)
	}

	conflictBody := `{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","to_driver_id":"drv_factory_2","to_vehicle_id":"veh_factory_2","reason":"route-mismatch-after-success"}`
	conflictReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/rebalance", strings.NewReader(conflictBody))
	conflictRR := httptest.NewRecorder()
	svc.HandleManifestRebalance(conflictRR, conflictReq)

	if conflictRR.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", conflictRR.Code, conflictRR.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(conflictRR.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode conflict response: %v", err)
	}
	if got, _ := body["error"].(string); got != "transfer_route_mismatch" {
		t.Fatalf("expected transfer_route_mismatch, got %q", got)
	}

	if repo.applyCalls != 2 {
		t.Fatalf("expected two repo apply calls, got %d", repo.applyCalls)
	}
	if len(repo.events) != eventsAfterFirst {
		t.Fatalf("expected no extra outbox events on route-mismatch conflict")
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterFirst) {
		t.Fatalf("expected outbox contract sequence unchanged, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterFirst[i] {
				t.Fatalf("expected outbox contract sequence unchanged, got %#v want %#v", got, outboxTypesAfterFirst)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterFirst {
		t.Fatalf("expected no extra cache invalidation on route-mismatch conflict")
	}
	if len(supplierConn.messages) != supplierMessagesAfterFirst {
		t.Fatalf("expected no extra supplier websocket events on route-mismatch conflict")
	}
	if len(factoryConn.messages) != factoryMessagesAfterFirst {
		t.Fatalf("expected no extra factory websocket events on route-mismatch conflict")
	}
	if got := factoryWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterFirst) {
		t.Fatalf("expected supplier ws contract sequence unchanged, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterFirst[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged, got %#v want %#v", got, supplierTypesAfterFirst)
			}
		}
	}
	if got := factoryWSMessageTypes(factoryConn.messages); len(got) != len(factoryTypesAfterFirst) {
		t.Fatalf("expected factory ws contract sequence unchanged, got %#v", got)
	} else {
		for i := range got {
			if got[i] != factoryTypesAfterFirst[i] {
				t.Fatalf("expected factory ws contract sequence unchanged, got %#v want %#v", got, factoryTypesAfterFirst)
			}
		}
	}
}

func TestHandleManifestCancelTransfer_IdempotentAlreadyCancelled(t *testing.T) {
	repo := &factoryRepoSpy{}
	// svc will be assigned later
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	body := `{"manifest_id":"mf_factory_1","transfer_id":"tr_factory_1","reason":"ops"}`
	firstReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/cancel-transfer", strings.NewReader(body))
	firstRR := httptest.NewRecorder()

	svc.HandleManifestCancelTransfer(firstRR, firstReq)

	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first status 200, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected one outbox event after first cancel, got %d", len(repo.events))
	}
	outboxTypesAfterFirst := factoryOutboxEventTypes(repo.events)

	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	supplierMessagesAfterFirst := len(supplierConn.messages)
	factoryMessagesAfterFirst := len(factoryConn.messages)
	supplierTypesAfterFirst := factoryWSMessageTypes(supplierConn.messages)
	factoryTypesAfterFirst := factoryWSMessageTypes(factoryConn.messages)

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/factory/manifests/cancel-transfer", strings.NewReader(body))
	secondRR := httptest.NewRecorder()

	svc.HandleManifestCancelTransfer(secondRR, secondReq)

	if secondRR.Code != http.StatusOK {
		t.Fatalf("expected second status 200, got %d body=%s", secondRR.Code, secondRR.Body.String())
	}

	var secondBody map[string]any
	if err := json.Unmarshal(secondRR.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if got, _ := secondBody["status"].(string); got != "already_cancelled" {
		t.Fatalf("expected status already_cancelled, got %q", got)
	}

	if len(repo.events) != 1 {
		t.Fatalf("expected no extra outbox events on second cancel, got %d", len(repo.events))
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterFirst) {
		t.Fatalf("expected outbox contract sequence unchanged on second cancel, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterFirst[i] {
				t.Fatalf("expected outbox contract sequence unchanged on second cancel, got %#v want %#v", got, outboxTypesAfterFirst)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterFirst {
		t.Fatalf("expected no extra cache invalidation on second cancel")
	}
	if len(supplierConn.messages) != supplierMessagesAfterFirst {
		t.Fatalf("expected no extra supplier ws events on second cancel")
	}
	if len(factoryConn.messages) != factoryMessagesAfterFirst {
		t.Fatalf("expected no extra factory ws events on second cancel")
	}
	if got := factoryWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterFirst) {
		t.Fatalf("expected supplier ws contract sequence unchanged on second cancel, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterFirst[i] {
				t.Fatalf("expected supplier ws contract sequence unchanged on second cancel, got %#v want %#v", got, supplierTypesAfterFirst)
			}
		}
	}
	if got := factoryWSMessageTypes(factoryConn.messages); len(got) != len(factoryTypesAfterFirst) {
		t.Fatalf("expected factory ws contract sequence unchanged on second cancel, got %#v", got)
	} else {
		for i := range got {
			if got[i] != factoryTypesAfterFirst[i] {
				t.Fatalf("expected factory ws contract sequence unchanged on second cancel, got %#v want %#v", got, factoryTypesAfterFirst)
			}
		}
	}
}

func newFactoryTestService(repo *factoryRepoSpy, cacheBackend *factoryCacheBackendSpy) *Service {
	now := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	cacheClient := cache.New(cacheBackend, nil)
	supplierHub := ws.NewHub("supplier", nil, nil)
	factoryHub := ws.NewHub("factory", nil, nil)

	return NewService(ServiceConfig{
		Repo:        repo,
		Cache:       cacheClient,
		SupplierHub: supplierHub,
		FactoryHub:  factoryHub,
		SupplierID:  "supplier-test",
		Currency:    "UZS",
		Now:         func() time.Time { return now },
	})
}

func withFactoryRouteParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func factoryOutboxEventTypes(eventsList []outbox.Event) []string {
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

func factoryWSMessageTypes(messages [][]byte) []string {
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

func assertFactoryCacheDeletedKeys(t *testing.T, deleted [][]string, expected ...string) {
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

func assertFactoryWSMessageContainsType(t *testing.T, messages [][]byte, wantType string) {
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

type factoryRepoSpy struct {
	svc *Service
	applyCalls int
	events     []outbox.Event
}

func (r *factoryRepoSpy) RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if fn != nil {
		if err := fn(ctx, &dummyFactoryTx{svc: r.svc}); err != nil {
			return err
		}
	}
	if emit != nil {
		buf := &factoryTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	_ = ctx
	return nil
}

type factoryTxnBufferSpy struct {
	events []outbox.Event
}

func (b *factoryTxnBufferSpy) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (r *factoryRepoSpy) UpdateSupplyRequestState(ctx context.Context, requestID, state string, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if emit != nil {
		buf := &factoryTxnBufferSpy{}
		_ = emit(buf)
		r.events = append(r.events, buf.events...)
	}
	return nil
}

type factoryCacheBackendSpy struct {
	deletedKeys [][]string
}

func (b *factoryCacheBackendSpy) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (b *factoryCacheBackendSpy) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (b *factoryCacheBackendSpy) Delete(_ context.Context, keys ...string) error {
	copyKeys := append([]string(nil), keys...)
	b.deletedKeys = append(b.deletedKeys, copyKeys)
	return nil
}

func (b *factoryCacheBackendSpy) Publish(context.Context, string, []byte) error {
	return nil
}

func (b *factoryCacheBackendSpy) Subscribe(context.Context, string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	close(ch)
	return ch, func() {}, nil
}

type factoryWSConnSpy struct {
	id       string
	messages [][]byte
}

func (c *factoryWSConnSpy) ID() string {
	return c.id
}

func (c *factoryWSConnSpy) Identity() auth.Claims {
	return auth.Claims{}
}

func (c *factoryWSConnSpy) Send(_ context.Context, payload []byte) error {
	copyPayload := append([]byte(nil), payload...)
	c.messages = append(c.messages, copyPayload)
	return nil
}

func (r *factoryRepoSpy) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}

type dummyFactoryTx struct{ svc *Service }
func (d *dummyFactoryTx) ListManifests(ctx context.Context) ([]ManifestRow, error) { return append([]ManifestRow(nil), d.svc.manifests...), nil }
func (d *dummyFactoryTx) SaveManifest(ctx context.Context, m ManifestRow) error { return nil }
func (d *dummyFactoryTx) ListTransfers(ctx context.Context) ([]TransferRow, error) { return append([]TransferRow(nil), d.svc.transfers...), nil }
func (d *dummyFactoryTx) SaveTransfer(ctx context.Context, t TransferRow) error { return nil }
