package factory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlanningFlagsDefaultOff(t *testing.T) {
	t.Setenv(FlagFactoryPlanning, "")
	t.Setenv(FlagFactoryBatcher, "")
	if PlanningEnabled() || BatcherEnabled() {
		t.Fatal("P5 flags must default off")
	}
}

func TestHandleDispatch_BatcherFlagDoesNot503WithoutSpanner(t *testing.T) {
	t.Setenv(FlagFactoryBatcher, "true")
	repo := &factoryRepoSpy{}
	svc := newFactoryTestService(repo, &factoryCacheBackendSpy{})
	repo.svc = svc
	factoryDisablePortalSeed(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/factory/dispatch", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	svc.HandleDispatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("batcher flag is not the dispatch engine gate; empty in-memory must 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}
