package inventory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

// inMemoryInventoryRepo is a thread-safe in-memory implementation of Repository
// that strictly enforces CAS version checks and inventory bounds.
type inMemoryInventoryRepo struct {
	mu     sync.Mutex
	levels map[string]*Level
}

func newInMemoryInventoryRepo() *inMemoryInventoryRepo {
	return &inMemoryInventoryRepo{
		levels: make(map[string]*Level),
	}
}

func (r *inMemoryInventoryRepo) ListByWarehouse(ctx context.Context, warehouseID string) ([]Level, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []Level
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID {
			res = append(res, *l)
		}
	}
	return res, nil
}

func (r *inMemoryInventoryRepo) ListBySupplier(ctx context.Context, supplierID string) ([]Level, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []Level
	for _, l := range r.levels {
		if l.SupplierID == supplierID {
			res = append(res, *l)
		}
	}
	return res, nil
}

func (r *inMemoryInventoryRepo) Get(ctx context.Context, inventoryID string) (*Level, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return nil, fmt.Errorf("inventory %s not found", inventoryID)
	}
	cp := *l
	return &cp, nil
}

func (r *inMemoryInventoryRepo) GetByWarehouseProduct(ctx context.Context, warehouseID, productID string) (*Level, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID && l.ProductID == productID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *inMemoryInventoryRepo) Upsert(ctx context.Context, l Level) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := l
	cp.UpdatedAt = time.Now().UTC()
	r.levels[l.InventoryID] = &cp
	return nil
}

func (r *inMemoryInventoryRepo) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return fmt.Errorf("read inventory %s: not found", inventoryID)
	}
	if l.Version != expectedVersion {
		return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, l.Version)
	}
	newOnHand := l.QuantityOnHand + delta
	if newOnHand < 0 {
		return fmt.Errorf("insufficient stock for %s: on_hand %d + delta %d would go negative", inventoryID, l.QuantityOnHand, delta)
	}
	l.QuantityOnHand = newOnHand
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *inMemoryInventoryRepo) ReserveForOrder(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return fmt.Errorf("read inventory %s: not found", inventoryID)
	}
	if l.Version != expectedVersion {
		return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, l.Version)
	}
	available := l.QuantityOnHand - l.QuantityReserved
	if available < quantity {
		return fmt.Errorf("inventory %s insufficient stock: %d available, %d requested", inventoryID, available, quantity)
	}
	l.QuantityReserved += quantity
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *inMemoryInventoryRepo) ReserveTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID && l.ProductID == productID {
			available := l.QuantityOnHand - l.QuantityReserved
			if available < quantity {
				return fmt.Errorf("insufficient stock for product %s in warehouse %s", productID, warehouseID)
			}
			l.QuantityReserved += quantity
			l.Version++
			return nil
		}
	}
	return fmt.Errorf("inventory not found for warehouse %s product %s", warehouseID, productID)
}

func (r *inMemoryInventoryRepo) ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return fmt.Errorf("read inventory %s: not found", inventoryID)
	}
	if l.Version != expectedVersion {
		return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, l.Version)
	}
	releaseQty := quantity
	if releaseQty > l.QuantityReserved {
		releaseQty = l.QuantityReserved
	}
	l.QuantityOnHand += releaseQty
	l.QuantityReserved -= releaseQty
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

