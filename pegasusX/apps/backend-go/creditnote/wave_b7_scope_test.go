package creditnote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// B7 WH-P0-3 unit tests for resolveReverseWarehouseID (no Spanner).

func TestResolveReverseWarehouseID_PinsHomeNode(t *testing.T) {
	claims := auth.Claims{
		Subject:      "wh-user",
		Role:         auth.RoleWarehouse,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-home",
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	wh, errCode := resolveReverseWarehouseID(req, claims, "wh-other")
	if errCode != "warehouse_scope_violation" || wh != "" {
		t.Fatalf("mismatch body: wh=%q err=%q", wh, errCode)
	}

	wh, errCode = resolveReverseWarehouseID(req, claims, "wh-home")
	if errCode != "" || wh != "wh-home" {
		t.Fatalf("matching body: wh=%q err=%q", wh, errCode)
	}

	wh, errCode = resolveReverseWarehouseID(req, claims, "")
	if errCode != "" || wh != "wh-home" {
		t.Fatalf("empty body uses home: wh=%q err=%q", wh, errCode)
	}
}

func TestResolveReverseWarehouseID_MissingHome(t *testing.T) {
	claims := auth.Claims{Subject: "wh-user", Role: auth.RoleWarehouse}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	wh, errCode := resolveReverseWarehouseID(req, claims, "wh-x")
	if errCode != "warehouse_scope_missing" || wh != "" {
		t.Fatalf("staff without home: wh=%q err=%q", wh, errCode)
	}
}

func TestHandleReceiveReverse_BodyMismatch_403(t *testing.T) {
	h := &Handlers{Svc: &Service{}}
	claims := auth.Claims{
		Subject:      "u1",
		Role:         auth.RoleWarehouse,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-home",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/reverse-logistics/t1/receive",
		strings.NewReader(`{"warehouse_id":"wh-evil","received_qty":{"sku":1}}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", "t1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(auth.WithClaims(ctx, claims))

	rr := httptest.NewRecorder()
	h.HandleReceiveReverse(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "warehouse_scope_violation" {
		t.Fatalf("error=%q", body["error"])
	}
}

func TestReverseWarehouseStaff_IncludesAdmin(t *testing.T) {
	if !reverseWarehouseStaff(auth.Claims{Role: auth.RoleWarehouseAdmin}) {
		t.Fatal("WAREHOUSE_ADMIN should be allowed")
	}
	if !reverseWarehouseStaff(auth.Claims{Role: auth.RoleWarehouse}) {
		t.Fatal("WAREHOUSE should be allowed")
	}
	if reverseWarehouseStaff(auth.Claims{Role: auth.RoleDriver}) {
		t.Fatal("DRIVER must be denied")
	}
}
