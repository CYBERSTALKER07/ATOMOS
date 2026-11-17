package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

type memoryMultiOrderRepo struct {
	mu           sync.Mutex
	orders       map[string]Order
	idemKeys     map[string]string // key -> orderID
	createCalls  int
	failOnChild2 bool
}

func newMemoryMultiOrderRepo() *memoryMultiOrderRepo {
	return &memoryMultiOrderRepo{
		orders:   make(map[string]Order),
		idemKeys: make(map[string]string),
	}
}

func (r *memoryMultiOrderRepo) CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error, _ StockReservationOpts) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.createCalls++
	if r.failOnChild2 && o.SupplierID == "sup-b" {
		return errors.New("simulated supplier-b downstream outage")
	}

	key := fmt.Sprintf("%s:%s:%s", o.RetailerID, o.SupplierID, o.IdempotencyKey)
	if o.IdempotencyKey != "" {
		if existingID, exists := r.idemKeys[key]; exists {
			// Spanner unique constraint violation simulation
			return fmt.Errorf("duplicate idempotency key %s (already held by order %s)", o.IdempotencyKey, existingID)
		}
		r.idemKeys[key] = o.OrderID
	}

	if o.Status == "" {
		o.Status = StatusPending
	}
	r.orders[o.OrderID] = *o
	return nil
}

func (r *memoryMultiOrderRepo) CreateOrderWithBackorder(ctx context.Context, o *Order, backorder *Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {
	return r.CreateOrder(ctx, o, emit, stockOpts)
}

func (r *memoryMultiOrderRepo) UpdateOrder(ctx context.Context, o Order, _ []DeliveryProofArtifact, _ func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.OrderID] = o
	return nil
}

func (r *memoryMultiOrderRepo) UpdateOrderWithTxn(ctx context.Context, o Order, proofs []DeliveryProofArtifact, _ func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error) error {
	return r.UpdateOrder(ctx, o, proofs, emit)
}

func (r *memoryMultiOrderRepo) GetOrder(_ context.Context, orderID string) (Order, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, found := r.orders[orderID]
	return o, found, nil
}

func (r *memoryMultiOrderRepo) GetOrderTxn(ctx context.Context, _ *spanner.ReadWriteTransaction, orderID string) (Order, bool, error) {
	return r.GetOrder(ctx, orderID)
}

