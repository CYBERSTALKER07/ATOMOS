package paymentroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

func paymentRouter(svc *payment.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("payment-test-secret"))
	RegisterRoutes(r, Deps{Service: svc, JWTSecret: "payment-test-secret"})
	return r
}

func issuePaymentToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "payment-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "payment-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestPaymentRoutes_NilService(t *testing.T) {
	r := paymentRouter(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/checkout/b2b", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestPaymentRoutes_Unauthenticated(t *testing.T) {
	svc := &payment.Service{}
	r := paymentRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/checkout/b2b"},
		{http.MethodPost, "/v1/checkout/preview"},
		{http.MethodPost, "/v1/payment/chargeback"},
		{http.MethodGet, "/v1/payment/ledger"},
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

func TestPaymentRoutes_RoleGuard(t *testing.T) {
	svc := &payment.Service{}
	r := paymentRouter(svc)

	retailerToken := issuePaymentToken(t, auth.RoleRetailer)

	// Retailer cannot trigger chargeback (Admin required)
	req := httptest.NewRequest(http.MethodPost, "/v1/payment/chargeback", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer triggering chargeback returned %d, want 403 Forbidden", rr.Code)
	}
}
