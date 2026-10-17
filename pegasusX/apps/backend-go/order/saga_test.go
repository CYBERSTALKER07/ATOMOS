package order

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
