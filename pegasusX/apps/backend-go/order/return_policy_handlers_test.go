package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func warehousePolicyService() (*Service, *memoryReturnPolicies) {
	store := newMemoryReturnPolicies()
	return &Service{returnPolicies: store}, store
}

func withWarehouseCaller(req *http.Request, claims auth.Claims, pin bool) *http.Request {
	ctx := auth.WithClaims(req.Context(), claims)
	if pin {
		ctx = context.WithValue(ctx, auth.WarehouseOpsKey, &auth.WarehouseOps{
			WarehouseID: claims.HomeNodeID,
			SupplierID:  claims.SupplierID,
			Subject:     claims.Subject,
		})
	}
	return req.WithContext(ctx)
}

func policyErr(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rr.Body.String(), err)
	}
	return body["error"]
}

func TestHandleWarehouseReturnPolicy_RejectsBodySupplier(t *testing.T) {
	svc, store := warehousePolicyService()
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/return-policy",
		strings.NewReader(`{"supplier_id":"sup-other","can_override_retailer_window":false}`))
	req = withWarehouseCaller(req, auth.Claims{
		Subject: "wh-user", Role: auth.RoleWarehouseAdmin, SupplierID: "sup-a",
		HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-1",
	}, true)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseReturnPolicy(rr, req)
	if rr.Code != http.StatusForbidden || policyErr(t, rr) != "supplier_scope_forbidden" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(store.warehouses) != 0 {
		t.Fatalf("wrote %d policies", len(store.warehouses))
	}
}

func TestHandleWarehouseReturnPolicy_RejectsQuerySupplier(t *testing.T) {
	svc, store := warehousePolicyService()
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/return-policy?supplier_id=sup-other", nil)
	req = withWarehouseCaller(req, auth.Claims{
		Subject: "wh-user", Role: auth.RoleWarehouse, SupplierID: "sup-a",
		HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-1",
	}, true)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseReturnPolicy(rr, req)
	if rr.Code != http.StatusForbidden || policyErr(t, rr) != "supplier_scope_forbidden" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(store.warehouses) != 0 {
		t.Fatalf("store mutated")
	}
}

func TestHandleWarehouseReturnPolicy_RejectsWarehouseOverride(t *testing.T) {
	svc, store := warehousePolicyService()
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/return-policy?warehouse_id=wh-other",
		strings.NewReader(`{"supplier_id":"sup-a"}`))
	req = withWarehouseCaller(req, auth.Claims{
		Subject: "wh-user", Role: auth.RoleWarehouseAdmin, SupplierID: "sup-a",
		HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-1",
	}, true)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseReturnPolicy(rr, req)
	if rr.Code != http.StatusForbidden || policyErr(t, rr) != "warehouse_scope_forbidden" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(store.warehouses) != 0 {
		t.Fatalf("wrote %d policies", len(store.warehouses))
	}
}

func TestHandleWarehouseReturnPolicy_PinsTokenSupplier(t *testing.T) {
	svc, store := warehousePolicyService()
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/return-policy?warehouse_id=wh-1",
		strings.NewReader(`{"supplier_id":"sup-a","can_override_retailer_window":true,"retailer_file_window_hours":48}`))
	req = withWarehouseCaller(req, auth.Claims{
		Subject: "wh-user", Role: auth.RoleWarehouseAdmin, SupplierID: "sup-a",
		HomeNodeType: auth.HomeNodeWarehouse, HomeNodeID: "wh-1",
	}, true)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseReturnPolicy(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	saved, ok := store.warehouses[store.key("wh-1", "sup-a")]
	if !ok {
		t.Fatalf("missing saved policy, keys=%v", store.warehouses)
	}
	if saved.SupplierID != "sup-a" || saved.UpdatedByUserID != "wh-user" {
		t.Fatalf("saved %+v", saved)
	}
	if saved.RetailerFileWindowHours == nil || *saved.RetailerFileWindowHours != 48 {
		t.Fatalf("hours %+v", saved.RetailerFileWindowHours)
	}
}

func TestHandleWarehouseReturnPolicy_CEOUsesTokenSupplier(t *testing.T) {
	svc, store := warehousePolicyService()
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/return-policy?warehouse_id=wh-9",
		strings.NewReader(`{}`))
	req = withWarehouseCaller(req, auth.Claims{
		Subject: "ceo", Role: auth.RoleAdmin, SupplierID: "sup-a",
	}, false)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseReturnPolicy(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if _, ok := store.warehouses[store.key("wh-9", "sup-a")]; !ok {
		t.Fatalf("expected sup-a policy, got %#v", store.warehouses)
	}
}

func TestHandleWarehouseReturnPolicy_CEORejectsForeignSupplier(t *testing.T) {
	svc, store := warehousePolicyService()
	req := httptest.NewRequest(http.MethodPut, "/v1/warehouse/return-policy?warehouse_id=wh-9&supplier_id=sup-other",
		strings.NewReader(`{"supplier_id":"sup-other"}`))
	req = withWarehouseCaller(req, auth.Claims{
		Subject: "ceo", Role: auth.RoleAdmin, SupplierID: "sup-a",
	}, false)
	rr := httptest.NewRecorder()
	svc.HandleWarehouseReturnPolicy(rr, req)
	if rr.Code != http.StatusForbidden || policyErr(t, rr) != "supplier_scope_forbidden" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(store.warehouses) != 0 {
		t.Fatalf("wrote %d policies", len(store.warehouses))
	}
}