func (r *memoryMultiOrderRepo) GetFiscalAttempt(_ context.Context, _, _ string) (FiscalReceiptRow, bool, error) {
	return FiscalReceiptRow{}, false, nil
}
func (r *memoryMultiOrderRepo) GetFiscalByReceiptID(_ context.Context, _ string) (FiscalReceiptRow, bool, error) {
	return FiscalReceiptRow{}, false, nil
}
func (r *memoryMultiOrderRepo) FindPendingBuyerAcceptance(_ context.Context, _ int) ([]*Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) CountFiscalAttemptsByStatus(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}
func (r *memoryMultiOrderRepo) ListRetailerOrders(_ context.Context, _ string, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ListWarehouseOrdersByDeliveryWindow(_ context.Context, _ string, _, _ time.Time, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ListDueAutoConfirmOrders(_ context.Context, _ time.Time, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ListManifestOrders(_ context.Context, _ string) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ListWarehousePreorders(_ context.Context, _ string, _, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ListOrdersForStockCommitment(_ context.Context, _ string, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ListBackorderedOrders(_ context.Context, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) ClearBackorder(_ context.Context, _ string, _ func(outbox.TxnBuffer) error, _ StockReservationOpts) error {
	return nil
}
func (r *memoryMultiOrderRepo) ListOrdersByStatus(_ context.Context, _, _ string, _ int) ([]Order, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) CreateConditionReport(_ context.Context, _ ConditionReport, _ func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *memoryMultiOrderRepo) ListConditionReports(_ context.Context, _ string) ([]ConditionReport, error) {
	return nil, nil
}
func (r *memoryMultiOrderRepo) FindSiblingDriversForOrder(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

type memoryWarehouseResolver struct{}

func (m *memoryWarehouseResolver) ResolveNearestWarehouseID(ctx context.Context, supplierID string, store proximity.StorePoint) (string, error) {
	return "wh-" + supplierID, nil
}

func TestUnifiedCheckout_MultiSupplierRetryAfterCompensation(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "true")
	t.Setenv("PEGASUSX_ENV", "test")

	repo := newMemoryMultiOrderRepo()
	svc := NewService(ServiceConfig{
		Repo:         repo,
		Warehouse:    &memoryWarehouseResolver{},
		SupplierID:   "sup-seed",
		SupplierName: "Seed Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	clientOrderID := "client-order-retry-test-12345"
	checkoutReq := UnifiedCheckoutRequest{
		ClientOrderID: clientOrderID,
		Latitude:      41.31,
		Longitude:     69.24,
		Items: []UnifiedCheckoutLineItem{
			{SkuID: "sku-a", Quantity: 2, UnitPrice: 10000, SupplierID: "sup-a"},
			{SkuID: "sku-b", Quantity: 3, UnitPrice: 20000, SupplierID: "sup-b"},
		},
	}

	// -------------------------------------------------------------
	// 1. Attempt 1: Supplier B fails during child order creation.
	// Saga compensation should run and cancel child A.
	// -------------------------------------------------------------
	repo.failOnChild2 = true
	_, err := svc.UnifiedCheckout(context.Background(), "ret-1", checkoutReq)
	if err == nil {
		t.Fatal("expected error on attempt 1 due to sup-b failure, got nil")
	}

	// Verify child A was created then cancelled
	var childAOrder *Order
	repo.mu.Lock()
	for _, o := range repo.orders {
		if o.SupplierID == "sup-a" {
			childAOrder = &o
			break
		}
	}
	repo.mu.Unlock()

	if childAOrder == nil {
		t.Fatal("expected child A to have been created before failure")
	}
	if childAOrder.Status != StatusCancelled {
		t.Fatalf("expected child A order to be CANCELLED after compensation, got %s", childAOrder.Status)
	}

	// -------------------------------------------------------------
	// 2. Attempt 2: Supplier B is recovered. Same ClientOrderID is retried.
	// Unified checkout should detect generation 0 was cancelled,
	// bump to generation :r1 without colliding with Spanner idempotency index,
	// and successfully create active orders for sup-a and sup-b.
	// -------------------------------------------------------------
	repo.failOnChild2 = false
	resp2, err := svc.UnifiedCheckout(context.Background(), "ret-1", checkoutReq)
	if err != nil {
		t.Fatalf("attempt 2 UnifiedCheckout failed: %v", err)
	}
	if resp2.Status != "ok" {
		t.Fatalf("attempt 2 status = %q, want ok", resp2.Status)
	}
	if len(resp2.SupplierOrders) != 2 {
		t.Fatalf("attempt 2 expected 2 supplier orders, got %d", len(resp2.SupplierOrders))
	}
	if resp2.Total != 80000 {
		t.Fatalf("attempt 2 expected total 80000, got %d", resp2.Total)
	}

	for _, so := range resp2.SupplierOrders {
		o, found, getErr := repo.GetOrder(context.Background(), so.OrderID)
		if getErr != nil || !found {
			t.Fatalf("failed retrieving active child order %s: %v", so.OrderID, getErr)
		}
		if o.Status != StatusPending {
			t.Fatalf("expected child order %s to be PENDING, got %s", o.OrderID, o.Status)
		}
		// Must not be the original cancelled order ID
		if o.OrderID == childAOrder.OrderID {
			t.Fatalf("attempt 2 reused cancelled child order ID %s", o.OrderID)
		}
	}

	callsAfterAttempt2 := repo.createCalls

	// -------------------------------------------------------------
	// 3. Attempt 3: Idempotent replay with same ClientOrderID.
	// Should return identical supplier orders without creating new ones.
	// -------------------------------------------------------------
	resp3, err := svc.UnifiedCheckout(context.Background(), "ret-1", checkoutReq)
	if err != nil {
		t.Fatalf("attempt 3 UnifiedCheckout failed: %v", err)
	}
	if resp3.Total != resp2.Total {
		t.Fatalf("attempt 3 total %d != attempt 2 total %d", resp3.Total, resp2.Total)
	}
	if len(resp3.SupplierOrders) != len(resp2.SupplierOrders) {
		t.Fatalf("attempt 3 supplier orders length mismatch")
	}
	for i := range resp2.SupplierOrders {
		if resp3.SupplierOrders[i].OrderID != resp2.SupplierOrders[i].OrderID {
			t.Fatalf("attempt 3 replay mismatch: order %s != %s",
				resp3.SupplierOrders[i].OrderID, resp2.SupplierOrders[i].OrderID)
		}
	}
	if repo.createCalls != callsAfterAttempt2 {
		t.Fatalf("attempt 3 made %d CreateOrder calls, want 0", repo.createCalls-callsAfterAttempt2)
	}
}
