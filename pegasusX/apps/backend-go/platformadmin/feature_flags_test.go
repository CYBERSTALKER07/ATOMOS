package platformadmin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestFeatureFlagDualControl_Service(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// 1. Setting regular flag should be ACTIVE immediately
	if err := svc.SetFeatureFlag(ctx, TenantRetailer, "ret-1", "GENERAL_FLAG", SetFeatureFlagRequest{Enabled: true}, "admin-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, ok, err := repo.GetFeatureFlag(ctx, TenantRetailer, "ret-1", "GENERAL_FLAG")
	if err != nil || !ok || f.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE for GENERAL_FLAG, got ok=%v, status=%s, err=%v", ok, f.Status, err)
	}

	// 2. Setting dual-control flags should be PENDING
	dualFlags := []string{
		"PAYMENT_GATEWAY_V2",
		"AR_INVOICES_ENABLED",
		"AUTO_ORDER_PLACE_ENABLED",
		"FISCAL_PROVIDER",
	}
	for _, df := range dualFlags {
		if err := svc.SetFeatureFlag(ctx, TenantRetailer, "ret-1", df, SetFeatureFlagRequest{Enabled: true}, "admin-1"); err != nil {
			t.Fatalf("unexpected error for %s: %v", df, err)
		}
		flag, ok, err := repo.GetFeatureFlag(ctx, TenantRetailer, "ret-1", df)
		if err != nil || !ok || flag.Status != "PENDING" {
			t.Fatalf("expected PENDING for %s, got ok=%v, status=%s", df, ok, flag.Status)
		}

		// 3. Same admin cannot approve (dual control)
		err = svc.ApproveFeatureFlag(ctx, TenantRetailer, "ret-1", df, "admin-1")
		if err != ErrApproverMustDiffer {
			t.Fatalf("expected ErrApproverMustDiffer for same admin on %s, got %v", df, err)
		}

		// 4. Different admin approves successfully
		err = svc.ApproveFeatureFlag(ctx, TenantRetailer, "ret-1", df, "admin-2")
		if err != nil {
			t.Fatalf("expected success for admin-2 on %s, got %v", df, err)
		}

		flag, ok, err = repo.GetFeatureFlag(ctx, TenantRetailer, "ret-1", df)
		if err != nil || !ok || flag.Status != "ACTIVE" || flag.ApprovedBy != "admin-2" {
			t.Fatalf("expected ACTIVE with ApprovedBy admin-2, got ok=%v, status=%s, approvedBy=%s", ok, flag.Status, flag.ApprovedBy)
		}

		// 5. Approving again fails because already ACTIVE
		err = svc.ApproveFeatureFlag(ctx, TenantRetailer, "ret-1", df, "admin-3")
		if err != ErrFeatureFlagNotPending {
			t.Fatalf("expected ErrFeatureFlagNotPending, got %v", err)
		}
	}

	// 6. Non-existent flag approval fails
	err = svc.ApproveFeatureFlag(ctx, TenantRetailer, "ret-1", "NONEXISTENT", "admin-2")
	if err != ErrFeatureFlagNotFound {
		t.Fatalf("expected ErrFeatureFlagNotFound, got %v", err)
	}
}

func TestHandleApproveFeatureFlag_HTTP(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)
	handlers := &Handlers{Svc: svc}
	ctx := context.Background()

	// Seed a pending flag
	_ = svc.SetFeatureFlag(ctx, TenantRetailer, "ret-1", "AUTO_ORDER_PLACE_ENABLED", SetFeatureFlagRequest{Enabled: true}, "admin-1")

	// Same admin tries to approve via HTTP -> 403 Forbidden
	req := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/tenants/RETAILER/ret-1/flags/AUTO_ORDER_PLACE_ENABLED/approve", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenantType", "RETAILER")
	rctx.URLParams.Add("tenantID", "ret-1")
	rctx.URLParams.Add("flagKey", "AUTO_ORDER_PLACE_ENABLED")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1",
		Role:    auth.RolePlatformAdmin,
	}))
	rr := httptest.NewRecorder()
	handlers.HandleApproveFeatureFlag(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for same admin, got %d: %s", rr.Code, rr.Body.String())
	}

	// Distinct admin approves via HTTP -> 200 OK
	req = httptest.NewRequest(http.MethodPost, "/v1/platform-admin/tenants/RETAILER/ret-1/flags/AUTO_ORDER_PLACE_ENABLED/approve", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-2",
		Role:    auth.RolePlatformAdmin,
	}))
	rr = httptest.NewRecorder()
	handlers.HandleApproveFeatureFlag(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}
}
