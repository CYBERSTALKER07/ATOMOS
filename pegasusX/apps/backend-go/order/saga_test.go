package order

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestSagaConstants(t *testing.T) {
	if SagaStatePending != "PENDING" {
		t.Errorf("expected PENDING, got %s", SagaStatePending)
	}
	if SagaStateReserving != "RESERVING" {
		t.Errorf("expected RESERVING, got %s", SagaStateReserving)
	}
	if SagaStateCommitted != "COMMITTED" {
		t.Errorf("expected COMMITTED, got %s", SagaStateCommitted)
	}
	if SagaStateCompensating != "COMPENSATING" {
		t.Errorf("expected COMPENSATING, got %s", SagaStateCompensating)
	}
	if SagaStateCompensated != "COMPENSATED" {
		t.Errorf("expected COMPENSATED, got %s", SagaStateCompensated)
	}
	if SagaLeaseDuration < 30*time.Second {
		t.Errorf("expected lease at least 30s, got %v", SagaLeaseDuration)
	}
}

func TestSagaCoordinator_CompensateInFlightOrder(t *testing.T) {
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{
		SupplierID: "sup-saga-1",
		Source:     "saga_test",
	})

	repo := &testRepo{}
	warehouse := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		Warehouse:    warehouse,
		SupplierID:   "sup-saga-1",
		SupplierName: "Saga Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	// Create child order #1
	created1, err := svc.Create(ctx, "ret-1", CreateRequest{
		H3Cell: "8720a52d2ffffff",
		Lat:    41.31,
		Lng:    69.24,
		LineItems: []LineItem{
			{SKU: "sku-1", Name: "Item 1", Quantity: 2, UnitPrice: 5000},
		},
	})
	if err != nil {
		t.Fatalf("create child 1: %v", err)
	}

	parentID := "par_test_saga_1"
	childOrders := []Order{
		{
			OrderID:       created1.OrderID,
			SupplierID:    "sup-saga-1",
			RetailerID:    "ret-1",
			ParentOrderID: parentID,
			Status:        created1.Status,
			TotalMinor:    created1.TotalMinor,
		},
	}

	// Mid-flight failure: child order #2 failed, coordinator triggers CompensateSaga
	err = svc.CompensateSaga(ctx, parentID, childOrders, "simulated_child_2_stockout")
	if err != nil {
		t.Fatalf("compensate saga failed: %v", err)
	}

	// Verify child order #1 was cancelled via repo
	if repo.captured.Status != StatusCancelled {
		t.Errorf("expected child order status CANCELLED, got %s", repo.captured.Status)
	}
	if !strings.Contains(repo.captured.WarehouseNotes, "multi_supplier_checkout_abort") {
		t.Errorf("expected warehouse notes to contain multi_supplier_checkout_abort, got %s", repo.captured.WarehouseNotes)
	}
}

func TestSagaCoordinator_NilSpannerFallbacks(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID:   "sup-test",
		SupplierName: "Test",
		Currency:     "UZS",
		Log:          slog.Default(),
	})
	ctx := context.Background()

	// Should safely no-op without panics when spanner is not connected
	if err := svc.RecordSagaChildCreated(ctx, "par-1", "ord-1"); err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if err := svc.CompleteSaga(ctx, "par-1", "UZS", 1000, 1); err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	stalled, err := svc.SweepStalledSagas(ctx)
	if err != nil || len(stalled) != 0 {
		t.Errorf("expected empty stalled slice, got %v, err: %v", stalled, err)
	}
}

type failingUpdateRepo struct {
	testRepo
	failOrderID string
}

func (r *failingUpdateRepo) UpdateOrder(ctx context.Context, o Order, proofs []DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error {
	if o.OrderID == r.failOrderID {
		return errors.New("simulated repo update failure")
	}
	return r.testRepo.UpdateOrder(ctx, o, proofs, emit)
}

func TestSagaCoordinator_CompensateChildFailureReturnsError(t *testing.T) {
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{
		SupplierID: "sup-saga-1",
		Source:     "saga_test",
	})

	repo := &failingUpdateRepo{
		failOrderID: "ord-fail-1",
	}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		SupplierID:   "sup-saga-1",
		SupplierName: "Saga Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	childOrders := []Order{
		{
			OrderID:       "ord-fail-1",
			SupplierID:    "sup-saga-1",
			RetailerID:    "ret-1",
			ParentOrderID: "par-fail-1",
			Status:        StatusPending,
		},
	}

	err := svc.CompensateSaga(ctx, "par-fail-1", childOrders, "abort_due_to_timeout")
	if err == nil {
		t.Fatal("expected error from CompensateSaga when child cancellation fails, got nil")
	}
	if !strings.Contains(err.Error(), "compensate child orders failed") {
		t.Errorf("expected 'compensate child orders failed', got: %v", err)
	}
}

