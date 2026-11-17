package compliance

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestGetDashboard_requiresSessionSupplier(t *testing.T) {
	repo := &mockRepo{stats: DashboardStats{FiscalFailed: 1}}
	h := NewHandler(NewService(repo, slog.Default()))

	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard?supplierId=tenant-abc-123", nil)
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without session, got %d", rr.Code)
	}
}

func TestGetDashboard_usesJWTNotQuery(t *testing.T) {
	repo := &mockRepo{
		stats: DashboardStats{FiscalFailed: 2},
		orders: []ProblemOrder{
			{OrderID: "o1", Status: "COMPLETED", FiscalStatus: "FAILED", CreatedAt: time.Now().UTC()},
		},
	}
	h := NewHandler(NewService(repo, slog.Default()))

	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard?supplierId=tenant-abc-123", nil)
	req = req.WithContext(auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-real-9",
	}))
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.lastFilter.SupplierID != "sup-real-9" {
		t.Fatalf("expected JWT supplier, got %q", repo.lastFilter.SupplierID)
	}
}

func TestExportCSV_requiresSessionSupplier(t *testing.T) {
	repo := &mockRepo{}
	h := NewHandler(NewService(repo, slog.Default()))
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/export?supplierId=tenant-abc-123", nil)
	rr := httptest.NewRecorder()
	h.ExportCSV(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without session, got %d", rr.Code)
	}
}
