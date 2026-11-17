package warehouseroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

func warehouseRouter(svc *warehouse.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("warehouse-test-secret"))
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "warehouse-test-secret"})
	return r
}

func issueWarehouseToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:      "warehouse-user",
		Role:         role,
		SupplierID:   "sup-1",
		HomeNodeType: "WAREHOUSE",
		HomeNodeID:   "wh-1",
	}, auth.IssueOptions{Secret: "warehouse-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestWarehouseRoutes_NilService(t *testing.T) {
	r := warehouseRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouses", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestWarehouseRoutes_Unauthenticated(t *testing.T) {
	svc := &warehouse.Service{}
	r := warehouseRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/warehouses"},
		{http.MethodGet, "/v1/warehouse/ops/dashboard"},
		{http.MethodGet, "/v1/warehouse/ops/settings"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestWarehouseRoutes_RoleGuard(t *testing.T) {
	svc := &warehouse.Service{}
	r := warehouseRouter(svc)

	retailerToken := issueWarehouseToken(t, auth.RoleRetailer)

	// Retailer cannot access warehouse routes
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouses", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer accessing warehouse routes returned %d, want 403 Forbidden", rr.Code)
	}
}

func TestWarehouseSetup_UnauthorizedWithoutToken(t *testing.T) {
	svc := &warehouse.Service{}
	r := warehouseRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/setup", strings.NewReader(`{"address":"Warehouse 1","lat":41.3,"lng":69.2}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated warehouse setup, got %d", rr.Code)
	}
}

func TestWarehouseSetup_ForbiddenForRetailerAndDriver(t *testing.T) {
	svc := &warehouse.Service{}
	r := warehouseRouter(svc)

	for _, role := range []auth.Role{auth.RoleRetailer, auth.RoleDriver} {
		tok := issueWarehouseToken(t, role)
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/setup", strings.NewReader(`{"address":"Warehouse 1","lat":41.3,"lng":69.2}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for role %s attempting warehouse setup, got %d", role, rr.Code)
		}
	}
}

func TestWarehouseSetup_AllowedForAuthorizedRoles(t *testing.T) {
	svc := &warehouse.Service{}
	r := warehouseRouter(svc)

	for _, role := range []auth.Role{auth.RoleWarehouse, auth.RoleWarehouseAdmin, auth.RoleAdmin} {
		tok := issueWarehouseToken(t, role)
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/setup", strings.NewReader(`{"address":"Warehouse 1","lat":41.3,"lng":69.2}`))
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// With an empty service (no Spanner), it should reach the handler and return 503 (Spanner not configured),
		// proving the role guard allowed the request through (not 401 Unauthorized or 403 Forbidden).
		if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
			t.Fatalf("role %s blocked by auth guard with status %d", role, rr.Code)
		}
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected handler 503 (Spanner not configured), got %d", rr.Code)
		}
	}
}
