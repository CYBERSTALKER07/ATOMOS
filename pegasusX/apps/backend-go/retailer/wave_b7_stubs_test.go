package retailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func newB7RetailerService() *Service {
	return NewService(ServiceConfig{
		Repo: &testRetailerRepo{},
		Now:  func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) },
	})
}

// B7 R-P0-3: cancel stubs fail-closed when OrderService unwired.
func TestHandleCancelOrder_Unwired_503(t *testing.T) {
	svc := newB7RetailerService()
	req := httptest.NewRequest(http.MethodPost, "/v1/order/cancel", strings.NewReader(`{"order_id":"o1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "user-1", Role: auth.RoleRetailer, RetailerOrgID: "ret-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleCancelOrder(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "order_service_unwired" {
		t.Fatalf("error=%q", body["error"])
	}
}

func TestHandleRequestCancel_Unwired_503(t *testing.T) {
	svc := newB7RetailerService()
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/request-cancel", strings.NewReader(`{"order_id":"o1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "user-1", Role: auth.RoleRetailer, RetailerOrgID: "ret-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleRequestCancel(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateOrder_Unwired_503(t *testing.T) {
	svc := newB7RetailerService()
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	svc.HandleCreateOrder(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
}
