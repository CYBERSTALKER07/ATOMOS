package warehouse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleDeleteOpsStaff_RoleEnforcementAndRevocation(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "test-secret", SeedSupplierID: "sup-1"})
	svc.mu.Lock()
	svc.staff = []portalStaff{
		{StaffID: "wh-del-1", Name: "Ops Worker", Role: "PICKER"},
	}
	svc.mu.Unlock()

	// 1. Non-admin forbidden
	req := httptest.NewRequest(http.MethodDelete, "/v1/warehouse/ops/staff/wh-del-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", "wh-del-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "worker-1",
		Role:    auth.RoleWarehouse,
	}))
	rr := httptest.NewRecorder()
	svc.HandleDeleteOpsStaff(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 forbidden for non-admin, got %d", rr.Code)
	}

	// 2. Warehouse Admin can delete
	req = httptest.NewRequest(http.MethodDelete, "/v1/warehouse/ops/staff/wh-del-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:      "wh-admin-1",
		Role:         auth.RoleWarehouseAdmin,
		SupplierID:   "sup-1",
		SupplierRole: auth.RoleWarehouseAdmin,
	}))
	rr = httptest.NewRecorder()
	svc.HandleDeleteOpsStaff(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify staff removed from memory
	svc.mu.Lock()
	if len(svc.staff) != 0 {
		t.Fatalf("expected staff list to be empty, got %d", len(svc.staff))
	}
	svc.mu.Unlock()

	// Verify user is marked revoked in revocation store
	revoked, err := auth.GetRevocationStore().IsUserRevoked(context.Background(), "wh-del-1")
	if err != nil {
		t.Fatalf("unexpected error checking revocation: %v", err)
	}
	if !revoked {
		t.Fatalf("expected wh-del-1 to be revoked in auth revocation store")
	}
}

func TestHandleUpdateOpsStaff_DeactivationRevokesToken(t *testing.T) {
	svc := NewService(ServiceConfig{JWTSecret: "test-secret", SeedSupplierID: "sup-1"})
	svc.mu.Lock()
	svc.staff = []portalStaff{
		{StaffID: "wh-patch-1", Name: "Ops Worker 2", Role: "SORTER"},
	}
	svc.mu.Unlock()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", "wh-patch-1")

	// Update with is_active: false
	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouse/ops/staff/wh-patch-1", strings.NewReader(`{"is_active":false}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:      "admin-1",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-1",
		SupplierRole: auth.RoleAdmin,
	}))
	rr := httptest.NewRecorder()
	svc.HandleUpdateOpsStaff(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	revoked, err := auth.GetRevocationStore().IsUserRevoked(context.Background(), "wh-patch-1")
	if err != nil {
		t.Fatalf("unexpected error checking revocation: %v", err)
	}
	if !revoked {
		t.Fatalf("expected wh-patch-1 to be revoked")
	}
}
