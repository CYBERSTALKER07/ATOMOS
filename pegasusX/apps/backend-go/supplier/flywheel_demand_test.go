package supplier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleAnalyticsDemandFlywheel_emptyWithoutSpanner(t *testing.T) {
	t.Parallel()
	svc := &Service{
		supplierID: "sup-fly",
		now:        func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) },
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/analytics/demand/flywheel?days=7", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin", Role: auth.RoleAdmin, SupplierID: "sup-fly",
	}))
	rr := httptest.NewRecorder()
	svc.HandleAnalyticsDemandFlywheel(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Source string               `json:"source"`
		Items  []FlywheelDemandItem `json:"items"`
		Days   int                  `json:"days"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Source != "STORE_POS" {
		t.Fatalf("source=%q", body.Source)
	}
	if body.Days != 7 {
		t.Fatalf("days=%d", body.Days)
	}
	if body.Items == nil {
		t.Fatal("items should be empty slice not null")
	}
	if len(body.Items) != 0 {
		t.Fatalf("items=%+v", body.Items)
	}
}

func TestHandleAnalyticsDemandFlywheel_requiresSupplierScope(t *testing.T) {
	t.Parallel()
	svc := &Service{now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/analytics/demand/flywheel", nil)
	// No supplier scope
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "x", Role: auth.RoleAdmin,
	}))
	rr := httptest.NewRecorder()
	svc.HandleAnalyticsDemandFlywheel(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}
