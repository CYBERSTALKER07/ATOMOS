package factory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func factoryDisablePortalSeed(t *testing.T) {
	t.Helper()
	t.Setenv("FACTORY_PORTAL_SEED", "false")
	t.Setenv("USE_DEMO_SEED", "false")
}

func TestHandleStaff_Create_ClassA(t *testing.T) {
	repo := &factoryRepoSpy{}
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/staff", strings.NewReader(`{"name":"Bay Lead","role":"FACTORY_OPERATOR","phone":"+998901112233"}`))
	rr := httptest.NewRecorder()
	svc.HandleStaff(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.savedStaff) != 1 {
		t.Fatalf("expected SaveStaff once, got %d", len(repo.savedStaff))
	}
	if repo.savedStaff[0].Name != "Bay Lead" || repo.savedStaff[0].Role != "FACTORY_OPERATOR" {
		t.Fatalf("unexpected saved staff: %+v", repo.savedStaff[0])
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != 1 || got[0] != events.EventFactoryStaffCreated {
		t.Fatalf("unexpected outbox events: %#v", got)
	}
	if hash := repo.savedStaff[0].PasswordHash; hash == "" || hash == "unset" || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}
	assertFactoryCacheDeletedKeys(t, cacheBackend.deletedKeys, factoryStaffListKey("supplier-test"))
	assertFactoryWSMessageContainsType(t, supplierConn.messages, events.EventFactoryStaffCreated)
	assertFactoryWSMessageContainsType(t, factoryConn.messages, events.EventFactoryStaffCreated)
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := body["name"].(string); got != "Bay Lead" {
		t.Fatalf("expected name Bay Lead, got %v", body)
	}
	if body["invite_token"] == nil || body["invite_token"] == "" {
		t.Fatal("create without pin must return invite_token once")
	}
}

func TestHandleStaff_Create_NameRequired(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/staff", strings.NewReader(`{"role":"FACTORY_OPERATOR"}`))
	rr := httptest.NewRecorder()
	svc.HandleStaff(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no persist on validation fail, got applyCalls=%d", repo.applyCalls)
	}
}

