package retailer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleCheckShelfAlertsRequiresSessionAndStore(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/shelf/check", strings.NewReader(`{"location_id":"loc","items":[{"global_product_id":"sku","capacity_threshold":2}]}`))
	rr := httptest.NewRecorder()
	svc.HandleCheckShelfAlerts(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/retailer/shelf/check", strings.NewReader(`{"location_id":"loc","items":[{"global_product_id":"sku","capacity_threshold":2}]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr = httptest.NewRecorder()
	svc.HandleCheckShelfAlerts(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil store status = %d body %s", rr.Code, rr.Body.String())
	}
}

func TestHandleResolveShelfAlertRequiresStore(t *testing.T) {
	svc := &Service{}
	r := chi.NewRouter()
	r.Post("/v1/retailer/shelf/alerts/{alertID}/resolve", svc.HandleResolveShelfAlert)
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/shelf/alerts/alert-1/resolve", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
