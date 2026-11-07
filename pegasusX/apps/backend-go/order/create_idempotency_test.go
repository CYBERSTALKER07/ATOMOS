package order

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestHandleCreate_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{"line_items":[]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1",
		Role:    auth.RoleRetailer,
	}))
	rec := httptest.NewRecorder()
	svc.HandleCreate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "idempotency_key_required" {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestCustodyMutations_RequireIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	cases := []struct {
		name string
		path string
		body string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"arrive", "/v1/delivery/arrive", `{"order_id":"ord-1"}`, svc.HandleMarkArrived},
		{"deliver", "/v1/order/deliver", `{"order_id":"ord-1"}`, svc.HandleSubmitDelivery},
		{"offload", "/v1/order/confirm-offload", `{"order_id":"ord-1"}`, svc.HandleConfirmOffload},
		{"complete", "/v1/order/complete", `{"order_id":"ord-1"}`, svc.HandleCompleteOrder},
		{"shop-closed", "/v1/delivery/shop-closed", `{"order_id":"ord-1"}`, svc.HandleReportShopClosed},
		{"partial-offload", "/v1/delivery/partial-offload", `{"order_id":"ord-1"}`, svc.HandlePartialOffload},
		{"sync-batch", "/v1/sync/batch", `{"deliveries":[]}`, svc.HandleSyncBatch},
		{"credit-leave", "/v1/driver/orders/ord-1/credit-leave", `{}`, svc.HandleCreditLeave},
		{"bypass-offload", "/v1/delivery/bypass-offload", `{"order_id":"ord-1"}`, svc.HandleBypassOffload},
		{"proximity", "/v1/delivery/proximity-unlock", `{"order_id":"ord-1"}`, svc.HandleProximityUnlock},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
				Subject: "drv-1",
				Role:    auth.RoleDriver,
			}))
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "idempotency_key_required") {
				t.Fatalf("body=%s", rec.Body.String())
			}
		})
	}
}

func TestHandleCollectCash_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/collect-cash", strings.NewReader(`{"order_id":"ord-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	}))
	rec := httptest.NewRecorder()
	svc.HandleCollectCash(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}
