package payload

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestHandleFleetReassign_PersistsOrderAssignment(t *testing.T) {
	repo := &payloadRepoSpy{}
	svc := newPayloadTestService(repo, &payloadCacheBackendSpy{})
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/fleet/reassign", strings.NewReader(`{"order_ids":["ord_payload_1"],"new_route_id":"route_veh_payload_2"}`))
	rr := httptest.NewRecorder()
	svc.HandleFleetReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.applyCalls != 1 {
		t.Fatalf("expected 1 RunTx, got %d", repo.applyCalls)
	}
	if len(repo.assignments) != 1 {
		t.Fatalf("expected 1 Orders assignment persist, got %#v", repo.assignments)
	}
	got := repo.assignments[0]
	if got.orderID != "ord_payload_1" || got.routeID != "route_veh_payload_2" {
		t.Fatalf("unexpected assignment %#v", got)
	}
	if got.driverID != "drv_payload_2" {
		t.Fatalf("expected driver from target manifest, got %q", got.driverID)
	}
	if types := payloadOutboxEventTypes(repo.events); len(types) != 1 || types[0] != events.EventOrderReassigned {
		t.Fatalf("unexpected outbox %#v", types)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if n, _ := body["reassigned"].(float64); n != 1 {
		t.Fatalf("expected reassigned=1, got %v", body)
	}
}

func TestHandleFleetReassign_NoPersistOnConflict(t *testing.T) {
	repo := &payloadRepoSpy{}
	svc := newPayloadTestService(repo, &payloadCacheBackendSpy{})
	repo.svc = svc

	req := httptest.NewRequest(http.MethodPost, "/v1/fleet/reassign", strings.NewReader(`{"order_ids":["ord_missing"],"new_route_id":"route_veh_payload_2"}`))
	rr := httptest.NewRecorder()
	svc.HandleFleetReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(repo.assignments) != 0 {
		t.Fatalf("expected no persist on missing order, got %#v", repo.assignments)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no outbox when nothing reassigned, got %#v", payloadOutboxEventTypes(repo.events))
	}
}
