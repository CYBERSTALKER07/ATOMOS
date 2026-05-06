package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireWarehouseScope_NodeAdminRejectsCrossWarehouseOverride(t *testing.T) {
	req := requestWithClaims(&PegasusClaims{
		UserID:       "user-node",
		SupplierID:   "sup-1",
		Role:         "SUPPLIER",
		SupplierRole: "NODE_ADMIN",
		WarehouseID:  "wh-node",
	}, "wh-other")

	nextCalled := false
	handler := requireWarehouseScope(nil, func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if nextCalled {
		t.Fatalf("expected next handler to be blocked")
	}
}

func TestRequireWarehouseScope_FactoryAdminAllowsLinkedWarehouseQuery(t *testing.T) {
	req := requestWithClaims(&PegasusClaims{
		UserID:       "user-factory",
		SupplierID:   "sup-1",
		Role:         "SUPPLIER",
		SupplierRole: "FACTORY_ADMIN",
		FactoryID:    "fac-1",
	}, "wh-2")

	var seenScope *WarehouseScope
	handler := requireWarehouseScope(func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		if supplierID != "sup-1" {
			t.Fatalf("unexpected supplierID: %s", supplierID)
		}
		if factoryID != "fac-1" {
			t.Fatalf("unexpected factoryID: %s", factoryID)
		}
		return map[string]struct{}{"wh-1": {}, "wh-2": {}}, nil
	}, func(w http.ResponseWriter, r *http.Request) {
		seenScope = GetWarehouseScope(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if seenScope == nil {
		t.Fatalf("expected warehouse scope in context")
	}
	if seenScope.WarehouseID != "wh-2" {
		t.Fatalf("expected warehouse scope wh-2, got %q", seenScope.WarehouseID)
	}
}

func TestRequireWarehouseScope_FactoryAdminRejectsWarehouseOutsideScope(t *testing.T) {
	req := requestWithClaims(&PegasusClaims{
		UserID:       "user-factory",
		SupplierID:   "sup-1",
		Role:         "SUPPLIER",
		SupplierRole: "FACTORY_ADMIN",
		FactoryID:    "fac-1",
	}, "wh-outside")

	nextCalled := false
	handler := requireWarehouseScope(func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		return map[string]struct{}{"wh-1": {}}, nil
	}, func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if nextCalled {
		t.Fatalf("expected next handler to be blocked")
	}
}

func TestRequireWarehouseScope_FactoryAdminAutoPinsSingleLinkedWarehouse(t *testing.T) {
	req := requestWithClaims(&PegasusClaims{
		UserID:       "user-factory",
		SupplierID:   "sup-1",
		Role:         "SUPPLIER",
		SupplierRole: "FACTORY_ADMIN",
		FactoryID:    "fac-1",
	}, "")

	var seenScope *WarehouseScope
	handler := requireWarehouseScope(func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		return map[string]struct{}{"wh-only": {}}, nil
	}, func(w http.ResponseWriter, r *http.Request) {
		seenScope = GetWarehouseScope(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if seenScope == nil || seenScope.WarehouseID != "wh-only" {
		t.Fatalf("expected auto-pinned warehouse scope wh-only, got %+v", seenScope)
	}
}

func TestRequireWarehouseScope_FactoryAdminRequiresWarehouseSelectionWhenMultipleLinked(t *testing.T) {
	req := requestWithClaims(&PegasusClaims{
		UserID:       "user-factory",
		SupplierID:   "sup-1",
		Role:         "SUPPLIER",
		SupplierRole: "FACTORY_ADMIN",
		FactoryID:    "fac-1",
	}, "")

	nextCalled := false
	handler := requireWarehouseScope(func(ctx context.Context, supplierID, factoryID string) (map[string]struct{}, error) {
		return map[string]struct{}{"wh-1": {}, "wh-2": {}}, nil
	}, func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if nextCalled {
		t.Fatalf("expected next handler to be blocked")
	}
}

func requestWithClaims(claims *PegasusClaims, warehouseID string) *http.Request {
	path := "/v1/test"
	if warehouseID != "" {
		path += "?warehouse_id=" + warehouseID
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	ctx := context.WithValue(req.Context(), ClaimsContextKey, claims)
	return req.WithContext(ctx)
}
