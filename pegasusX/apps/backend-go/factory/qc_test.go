package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type memoryQCRepo struct {
	mu        sync.Mutex
	requests  map[string]qcRequestMeta
	rows      map[string]SupplyRequestQCResponse
	events    []outbox.Event
	lastState string
}

func (m *memoryQCRepo) GetVisibleRequest(_ context.Context, requestID, supplierID, factoryID string) (qcRequestMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.requests[requestID]
	if !ok || row.SupplierID != supplierID || (factoryID != "" && row.FactoryID != factoryID) {
		return qcRequestMeta{}, errQCRequestNotFound
	}
	return row, nil
}

func (m *memoryQCRepo) GetQC(_ context.Context, requestID string) (SupplyRequestQCResponse, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[requestID]
	return row, ok, nil
}

func (m *memoryQCRepo) UpsertQC(ctx context.Context, row qcUpsert, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	meta := m.requests[row.RequestID]
	m.lastState = meta.State
	m.rows[row.RequestID] = SupplyRequestQCResponse{
		RequestID:   row.RequestID,
		Result:      row.Result,
		Notes:       row.Notes,
		InspectedBy: row.InspectedBy,
	}
	m.mu.Unlock()
	if emit != nil {
		buf := &spannerTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		m.mu.Lock()
		m.events = append(m.events, buf.events...)
		m.mu.Unlock()
	}
	return nil
}

func TestValidQCResult(t *testing.T) {
	if !validQCResult("PASS") || !validQCResult("fail") {
		t.Fatal("PASS/FAIL")
	}
	if validQCResult("OK") || validQCResult("") {
		t.Fatal("reject other")
	}
}

func TestHandleSupplyRequestQC_NoSpanner503(t *testing.T) {
	svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})
	req := withFactoryRouteParam(httptest.NewRequest(http.MethodGet, "/v1/factory/supply-requests/req-1/qc", nil), "id", "req-1")
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSupplyRequestQC_InvalidResult400(t *testing.T) {
	svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})
	svc.qcRepo = &memoryQCRepo{requests: map[string]qcRequestMeta{}, rows: map[string]SupplyRequestQCResponse{}}
	req := withFactoryRouteParam(httptest.NewRequest(http.MethodPost, "/v1/factory/supply-requests/req-1/qc", strings.NewReader(`{"result":"MAYBE"}`)), "id", "req-1")
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSupplyRequestQC_MissingRequest404(t *testing.T) {
	svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})
	svc.qcRepo = &memoryQCRepo{requests: map[string]qcRequestMeta{}, rows: map[string]SupplyRequestQCResponse{}}
	req := withFactoryRouteParam(httptest.NewRequest(http.MethodGet, "/v1/factory/supply-requests/missing/qc", nil), "id", "missing")
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSupplyRequestQC_EmptyResultWhenNoRow(t *testing.T) {
	svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})
	svc.factoryNodeID = "factory-1"
	svc.qcRepo = &memoryQCRepo{
		requests: map[string]qcRequestMeta{
			"req-1": {RequestID: "req-1", WarehouseID: "wh-1", SupplierID: "supplier-test", FactoryID: "factory-1", State: "ACKNOWLEDGED"},
		},
		rows: map[string]SupplyRequestQCResponse{},
	}
	req := withFactoryRouteParam(httptest.NewRequest(http.MethodGet, "/v1/factory/supply-requests/req-1/qc", nil), "id", "req-1")
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body SupplyRequestQCResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.RequestID != "req-1" || body.Result != "" {
		t.Fatalf("want empty result, got %+v", body)
	}
}

func TestHandleSupplyRequestQC_PostPASSPersistsWithoutChangingState(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	svc.factoryNodeID = "factory-1"
	mem := &memoryQCRepo{
		requests: map[string]qcRequestMeta{
			"req-1": {RequestID: "req-1", WarehouseID: "wh-1", SupplierID: "supplier-test", FactoryID: "factory-1", State: "ACKNOWLEDGED"},
		},
		rows: map[string]SupplyRequestQCResponse{},
	}
	svc.qcRepo = mem
	req := withFactoryRouteParam(httptest.NewRequest(http.MethodPost, "/v1/factory/supply-requests/req-1/qc", strings.NewReader(`{"result":"PASS","notes":"ok"}`)), "id", "req-1")
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "inspector-1", Role: auth.RoleFactoryAdmin, SupplierID: "supplier-test"}))
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	stored, ok, _ := mem.GetQC(req.Context(), "req-1")
	if !ok || stored.Result != "PASS" {
		t.Fatalf("persist %+v ok=%v", stored, ok)
	}
	if mem.lastState != "ACKNOWLEDGED" {
		t.Fatalf("must not change request state, lastState=%s", mem.lastState)
	}
	types := factoryOutboxEventTypes(mem.events)
	if len(types) != 1 || types[0] != events.EventFactorySupplyRequestUpdate {
		t.Fatalf("outbox=%v", types)
	}
	var payload map[string]any
	if err := json.Unmarshal(mem.events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["qc_result"] != "PASS" || payload["state"] != "ACKNOWLEDGED" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestHandleSupplyRequestQC_MissingID(t *testing.T) {
	svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})
	req := httptest.NewRequest(http.MethodGet, "/v1/factory/supply-requests//qc", nil)
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestQC(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}
