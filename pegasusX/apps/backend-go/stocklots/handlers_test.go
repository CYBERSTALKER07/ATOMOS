package stocklots

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func recallRequest(method, target, body string, claims auth.Claims) *http.Request {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	ctx := auth.WithClaims(req.Context(), claims)
	if claims.Role == auth.RoleWarehouse || claims.Role == auth.RoleWarehouseAdmin {
		ctx = context.WithValue(ctx, auth.WarehouseOpsKey, &auth.WarehouseOps{
			WarehouseID: claims.HomeNodeID,
			SupplierID:  claims.SupplierID,
			Subject:     claims.Subject,
		})
	}
	return req.WithContext(ctx)
}

func decodeErr(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rr.Body.String(), err)
	}
	return body["error"]
}

func TestHandleRecalls_RejectsBodySupplierMismatch(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodPost, "/v1/supplier/recalls",
		`{"supplier_id":"sup-other","product_id":"p1","recall_reason":"contamination"}`,
		auth.Claims{Subject: "ceo", Role: auth.RoleAdmin, SupplierID: "sup-a"},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_scope_forbidden" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_RejectsQuerySupplierMismatch(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodGet, "/v1/warehouse/ops/recalls?supplier_id=sup-other", "",
		auth.Claims{Subject: "wh", Role: auth.RoleWarehouse, SupplierID: "sup-a", HomeNodeID: "wh-1", HomeNodeType: auth.HomeNodeWarehouse},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_scope_forbidden" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_BodyDoesNotCreateSupplier(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/recalls",
		strings.NewReader(`{"supplier_id":"sup-other","product_id":"p1","recall_reason":"x"}`))
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_id_required" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_SeedDoesNotAcceptForeignBody(t *testing.T) {
	h := &Handler{SupplierID: "seed-supplier"}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/recalls",
		strings.NewReader(`{"supplier_id":"sup-other","product_id":"p1","recall_reason":"x"}`))
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_scope_forbidden" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_MatchingSupplierReachesStore(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodPost, "/v1/supplier/recalls",
		`{"supplier_id":"sup-a","product_id":"p1","recall_reason":"contamination"}`,
		auth.Claims{Subject: "ceo", Role: auth.RoleAdmin, SupplierID: "sup-a"},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleConfirmPickTask_RejectsNegativeQuantity(t *testing.T) {
	SetPickWavesEnabled(true)
	t.Cleanup(func() { SetPickWavesEnabled(false) })

	h := &Handler{Spanner: &spanner.Client{}}
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/pick-waves/w1/tasks/t1/confirm?warehouse_id=wh1",
		strings.NewReader(`{"quantity_picked":-1,"picker_id":"p1"}`))
	rr := httptest.NewRecorder()
	h.HandleConfirmPickTask(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s want 400", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "quantity_picked_negative" {
		t.Fatalf("error=%q want quantity_picked_negative", got)
	}
}

func TestHandleRecalls_BlocksAuditorWrite(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodPost, "/v1/supplier/recalls",
		`{"supplier_id":"sup-a","product_id":"p1","recall_reason":"contamination"}`,
		auth.Claims{Subject: "aud-1", Role: auth.RoleAdmin, SupplierRole: auth.SupplierRoleAuditor, SupplierID: "sup-a"},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "auditor_read_only" {
		t.Fatalf("error=%q want auditor_read_only", got)
	}
}

func TestRequireAdjustmentApprovalAccess_SoDAndRoles(t *testing.T) {
	h := &Handler{SupplierID: "sup-a"}

	makeReq := func(claims auth.Claims) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/inventory-adjustments/adj-1/approve?warehouse_id=wh-1", nil)
		ctx := auth.WithClaims(req.Context(), claims)
		ctx = context.WithValue(ctx, auth.WarehouseOpsKey, &auth.WarehouseOps{
			SupplierID:  "sup-a",
			WarehouseID: "wh-1",
		})
		return req.WithContext(ctx)
	}

	t.Run("auditor blocked", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := h.requireAdjustmentApprovalAccess(rr, makeReq(auth.Claims{
			Subject: "aud-1", Role: auth.RoleAdmin, SupplierRole: auth.SupplierRoleAuditor, SupplierID: "sup-a",
		}), "sup-a", "adj-1", "counter-1", "/v1/warehouse/ops/inventory-adjustments/adj-1/approve")
		if ok || rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 got ok=%v code=%d body=%s", ok, rr.Code, rr.Body.String())
		}
	})

	t.Run("agent approval blocked", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := h.requireAdjustmentApprovalAccess(rr, makeReq(auth.Claims{
			Subject: "agent-1", Role: auth.RoleAdmin, ActorType: auth.ActorTypeAgent, SupplierRole: auth.SupplierRoleAgent, SupplierID: "sup-a",
		}), "sup-a", "adj-1", "counter-1", "/v1/warehouse/ops/inventory-adjustments/adj-1/approve")
		if ok || rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 got ok=%v code=%d body=%s", ok, rr.Code, rr.Body.String())
		}
	})

	t.Run("dispatcher lacks stock.adjust.approve", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := h.requireAdjustmentApprovalAccess(rr, makeReq(auth.Claims{
			Subject: "disp-1", Role: auth.RoleWarehouse, SupplierRole: auth.SupplierRoleDispatcher, SupplierID: "sup-a",
		}), "sup-a", "adj-1", "counter-1", "/v1/warehouse/ops/inventory-adjustments/adj-1/approve")
		if ok || rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 got ok=%v code=%d body=%s", ok, rr.Code, rr.Body.String())
		}
	})

	t.Run("self approval blocked with 403 sod_violation", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := h.requireAdjustmentApprovalAccess(rr, makeReq(auth.Claims{
			Subject: "whm-1", Role: auth.RoleWarehouseAdmin, SupplierRole: auth.SupplierRoleWarehouseManager, SupplierID: "sup-a",
		}), "sup-a", "adj-1", "whm-1", "/v1/warehouse/ops/inventory-adjustments/adj-1/approve")
		if ok || rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 got ok=%v code=%d body=%s", ok, rr.Code, rr.Body.String())
		}
		if got := decodeErr(t, rr); got != "sod_violation" {
			t.Fatalf("error=%q want sod_violation", got)
		}
	})

	t.Run("distinct warehouse manager allowed", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := h.requireAdjustmentApprovalAccess(rr, makeReq(auth.Claims{
			Subject: "whm-checker", Role: auth.RoleWarehouseAdmin, SupplierRole: auth.SupplierRoleWarehouseManager, SupplierID: "sup-a",
		}), "sup-a", "adj-1", "counter-maker", "/v1/warehouse/ops/inventory-adjustments/adj-1/approve")
		if !ok || rr.Code != http.StatusOK {
			t.Fatalf("expected allowed got ok=%v code=%d body=%s", ok, rr.Code, rr.Body.String())
		}
	})
}
