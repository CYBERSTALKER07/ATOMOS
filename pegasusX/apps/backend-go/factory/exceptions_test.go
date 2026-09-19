package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

type memoryExceptionRepo struct {
	rows []ManifestException
}

func (m *memoryExceptionRepo) List(_ context.Context, _, _ string) ([]ManifestException, error) {
	out := make([]ManifestException, len(m.rows))
	copy(out, m.rows)
	return out, nil
}

func (m *memoryExceptionRepo) Get(_ context.Context, exceptionID, _, _ string) (ManifestException, bool, error) {
	for _, row := range m.rows {
		if row.ExceptionID == exceptionID {
			return row, true, nil
		}
	}
	return ManifestException{}, false, nil
}

func TestHandleManifestExceptions_SeedOffEmpty(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/factory/manifest-exceptions", nil)
	rr := httptest.NewRecorder()
	svc.HandleManifestExceptions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Exceptions []ManifestException `json:"exceptions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Exceptions == nil {
		t.Fatal("exceptions must be [] not null")
	}
	if len(body.Exceptions) != 0 {
		t.Fatalf("seed-off GET must not serve demo mex_factory_demo_1: %+v", body.Exceptions)
	}
}

func TestHandleManifestExceptions_BackendMapsOrderIdToTransferID(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)
	svc.exceptionRepo = &memoryExceptionRepo{rows: []ManifestException{{
		ExceptionID:  "mex-spanner-1",
		ManifestID:   "mf-factory-1",
		TransferID:   "ord-as-transfer",
		Reason:       "OVERFLOW",
		AttemptCount: 1,
		CreatedAt:    "2026-08-13T10:00:00Z",
	}}}

	req := httptest.NewRequest(http.MethodGet, "/v1/factory/manifest-exceptions", nil)
	rr := httptest.NewRecorder()
	svc.HandleManifestExceptions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	list, _ := raw["exceptions"].([]any)
	if len(list) != 1 {
		t.Fatalf("want 1 exception, got %v", raw)
	}
	row, _ := list[0].(map[string]any)
	if row["transfer_id"] != "ord-as-transfer" {
		t.Fatalf("factory JSON must keep transfer_id mapped from OrderId, got %v", row)
	}
	if _, has := row["order_id"]; has {
		t.Fatal("factory JSON must not grow supplier order_id field")
	}
}

func TestHandleResolveManifestException_SpannerShapedRow(t *testing.T) {
	repo := &factoryRepoSpy{}
	cacheBackend := &factoryCacheBackendSpy{}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	factoryDisablePortalSeed(t)
	svc.exceptionRepo = &memoryExceptionRepo{rows: []ManifestException{{
		ExceptionID: "mex-spanner-1",
		ManifestID:  "mf-factory-1",
		TransferID:  "tr-from-order",
		Reason:      "OVERFLOW",
	}}}

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifest-exceptions/mex-spanner-1/resolve", strings.NewReader(`{"resolution":"RESOLVED"}`))
	req = withFactoryRouteParam(req, "exceptionID", "mex-spanner-1")
	rr := httptest.NewRecorder()
	svc.HandleResolveManifestException(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(repo.resolvedExceptions) != 1 || repo.resolvedExceptions[0].ExceptionID != "mex-spanner-1" {
		t.Fatalf("persist %+v", repo.resolvedExceptions)
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != 1 || got[0] != events.EventManifestExceptionResolved {
		t.Fatalf("outbox=%v", got)
	}
}

func TestHandleDispatch_EmptyQueueNoInvent(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/dispatch", strings.NewReader(`{"reason":"wave"}`))
	rr := httptest.NewRecorder()
	svc.HandleDispatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["created_manifest_count"] != float64(0) || body["manifests_created"] != float64(0) {
		t.Fatalf("empty queue must not invent, got %v", body)
	}
	if body["optimizer_class"] != "HEURISTIC" || body["dispatch_algo"] != "pick_n_created_v1" {
		t.Fatalf("honesty labels missing: %v", body)
	}
	if repo.applyCalls != 0 {
		t.Fatalf("empty queue must not start a txn, applyCalls=%d", repo.applyCalls)
	}
	if len(repo.events) != 0 {
		t.Fatalf("empty queue must not emit outbox: %#v", factoryOutboxEventTypes(repo.events))
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	for _, tr := range svc.transfers {
		if strings.HasPrefix(tr.TransferID, "tr_") && tr.State == "CREATED" && tr.OrderID != "" && tr.TotalVU == 20 {
			t.Fatalf("invented transfer still present: %+v", tr)
		}
	}
}

func TestLookupManifestException_SpannerFirst(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)
	svc.exceptionRepo = &memoryExceptionRepo{rows: []ManifestException{{
		ExceptionID: "mex-spanner-1",
		ManifestID:  "mf-1",
		TransferID:  "tr-1",
		Reason:      "OVERFLOW",
		CreatedAt:   "2026-08-13T10:00:00Z",
	}}}
	row, fromMemory, ok, err := svc.lookupManifestException(t.Context(), "mex-spanner-1")
	if err != nil || !ok || fromMemory {
		t.Fatalf("want spanner hit fromMemory=false ok=true err=%v fromMemory=%v ok=%v", err, fromMemory, ok)
	}
	if row.ExceptionID != "mex-spanner-1" {
		t.Fatalf("row=%+v", row)
	}
	_, _, ok, err = svc.lookupManifestException(t.Context(), "mex_factory_demo_1")
	if err != nil || ok {
		t.Fatalf("seed-off must not fall back to demo memory, ok=%v err=%v", ok, err)
	}
}
