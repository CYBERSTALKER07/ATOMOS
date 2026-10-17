package orderroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func orderRouter(svc *order.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("order-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueOrderToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "order-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "order-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestOrderRoutes_NilService(t *testing.T) {
	r := orderRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/platform/receipts/rcpt-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestOrderRoutes_Unauthenticated(t *testing.T) {
	svc := &order.Service{}
	r := orderRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/order/create"},
		{http.MethodGet, "/v1/order/currencies"},
		{http.MethodPost, "/v1/delivery/arrive"},
		{http.MethodPost, "/v1/supplier/shop-closed/resolve"},
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

func TestOrderRoutes_RoleGuard(t *testing.T) {
	svc := &order.Service{}
	r := orderRouter(svc)

	driverToken := issueOrderToken(t, auth.RoleDriver)

	// Driver cannot create orders (Retailer required)
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver creating order returned %d, want 403 Forbidden", rr.Code)
	}

	retailerToken := issueOrderToken(t, auth.RoleRetailer)

	// Retailer cannot arrive at delivery (Driver required)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/delivery/arrive", strings.NewReader(`{}`))
	req2.Header.Set("Authorization", "Bearer "+retailerToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusForbidden {
		t.Fatalf("retailer arriving delivery returned %d, want 403 Forbidden", rr2.Code)
	}
}

func TestOrderRoutes_HandlerExecutionAndValidation(t *testing.T) {
	svc := &order.Service{}
	r := orderRouter(svc)
	retailerToken := issueOrderToken(t, auth.RoleRetailer)

	// GET /v1/order/currencies executes handler and returns 200 OK with supported currencies
	req := httptest.NewRequest(http.MethodGet, "/v1/order/currencies", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /v1/order/currencies returned %d, want 200 OK", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "UZS") {
		t.Fatalf("GET /v1/order/currencies body missing UZS: %s", rr.Body.String())
	}

	// POST /v1/order/create with malformed JSON returns 400 Bad Request from handler
	req2 := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{bad-json`))
	req2.Header.Set("Authorization", "Bearer "+retailerToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/order/create with malformed JSON returned %d, want 400", rr2.Code)
	}
}
