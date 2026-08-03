package order

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/allocation"
)

type mockAllocator struct {
	result *allocation.AllocationResult
	err    error
}

func (m *mockAllocator) AllocateOrder(ctx context.Context, req *allocation.AllocationRequest) (*allocation.AllocationResult, error) {
	return m.result, m.err
}

func (m *mockAllocator) AllocateOrderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req *allocation.AllocationRequest) (*allocation.AllocationResult, error) {
	return m.AllocateOrder(ctx, req)
}

func seedActiveWarehouse(t *testing.T, ctx context.Context, client *spanner.Client, supplierID, warehouseID string) {
	t.Helper()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId":      warehouseID,
			"SupplierId":       supplierID,
			"Name":             warehouseID,
			"CoverageRadiusKm": 10.0,
			"IsActive":         true,
			"IsOnShift":        true,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed warehouse: %v", err)
	}
}

func seedSupplierInventorySKU(t *testing.T, ctx context.Context, client *spanner.Client, supplierID, warehouseID, sku string, onHand, reserved int64) {
	t.Helper()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        sku,
			"QuantityOnHand":   onHand,
			"QuantityReserved": reserved,
			"ReorderThreshold": 0,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed supplier inventory: %v", err)
	}
}

func seedAllocationInventory(t *testing.T, ctx context.Context, client *spanner.Client, supplierID, warehouseID string, sku string, onHand, reserved int64) {
	t.Helper()
	seedActiveWarehouse(t, ctx, client, supplierID, warehouseID)
	seedSupplierInventorySKU(t, ctx, client, supplierID, warehouseID, sku, onHand, reserved)
}

func cleanupAllocationInventory(t *testing.T, ctx context.Context, client *spanner.Client, supplierID, warehouseID, sku string) {
	t.Helper()
	client.Apply(ctx, []*spanner.Mutation{
		spanner.Delete("SupplierInventoryV2", spanner.Key{supplierID, warehouseID, sku}),
		spanner.Delete("Warehouses", spanner.Key{warehouseID}),
	})
}

