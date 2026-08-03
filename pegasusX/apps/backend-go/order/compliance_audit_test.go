package order

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleComplianceDashboard_unauthorized(t *testing.T) {
	svc := newTestService(&testRepo{}, time.Now().UTC())
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard", nil)
	rr := httptest.NewRecorder()
	svc.HandleComplianceDashboard(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rr.Code)
	}
}

func TestHandleComplianceDashboard_noSpanner(t *testing.T) {
	svc := newTestService(&testRepo{}, time.Now().UTC())
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleComplianceDashboard(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 (nil spanner)", rr.Code)
	}
}

func TestComplianceLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard?limit=10", nil)
	if got := complianceLimit(req, 50, 200); got != 10 {
		t.Fatalf("limit=%d", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard?limit=9999", nil)
	if got := complianceLimit(req, 50, 200); got != 200 {
		t.Fatalf("capped limit=%d", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard", nil)
	if got := complianceLimit(req, 50, 200); got != 50 {
		t.Fatalf("default limit=%d", got)
	}
}
