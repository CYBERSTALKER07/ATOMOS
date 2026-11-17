package order

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestWarehouseOrderMutations_RequireIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	cases := []struct {
		name string
		path string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"edit-preorder", "/v1/warehouse/ops/preorders/ord-1/edit", svc.HandleWarehouseEditPreorder},
		{"delay", "/v1/warehouse/ops/orders/ord-1/delay", svc.HandleWarehouseMarkDelayed},
		{"reject", "/v1/warehouse/ops/orders/ord-1/reject", svc.HandleWarehouseRejectOrder},
		{"overflow", "/v1/warehouse/ops/orders/ord-1/overflow", svc.HandleWarehousePayloadOverflow},
		{"propose", "/v1/warehouse/ops/orders/ord-1/propose-delivery", svc.HandleWarehouseProposeDelivery},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"reason":"late"}`))
			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("id", "ord-1")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "idempotency_key_required") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
