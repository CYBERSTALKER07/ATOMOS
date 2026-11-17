package driver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleCreateDriver_HomeNodeEnforcement(t *testing.T) {
	repo := &driverRepoSpy{}
	cacheBackend := &driverCacheBackendSpy{}
	svc := newDriverTestService(repo, cacheBackend)

	// 1. Warehouse Admin with HomeNodeID="wh-1" trying to create driver on "wh-2" -> 403 Forbidden
	req := httptest.NewRequest(http.MethodPost, "/v1/drivers", strings.NewReader(`{
		"name": "Driver One",
		"phone": "+998901112233",
		"home_node_id": "wh-2"
	}`))
	req = withDriverClaims(req, auth.Claims{
		Subject:      "admin-wh",
		Role:         auth.RoleWarehouseAdmin,
		SupplierID:   "supplier-test",
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeID:   "wh-1",
	})
	rr := httptest.NewRecorder()
	svc.HandleCreateDriver(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 forbidden for mismatched home node, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Warehouse Admin with HomeNodeID="wh-1" omitting home_node_id -> defaults to wh-1 and succeeds
	req = httptest.NewRequest(http.MethodPost, "/v1/drivers", strings.NewReader(`{
		"name": "Driver One",
		"phone": "+998901112233"
	}`))
	req = withDriverClaims(req, auth.Claims{
		Subject:      "admin-wh",
		Role:         auth.RoleWarehouseAdmin,
		SupplierID:   "supplier-test",
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeID:   "wh-1",
	})
	rr = httptest.NewRecorder()
	svc.HandleCreateDriver(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", rr.Code, rr.Body.String())
	}

	// 3. Factory Admin with HomeNodeID="fac-1" updating driver with mismatched home_node_id -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPut, "/v1/drivers/drv-1", strings.NewReader(`{
		"name": "Driver Updated",
		"home_node_id": "fac-2"
	}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("driverId", "drv-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = withDriverClaims(req, auth.Claims{
		Subject:      "admin-fac",
		Role:         auth.RoleFactoryAdmin,
		SupplierID:   "supplier-test",
		SupplierRole: auth.RoleFactoryAdmin,
		HomeNodeID:   "fac-1",
	})
	rr = httptest.NewRecorder()
	svc.HandleUpdateDriver(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 forbidden for mismatched home node on update, got %d: %s", rr.Code, rr.Body.String())
	}
}