func TestHandleStaff_Create_RoleInvalid(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/staff", strings.NewReader(`{"name":"X","role":"ADMIN"}`))
	rr := httptest.NewRecorder()
	svc.HandleStaff(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 0 {
		t.Fatalf("expected no persist on invalid role, got applyCalls=%d", repo.applyCalls)
	}
}

func TestHandleStaff_GET_SeedOffEmpty(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/factory/staff", nil)
	rr := httptest.NewRecorder()
	svc.HandleStaff(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Staff []map[string]any `json:"staff"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Staff) != 0 {
		t.Fatalf("seed-off GET must not serve demo staff: %+v", body.Staff)
	}
}

func TestHandleResolveManifestException_ClassA(t *testing.T) {
	repo := &factoryRepoSpy{}
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifest-exceptions/mex_factory_demo_1/resolve", strings.NewReader(`{"resolution":"RESOLVED","note":"ok"}`))
	req = withFactoryRouteParam(req, "exceptionID", "mex_factory_demo_1")
	rr := httptest.NewRecorder()
	svc.HandleResolveManifestException(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}
	if len(repo.resolvedExceptions) != 1 {
		t.Fatalf("expected ResolveException once, got %d", len(repo.resolvedExceptions))
	}
	if repo.resolvedExceptions[0].ExceptionID != "mex_factory_demo_1" {
		t.Fatalf("unexpected exception: %+v", repo.resolvedExceptions[0])
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != 1 || got[0] != events.EventManifestExceptionResolved {
		t.Fatalf("unexpected outbox events: %#v", got)
	}
	assertFactoryCacheDeletedKeys(t, cacheBackend.deletedKeys, factoryExceptionListKey("supplier-test"), factoryManifestKey("mf_factory_1"))
	assertFactoryWSMessageContainsType(t, supplierConn.messages, events.EventManifestExceptionResolved)
	assertFactoryWSMessageContainsType(t, factoryConn.messages, events.EventManifestExceptionResolved)
}

func TestHandleResolveManifestException_NotFound(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/manifest-exceptions/mex_missing/resolve", strings.NewReader(`{"resolution":"RESOLVED"}`))
	req = withFactoryRouteParam(req, "exceptionID", "mex_missing")
	rr := httptest.NewRecorder()
	svc.HandleResolveManifestException(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox on missing exception, got %#v", factoryOutboxEventTypes(repo.events))
	}
	if len(repo.resolvedExceptions) != 0 {
		t.Fatalf("expected no persist on missing exception, got %d", len(repo.resolvedExceptions))
	}
}

func TestHandleTransfers_Create_EmitsTransferCreated(t *testing.T) {
	repo := &factoryRepoSpy{}
	cacheBackend := &factoryCacheBackendSpy{}
	supplierConn := &factoryWSConnSpy{id: "supplier-conn"}
	factoryConn := &factoryWSConnSpy{id: "factory-conn"}
	svc := newFactoryTestService(repo, cacheBackend)
	repo.svc = svc
	svc.supplierHub.Subscribe("supplier:supplier-test", supplierConn)
	svc.factoryHub.Subscribe("factory:supplier-test", factoryConn)

	req := httptest.NewRequest(http.MethodPost, "/v1/factory/transfers/create", strings.NewReader(`{"order_id":"ord_x","total_vu":40}`))
	rr := httptest.NewRecorder()
	svc.HandleTransfers(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 repo apply call, got %d", repo.applyCalls)
	}
	if got := factoryOutboxEventTypes(repo.events); len(got) != 1 || got[0] != events.EventFactoryTransferCreated {
		t.Fatalf("unexpected outbox events: %#v", got)
	}
	var payload map[string]any
	if err := json.Unmarshal(repo.events[0].Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if got, _ := payload["transfer_id"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected transfer_id on TRANSFER_CREATED, got %v", payload)
	}
	assertFactoryCacheDeletedKeys(t, cacheBackend.deletedKeys, factoryTransferListKey("supplier-test"))
	assertFactoryWSMessageContainsType(t, supplierConn.messages, events.EventFactoryTransferCreated)
	assertFactoryWSMessageContainsType(t, factoryConn.messages, events.EventFactoryTransferCreated)
}

func TestHandleTransfers_GET_SeedOffEmpty(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/factory/transfers", nil)
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
	if body.Total != 0 || len(body.Transfers) != 0 {
		t.Fatalf("seed-off GET must not serve demo transfers: %+v", body)
	}
}

func TestHandleAcceptSupplyRequest_RequiresQCPass(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	req := withFactoryRouteParam(
		httptest.NewRequest(http.MethodPost, "/v1/factory/supply-requests/srq_factory_1/accept", strings.NewReader(`{}`)),
		"requestID", "srq_factory_1",
	)
	rr := httptest.NewRecorder()
	svc.HandleAcceptSupplyRequest(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d want 409 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "qc_pass_required" {
		t.Fatalf("error=%v", body["error"])
	}
}

func TestHandleAcceptSupplyRequest_PassAllows(t *testing.T) {
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	svc.qcRepo = &memoryQCRepo{
		requests: map[string]qcRequestMeta{"srq_factory_1": {RequestID: "srq_factory_1", SupplierID: "supplier-test", FactoryID: svc.factoryNodeID, State: "SUBMITTED"}},
		rows:     map[string]SupplyRequestQCResponse{"srq_factory_1": {RequestID: "srq_factory_1", Result: "PASS"}},
	}
	req := withFactoryRouteParam(
		httptest.NewRequest(http.MethodPost, "/v1/factory/supply-requests/srq_factory_1/accept", strings.NewReader(`{}`)),
		"requestID", "srq_factory_1",
	)
	rr := httptest.NewRecorder()
	svc.HandleAcceptSupplyRequest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", rr.Code, rr.Body.String())
	}
}