func TestLevel_Available(t *testing.T) {
	cases := []struct {
		name      string
		onHand    int64
		reserved  int64
		expected  int64
	}{
		{name: "standard availability", onHand: 100, reserved: 20, expected: 80},
		{name: "fully reserved", onHand: 100, reserved: 100, expected: 0},
		{name: "over-reserved clamp to zero", onHand: 50, reserved: 80, expected: 0},
		{name: "zero on hand", onHand: 0, reserved: 0, expected: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := Level{QuantityOnHand: tc.onHand, QuantityReserved: tc.reserved}
			if got := l.Available(); got != tc.expected {
				t.Fatalf("Available() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestLevel_IsBelowThreshold(t *testing.T) {
	cases := []struct {
		name      string
		onHand    int64
		threshold int64
		expected  bool
	}{
		{name: "above threshold", onHand: 50, threshold: 20, expected: false},
		{name: "at threshold boundary", onHand: 20, threshold: 20, expected: true},
		{name: "below threshold", onHand: 10, threshold: 20, expected: true},
		{name: "zero on hand", onHand: 0, threshold: 10, expected: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := Level{QuantityOnHand: tc.onHand, ReorderThreshold: tc.threshold}
			if got := l.IsBelowThreshold(); got != tc.expected {
				t.Fatalf("IsBelowThreshold() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestStockLot_Available(t *testing.T) {
	cases := []struct {
		name      string
		onHand    int64
		allocated int64
		expected  int64
	}{
		{name: "unallocated stock", onHand: 50, allocated: 15, expected: 35},
		{name: "all allocated", onHand: 50, allocated: 50, expected: 0},
		{name: "over-allocated clamp to zero", onHand: 10, allocated: 25, expected: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lot := StockLot{QuantityOnHand: tc.onHand, QuantityAllocated: tc.allocated}
			if got := lot.Available(); got != tc.expected {
				t.Fatalf("Available() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestService_TwoPhaseReservationAndRelease_Lifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newInMemoryInventoryRepo()
	svc := NewService(repo, nil, nil)

	inv := Level{
		InventoryID:      "inv-sku-001",
		ProductID:        "prod-101",
		WarehouseID:      "wh-central",
		SupplierID:       "sup-alpha",
		QuantityOnHand:   100,
		QuantityReserved: 0,
		ReorderThreshold: 10,
		Version:          1,
	}
	if err := svc.Upsert(ctx, inv); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// 1. Phase 1 - Prepare/Reserve 30 units with expectedVersion = 1
	if err := svc.ReserveForOrder(ctx, "inv-sku-001", 30, 1); err != nil {
		t.Fatalf("reserve phase 1 failed: %v", err)
	}
	cur, err := repo.Get(ctx, "inv-sku-001")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if cur.QuantityReserved != 30 || cur.Version != 2 {
		t.Fatalf("unexpected state after reserve: reserved=%d, version=%d", cur.QuantityReserved, cur.Version)
	}
	if cur.Available() != 70 {
		t.Fatalf("expected 70 available, got %d", cur.Available())
	}

	// 2. CAS Version Check: stale expectedVersion = 1 must be rejected
	err = svc.ReserveForOrder(ctx, "inv-sku-001", 10, 1)
	if err == nil {
		t.Fatal("expected version conflict error for stale version 1, got nil")
	}

	// 3. Phase 1b - Reserve another 40 units with valid version 2
	if err := svc.ReserveForOrder(ctx, "inv-sku-001", 40, 2); err != nil {
		t.Fatalf("reserve phase 1b failed: %v", err)
	}
	cur, _ = repo.Get(ctx, "inv-sku-001")
	if cur.QuantityReserved != 70 || cur.Version != 3 || cur.Available() != 30 {
		t.Fatalf("unexpected state: reserved=%d, ver=%d, avail=%d", cur.QuantityReserved, cur.Version, cur.Available())
	}

	// 4. Insufficient stock: request 50 when only 30 available
	err = svc.ReserveForOrder(ctx, "inv-sku-001", 50, 3)
	if err == nil {
		t.Fatal("expected insufficient stock error, got nil")
	}

	// 5. Phase 2 - Rollback / Release on cancellation: release 30 units with version 3
	if err := svc.ReleaseReservation(ctx, "inv-sku-001", 30, 3); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	cur, _ = repo.Get(ctx, "inv-sku-001")
	if cur.QuantityReserved != 40 || cur.Version != 4 {
		t.Fatalf("unexpected state after release: reserved=%d, ver=%d", cur.QuantityReserved, cur.Version)
	}

	// 6. Phase 2 - Stock Outflow / Adjustment: ship 40 units
	if err := svc.AdjustStock(ctx, "inv-sku-001", -40, 4); err != nil {
		t.Fatalf("adjust stock failed: %v", err)
	}
	cur, _ = repo.Get(ctx, "inv-sku-001")
	if cur.QuantityOnHand != 90 || cur.Version != 5 {
		t.Fatalf("unexpected state after adjust: onHand=%d, ver=%d", cur.QuantityOnHand, cur.Version)
	}
}

func TestService_FlashSaleConcurrentReservationCollisions(t *testing.T) {
	ctx := context.Background()
	repo := newInMemoryInventoryRepo()
	svc := NewService(repo, nil, nil)

	// High-demand flash sale: 100 units total.
	// 50 concurrent buyers each attempting to reserve 5 units.
	// Total demand = 250 units. Available = 100 units.
	// Exactly 20 buyers must succeed, 30 must fail.
	inv := Level{
		InventoryID:      "inv-flash-sale-999",
		ProductID:        "prod-flash-phone",
		WarehouseID:      "wh-tashkent",
		SupplierID:       "sup-flash",
		QuantityOnHand:   100,
		QuantityReserved: 0,
		ReorderThreshold: 10,
		Version:          1,
	}
	if err := svc.Upsert(ctx, inv); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	concurrency := 50
	unitsPerOrder := int64(5)

	var wg sync.WaitGroup
	var successCount int64
	var collisionRetries int64
	var exhaustedCount int64
	var countMu sync.Mutex

	maxRetries := 25

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(buyerID int) {
			defer wg.Done()

			retries := 0
			for {
				cur, err := repo.Get(ctx, "inv-flash-sale-999")
				if err != nil {
					return
				}
				if cur.Available() < unitsPerOrder {
					countMu.Lock()
					exhaustedCount++
					countMu.Unlock()
					return
				}

				// Attempt CAS reservation
				err = svc.ReserveForOrder(ctx, "inv-flash-sale-999", unitsPerOrder, cur.Version)
				if err == nil {
					countMu.Lock()
					successCount++
					countMu.Unlock()
					return
				}

				// Collision occurred (version conflict) -> retry
				retries++
				countMu.Lock()
				collisionRetries++
				countMu.Unlock()

				if retries >= maxRetries {
					countMu.Lock()
					exhaustedCount++
					countMu.Unlock()
					return
				}
				// Jittered backoff to simulate Spanner retry behavior
				time.Sleep(time.Duration(retries) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	final, err := repo.Get(ctx, "inv-flash-sale-999")
	if err != nil {
		t.Fatalf("get final inventory failed: %v", err)
	}

	t.Logf("Flash sale results: successes=%d, exhausted=%d, collisions=%d, final_reserved=%d, final_version=%d",
		successCount, exhaustedCount, collisionRetries, final.QuantityReserved, final.Version)

	// Invariant 1: Exactly 20 successful orders (20 * 5 = 100 units reserved)
	if successCount != 20 {
		t.Fatalf("expected exactly 20 successful reservations, got %d", successCount)
	}

	// Invariant 2: Total reserved must equal 100
	if final.QuantityReserved != 100 {
		t.Fatalf("expected QuantityReserved == 100, got %d", final.QuantityReserved)
	}

	// Invariant 3: Remaining available stock must be 0
	if final.Available() != 0 {
		t.Fatalf("expected 0 available stock, got %d", final.Available())
	}

	// Invariant 4: No overselling occurred (Reserved <=OnHand)
	if final.QuantityReserved > final.QuantityOnHand {
		t.Fatalf("oversell detected! reserved=%d > onHand=%d", final.QuantityReserved, final.QuantityOnHand)
	}

	// Invariant 5: Exactly 30 buyers rejected due to stock exhaustion
	if exhaustedCount != 30 {
		t.Fatalf("expected 30 rejected buyers, got %d", exhaustedCount)
	}

	// Invariant 6: Concurrent cancellation recovery
	// Simulate 5 buyers cancelling their flash sale orders concurrently
	var cancelWG sync.WaitGroup
	cancelUnits := int64(5)
	cancellations := 5

	for c := 0; c < cancellations; c++ {
		cancelWG.Add(1)
		go func() {
			defer cancelWG.Done()
			for {
				cur, err := repo.Get(ctx, "inv-flash-sale-999")
				if err != nil {
					return
				}
				err = svc.ReleaseReservation(ctx, "inv-flash-sale-999", cancelUnits, cur.Version)
				if err == nil {
					return
				}
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}
	cancelWG.Wait()

	afterCancel, err := repo.Get(ctx, "inv-flash-sale-999")
	if err != nil {
		t.Fatalf("get after cancel failed: %v", err)
	}

	// 5 cancellations * 5 units = 25 units freed
	expectedReserved := int64(100 - (cancellations * int(cancelUnits))) // 75
	if afterCancel.QuantityReserved != expectedReserved {
		t.Fatalf("expected %d reserved after cancellations, got %d", expectedReserved, afterCancel.QuantityReserved)
	}
	t.Logf("After concurrent cancellation: reserved=%d, available=%d", afterCancel.QuantityReserved, afterCancel.Available())
}

func TestService_CASVersionConflict_MultipleCollisions(t *testing.T) {
	ctx := context.Background()
	repo := newInMemoryInventoryRepo()
	svc := NewService(repo, nil, nil)

	inv := Level{
		InventoryID:      "inv-cas-1",
		ProductID:        "prod-flash-laptop",
		WarehouseID:      "wh-samarkand",
		SupplierID:       "sup-gadgets",
		QuantityOnHand:   100,
		QuantityReserved: 0,
		ReorderThreshold: 5,
		Version:          1,
	}
	if err := svc.Upsert(ctx, inv); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	concurrency := 10
	startGate := make(chan struct{})
	var wg sync.WaitGroup

	var successes int64
	var conflicts int64
	var countMu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startGate // Synchronized simultaneous attempt
			err := svc.ReserveForOrder(ctx, "inv-cas-1", 5, 1)
			countMu.Lock()
			defer countMu.Unlock()
			if err == nil {
				successes++
			} else {
				conflicts++
			}
		}()
	}

	close(startGate) // Release all 10 goroutines at the exact same instant
	wg.Wait()

	if successes != 1 {
		t.Fatalf("expected exactly 1 success out of %d concurrent attempts, got %d", concurrency, successes)
	}
	if conflicts != int64(concurrency-1) {
		t.Fatalf("expected exactly %d CAS version conflicts, got %d", concurrency-1, conflicts)
	}

	final, err := repo.Get(ctx, "inv-cas-1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if final.QuantityReserved != 5 || final.Version != 2 {
		t.Fatalf("final state mismatch: reserved=%d, version=%d", final.QuantityReserved, final.Version)
	}
	if final.Available() != 95 {
		t.Fatalf("available mismatch: got %d, want 95", final.Available())
	}
}

