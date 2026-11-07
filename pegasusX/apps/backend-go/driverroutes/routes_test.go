package driverroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func driverRouter(svc *driver.Service, orderSvc *order.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("driver-test-secret"))
	RegisterRoutes(r, Deps{Service: svc, OrderService: orderSvc})
	return r
}

func issueDriverToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "driver-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "driver-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestDriverRoutes_NilService(t *testing.T) {
	r := driverRouter(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/profile", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestDriverRoutes_NilOrderServicePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when OrderService is nil")
		}
	}()

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: &driver.Service{}})
}

func TestDriverRoutes_Unauthenticated(t *testing.T) {
	svc := &driver.Service{}
	orderSvc := &order.Service{}
	r := driverRouter(svc, orderSvc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/driver/profile"},
		{http.MethodGet, "/v1/driver/history"},
		{http.MethodGet, "/v1/fleet/orders"},
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

func TestDriverRoutes_RoleGuard(t *testing.T) {
	svc := &driver.Service{}
	orderSvc := &order.Service{}
	r := driverRouter(svc, orderSvc)

	retailerToken := issueDriverToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/profile", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}
