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

func TestHandleManifestStartLoading_SeamParity(t *testing.T) {
	repo := &factoryRepoSpy{}
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
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
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}

	svc := newFactoryTestService(repo, cacheBackend)
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
	cacheBackend := &factoryCacheBackendSpy{}
	svc := newFactoryTestService(repo, cacheBackend)

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
	applyCalls int
	events     []outbox.Event
}

func (r *factoryRepoSpy) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if mutate != nil {
		if err := mutate(); err != nil {
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
