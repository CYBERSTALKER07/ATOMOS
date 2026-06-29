package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestHandleAvailability_PatchEmitsOutboxCacheWSOnce(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	driverConn := &driverWSConnSpy{id: "driver-conn"}
	supplierConn := &driverWSConnSpy{id: "supplier-conn"}

	svc := newDriverTestService(repo, cacheBackend)
	svc.driverHub.Subscribe("driver:drv-1", driverConn)
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)

	req := httptest.NewRequest(http.MethodPatch, "/v1/driver/availability", strings.NewReader(`{"on_shift":true}`))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1", HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-7"})
	rr := httptest.NewRecorder()

	svc.HandleAvailability(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo.Apply call, got %d", repo.applyCalls)
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(repo.events))
	}

	payload := decodeDriverOutboxPayload(t, repo.events[0])
	if got, _ := payload["type"].(string); got != events.EventDriverAvailabilityChanged {
		t.Fatalf("expected event type %q, got %q", events.EventDriverAvailabilityChanged, got)
	}
	if got, _ := payload["version"].(float64); got != 1 {
		t.Fatalf("expected version=1, got %v", payload["version"])
	}
	if got, _ := payload["driver_id"].(string); got != "drv-1" {
		t.Fatalf("expected driver_id drv-1, got %q", got)
	}
	if got, _ := payload["available"].(bool); !got {
		t.Fatalf("expected available true, got %v", payload["available"])
	}
	if got, _ := payload["on_shift"].(bool); !got {
		t.Fatalf("expected on_shift true, got %v", payload["on_shift"])
	}
	if got, _ := payload["home_node_type"].(string); got != "WAREHOUSE" {
		t.Fatalf("expected home_node_type WAREHOUSE, got %q", got)
	}
	if got, _ := payload["home_node_id"].(string); got != "wh-7" {
		t.Fatalf("expected home_node_id wh-7, got %q", got)
	}

	assertDriverCacheDeletedKeys(t, cacheBackend.deletedKeys, driverAvailabilityKey("drv-1"))
	assertDriverWSMessageContainsType(t, driverConn.messages, events.EventDriverAvailabilityChanged)
	assertDriverWSMessageContainsType(t, supplierConn.messages, events.EventDriverAvailabilityChanged)
}