func TestSagaCoordinator_CompensateMultipleChildren_PartialFailure(t *testing.T) {
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{
		SupplierID: "sup-saga-1",
		Source:     "saga_test",
	})

	repo := &failingUpdateRepo{
		failOrderID: "ord-child-2",
	}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		SupplierID:   "sup-saga-1",
		SupplierName: "Saga Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	childOrders := []Order{
		{
			OrderID:       "ord-child-1",
			SupplierID:    "sup-saga-1",
			RetailerID:    "ret-1",
			ParentOrderID: "par-multi-1",
			Status:        StatusPending,
		},
		{
			OrderID:       "ord-child-2",
			SupplierID:    "sup-saga-1",
			RetailerID:    "ret-1",
			ParentOrderID: "par-multi-1",
			Status:        StatusPending,
		},
	}

	err := svc.CompensateSaga(ctx, "par-multi-1", childOrders, "abort_due_to_leg_failure")
	if err == nil {
		t.Fatal("expected error from CompensateSaga when partial child cancellation fails, got nil")
	}
	if !strings.Contains(err.Error(), "ord-child-2") {
		t.Errorf("expected error to name failed child ord-child-2, got: %v", err)
	}
}

func TestReclaimPendingCancelCompensations_NilGuards(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID:   "sup-test",
		SupplierName: "Test",
		Currency:     "UZS",
		Log:          slog.Default(),
	})
	ctx := context.Background()

	// Should safely return 0, nil when spanner/credit are nil
	n, err := svc.ReclaimPendingCancelCompensations(ctx)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 reclaimed, got %d", n)
	}
}

func TestReclaimPendingCancelCompensations_Hook(t *testing.T) {
	testReclaimPendingCancelCompensationsHook = func(ctx context.Context) (int, error) {
		return 3, nil
	}
	defer func() { testReclaimPendingCancelCompensationsHook = nil }()

	svc := NewService(ServiceConfig{
		SupplierID:   "sup-test",
		SupplierName: "Test",
		Currency:     "UZS",
		Log:          slog.Default(),
	})
	ctx := context.Background()

	n, err := svc.ReclaimPendingCancelCompensations(ctx)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 reclaimed, got %d", n)
	}
}

func TestStartSagaRecoveryWorker_ContextShutdown(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID:   "sup-test",
		SupplierName: "Test",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		StartSagaRecoveryWorker(ctx, svc, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// success: worker shut down cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("StartSagaRecoveryWorker did not shut down on context cancellation")
	}
}

func TestStartSagaRecoveryWorker_ExecutesBothSweepAndReclaim(t *testing.T) {
	sweepCalls := 0
	reclaimCalls := 0

	testSweepStalledSagasHook = func(ctx context.Context) ([]StalledSagaRecord, error) {
		sweepCalls++
		return nil, nil
	}
	defer func() { testSweepStalledSagasHook = nil }()

	testReclaimPendingCancelCompensationsHook = func(ctx context.Context) (int, error) {
		reclaimCalls++
		return 1, nil
	}
	defer func() { testReclaimPendingCancelCompensationsHook = nil }()

	svc := NewService(ServiceConfig{
		SupplierID:   "sup-test",
		SupplierName: "Test",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		StartSagaRecoveryWorker(ctx, svc, 15*time.Millisecond)
		close(done)
	}()

	// Wait long enough for initial execution plus at least 1 tick
	time.Sleep(40 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not exit")
	}

	if sweepCalls < 2 {
		t.Errorf("expected at least 2 sweep calls (startup + tick), got %d", sweepCalls)
	}
	if reclaimCalls < 2 {
		t.Errorf("expected at least 2 reclaim calls (startup + tick), got %d", reclaimCalls)
	}
}
