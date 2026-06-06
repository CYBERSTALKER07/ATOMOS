package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type testDemandPlanner struct {
	warehouseID string
	start       time.Time
	days        int
	series      []order.WarehouseDemandDay
	err         error
}

func (p *testDemandPlanner) WarehouseDemandForecast(_ context.Context, warehouseID string, start time.Time, days int) ([]order.WarehouseDemandDay, error) {
	p.warehouseID = warehouseID
	p.start = start
	p.days = days
	return p.series, p.err
}

func TestHandleSupplyRequests_PostSeamParity(t *testing.T) {
	repo := &warehouseRepoSpy{}
	cacheBackend := &warehouseCacheBackendSpy{}
	supplierConn := &warehouseWSConnSpy{id: "supplier-conn"}
	warehouseConn := &warehouseWSConnSpy{id: "warehouse-conn"}
	planner := &testDemandPlanner{series: []order.WarehouseDemandDay{{
		Date:                     "2026-05-19",
		ProjectedUnits:           12,
		CommittedUnits:           7,
		PendingConfirmationUnits: 5,
		Currency:                 "UZS",
	}}}

	svc := newWarehouseTestServiceWithPlanner(repo, cacheBackend, planner)
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.warehouseHub.Subscribe("warehouse:wh-1", warehouseConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/supply-requests?warehouse_id=wh-1&start_date=2026-05-19&days=3", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-claims"})
	rr := httptest.NewRecorder()

	svc.HandleSupplyRequests(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected 1 repo create call, got %d", repo.createCalls)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected 0 repo apply calls, got %d", repo.applyCalls)
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(repo.events))
	}
	if len(repo.createdRequests) != 1 {
		t.Fatalf("expected 1 persisted request, got %d", len(repo.createdRequests))
	}
	created := repo.createdRequests[0]
	if created.WarehouseID != "wh-1" {
		t.Fatalf("expected warehouse_id wh-1, got %q", created.WarehouseID)
	}
	if created.CoverageStartDate != "2026-05-19" || created.CoverageDays != 3 {
		t.Fatalf("unexpected coverage window: %+v", created)
	}
	if created.ProjectedUnits != 12 || created.CommittedUnits != 7 || created.PendingConfirmationUnits != 5 {
		t.Fatalf("unexpected forecast snapshot: %+v", created)
	}
	if planner.warehouseID != "wh-1" || planner.days != 3 || planner.start.Format("2006-01-02") != "2026-05-19" {
		t.Fatalf("unexpected planner call warehouse=%s start=%s days=%d", planner.warehouseID, planner.start.Format("2006-01-02"), planner.days)
	}

	payload := decodeWarehouseOutboxPayload(t, repo.events[0])
	if got, _ := payload["type"].(string); got != events.EventWarehouseSupplyRequestOpened {
		t.Fatalf("expected event type %q, got %q", events.EventWarehouseSupplyRequestOpened, got)
	}
	if got, _ := payload["warehouse_id"].(string); got != "wh-1" {
		t.Fatalf("expected warehouse_id wh-1, got %q", got)
	}
	if got, _ := payload["requested_by"].(string); got != "ops-1" {
		t.Fatalf("expected requested_by ops-1, got %q", got)
	}
	if got, _ := payload["projected_units"].(float64); got != 12 {
		t.Fatalf("expected projected_units 12, got %v", payload["projected_units"])
	}
	if got, _ := payload["coverage_days"].(float64); got != 3 {
		t.Fatalf("expected coverage_days 3, got %v", payload["coverage_days"])
	}

	var response SupplyRequest
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode supply request response: %v", err)
	}
	if response.State != "OPEN" || response.ProjectedUnits != 12 {
		t.Fatalf("unexpected response payload: %+v", response)
	}

	assertWarehouseCacheDeletedKeys(t, cacheBackend.deletedKeys, warehouseSupplyRequestsKey("supplier-test", "wh-1"))
	assertWarehouseWSMessageContainsType(t, supplierConn.messages, events.EventWarehouseSupplyRequestOpened)
	assertWarehouseWSMessageContainsType(t, warehouseConn.messages, events.EventWarehouseSupplyRequestOpened)
}