func TestHandleManifest_NotFound(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifest(repo, cacheBackend, func(driverID, manifestID, date string) (factory.ManifestDetailSnapshot, bool) {
		if driverID != "drv-1" || manifestID != "mf_missing" || date != "" {
			t.Fatalf("unexpected lookup args driver=%q manifest=%q date=%q", driverID, manifestID, date)
		}
		return factory.ManifestDetailSnapshot{}, false
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest?manifest_id=mf_missing", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifest(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "manifest_not_found" {
		t.Fatalf("expected manifest_not_found, got %#v", body)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleManifest_IOSRouteManifest(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifest(repo, cacheBackend, func(driverID, manifestID, date string) (factory.ManifestDetailSnapshot, bool) {
		t.Fatalf("iOS manifest path must not call detail lookup, got driver=%q manifest=%q date=%q", driverID, manifestID, date)
		return factory.ManifestDetailSnapshot{}, false
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest?date=2026-05-22", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["driver_id"].(string); got != "drv-1" {
		t.Fatalf("expected driver_id drv-1, got %#v", body)
	}
	if got, _ := body["date"].(string); got != "2026-05-22" {
		t.Fatalf("expected date 2026-05-22, got %#v", body)
	}
	hashes, ok := body["hashes"].(map[string]any)
	if !ok || len(hashes) == 0 {
		t.Fatalf("expected non-empty hashes map, got %#v", body["hashes"])
	}
}

func TestHandleManifest_DetailResponse(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifest(repo, cacheBackend, func(driverID, manifestID, date string) (factory.ManifestDetailSnapshot, bool) {
		if driverID != "drv-1" || manifestID != "mf_factory_1" || date != "2026-05-22" {
			t.Fatalf("unexpected lookup args driver=%q manifest=%q date=%q", driverID, manifestID, date)
		}
		return factory.ManifestDetailSnapshot{
			Manifest: factory.ManifestRow{
				ManifestID:    "mf_factory_1",
				State:         "DISPATCHED",
				TransferCnt:   2,
				TotalVolumeVU: 79,
				DriverID:      "drv-1",
				VehicleID:     "veh_factory_1",
				CreatedAt:     "2026-05-22T08:00:00Z",
				UpdatedAt:     "2026-05-22T09:00:00Z",
			},
			Transfers: []factory.TransferRow{
				{TransferID: "tr_factory_1", OrderID: "ord_factory_1", ManifestID: "mf_factory_1", State: "ASSIGNED", TotalVU: 42},
				{TransferID: "tr_factory_2", OrderID: "ord_factory_2", ManifestID: "mf_factory_1", State: "ASSIGNED", TotalVU: 37},
			},
			Transitions: []factory.ManifestTransition{{Action: "DISPATCH", FromState: "SEALED", ToState: "DISPATCHED", At: "2026-05-22T09:00:00Z"}},
			RouteID:     "route_veh_factory_1",
			StopCount:   2,
			OrderCount:  2,
		}, true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest?manifest_id=mf_factory_1&date=2026-05-22", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["manifest_id"].(string); got != "mf_factory_1" {
		t.Fatalf("expected manifest_id mf_factory_1, got %#v", body)
	}
	if got, _ := body["route_id"].(string); got != "route_veh_factory_1" {
		t.Fatalf("expected route_id route_veh_factory_1, got %#v", body)
	}
	if got, _ := body["stop_count"].(float64); int(got) != 2 {
		t.Fatalf("expected stop_count 2, got %#v", body)
	}
	if got, _ := body["order_count"].(float64); int(got) != 2 {
		t.Fatalf("expected order_count 2, got %#v", body)
	}
	hashes, ok := body["hashes"].(map[string]any)
	if !ok || len(hashes) == 0 {
		t.Fatalf("expected non-empty offline hashes map, got %#v", body["hashes"])
	}
	if available, _ := body["legacy_hashes_available"].(bool); !available {
		t.Fatalf("expected legacy_hashes_available=true when hashes populated, got %#v", body)
	}
	manifest, ok := body["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("expected manifest object, got %#v", body)
	}
	if got, _ := manifest["state"].(string); got != "DISPATCHED" {
		t.Fatalf("expected manifest state DISPATCHED, got %#v", manifest)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleAvailability_PatchIdempotentNoOpNoSideEffects(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	driverConn := &driverWSConnSpy{id: "driver-conn"}
	supplierConn := &driverWSConnSpy{id: "supplier-conn"}

	svc := newDriverTestService(repo, cacheBackend)
	svc.driverHub.Subscribe("driver:drv-1", driverConn)
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)

	// First PATCH lands the durable transition.
	first := httptest.NewRequest(http.MethodPatch, "/v1/driver/availability", strings.NewReader(`{"on_shift":true}`))
	first = withDriverClaims(first, auth.Claims{Subject: "drv-1"})
	svc.HandleAvailability(httptest.NewRecorder(), first)

	applyCallsAfterFirst := repo.applyCalls
	eventsAfterFirst := len(repo.events)
	cacheCallsAfterFirst := len(cacheBackend.deletedKeys)
	driverMsgsAfterFirst := len(driverConn.messages)
	supplierMsgsAfterFirst := len(supplierConn.messages)

	// Second PATCH with same target state -> idempotent no-op branch.
	second := httptest.NewRequest(http.MethodPatch, "/v1/driver/availability", strings.NewReader(`{"on_shift":true}`))
	second = withDriverClaims(second, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()
	svc.HandleAvailability(rr, second)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on idempotent no-op, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if noChange, _ := body["no_change"].(bool); !noChange {
		t.Fatalf("expected no_change=true on idempotent PATCH, got %#v", body)
	}

	if repo.applyCalls != applyCallsAfterFirst {
		t.Fatalf("expected no extra repo.Apply call on no-op, got %d (was %d)", repo.applyCalls, applyCallsAfterFirst)
	}
	if len(repo.events) != eventsAfterFirst {
		t.Fatalf("expected no extra outbox events on no-op, got %d (was %d)", len(repo.events), eventsAfterFirst)
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterFirst {
		t.Fatalf("expected no extra cache invalidations on no-op, got %d (was %d)", len(cacheBackend.deletedKeys), cacheCallsAfterFirst)
	}
	if len(driverConn.messages) != driverMsgsAfterFirst {
		t.Fatalf("expected no extra driver ws messages on no-op, got %d (was %d)", len(driverConn.messages), driverMsgsAfterFirst)
	}
	if len(supplierConn.messages) != supplierMsgsAfterFirst {
		t.Fatalf("expected no extra supplier ws messages on no-op, got %d (was %d)", len(supplierConn.messages), supplierMsgsAfterFirst)
	}
}

func TestHandleAvailability_PatchIdempotencyReplay(t *testing.T) {
	body := []byte(`{"on_shift":true}`)
	cached := map[string]any{"driver_id": "drv-1", "on_shift": true, "no_change": true}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "driver-availability:drv-1:test"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	svc.idem = store

	req := httptest.NewRequest(http.MethodPatch, "/v1/driver/availability", strings.NewReader(string(body)))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleAvailability(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("replay body = %s want %s", rr.Body.String(), string(cachedBytes))
	}
}

func TestHandleAvailability_PatchInvalidJSONRejectedNoSideEffects(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodPatch, "/v1/driver/availability", strings.NewReader(`not-json`))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleAvailability(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid json, got %d", rr.Code)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected 0 repo.Apply calls on bad request, got %d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected 0 outbox events on bad request, got %d", len(repo.events))
	}
	if len(cacheBackend.deletedKeys) != 0 {
		t.Fatalf("expected 0 cache invalidations on bad request, got %d", len(cacheBackend.deletedKeys))
	}
}

func TestHandleManifestGate_MissingManifestIDRejected(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest-gate", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifestGate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleManifestGate_NotFound(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifestGate(repo, cacheBackend, func(manifestID string) (ManifestGateResult, bool) {
		return ManifestGateResult{}, false
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest-gate?manifest_id=mf_missing", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifestGate(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "manifest_not_found" {
		t.Fatalf("expected manifest_not_found, got %#v", body)
	}
}

func TestHandleManifestGate_ClearedWhenSealed(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifestGate(repo, cacheBackend, func(manifestID string) (ManifestGateResult, bool) {
		if manifestID != "mf_factory_1" {
			return ManifestGateResult{}, false
		}
		return ManifestGateResult{
			ManifestID: manifestID,
			State:      "SEALED",
			StopCount:  2,
			VolumeVU:   79,
		}, true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest-gate?manifest_id=mf_factory_1", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifestGate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if cleared, _ := body["cleared"].(bool); !cleared {
		t.Fatalf("expected cleared=true, got %#v", body)
	}
	if allowed, _ := body["allowed"].(bool); !allowed {
		t.Fatalf("expected allowed=true, got %#v", body)
	}
	if got, _ := body["state"].(string); got != "SEALED" {
		t.Fatalf("expected state SEALED, got %#v", body)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleManifestGate_BlockedWhenDraft(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifestGate(repo, cacheBackend, func(manifestID string) (ManifestGateResult, bool) {
		if manifestID != "mf_draft_1" {
			return ManifestGateResult{}, false
		}
		return ManifestGateResult{
			ManifestID: manifestID,
			State:      "DRAFT",
			StopCount:  1,
			VolumeVU:   12,
		}, true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest-gate?manifest_id=mf_draft_1", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifestGate(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["error"].(string); got != "AWAITING_PAYLOAD_SEAL" {
		t.Fatalf("expected AWAITING_PAYLOAD_SEAL, got %#v", body)
	}
}

func TestHandleManifestGate_BlockedWhenLoading(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithManifestGate(repo, cacheBackend, func(manifestID string) (ManifestGateResult, bool) {
		if manifestID != "mf_factory_1" {
			return ManifestGateResult{}, false
		}
		return ManifestGateResult{
			ManifestID: manifestID,
			State:      "LOADING",
			StopCount:  2,
			VolumeVU:   79,
		}, true
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/manifest-gate?manifest_id=mf_factory_1", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleManifestGate(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if cleared, _ := body["cleared"].(bool); cleared {
		t.Fatalf("expected cleared=false, got %#v", body)
	}
	if allowed, _ := body["allowed"].(bool); allowed {
		t.Fatalf("expected allowed=false, got %#v", body)
	}
	if got, _ := body["error"].(string); got != "AWAITING_PAYLOAD_SEAL" {
		t.Fatalf("expected AWAITING_PAYLOAD_SEAL, got %#v", body)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleEarnings_Unauthorized(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/earnings", nil)
	rr := httptest.NewRecorder()

	svc.HandleEarnings(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleEarnings_CompatibilityResponse(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithEarnings(repo, cacheBackend, func(ctx context.Context, driverID string) (DriverEarningsResponse, error) {
		_ = ctx
		if driverID != "drv-1" {
			return DriverEarningsResponse{}, nil
		}
		return DriverEarningsResponse{
			TotalDeliveries: 3,
			TotalVolume:     7200,
			TotalRoutes:     2,
			Last30Days: []DailyEarning{
				{Date: "2026-05-22", DeliveryCount: 2, Volume: 5000},
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/earnings", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleEarnings(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body DriverEarningsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.DriverID != "drv-1" || body.Currency != "UZS" {
		t.Fatalf("expected driver/currency normalization, got %#v", body)
	}
	if body.TotalDeliveries != 3 || body.TotalVolume != 7200 || body.TotalRoutes != 2 {
		t.Fatalf("expected Pegasus totals, got %#v", body)
	}
	if len(body.Last30Days) != 1 {
		t.Fatalf("expected one daily row, got %#v", body)
	}
	if body.Last30Days[0].Currency != "UZS" || body.Last30Days[0].VolumeMinor != 5000 {
		t.Fatalf("expected daily currency/minor aliases, got %#v", body.Last30Days[0])
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandleEarnings_EmptyFallbackResponse(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/earnings", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandleEarnings(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body DriverEarningsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.DriverID != "drv-1" || body.Currency != "UZS" {
		t.Fatalf("expected driver/currency fallback, got %#v", body)
	}
	if body.TotalDeliveries != 0 || body.TotalVolume != 0 || body.TotalRoutes != 0 {
		t.Fatalf("expected zero Pegasus totals, got %#v", body)
	}
	if body.Last30Days == nil || len(body.Last30Days) != 0 {
		t.Fatalf("expected empty last_30_days array, got %#v", body.Last30Days)
	}
	if body.TodayMinor != 0 || body.WeekMinor != 0 || body.MonthMinor != 0 {
		t.Fatalf("expected zero legacy buckets, got %#v", body)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandlePendingCollections_Unauthorized(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/pending-collections", nil)
	rr := httptest.NewRecorder()

	svc.HandlePendingCollections(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandlePendingCollections_CompatibilityEnvelope(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithPendingCollections(repo, cacheBackend, func(driverID string) []PendingCollection {
		if driverID != "drv-1" {
			return nil
		}
		return []PendingCollection{
			{
				OrderID:    "ord-older",
				RetailerID: "ret-2",
				Amount:     900,
				State:      "PENDING_CASH_COLLECTION",
				UpdatedAt:  "2026-05-22T08:00:00Z",
			},
			{
				OrderID:     "ord-newer",
				RetailerID:  "ret-1",
				AmountMinor: 1200,
				Currency:    "UZS",
				DueAt:       "2026-05-22T09:00:00Z",
			},
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/pending-collections?envelope=1", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandlePendingCollections(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Count              int                 `json:"count"`
		PendingCollections []PendingCollection `json:"pending_collections"`
		Pending            []PendingCollection `json:"pending"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count != 2 {
		t.Fatalf("expected count 2, got %#v", body)
	}
	if len(body.PendingCollections) != 2 {
		t.Fatalf("expected 2 pending_collections, got %#v", body)
	}
	if len(body.Pending) != 2 {
		t.Fatalf("expected 2 legacy pending items, got %#v", body)
	}
	if body.PendingCollections[0].OrderID != "ord-newer" {
		t.Fatalf("expected newest item first, got %#v", body.PendingCollections)
	}
	if body.PendingCollections[0].Amount != 1200 || body.PendingCollections[0].AmountMinor != 1200 {
		t.Fatalf("expected amount alias normalized to 1200, got %#v", body.PendingCollections[0])
	}
	if body.PendingCollections[0].State != "PENDING_CASH_COLLECTION" {
		t.Fatalf("expected default state set, got %#v", body.PendingCollections[0])
	}
	if body.PendingCollections[0].UpdatedAt != "2026-05-22T09:00:00Z" {
		t.Fatalf("expected updated_at derived from due_at, got %#v", body.PendingCollections[0])
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

func TestHandlePendingCollections_EmptyEnvelope(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestServiceWithPendingCollections(repo, cacheBackend, func(driverID string) []PendingCollection {
		_ = driverID
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/pending-collections?envelope=1", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
	rr := httptest.NewRecorder()

	svc.HandlePendingCollections(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Count              int                 `json:"count"`
		PendingCollections []PendingCollection `json:"pending_collections"`
		Pending            []PendingCollection `json:"pending"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count != 0 {
		t.Fatalf("expected count 0, got %#v", body)
	}
	if len(body.PendingCollections) != 0 || len(body.Pending) != 0 {
		t.Fatalf("expected empty arrays, got %#v", body)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no repo.Apply calls, got %d", repo.applyCalls)
	}
}

// ---------- test harness ----------

func newDriverTestService(repo *driverRepoSpy, cacheBackend *driverCacheBackendSpy) *Service {
	return newDriverTestServiceWithLookups(repo, cacheBackend, nil, nil, nil, nil)
}

func newDriverTestServiceWithManifestGate(repo *driverRepoSpy, cacheBackend *driverCacheBackendSpy, lookup ManifestGateLookup) *Service {
	return newDriverTestServiceWithLookups(repo, cacheBackend, lookup, nil, nil, nil)
}

func newDriverTestServiceWithManifest(repo *driverRepoSpy, cacheBackend *driverCacheBackendSpy, lookup ManifestLookup) *Service {
	return newDriverTestServiceWithLookups(repo, cacheBackend, nil, lookup, nil, nil)
}

func newDriverTestServiceWithPendingCollections(repo *driverRepoSpy, cacheBackend *driverCacheBackendSpy, lookup PendingCollectionsLookup) *Service {
	return newDriverTestServiceWithLookups(repo, cacheBackend, nil, nil, lookup, nil)
}

func newDriverTestServiceWithEarnings(repo *driverRepoSpy, cacheBackend *driverCacheBackendSpy, lookup EarningsLookup) *Service {
	return newDriverTestServiceWithLookups(repo, cacheBackend, nil, nil, nil, lookup)
}

func newDriverTestServiceWithLookups(repo *driverRepoSpy, cacheBackend *driverCacheBackendSpy, manifestGateLookup ManifestGateLookup, manifestLookup ManifestLookup, pendingLookup PendingCollectionsLookup, earningsLookup EarningsLookup) *Service {
	now := time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)
	cacheClient := cache.New(cacheBackend, nil)
	return NewService(ServiceConfig{
		Repo:         repo,
		Cache:        cacheClient,
		SupplierHub:  ws.NewHub("supplier", nil, nil),
		DriverHub:    ws.NewHub("driver", nil, nil),
		ManifestGate: manifestGateLookup,
		Manifest:     manifestLookup,
		PendingQuery: pendingLookup,
		Earnings:     earningsLookup,
		SupplierID:   "supplier-test",
		Currency:     "UZS",
		Now:          func() time.Time { return now },
	})
}

func withDriverClaims(req *http.Request, claims auth.Claims) *http.Request {
	return req.WithContext(auth.WithClaims(req.Context(), claims))
}

func decodeDriverOutboxPayload(t *testing.T, event outbox.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	return payload
}

func assertDriverCacheDeletedKeys(t *testing.T, deleted [][]string, expected ...string) {
	t.Helper()
	flat := make(map[string]struct{})
	for _, call := range deleted {
		for _, key := range call {
			flat[key] = struct{}{}
		}
	}
	for _, key := range expected {
		if _, ok := flat[key]; !ok {
			t.Fatalf("expected cache key %q invalidated, got %#v", key, deleted)
		}
	}
}

func assertDriverWSMessageContainsType(t *testing.T, messages [][]byte, wantType string) {
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
	t.Fatalf("expected ws event %q in messages: %q", wantType, messages)
}

type driverRepoSpy struct {
	applyCalls int
	events     []outbox.Event
}

func (r *driverRepoSpy) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	if emit != nil {
		buf := &driverTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	_ = ctx
	return nil
}

func (r *driverRepoSpy) ApplyAvailability(ctx context.Context, _ AvailabilityUpdate, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if emit != nil {
		buf := &driverTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	_ = ctx
	return nil
}

type driverTxnBufferSpy struct {
	events []outbox.Event
}

func (b *driverTxnBufferSpy) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

type driverCacheBackendSpy struct {
	deletedKeys [][]string
}

func (b *driverCacheBackendSpy) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (b *driverCacheBackendSpy) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (b *driverCacheBackendSpy) Delete(_ context.Context, keys ...string) error {
	copyKeys := append([]string(nil), keys...)
	b.deletedKeys = append(b.deletedKeys, copyKeys)
	return nil
}

func (b *driverCacheBackendSpy) Publish(context.Context, string, []byte) error {
	return nil
}

func (b *driverCacheBackendSpy) Subscribe(context.Context, string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	close(ch)
	return ch, func() {}, nil
}

type driverWSConnSpy struct {
	id       string
	messages [][]byte
}

func (c *driverWSConnSpy) ID() string            { return c.id }
func (c *driverWSConnSpy) Identity() auth.Claims { return auth.Claims{} }
func (c *driverWSConnSpy) Send(_ context.Context, payload []byte) error {
	copyPayload := append([]byte(nil), payload...)
	c.messages = append(c.messages, copyPayload)
	return nil
}
