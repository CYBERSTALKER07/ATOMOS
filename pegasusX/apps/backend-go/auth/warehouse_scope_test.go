package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireWarehouseScope_WarehouseAdminRejectsCrossWarehouseOverride(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/dispatch/preview?warehouse_id=wh-other", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject:      "user-wh",
		SupplierID:   "sup-1",
		Role:         RoleAdmin,
		SupplierRole: RoleWarehouseAdmin,
		HomeNodeType: HomeNodeWarehouse,
		HomeNodeID:   "wh-node",
	}))

	nextCalled := false
	handler := requireWarehouseScope(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	if nextCalled {
		t.Fatal("expected handler blocked")
	}
}

func TestRequireWarehouseScope_FactoryAdminAllowsLinkedWarehouseQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/dispatch/preview?warehouse_id=wh-2", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject:      "user-factory",
		SupplierID:   "sup-1",
		Role:         RoleAdmin,
		SupplierRole: RoleFactoryAdmin,
		HomeNodeType: HomeNodeFactory,
		HomeNodeID:   "fac-1",
	}))

	var seen *WarehouseScope
	handler := requireWarehouseScope(func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		return map[string]struct{}{"wh-1": {}, "wh-2": {}}, nil
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetWarehouseScope(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if seen == nil || seen.WarehouseID != "wh-2" {
		t.Fatalf("expected wh-2 scope, got %+v", seen)
	}
}

func TestRequireWarehouseOpsScope_GlobalSupplierAdminPassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/dispatch/preview", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject:    "ceo",
		SupplierID: "sup-1",
		Role:       RoleAdmin,
	}))

	nextCalled := false
	handler := RequireWarehouseOpsScope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !nextCalled {
		t.Fatal("expected global admin through")
	}
}

func TestRequireWarehouseOpsScope_RejectsQueryOverride(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/orders?warehouse_id=wh-other", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject:      "wh-staff",
		SupplierID:   "sup-1",
		Role:         RoleWarehouse,
		HomeNodeType: HomeNodeWarehouse,
		HomeNodeID:   "wh-demo-1",
	}))

	nextCalled := false
	handler := RequireWarehouseOpsScope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	if nextCalled {
		t.Fatal("expected handler blocked")
	}
}
