package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func TestHandleIssuePaymentBypass_Unauthorized(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	// 1. Missing claims -> 401
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/orders/payment-bypass", strings.NewReader(`{"order_id":"ord-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleIssuePaymentBypass(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. RoleDriver (not Admin) -> 401
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/orders/payment-bypass", strings.NewReader(`{"order_id":"ord-1"}`)).WithContext(ctx)
	rr = httptest.NewRecorder()
	svc.HandleIssuePaymentBypass(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleIssuePaymentBypass_UnavailableWhenNilSpanner(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1",
		Role:    auth.RoleAdmin,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/orders/payment-bypass", strings.NewReader(`{"order_id":"ord-1"}`)).WithContext(ctx)
	rr := httptest.NewRecorder()
	svc.HandleIssuePaymentBypass(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when spanner is nil, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleConfirmPaymentBypass_Unauthorized(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	// 1. Missing claims -> 401
	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/confirm-payment-bypass", strings.NewReader(`{"order_id":"ord-1","bypass_token":"123456"}`))
	rr := httptest.NewRecorder()
	svc.HandleConfirmPaymentBypass(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Admin role (not Driver) -> 401
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1",
		Role:    auth.RoleAdmin,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/delivery/confirm-payment-bypass", strings.NewReader(`{"order_id":"ord-1","bypass_token":"123456"}`)).WithContext(ctx)
	rr = httptest.NewRecorder()
	svc.HandleConfirmPaymentBypass(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleConfirmPaymentBypass_UnavailableWhenNilSpanner(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/confirm-payment-bypass", strings.NewReader(`{"order_id":"ord-1","bypass_token":"123456"}`)).WithContext(ctx)
	rr := httptest.NewRecorder()
	svc.HandleConfirmPaymentBypass(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when spanner is nil, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleApproveEarlyComplete_Unauthorized(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	// 1. Missing claims -> 401
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/route/approve-early-complete", strings.NewReader(`{"driver_id":"drv-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleApproveEarlyComplete(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Driver role (not Admin) -> 401
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/route/approve-early-complete", strings.NewReader(`{"driver_id":"drv-1"}`)).WithContext(ctx)
	rr = httptest.NewRecorder()
	svc.HandleApproveEarlyComplete(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleApproveEarlyComplete_UnavailableWhenNilSpanner(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1",
		Role:    auth.RoleAdmin,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/route/approve-early-complete", strings.NewReader(`{"driver_id":"drv-1"}`)).WithContext(ctx)
	rr := httptest.NewRecorder()
	svc.HandleApproveEarlyComplete(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when spanner is nil, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleRequestEarlyComplete_UnauthorizedAndUnavailable(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	// 1. Missing claims -> 401
	req := httptest.NewRequest(http.MethodPost, "/v1/fleet/route/request-early-complete", strings.NewReader(`{"route_id":"rt-1"}`))
	rr := httptest.NewRecorder()
	svc.HandleRequestEarlyComplete(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Admin role (not Driver) -> 401
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1",
		Role:    auth.RoleAdmin,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/fleet/route/request-early-complete", strings.NewReader(`{"route_id":"rt-1"}`)).WithContext(ctx)
	rr = httptest.NewRecorder()
	svc.HandleRequestEarlyComplete(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}

	// 3. Driver role with nil spanner -> 503
	driverCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/fleet/route/request-early-complete", strings.NewReader(`{"route_id":"rt-1"}`)).WithContext(driverCtx)
	rr = httptest.NewRecorder()
	svc.HandleRequestEarlyComplete(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when spanner is nil, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPaymentBypassFSMTransitionRules(t *testing.T) {
	// ADR-009 / Spine Law 4 Fiscal Hard Gate:
	// AWAITING_PAYMENT cannot jump directly to COMPLETED
	err := ValidateStatusTransition(string(StatusAwaitingPayment), string(StatusCompleted), TransitionOpts{
		Actor:  "drv-1",
		Reason: "direct_complete_attempt",
	})
	if err == nil {
		t.Fatal("expected error for direct transition AWAITING_PAYMENT -> COMPLETED, got nil")
	}

	// AWAITING_PAYMENT -> FISCALIZING is valid for payment bypass
	err = ValidateStatusTransition(string(StatusAwaitingPayment), string(StatusFiscalizing), TransitionOpts{
		Actor:  "drv-1",
		Reason: "payment_bypass_confirmed",
	})
	if err != nil {
		t.Fatalf("expected valid transition AWAITING_PAYMENT -> FISCALIZING, got: %v", err)
	}

	// FISCALIZING -> COMPLETED is valid once OFD completes
	err = ValidateStatusTransition(string(StatusFiscalizing), string(StatusCompleted), TransitionOpts{
		Actor:  "system",
		Reason: "fiscal_succeeded",
	})
	if err != nil {
		t.Fatalf("expected valid transition FISCALIZING -> COMPLETED, got: %v", err)
	}
}

func TestEarlyCompleteFSMTransitionRules(t *testing.T) {
	// Active order statuses transitioning to CANCELLED on early route return
	activeStatuses := []Status{
		StatusPending,
		StatusLoaded,
		StatusInTransit,
		StatusShopClosedPending,
		StatusCancelRequested,
		StatusScheduled,
		StatusAutoAccepted,
	}

	for _, st := range activeStatuses {
		err := ValidateStatusTransition(string(st), string(StatusCancelled), TransitionOpts{
			Actor:  "admin-1",
			Reason: "early_route_complete_approved",
		})
		if err != nil {
			t.Errorf("expected status %s -> CANCELLED to be valid for early complete, got %v", st, err)
		}
	}
}

func TestG03_ZeroStatusCompletedInSupplierOps(t *testing.T) {
	content, err := os.ReadFile("supplier_ops.go")
	if err != nil {
		t.Fatalf("failed to read supplier_ops.go: %v", err)
	}
	if strings.Contains(string(content), "StatusCompleted") {
		t.Fatal("G-03 violation: supplier_ops.go must not contain any occurrences of StatusCompleted")
	}
}

func TestGeneratePaymentBypassToken(t *testing.T) {
	token, err := generatePaymentBypassToken()
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	if len(token) != 6 {
		t.Fatalf("expected 6 digit token, got length %d: %s", len(token), token)
	}
	num, err := strconv.Atoi(token)
	if err != nil {
		t.Fatalf("expected numeric token, got %s", token)
	}
	if num < 100000 || num > 999999 {
		t.Fatalf("token %d out of 6-digit range [100000, 999999]", num)
	}
}

func TestStoreEarlyCompleteRequest(t *testing.T) {
	c := cache.New(cache.NewInMemoryBackend(), nil)
	svc := NewService(ServiceConfig{
		Cache: c,
	})

	ctx := context.Background()

	// 1. Missing driver ID -> error
	err := svc.StoreEarlyCompleteRequest(ctx, earlyCompleteRecord{
		OrderIDs: []string{"ord-1"},
	})
	if err == nil {
		t.Fatal("expected error for empty driver ID, got nil")
	}

	// 2. Valid record stored in cache
	rec := earlyCompleteRecord{
		DriverID: "drv-1",
		OrderIDs: []string{"ord-1", "ord-2"},
		Reason:   "TRUCK_BREAKDOWN",
		Note:     "Flat tire on highway",
	}
	err = svc.StoreEarlyCompleteRequest(ctx, rec)
	if err != nil {
		t.Fatalf("unexpected error storing record: %v", err)
	}

	raw, found, err := c.Get(ctx, cacheKeyEarlyCompletePrefix+"drv-1")
	if err != nil || !found {
		t.Fatalf("expected cached record for drv-1, found=%v err=%v", found, err)
	}
	if !strings.Contains(string(raw), "TRUCK_BREAKDOWN") {
		t.Fatalf("expected cached content to contain reason, got %s", string(raw))
	}
}
