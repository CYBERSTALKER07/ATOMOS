package payout

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestNormalizePayoutMode(t *testing.T) {
	got, ok := normalizePayoutMode("warehouse_local")
	if !ok || got != PayoutModeWarehouseLocal {
		t.Fatalf("got=%s ok=%v", got, ok)
	}
	if _, ok := normalizePayoutMode("LIVE_PSP"); ok {
		t.Fatal("must not accept invented live PSP mode")
	}
}

func TestDefaultPolicyIsHQSupplier(t *testing.T) {
	p := defaultPolicy("sup-1")
	if p.PayoutMode != PayoutModeHQSupplier || p.Source != payoutPolicySourceDefault {
		t.Fatalf("%+v", p)
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), `"status":"ok"`) {
		t.Fatal("must not wrap status:ok")
	}
}

func TestHandlePayoutPolicy_NoService503(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/payout-policy", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "u1",
		SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	h.HandlePayoutPolicy(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandlePayoutPolicy_PatchRequiresReason(t *testing.T) {
	h := &Handlers{Svc: NewService(NewRepository(nil))}
	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/payout-policy", strings.NewReader(`{"payout_mode":"HQ_SUPPLIER"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "u1",
		SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	h.HandlePayoutPolicy(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "reason_required") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}
