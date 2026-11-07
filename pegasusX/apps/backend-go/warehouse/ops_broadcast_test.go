package warehouse

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/go-chi/chi/v5"
)

func warehouseTestClaims(warehouseID string) auth.Claims {
	return auth.Claims{
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
	}
}

func TestHandleWarehouseBroadcastTemplates_ListBuiltin(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/broadcast/templates", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), warehouseTestClaims("wh-1")))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseBroadcastTemplates(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var resp broadcastTemplatesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Templates) < len(builtinWarehouseBroadcastTemplates) {
		t.Fatalf("expected builtin templates, got %d", len(resp.Templates))
	}
	if resp.Templates[0].Scope != "warehouse" || resp.Templates[0].Source != "builtin" {
		t.Fatalf("unexpected first template: %+v", resp.Templates[0])
	}
}

func TestHandleWarehouseBroadcastTemplates_CreateCustom(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	body := []byte(`{"title":"Bay 2 closed","body":"Use bay 3 for check-in.","default_role":"DRIVER","category":"custom"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/broadcast/templates", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:      "ops-1",
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseBroadcastTemplates(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var wire BroadcastTemplateWire
	if err := json.Unmarshal(rr.Body.Bytes(), &wire); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if wire.Source != "custom" || wire.WarehouseID != "wh-1" {
		t.Fatalf("wire = %+v", wire)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/broadcast/templates", nil)
	listReq = listReq.WithContext(auth.WithClaims(listReq.Context(), warehouseTestClaims("wh-1")))
	listRR := httptest.NewRecorder()
	svc.HandleWarehouseBroadcastTemplates(listRR, listReq)
	var list broadcastTemplatesResponse
	_ = json.Unmarshal(listRR.Body.Bytes(), &list)
	if len(list.Templates) <= len(builtinWarehouseBroadcastTemplates) {
		t.Fatalf("expected custom template in list")
	}
}

func TestHandleWarehouseBroadcast_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"title":"Delay","body":"Gate slow","role":"DRIVER"}`)
	store := idempotency.NewInMemoryStore()
	key := "warehouse-broadcast:wh-1:test"
	cached := map[string]string{"status": "broadcast_sent", "warehouse_id": "wh-1"}
	cachedBytes, _ := json.Marshal(cached)
	_ = store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour)

	svc := NewService(ServiceConfig{SupplierID: "sup-1", Idem: store})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/broadcast", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), warehouseTestClaims("wh-1")))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseBroadcast(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func TestHandleWarehouseBroadcast_NoSpanner503(t *testing.T) {
	body := []byte(`{"title":"Delay","body":"Gate slow","role":"DRIVER"}`)
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/broadcast", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), warehouseTestClaims("wh-1")))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseBroadcast(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d want 503 body = %s", rr.Code, rr.Body.String())
	}
}

func TestWarehouseBroadcastOutboxEventType(t *testing.T) {
	if events.EventWarehouseBroadcast != "WAREHOUSE_BROADCAST" {
		t.Fatalf("event type %q", events.EventWarehouseBroadcast)
	}
}

func TestHandleWarehouseBroadcastTemplateDelete_BuiltinRejected(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodDelete, "/v1/warehouse/ops/broadcast/templates/wh_yard_hold", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "wh_yard_hold")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), warehouseTestClaims("wh-1")))
	rr := httptest.NewRecorder()
	svc.HandleWarehouseBroadcastTemplateDelete(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rr.Code)
	}
}
