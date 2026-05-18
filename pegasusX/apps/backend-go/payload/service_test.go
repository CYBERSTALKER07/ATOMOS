package payload

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
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestHandleManifestException_EscalationSeamParity(t *testing.T) {
	repo := &payloadRepoSpy{}
	cacheBackend := &payloadCacheBackendSpy{}
	supplierConn := &payloadWSConnSpy{id: "supplier-conn"}
	payloadConn := &payloadWSConnSpy{id: "payload-conn"}

	svc := newPayloadTestService(repo, cacheBackend)
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
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.payloadHub.Subscribe("payload:supplier-test", payloadConn)

	// Add a second target manifest so reassignment updates concrete manifest keys.
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

func newPayloadTestService(repo *payloadRepoSpy, cacheBackend *payloadCacheBackendSpy) *Service {
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
	applyCalls int
	events     []outbox.Event
}

func (r *payloadRepoSpy) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if mutate != nil {
		if err := mutate(); err != nil {
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
