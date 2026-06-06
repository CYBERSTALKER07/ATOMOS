package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireReplenishmentInsightsScope_FactoryRolePasses(t *testing.T) {
	t.Parallel()
	var called bool
	handler := RequireReplenishmentInsightsScope(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/replenishment/insights", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject:      "factory-staff-demo",
		Role:         RoleFactory,
		SupplierRole: RoleFactoryAdmin,
		HomeNodeType: HomeNodeFactory,
		HomeNodeID:   "factory-demo-1",
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestRequireReplenishmentInsightsScope_WarehouseRoleRequiresHomeNode(t *testing.T) {
	t.Parallel()
	handler := RequireReplenishmentInsightsScope(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/replenishment/insights", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject: "warehouse-staff",
		Role:    RoleWarehouse,
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
