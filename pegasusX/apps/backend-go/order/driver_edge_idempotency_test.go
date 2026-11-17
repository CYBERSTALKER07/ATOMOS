package order

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestDriverEdgeMutations_RequireIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	driver := auth.Claims{Subject: "drv-1", Role: auth.RoleDriver}
	cases := []struct {
		name   string
		body   string
		claims auth.Claims
		call   func(http.ResponseWriter, *http.Request)
	}{
		{"reorder", `{"route_id":"r1","order_sequence":["o1"]}`, driver, svc.HandleFleetRouteReorder},
		{"credit-delivery", `{"order_id":"ord-1"}`, driver, svc.HandleCreditDelivery},
		{"split", `{"order_id":"ord-1","cash_minor":1,"card_minor":1}`, driver, svc.HandleSplitPayment},
		{"exception", `{"order_id":"ord-1","items":[]}`, driver, svc.HandleExceptionReport},
		{"shop-closed-resolve", `{"attempt_id":"a1","action":"CANCEL"}`, auth.Claims{Subject: "sup-1", Role: auth.RoleAdmin}, svc.HandleResolveShopClosed},
		{"shop-closed-response", `{"order_id":"ord-1","response":"CANCEL"}`, auth.Claims{Subject: "ret-1", Role: auth.RoleRetailer}, svc.HandleShopClosedResponse},
		{"reconciliation-resolve", `{"order_id":"ord-1","action":"CANCEL"}`, auth.Claims{Subject: "sup-1", Role: auth.RoleAdmin}, svc.HandleResolveReconciliation},
		{"report-damage", `{"order_id":"ord-1"}`, driver, svc.HandleReportDamage},
		{"report-condition", `{"order_id":"ord-1","condition_type":"DAMAGED"}`, driver, svc.HandleReportCondition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/test", strings.NewReader(tc.body))
			req = req.WithContext(auth.WithClaims(req.Context(), tc.claims))
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "idempotency_key_required") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestNegotiationMutations_RequireIdempotencyKey(t *testing.T) {
	t.Setenv("QUANTITY_NEGOTIATION_ENABLED", "true")
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	driver := auth.Claims{Subject: "drv-1", Role: auth.RoleDriver}
	admin := auth.Claims{Subject: "sup-1", Role: auth.RoleAdmin}

	cases := []struct {
		name   string
		body   string
		claims auth.Claims
		call   func(http.ResponseWriter, *http.Request)
	}{
		{"negotiate-propose", `{"order_id":"ord-1","items":[{"sku_id":"s1","original_qty":2,"proposed_qty":1}]}`, driver, svc.HandleProposeNegotiation},
		{"negotiate-resolve", `{"proposal_id":"p1","action":"APPROVE"}`, admin, svc.HandleResolveNegotiation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/test", strings.NewReader(tc.body))
			req = req.WithContext(auth.WithClaims(req.Context(), tc.claims))
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "idempotency_key_required") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
