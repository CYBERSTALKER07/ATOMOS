package warehouse

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleEmergencyTransfer_MemoryFallback(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "supplier-test"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/transfers/emergency", bytes.NewReader([]byte(`{"total_volume_vu":15,"notes":"ops"}`)))
	req = req.WithContext(context.WithValue(req.Context(), auth.WarehouseOpsKey, &auth.WarehouseOps{
		WarehouseID: "wh-demo-1",
		SupplierID:  "supplier-test",
	}))
	rr := httptest.NewRecorder()
	svc.HandleEmergencyTransfer(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, _ := body["state"].(string); got != "APPROVED" {
		t.Fatalf("expected APPROVED, got %q", got)
	}
}

func TestHandleReceiveTransfer_MemoryDemoRow(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "supplier-test"})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/transfers/ssmr-wh-transfer-receive/receive", nil)
	req = withTransferIDParam(req, "ssmr-wh-transfer-receive")
	req = req.WithContext(context.WithValue(req.Context(), auth.WarehouseOpsKey, &auth.WarehouseOps{
		WarehouseID: "wh-demo-1",
		SupplierID:  "supplier-test",
	}))
	rr := httptest.NewRecorder()
	svc.HandleReceiveTransfer(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func withTransferIDParam(req *http.Request, transferID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", transferID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
