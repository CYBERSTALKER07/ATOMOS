package payload

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleVehicleCapacity_GoneUnwired(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	req := httptest.NewRequest(http.MethodGet, "/v1/payload/capacity/v-752069247", nil)
	rr := httptest.NewRecorder()
	svc.HandleVehicleCapacity(rr, req)

	if rr.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "59") {
		t.Fatalf("must not return hardcoded 59 percent: %s", rr.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "capacity_unwired" {
		t.Fatalf("error=%q body=%s", body["error"], rr.Body.String())
	}
}

func TestHandleVehicleCapacity_MethodNotAllowed(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	req := httptest.NewRequest(http.MethodPost, "/v1/payload/capacity/v-1", nil)
	rr := httptest.NewRecorder()
	svc.HandleVehicleCapacity(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}
