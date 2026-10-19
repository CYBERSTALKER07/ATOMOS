package deliveryroutes

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

func deliveryRouter(svc *order.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("delivery-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueDeliveryToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "driver-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "delivery-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestDeliveryRoutes_NilService(t *testing.T) {
	// RegisterRoutes must safely no-op when Service is nil.
	r := deliveryRouter(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/verify-handshake", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for unregistered routes, got %d", rr.Code)
	}
}

func TestDeliveryRoutes_Unauthenticated(t *testing.T) {
	svc := &order.Service{}
	r := deliveryRouter(svc)

	endpoints := []string{
		"/v1/delivery/verify-handshake",
		"/v1/delivery/update-order-during-delivery",
		"/v1/delivery/submit-offline-pod",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodPost, ep, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s without auth returned %d, want 401", ep, rr.Code)
		}
	}
}

func TestDeliveryRoutes_RoleGuard(t *testing.T) {
	svc := &order.Service{}
	r := deliveryRouter(svc)

	retailerToken := issueDeliveryToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/verify-handshake", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}
