package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
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
	req.Header.Set("Idempotency-Key", "driver-rescue-request:drv-broken")
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
			req.Header.Set("Idempotency-Key", "driver-rescue-respond:"+tc.name)
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
	req.Header.Set("Idempotency-Key", "driver-rescue-respond:drv-rescue:resc-123:true")
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

func TestHandleRescueRequest_IdempotencyReplay(t *testing.T) {
	body := []byte(`{}`)
	cached := map[string]any{"status": "requested", "rescue_id": "resc-cached-1"}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "driver-rescue:drv-broken:test"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	svc.idem = store

	req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/request", strings.NewReader(string(body)))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-broken"})
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleRescueRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("expected replay body %s, got %s", string(cachedBytes), rr.Body.String())
	}
}

func TestHandleRescueRespond_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"rescue_id":"resc-101","broken_driver_id":"drv-broken","accept":true}`)
	cached := map[string]any{"status": "recorded", "rescue_id": "resc-101", "accepted": true}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "driver-rescue-respond:drv-rescue:test"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	svc.idem = store

	req := httptest.NewRequest(http.MethodPost, "/v1/driver/ops/rescue/respond", strings.NewReader(string(body)))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-rescue"})
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.HandleRescueRespond(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != string(cachedBytes) {
		t.Fatalf("expected replay body %s, got %s", string(cachedBytes), rr.Body.String())
	}
}

func TestRescueMutations_RequireIdempotencyKey(t *testing.T) {
	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	cases := []struct {
		name string
		body string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"request", `{"reason":"breakdown","note":"tire"}`, svc.HandleRescueRequest},
		{"respond", `{"rescue_id":"resc-1","broken_driver_id":"drv-broken","accept":true}`, svc.HandleRescueRespond},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/test", strings.NewReader(tc.body))
			req = withDriverClaims(req, auth.Claims{Subject: "drv-1"})
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "idempotency_key_required") {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}
