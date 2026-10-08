package driver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestHandleRescueRequest_MethodNotAllowed(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/ops/rescue/request", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-broken"})
	rr := httptest.NewRecorder()

	svc.HandleRescueRequest(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRescueRequest_Unauthorized(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/request", nil)
	rr := httptest.NewRecorder()

	svc.HandleRescueRequest(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRescueRequest_SpannerNotConfigured(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/request", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-broken"})
	rr := httptest.NewRecorder()

	svc.HandleRescueRequest(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if resp["error"] != "spanner_not_configured" {
		t.Fatalf("expected error spanner_not_configured, got %q", resp["error"])
	}
}

func TestHandleRescueRespond_MethodNotAllowed(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/ops/rescue/respond", nil)
	req = withDriverClaims(req, auth.Claims{Subject: "drv-rescue"})
	rr := httptest.NewRecorder()

	svc.HandleRescueRespond(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRescueRespond_Unauthorized(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/respond", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	svc.HandleRescueRespond(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRescueRespond_InvalidPayload(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	testCases := []struct {
		name string
		body string
	}{
		{"empty_json", `{}`},
		{"missing_rescue_id", `{"broken_driver_id":"drv-broken","accept":true}`},
		{"missing_broken_driver", `{"rescue_id":"resc-123","accept":true}`},
		{"malformed_json", `{"rescue_id":`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/respond", strings.NewReader(tc.body))
			req = withDriverClaims(req, auth.Claims{Subject: "drv-rescue"})
			rr := httptest.NewRecorder()

			svc.HandleRescueRespond(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d body=%s", tc.name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleRescueRespond_SpannerNotConfigured(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	payload := `{"rescue_id":"resc-123","broken_driver_id":"drv-broken","accept":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/respond", strings.NewReader(payload))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-rescue"})
	rr := httptest.NewRecorder()

	svc.HandleRescueRespond(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if resp["error"] != "spanner_not_configured" {
		t.Fatalf("expected error spanner_not_configured, got %q", resp["error"])
	}
}

func TestRescueEvent_SchemaAndTypeContracts(t *testing.T) {
	ev := events.RescueEvent{
		RescueID:       "resc-99",
		BrokenDriverID: "drv-1",
		RescueDriverID: "drv-2",
		Status:         "ACCEPTED",
		WarehouseID:    "wh-1",
		SupplierID:     "sup-1",
	}
	ev.Type = "RESCUE_ACCEPTED"

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if m["type"] != "RESCUE_ACCEPTED" {
		t.Fatalf("expected type RESCUE_ACCEPTED, got %v", m["type"])
	}
	if m["rescue_id"] != "resc-99" {
		t.Fatalf("expected rescue_id resc-99, got %v", m["rescue_id"])
	}
	if m["broken_driver_id"] != "drv-1" {
		t.Fatalf("expected broken_driver_id drv-1, got %v", m["broken_driver_id"])
	}
	if m["rescue_driver_id"] != "drv-2" {
		t.Fatalf("expected rescue_driver_id drv-2, got %v", m["rescue_driver_id"])
	}

	orderEv := events.OrderEvent{
		BaseEvent: events.BaseEvent{
			Type: events.EventOrderReassigned,
		},
		OrderID:      "ord-42",
		ToDriverID:   "drv-2",
		FromDriverID: "drv-1",
		LicensePlate: "01A777AA",
	}

	orderData, err := json.Marshal(orderEv)
	if err != nil {
		t.Fatalf("order marshal failed: %v", err)
	}
	var orderM map[string]any
	if err := json.Unmarshal(orderData, &orderM); err != nil {
		t.Fatalf("order unmarshal failed: %v", err)
	}
	if orderM["type"] != events.EventOrderReassigned {
		t.Fatalf("expected %s, got %v", events.EventOrderReassigned, orderM["type"])
	}
	if orderM["to_driver_id"] != "drv-2" || orderM["from_driver_id"] != "drv-1" {
		t.Fatalf("driver id mapping error: %v", orderM)
	}
}
