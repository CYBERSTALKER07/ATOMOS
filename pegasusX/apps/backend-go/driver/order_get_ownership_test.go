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

// newOrderGetService builds a minimal service with only the orderGet seam wired.
func newOrderGetService(order DriverOrderView, found bool) *Service {
	return NewService(ServiceConfig{
		SupplierID: "supplier-test",
		Currency:   "UZS",
		OrderGet: func(ctx context.Context, orderID string) (DriverOrderView, bool, error) {
			return order, found, nil
		},
	})
}

func orderGetRequest(t *testing.T, svc *Service, orderID string, claims *auth.Claims) *httptest.ResponseRecorder {
	t.Helper()
	// Route is mounted under chi with {orderID}; use a request that carries the param.
	req := httptest.NewRequest(http.MethodGet, "/v1/mobile/driver/orders/"+orderID, nil)
	if claims != nil {
		req = withDriverClaims(req, *claims)
	}
	rr := httptest.NewRecorder()
	mux := chi.NewRouter()
	mux.Get("/v1/mobile/driver/orders/{orderID}", svc.HandleOrderGet)
	mux.ServeHTTP(rr, req)
	return rr
}

func TestHandleOrderGet_DriverOwnsOrder_OK(t *testing.T) {
	order := DriverOrderView{OrderID: "o1", AssignedDriverID: "driver-1"}
	svc := newOrderGetService(order, true)
	claims := &auth.Claims{Subject: "driver-1", Role: auth.RoleDriver}
	rr := orderGetRequest(t, svc, "o1", claims)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestHandleOrderGet_DriverDoesNotOwnOrder_404(t *testing.T) {
	order := DriverOrderView{OrderID: "o1", AssignedDriverID: "driver-2"}
	svc := newOrderGetService(order, true)
	claims := &auth.Claims{Subject: "driver-1", Role: auth.RoleDriver}
	rr := orderGetRequest(t, svc, "o1", claims)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-driver read, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "order_not_found") {
		t.Fatalf("expected order_not_found body, got %s", rr.Body.String())
	}
}

func TestHandleOrderGet_EmptyAssigned_404(t *testing.T) {
	order := DriverOrderView{OrderID: "o1", AssignedDriverID: ""}
	svc := newOrderGetService(order, true)
	claims := &auth.Claims{Subject: "driver-1", Role: auth.RoleDriver}
	rr := orderGetRequest(t, svc, "o1", claims)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when assignment unknown (fail-closed), got %d", rr.Code)
	}
}

func TestHandleOrderGet_AdminBypassesOwnership_OK(t *testing.T) {
	order := DriverOrderView{OrderID: "o1", AssignedDriverID: "driver-2"}
	svc := newOrderGetService(order, true)
	claims := &auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}
	rr := orderGetRequest(t, svc, "o1", claims)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d", rr.Code)
	}
}

func TestHandleOrderGet_NotFound_404(t *testing.T) {
	svc := newOrderGetService(DriverOrderView{}, false)
	claims := &auth.Claims{Subject: "driver-1", Role: auth.RoleDriver}
	rr := orderGetRequest(t, svc, "o-missing", claims)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing order, got %d", rr.Code)
	}
}