func TestConfirmAndAllocate_PartialAllocation(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	repo := NewSpannerRepository(client)

	orderID := "ord-allocate-test-1"
	supplierID := "sup-partial-1"
	now := time.Now()
	err := repo.CreateOrder(ctx, &Order{
		OrderID:    orderID,
		SupplierID: supplierID,
		Status:     StatusPending,
		LineItems: []LineItem{
			{SKU: "sku-1", Quantity: 5},
			{SKU: "sku-2", Quantity: 2},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil, StockReservationOpts{})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	allocator := &mockAllocator{
		result: &allocation.AllocationResult{
			Fulfillments: map[string]string{
				"sku-1": "wh-1",
			},
		},
	}

	svc := NewService(ServiceConfig{
		Repo:          repo,
		Allocator:     allocator,
		SpannerClient: client,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           func() time.Time { return now },
	})
	svc.SetAllocationRequired(true)

	err = svc.ConfirmAndAllocate(ctx, orderID)
	if err == nil || !strings.Contains(err.Error(), "partial allocation") {
		t.Fatalf("expected partial allocation error, got %v", err)
	}
}

func TestConfirmAndAllocate_MultiWarehouseSplits(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	repo := NewSpannerRepository(client)

	orderID := "ord-allocate-test-2"
	supplierID := "sup-split-1"
	now := time.Now()
	err := repo.CreateOrder(ctx, &Order{
		OrderID:    orderID,
		SupplierID: supplierID,
		Status:     StatusPending,
		LineItems: []LineItem{
			{SKU: "sku-1", Quantity: 5},
			{SKU: "sku-2", Quantity: 2},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil, StockReservationOpts{})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	allocator := &mockAllocator{
		result: &allocation.AllocationResult{
			Fulfillments: map[string]string{
				"sku-1": "wh-1",
				"sku-2": "wh-2",
			},
		},
	}

	svc := NewService(ServiceConfig{
		Repo:          repo,
		Allocator:     allocator,
		SpannerClient: client,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           func() time.Time { return now },
	})
	svc.SetAllocationRequired(true)

	err = svc.ConfirmAndAllocate(ctx, orderID)
	if err == nil || !strings.Contains(err.Error(), "multi-warehouse splits") {
		t.Fatalf("expected multi-warehouse split error, got %v", err)
	}
}

func TestConfirmAndAllocate_Success(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	repo := NewSpannerRepository(client)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	supplierID := "sup-success-" + suffix
	warehouseID := "wh-success-" + suffix
	seedActiveWarehouse(t, ctx, client, supplierID, warehouseID)
	seedSupplierInventorySKU(t, ctx, client, supplierID, warehouseID, "sku-1", 100, 0)
	seedSupplierInventorySKU(t, ctx, client, supplierID, warehouseID, "sku-2", 100, 0)
	defer cleanupAllocationInventory(t, ctx, client, supplierID, warehouseID, "sku-1")
	defer cleanupAllocationInventory(t, ctx, client, supplierID, warehouseID, "sku-2")

	orderID := "ord-allocate-test-3-" + suffix
	now := time.Now()
	err := repo.CreateOrder(ctx, &Order{
		OrderID:     orderID,
		SupplierID:  supplierID,
		WarehouseID: "wh-default",
		Status:      StatusPending,
		LineItems: []LineItem{
			{SKU: "sku-1", Quantity: 5},
			{SKU: "sku-2", Quantity: 2},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil, StockReservationOpts{Skip: true})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo:          repo,
		Allocator:     allocation.NewAllocationService(client),
		SpannerClient: client,
		SupplierID:    supplierID,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           func() time.Time { return now },
	})
	svc.SetAllocationRequired(true)

	err = svc.ConfirmAndAllocate(ctx, orderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	o, found, err := repo.GetOrder(ctx, orderID)
	if err != nil || !found {
		t.Fatalf("failed to fetch updated order")
	}
	if o.WarehouseID != warehouseID {
		t.Fatalf("expected warehouse %s, got %s", warehouseID, o.WarehouseID)
	}

	row, err := client.Single().ReadRow(ctx, "OrderStockReservationMarkers", spanner.Key{orderID}, []string{"OrderId"})
	if err != nil {
		t.Fatalf("expected reservation marker: %v", err)
	}
	if row == nil {
		t.Fatal("expected reservation marker row")
	}
}

func TestConfirmAndAllocate_ConcurrentReservation(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	repo := NewSpannerRepository(client)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	supplierID := "sup-race-" + suffix
	warehouseID := "wh-race-" + suffix
	sku := "sku-race"
	seedAllocationInventory(t, ctx, client, supplierID, warehouseID, sku, 10, 0)
	defer cleanupAllocationInventory(t, ctx, client, supplierID, warehouseID, sku)

	now := time.Now()
	orderA := "ord-race-a-" + suffix
	orderB := "ord-race-b-" + suffix
	for _, spec := range []struct {
		id  string
		qty int64
	}{
		{id: orderA, qty: 6},
		{id: orderB, qty: 6},
	} {
		if err := repo.CreateOrder(ctx, &Order{
			OrderID:     spec.id,
			SupplierID:  supplierID,
			WarehouseID: warehouseID,
			Status:      StatusPending,
			LineItems:   []LineItem{{SKU: sku, Quantity: spec.qty}},
			CreatedAt:   now,
			UpdatedAt:   now,
		}, nil, StockReservationOpts{Skip: true}); err != nil {
			t.Fatalf("create order %s: %v", spec.id, err)
		}
	}

	svc := NewService(ServiceConfig{
		Repo:          repo,
		Allocator:     allocation.NewAllocationService(client),
		SpannerClient: client,
		SupplierID:    supplierID,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           func() time.Time { return now },
	})
	svc.SetAllocationRequired(true)

	errA := svc.ConfirmAndAllocate(ctx, orderA)
	errB := svc.ConfirmAndAllocate(ctx, orderB)
	if errA == nil && errB == nil {
		t.Fatal("expected one allocation to fail due to insufficient stock")
	}
	if errA != nil && errB != nil {
		t.Fatalf("expected one success, got both failed: %v / %v", errA, errB)
	}
}

func TestCreateOrder_LegacyFlagOffReservesAtCreate(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	repo := NewSpannerRepository(client)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	supplierID := "sup-legacy-" + suffix
	warehouseID := "wh-legacy-" + suffix
	sku := "sku-legacy"
	seedAllocationInventory(t, ctx, client, supplierID, warehouseID, sku, 50, 0)
	defer cleanupAllocationInventory(t, ctx, client, supplierID, warehouseID, sku)

	orderID := "ord-legacy-" + suffix
	now := time.Now()
	_ = NewService(ServiceConfig{
		Repo:          repo,
		SpannerClient: client,
		SupplierID:    supplierID,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           func() time.Time { return now },
	})

	err := repo.CreateOrder(ctx, &Order{
		OrderID:     orderID,
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		Status:      StatusPending,
		LineItems:   []LineItem{{SKU: sku, Quantity: 3}},
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil, StockReservationOpts{})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	row, err := client.Single().ReadRow(ctx, "OrderStockReservationMarkers", spanner.Key{orderID}, []string{"OrderId"})
	if err != nil {
		t.Fatalf("expected reservation marker with flag off: %v", err)
	}
	if row == nil {
		t.Fatal("expected reservation marker")
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OrderLineAllocations WHERE OrderId = @orderId`,
		Params: map[string]any{"orderId": orderID},
	})
	defer iter.Stop()
	rowCount, err := iter.Next()
	if err != nil {
		t.Fatalf("count allocations: %v", err)
	}
	var count int64
	if err := rowCount.Column(0, &count); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no allocation rows with flag off, got %d", count)
	}
}

func TestPromotePreorder_AllocationRequired(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	repo := NewSpannerRepository(client)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	supplierID := "sup-promote-" + suffix
	warehouseID := "wh-promote-" + suffix
	sku := "sku-promote"
	seedActiveWarehouse(t, ctx, client, supplierID, warehouseID)
	seedSupplierInventorySKU(t, ctx, client, supplierID, warehouseID, sku, 50, 0)
	defer cleanupAllocationInventory(t, ctx, client, supplierID, warehouseID, sku)

	orderID := "ord-promote-" + suffix
	now := time.Now()
	if err := repo.CreateOrder(ctx, &Order{
		OrderID:            orderID,
		SupplierID:         supplierID,
		WarehouseID:        warehouseID,
		Status:             StatusScheduled,
		Source:             OrderSourceManualPreorder,
		ConfirmationStatus: ConfirmationStatusConfirmed,
		LineItems:          []LineItem{{SKU: sku, Quantity: 2}},
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil, StockReservationOpts{Skip: true}); err != nil {
		t.Fatalf("create scheduled order: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo:          repo,
		Allocator:     allocation.NewAllocationService(client),
		SpannerClient: client,
		SupplierID:    supplierID,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           func() time.Time { return now },
	})
	svc.SetAllocationRequired(true)

	o, found, _ := repo.GetOrder(ctx, orderID)
	if !found {
		t.Fatal("order not found")
	}
	if err := svc.promotePreorderToPending(ctx, o, now); err != nil {
		t.Fatalf("promote failed: %v", err)
	}

	updated, found, err := repo.GetOrder(ctx, orderID)
	if err != nil || !found {
		t.Fatalf("fetch order: %v", err)
	}
	if updated.Status != StatusPending {
		t.Fatalf("expected PENDING, got %s", updated.Status)
	}
	if _, err := client.Single().ReadRow(ctx, "OrderStockReservationMarkers", spanner.Key{orderID}, []string{"OrderId"}); err != nil {
		t.Fatalf("expected reservation at promote: %v", err)
	}
}