func TestHandleSupplyRequests_GetUsesRepository(t *testing.T) {
	repo := &warehouseRepoSpy{listRequests: []SupplyRequest{{
		RequestID:                "req-1",
		WarehouseID:              "wh-1",
		Status:                   "OPEN",
		State:                    "OPEN",
		RequestedBy:              "ops-1",
		CoverageStartDate:        "2026-05-19",
		CoverageDays:             7,
		ProjectedUnits:           12,
		CommittedUnits:           7,
		PendingConfirmationUnits: 5,
		CreatedAt:                "2026-05-19T09:00:00Z",
		UpdatedAt:                "2026-05-19T09:00:00Z",
	}}}
	svc := newWarehouseTestService(repo, &warehouseCacheBackendSpy{})

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/supply-requests?warehouse_id=wh-1", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-claims"})
	rr := httptest.NewRecorder()

	svc.HandleSupplyRequests(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.listCalls != 1 || repo.listWarehouseID != "wh-1" {
		t.Fatalf("unexpected list invocation calls=%d warehouse=%q", repo.listCalls, repo.listWarehouseID)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode supply request list response: %v", err)
	}
	requests, ok := payload["requests"].([]any)
	if !ok || len(requests) != 1 {
		t.Fatalf("unexpected requests payload: %v", payload)
	}
	if payload["supply_requests"] == nil {
		t.Fatalf("expected additive supply_requests alias in payload: %v", payload)
	}
}

func TestHandleDispatchLock_AcquireReleaseSeamParity(t *testing.T) {
	repo := &warehouseRepoSpy{}
	cacheBackend := &warehouseCacheBackendSpy{}
	supplierConn := &warehouseWSConnSpy{id: "supplier-conn"}
	warehouseConn := &warehouseWSConnSpy{id: "warehouse-conn"}

	svc := newWarehouseTestService(repo, cacheBackend)
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.warehouseHub.Subscribe("warehouse:wh-1", warehouseConn)

	acquireReq := httptest.NewRequest(http.MethodPost, "/v1/warehouse/dispatch-lock?warehouse_id=wh-1", strings.NewReader(`{"entity_type":"ORDER","entity_id":"ord-1","reason":"manual"}`))
	acquireReq = withWarehouseClaims(acquireReq, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-claims"})
	acquireRR := httptest.NewRecorder()

	svc.HandleDispatchLock(acquireRR, acquireReq)

	if acquireRR.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", acquireRR.Code, acquireRR.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call after acquire, got %d", repo.applyCalls)
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 outbox event after acquire, got %d", len(repo.events))
	}

	acquirePayload := decodeWarehouseOutboxPayload(t, repo.events[0])
	if got, _ := acquirePayload["type"].(string); got != events.EventWarehouseDispatchLockChanged {
		t.Fatalf("expected event type %q, got %q", events.EventWarehouseDispatchLockChanged, got)
	}
	if got, _ := acquirePayload["action"].(string); got != "ACQUIRED" {
		t.Fatalf("expected action ACQUIRED, got %q", got)
	}

	var lock DispatchLock
	if err := json.Unmarshal(acquireRR.Body.Bytes(), &lock); err != nil {
		t.Fatalf("failed to decode acquire response: %v", err)
	}
	if strings.TrimSpace(lock.LockID) == "" {
		t.Fatalf("expected lock_id in acquire response")
	}

	releaseReq := httptest.NewRequest(http.MethodDelete, "/v1/warehouse/dispatch-lock?warehouse_id=wh-1&lock_id="+url.QueryEscape(lock.LockID), nil)
	releaseReq = withWarehouseClaims(releaseReq, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-claims"})
	releaseRR := httptest.NewRecorder()

	svc.HandleDispatchLock(releaseRR, releaseReq)

	if releaseRR.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", releaseRR.Code, releaseRR.Body.String())
	}
	if repo.applyCalls != 2 {
		t.Fatalf("expected 2 repo apply calls after release, got %d", repo.applyCalls)
	}
	if len(repo.events) != 2 {
		t.Fatalf("expected 2 outbox events after release, got %d", len(repo.events))
	}

	releasePayload := decodeWarehouseOutboxPayload(t, repo.events[1])
	if got, _ := releasePayload["type"].(string); got != events.EventWarehouseDispatchLockChanged {
		t.Fatalf("expected event type %q, got %q", events.EventWarehouseDispatchLockChanged, got)
	}
	if got, _ := releasePayload["action"].(string); got != "RELEASED" {
		t.Fatalf("expected action RELEASED, got %q", got)
	}
	if got, _ := releasePayload["lock_id"].(string); got != lock.LockID {
		t.Fatalf("expected lock_id %q, got %q", lock.LockID, got)
	}

	assertWarehouseCacheDeletedKeys(t, cacheBackend.deletedKeys, warehouseDispatchLocksKey("supplier-test"))
	assertWarehouseWSMessageContainsType(t, supplierConn.messages, events.EventWarehouseDispatchLockChanged)
	assertWarehouseWSMessageContainsType(t, warehouseConn.messages, events.EventWarehouseDispatchLockChanged)
}

