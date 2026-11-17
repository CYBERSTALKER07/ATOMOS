package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func TestAdminAndRetailerMutations_RequireIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Cache: cache.New(cache.NewInMemoryBackend(), nil),
	})
	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}
	retailer := auth.Claims{Subject: "ret-1", Role: auth.RoleRetailer}
	driver := auth.Claims{Subject: "drv-1", Role: auth.RoleDriver}

	cases := []struct {
		name   string
		method string
		body   string
		claims auth.Claims
		param  string
		call   func(http.ResponseWriter, *http.Request)
	}{
		{"retailer-cancel", http.MethodPost, `{"order_id":"ord-1"}`, retailer, "", svc.HandleRetailerCancel},
		{"assign", http.MethodPost, `{"driver_id":"drv-1"}`, admin, "orderID", svc.HandleAssignOrder},
		{"status", http.MethodPatch, `{"status":"CANCELLED"}`, admin, "orderID", svc.HandleUpdateStatus},
		{"issue-bypass", http.MethodPost, `{"order_id":"ord-1"}`, admin, "", svc.HandleIssuePaymentBypass},
		{"confirm-bypass", http.MethodPost, `{"order_id":"ord-1","bypass_token":"123456"}`, driver, "", svc.HandleConfirmPaymentBypass},
		{"approve-early-complete", http.MethodPost, `{"driver_id":"drv-1","action":"CANCEL"}`, admin, "", svc.HandleApproveEarlyComplete},
		{"request-early-complete", http.MethodPost, `{"route_id":"rt-1","reason":"fatigue"}`, driver, "", svc.HandleRequestEarlyComplete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/v1/test", strings.NewReader(tc.body))
			ctx := auth.WithClaims(req.Context(), tc.claims)
			if tc.param != "" {
				routeCtx := chi.NewRouteContext()
				routeCtx.URLParams.Add(tc.param, "ord-1")
				ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "idempotency_key_required") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
