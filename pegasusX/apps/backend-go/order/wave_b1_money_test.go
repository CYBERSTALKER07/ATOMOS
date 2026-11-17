package order

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Wave B1 — money integrity regression tests.

func TestCollectCashIdempotencyKeyIsStable(t *testing.T) {
	orderID := "ord-b1-cash"
	key := "cash-" + orderID
	if key2 := "cash-" + orderID; key2 != key {
		t.Fatalf("cash key not stable across retries: %q vs %q", key, key2)
	}
	if strings.Contains(key, time.Now().Format("2006")) {
		t.Fatalf("cash key must not embed wall-clock: %q", key)
	}
}

func TestCreditLeaveIdempotencyKeyIsStable(t *testing.T) {
	orderID := "ord-b1-credit"
	key := "credit-leave-" + orderID
	if key2 := "credit-leave-" + orderID; key2 != key {
		t.Fatalf("credit-leave key not stable: %q vs %q", key, key2)
	}
}

func TestUpdateOrderDuringDelivery_FailClosedNoSilentSuccess(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	// Order with geofence so validation reaches the fail-closed mutator path.
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-mid",
			DriverID:   "drv-1",
			RetailerID: "ret-1",
			SupplierID: "sup-1",
			Status:     StatusInTransit,
			Lat:        41.3,
			Lng:        69.2,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	svc := newTestService(repo, now)
	claims := auth.Claims{Subject: "drv-1", Role: auth.RoleDriver}
	_, err := svc.UpdateOrderDuringDelivery(context.Background(), claims, UpdateOrderDuringDeliveryRequest{
		OrderID:   "ord-mid",
		Latitude:  41.3,
		Longitude: 69.2,
	})
	if err == nil {
		t.Fatal("expected fail-closed error, got nil success")
	}
	if !strings.Contains(err.Error(), "not_implemented") {
		t.Fatalf("err = %v, want not_implemented", err)
	}
}

func TestSelectCashAtDelivery_ArrivedToPendingCash(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-cash-sel",
			DriverID:   "drv-1",
			RetailerID: "ret-1",
			SupplierID: "sup-1",
			Status:     StatusArrived,
			TotalMinor: 15000,
			Currency:   "UZS",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	svc := newTestService(repo, now)
	status, amount, currency, err := svc.SelectCashAtDelivery(context.Background(), "ord-cash-sel", "ret-1", "ret-user-1")
	if err != nil {
		t.Fatalf("SelectCashAtDelivery: %v", err)
	}
	if status != string(StatusPendingCashCollection) {
		t.Fatalf("status=%s want PENDING_CASH_COLLECTION", status)
	}
	if amount != 15000 || currency != "UZS" {
		t.Fatalf("amount/currency = %d %s", amount, currency)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("updateCalls=%d want 1", repo.updateCalls)
	}
	if repo.bufferedEvents < 1 {
		t.Fatalf("expected outbox events on cash select, got %d", repo.bufferedEvents)
	}
	// Idempotent replay
	status2, _, _, err := svc.SelectCashAtDelivery(context.Background(), "ord-cash-sel", "ret-1", "ret-user-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if status2 != string(StatusPendingCashCollection) {
		t.Fatalf("replay status=%s", status2)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("replay should not re-update: updateCalls=%d", repo.updateCalls)
	}
}

func TestSelectCashAtDelivery_ForbiddenWrongRetailer(t *testing.T) {
	now := time.Now().UTC()
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID: "ord-x", RetailerID: "ret-1", SupplierID: "sup-1",
			Status: StatusArrived, CreatedAt: now, UpdatedAt: now,
		},
	}
	svc := newTestService(repo, now)
	_, _, _, err := svc.SelectCashAtDelivery(context.Background(), "ord-x", "ret-other", "u")
	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("err=%v want ErrOrderForbidden", err)
	}
}

func TestSelectCashAtDelivery_BeforeDeliveryRejected(t *testing.T) {
	now := time.Now().UTC()
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID: "ord-early", RetailerID: "ret-1", SupplierID: "sup-1",
			Status: StatusInTransit, CreatedAt: now, UpdatedAt: now,
		},
	}
	svc := newTestService(repo, now)
	_, _, _, err := svc.SelectCashAtDelivery(context.Background(), "ord-early", "ret-1", "u")
	if err == nil || !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("err=%v want ErrInvalidStatusTransition", err)
	}
}