func TestHandleDispatchLock_ReleaseMissingReturnsNotFound(t *testing.T) {
	repo := &warehouseRepoSpy{}
	cacheBackend := &warehouseCacheBackendSpy{}
	supplierConn := &warehouseWSConnSpy{id: "supplier-conn"}
	warehouseConn := &warehouseWSConnSpy{id: "warehouse-conn"}
	svc := newWarehouseTestService(repo, cacheBackend)
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.warehouseHub.Subscribe("warehouse:wh-1", warehouseConn)

	acquireReq := httptest.NewRequest(http.MethodPost, "/v1/warehouse/dispatch-lock?warehouse_id=wh-1", strings.NewReader(`{"entity_type":"ORDER","entity_id":"ord-1","reason":"manual"}`))
	acquireReq = withWarehouseClaims(acquireReq, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-claims"})
	acquireRR := httptest.NewRecorder()
	svc.HandleDispatchLock(acquireRR, acquireReq)
	if acquireRR.Code != http.StatusCreated {
		t.Fatalf("expected acquire status 201, got %d body=%s", acquireRR.Code, acquireRR.Body.String())
	}

	outboxTypesAfterAcquire := warehouseOutboxEventTypes(repo.events)
	if len(outboxTypesAfterAcquire) != 1 || outboxTypesAfterAcquire[0] != events.EventWarehouseDispatchLockChanged {
		t.Fatalf("unexpected outbox events after acquire: %#v", outboxTypesAfterAcquire)
	}
	cacheCallsAfterAcquire := len(cacheBackend.deletedKeys)
	supplierTypesAfterAcquire := warehouseWSMessageTypes(supplierConn.messages)
	warehouseTypesAfterAcquire := warehouseWSMessageTypes(warehouseConn.messages)
	if len(supplierTypesAfterAcquire) != 1 || supplierTypesAfterAcquire[0] != events.EventWarehouseDispatchLockChanged {
		t.Fatalf("unexpected supplier ws events after acquire: %#v", supplierTypesAfterAcquire)
	}
	if len(warehouseTypesAfterAcquire) != 1 || warehouseTypesAfterAcquire[0] != events.EventWarehouseDispatchLockChanged {
		t.Fatalf("unexpected warehouse ws events after acquire: %#v", warehouseTypesAfterAcquire)
	}
	if got, _ := decodeWarehouseWSMessagePayload(t, supplierConn.messages[0])["action"].(string); got != "ACQUIRED" {
		t.Fatalf("expected supplier ws action ACQUIRED after acquire, got %q", got)
	}
	if got, _ := decodeWarehouseWSMessagePayload(t, warehouseConn.messages[0])["action"].(string); got != "ACQUIRED" {
		t.Fatalf("expected warehouse ws action ACQUIRED after acquire, got %q", got)
	}

	req := httptest.NewRequest(http.MethodDelete, "/v1/warehouse/dispatch-lock?warehouse_id=wh-1&lock_id=missing", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-claims"})
	rr := httptest.NewRecorder()

	svc.HandleDispatchLock(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 2 {
		t.Fatalf("expected two repo apply calls (acquire + missing release), got %d", repo.applyCalls)
	}
	if got := warehouseOutboxEventTypes(repo.events); len(got) != len(outboxTypesAfterAcquire) {
		t.Fatalf("expected no extra outbox events on missing lock release, got %#v", got)
	} else {
		for i := range got {
			if got[i] != outboxTypesAfterAcquire[i] {
				t.Fatalf("expected outbox event sequence unchanged, got %#v want %#v", got, outboxTypesAfterAcquire)
			}
		}
	}
	if len(cacheBackend.deletedKeys) != cacheCallsAfterAcquire {
		t.Fatalf("expected no extra cache invalidation on missing lock release, got %#v", cacheBackend.deletedKeys)
	}
	if got := warehouseWSMessageTypes(supplierConn.messages); len(got) != len(supplierTypesAfterAcquire) {
		t.Fatalf("expected no extra supplier ws events on missing lock release, got %#v", got)
	} else {
		for i := range got {
			if got[i] != supplierTypesAfterAcquire[i] {
				t.Fatalf("expected supplier ws event sequence unchanged, got %#v want %#v", got, supplierTypesAfterAcquire)
			}
		}
	}
	if got := warehouseWSMessageTypes(warehouseConn.messages); len(got) != len(warehouseTypesAfterAcquire) {
		t.Fatalf("expected no extra warehouse ws events on missing lock release, got %#v", got)
	} else {
		for i := range got {
			if got[i] != warehouseTypesAfterAcquire[i] {
				t.Fatalf("expected warehouse ws event sequence unchanged, got %#v want %#v", got, warehouseTypesAfterAcquire)
			}
		}
	}
}

func TestHandleDemandForecastUsesOrderPlanner(t *testing.T) {
	planner := &testDemandPlanner{series: []order.WarehouseDemandDay{{
		Date:                     "2026-05-19",
		ProjectedUnits:           12,
		ProjectedRevenue:         5400,
		CommittedUnits:           7,
		PendingConfirmationUnits: 5,
		Currency:                 "UZS",
	}}}
	svc := NewService(ServiceConfig{
		Repo:       &warehouseRepoSpy{},
		Planner:    planner,
		SupplierID: "supplier-test",
		Currency:   "UZS",
		Now:        func() time.Time { return time.Date(2026, 5, 19, 9, 0, 0, 0, time.UTC) },
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/demand/forecast?days=3", nil)
	req = withWarehouseClaims(req, auth.Claims{Subject: "ops-1", HomeNodeID: "wh-1"})
	rr := httptest.NewRecorder()

	svc.HandleDemandForecast(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if planner.warehouseID != "wh-1" || planner.days != 3 {
		t.Fatalf("planner call = warehouse %s days %d", planner.warehouseID, planner.days)
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode forecast response: %v", err)
	}
	if payload["warehouse_id"] != "wh-1" {
		t.Fatalf("warehouse_id=%v want wh-1", payload["warehouse_id"])
	}
	series, ok := payload["series"].([]any)
	if !ok || len(series) != 1 {
		t.Fatalf("series=%v", payload["series"])
	}
	day := series[0].(map[string]any)
	if day["pending_confirmation_units"] != float64(5) || day["committed_units"] != float64(7) {
		t.Fatalf("forecast day=%v", day)
	}
}

func newWarehouseTestService(repo *warehouseRepoSpy, cacheBackend *warehouseCacheBackendSpy) *Service {
	return newWarehouseTestServiceWithPlanner(repo, cacheBackend, nil)
}

func newWarehouseTestServiceWithPlanner(repo *warehouseRepoSpy, cacheBackend *warehouseCacheBackendSpy, planner DemandPlanner) *Service {
	now := time.Date(2026, 5, 19, 9, 0, 0, 0, time.UTC)
	cacheClient := cache.New(cacheBackend, nil)
	supplierHub := ws.NewHub("supplier", nil, nil)
	warehouseHub := ws.NewHub("warehouse", nil, nil)

	return NewService(ServiceConfig{
		Repo:         repo,
		Planner:      planner,
		Cache:        cacheClient,
		SupplierHub:  supplierHub,
		WarehouseHub: warehouseHub,
		SupplierID:   "supplier-test",
		Currency:     "UZS",
		Now:          func() time.Time { return now },
	})
}

func withWarehouseClaims(req *http.Request, claims auth.Claims) *http.Request {
	ctx := auth.WithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func decodeWarehouseOutboxPayload(t *testing.T, event outbox.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to decode outbox payload: %v", err)
	}
	return payload
}

func decodeWarehouseWSMessagePayload(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("failed to decode ws payload: %v", err)
	}
	return payload
}

func warehouseOutboxEventTypes(eventsList []outbox.Event) []string {
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

func warehouseWSMessageTypes(messages [][]byte) []string {
	types := make([]string, 0, len(messages))
	for _, raw := range messages {
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			types = append(types, "")
			continue
		}
		typeName, _ := payload["type"].(string)
		types = append(types, typeName)
	}
	return types
}

func assertWarehouseCacheDeletedKeys(t *testing.T, deleted [][]string, expected ...string) {
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

func assertWarehouseWSMessageContainsType(t *testing.T, messages [][]byte, wantType string) {
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

type warehouseRepoSpy struct {
	listCalls       int
	listWarehouseID string
	listRequests    []SupplyRequest
	createCalls     int
	createdRequests []SupplyRequest
	applyCalls      int
	events          []outbox.Event
}

func (r *warehouseRepoSpy) ListSupplyRequests(_ context.Context, warehouseID string, _ int) ([]SupplyRequest, error) {
	r.listCalls++
	r.listWarehouseID = warehouseID
	return append([]SupplyRequest(nil), r.listRequests...), nil
}

func (r *warehouseRepoSpy) CreateSupplyRequest(ctx context.Context, req SupplyRequest, emit func(outbox.TxnBuffer) error) error {
	r.createCalls++
	r.createdRequests = append(r.createdRequests, req)
	if emit != nil {
		buf := &warehouseTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	_ = ctx
	return nil
}

func (r *warehouseRepoSpy) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	r.applyCalls++
	if mutate != nil {
		if err := mutate(); err != nil {
			return err
		}
	}
	if emit != nil {
		buf := &warehouseTxnBufferSpy{}
		if err := emit(buf); err != nil {
			return err
		}
		r.events = append(r.events, buf.events...)
	}
	_ = ctx
	return nil
}

type warehouseTxnBufferSpy struct {
	events []outbox.Event
}

func (b *warehouseTxnBufferSpy) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

type warehouseCacheBackendSpy struct {
	deletedKeys [][]string
}

func (b *warehouseCacheBackendSpy) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (b *warehouseCacheBackendSpy) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (b *warehouseCacheBackendSpy) Delete(_ context.Context, keys ...string) error {
	copyKeys := append([]string(nil), keys...)
	b.deletedKeys = append(b.deletedKeys, copyKeys)
	return nil
}

func (b *warehouseCacheBackendSpy) Publish(context.Context, string, []byte) error {
	return nil
}

func (b *warehouseCacheBackendSpy) Subscribe(context.Context, string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	close(ch)
	return ch, func() {}, nil
}

type warehouseWSConnSpy struct {
	id       string
	messages [][]byte
}

func (c *warehouseWSConnSpy) ID() string {
	return c.id
}

func (c *warehouseWSConnSpy) Identity() auth.Claims {
	return auth.Claims{}
}

func (c *warehouseWSConnSpy) Send(_ context.Context, payload []byte) error {
	copyPayload := append([]byte(nil), payload...)
	c.messages = append(c.messages, copyPayload)
	return nil
}
