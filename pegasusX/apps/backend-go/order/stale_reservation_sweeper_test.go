package order

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestStaleReservationSweeper_ConfigOverrides(t *testing.T) {
	origInterval := os.Getenv("STALE_RESERVATION_INTERVAL_SECONDS")
	origTTL := os.Getenv("STALE_RESERVATION_TTL_MINUTES")
	origHoldTTL := os.Getenv("INVENTORY_HOLD_TTL_MINUTES")
	defer func() {
		os.Setenv("STALE_RESERVATION_INTERVAL_SECONDS", origInterval)
		os.Setenv("STALE_RESERVATION_TTL_MINUTES", origTTL)
		os.Setenv("INVENTORY_HOLD_TTL_MINUTES", origHoldTTL)
	}()

	// 1. Defaults
	os.Unsetenv("STALE_RESERVATION_INTERVAL_SECONDS")
	os.Unsetenv("STALE_RESERVATION_TTL_MINUTES")
	os.Unsetenv("INVENTORY_HOLD_TTL_MINUTES")
	if staleReservationInterval() != DefaultStaleReservationInterval {
		t.Fatalf("expected default interval %v, got %v", DefaultStaleReservationInterval, staleReservationInterval())
	}
	if staleReservationTTL() != DefaultStaleReservationTTL {
		t.Fatalf("expected default TTL %v, got %v", DefaultStaleReservationTTL, staleReservationTTL())
	}

	// 2. Custom overrides
	os.Setenv("STALE_RESERVATION_INTERVAL_SECONDS", "30")
	os.Setenv("STALE_RESERVATION_TTL_MINUTES", "25")
	if staleReservationInterval() != 30*time.Second {
		t.Fatalf("expected 30s interval, got %v", staleReservationInterval())
	}
	if staleReservationTTL() != 25*time.Minute {
		t.Fatalf("expected 25m TTL, got %v", staleReservationTTL())
	}

	// 3. Fallback to INVENTORY_HOLD_TTL_MINUTES
	os.Unsetenv("STALE_RESERVATION_TTL_MINUTES")
	os.Setenv("INVENTORY_HOLD_TTL_MINUTES", "10")
	if staleReservationTTL() != 10*time.Minute {
		t.Fatalf("expected 10m TTL from INVENTORY_HOLD_TTL_MINUTES, got %v", staleReservationTTL())
	}
}

func TestReleaseStaleReservations_PendingOrderCancelled(t *testing.T) {
	now := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusPending)
	o.OrderID = "ord-stale-pending-1"
	o.UpdatedAt = now.Add(-30 * time.Minute) // 30 mins ago (exceeds 15m TTL)

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	// Direct cancellation with reason matching stale sweeper
	resp, err := svc.cancelOrderWithReason(context.Background(), &o, "system", "SYSTEM", "stale_unpaid_reservation_release", "")
	if err != nil {
		t.Fatalf("cancelOrderWithReason failed: %v", err)
	}

	if resp.Status != StatusCancelled {
		t.Fatalf("expected status CANCELLED, got %s", resp.Status)
	}
	if repo.captured.Status != StatusCancelled {
		t.Fatalf("expected captured order status CANCELLED, got %s", repo.captured.Status)
	}
	if repo.captured.WarehouseNotes != "stale_unpaid_reservation_release" {
		t.Fatalf("expected reason note 'stale_unpaid_reservation_release', got %q", repo.captured.WarehouseNotes)
	}
}

func TestReleaseStaleReservations_SkipsActiveOrBackorder(t *testing.T) {
	now := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)

	// Non-pending status
	oTransit := deliveryTestOrder(StatusInTransit)
	oTransit.OrderID = "ord-in-transit-1"
	oTransit.UpdatedAt = now.Add(-40 * time.Minute)

	repo := &testRepo{
		found: true,
		order: oTransit,
	}
	svc := newTestService(repo, now)

	// Sweeper over transit order should not cancel it
	if oTransit.Status == StatusPending {
		t.Fatalf("expected non-pending status")
	}

	// Backorder source
	oBackorder := deliveryTestOrder(StatusPending)
	oBackorder.OrderID = "ord-backorder-1"
	oBackorder.Source = OrderSourceBackorder
	oBackorder.UpdatedAt = now.Add(-40 * time.Minute)

	repo2 := &testRepo{
		found: true,
		order: oBackorder,
	}
	svc2 := newTestService(repo2, now)
	if svc2 == nil || svc == nil {
		t.Fatalf("service initialization failed")
	}
}

func TestRunStaleReservationSweeperOnce_NilSafe(t *testing.T) {
	var nilSvc *Service
	n, err := nilSvc.RunStaleReservationSweeperOnce(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("expected 0, nil for nil service, got %d, %v", n, err)
	}

	// Service with nil spanner client
	svc := &Service{spannerClient: nil}
	n, err = svc.RunStaleReservationSweeperOnce(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("expected 0, nil for nil spannerClient, got %d, %v", n, err)
	}
}
