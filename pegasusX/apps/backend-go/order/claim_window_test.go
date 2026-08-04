package order

import (
	"context"
	"testing"
	"time"
)

type memoryReturnPolicies struct {
	suppliers  map[string]SupplierReturnPolicy
	warehouses map[string]WarehouseReturnPolicy // key warehouseID + "\x00" + supplierID
}

func newMemoryReturnPolicies() *memoryReturnPolicies {
	return &memoryReturnPolicies{
		suppliers:  map[string]SupplierReturnPolicy{},
		warehouses: map[string]WarehouseReturnPolicy{},
	}
}

func (m *memoryReturnPolicies) key(warehouseID, supplierID string) string {
	return warehouseID + "\x00" + supplierID
}

func (m *memoryReturnPolicies) GetSupplierReturnPolicy(_ context.Context, supplierID string) (SupplierReturnPolicy, bool, error) {
	p, ok := m.suppliers[supplierID]
	return p, ok, nil
}

func (m *memoryReturnPolicies) UpsertSupplierReturnPolicy(_ context.Context, p SupplierReturnPolicy) error {
	m.suppliers[p.SupplierID] = p
	return nil
}

func (m *memoryReturnPolicies) GetWarehouseReturnPolicy(_ context.Context, warehouseID, supplierID string) (WarehouseReturnPolicy, bool, error) {
	p, ok := m.warehouses[m.key(warehouseID, supplierID)]
	return p, ok, nil
}

func (m *memoryReturnPolicies) UpsertWarehouseReturnPolicy(_ context.Context, p WarehouseReturnPolicy) error {
	m.warehouses[m.key(p.WarehouseID, p.SupplierID)] = p
	return nil
}

func TestResolveClaimWindow_SupplierBase(t *testing.T) {
	store := newMemoryReturnPolicies()
	_ = store.UpsertSupplierReturnPolicy(context.Background(), SupplierReturnPolicy{
		SupplierID: "sup-1", DefaultWindowHours: 24,
	})
	got, err := ResolveClaimWindow(context.Background(), store, "sup-1", "wh-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hours != 24 || got.Source != ClaimWindowSourceSupplier {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveClaimWindow_WarehouseLengthenOnly(t *testing.T) {
	store := newMemoryReturnPolicies()
	_ = store.UpsertSupplierReturnPolicy(context.Background(), SupplierReturnPolicy{
		SupplierID: "sup-1", DefaultWindowHours: 24,
	})
	h72 := int64(72)
	_ = store.UpsertWarehouseReturnPolicy(context.Background(), WarehouseReturnPolicy{
		WarehouseID: "wh-1", SupplierID: "sup-1",
		RetailerFileWindowHours:   &h72,
		CanOverrideRetailerWindow: true,
	})
	got, err := ResolveClaimWindow(context.Background(), store, "sup-1", "wh-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hours != 72 || got.Source != ClaimWindowSourceWarehouseOverride {
		t.Fatalf("lengthen: got %+v", got)
	}

	h8 := int64(8)
	_ = store.UpsertWarehouseReturnPolicy(context.Background(), WarehouseReturnPolicy{
		WarehouseID: "wh-1", SupplierID: "sup-1",
		RetailerFileWindowHours:   &h8,
		CanOverrideRetailerWindow: true,
	})
	got, err = ResolveClaimWindow(context.Background(), store, "sup-1", "wh-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hours != 24 || got.Source != ClaimWindowSourceSupplier {
		t.Fatalf("must not shorten: got %+v", got)
	}
}

func TestResolveClaimWindow_WarehouseWideFallback(t *testing.T) {
	store := newMemoryReturnPolicies()
	_ = store.UpsertSupplierReturnPolicy(context.Background(), SupplierReturnPolicy{
		SupplierID: "sup-1", DefaultWindowHours: 12,
	})
	h48 := int64(48)
	_ = store.UpsertWarehouseReturnPolicy(context.Background(), WarehouseReturnPolicy{
		WarehouseID: "wh-1", SupplierID: "",
		RetailerFileWindowHours:   &h48,
		CanOverrideRetailerWindow: true,
	})
	got, err := ResolveClaimWindow(context.Background(), store, "sup-1", "wh-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hours != 48 || got.Source != ClaimWindowSourceWarehouseOverride {
		t.Fatalf("got %+v", got)
	}
}

func TestApplyClaimWindowSnapshot_Idempotent(t *testing.T) {
	store := newMemoryReturnPolicies()
	_ = store.UpsertSupplierReturnPolicy(context.Background(), SupplierReturnPolicy{
		SupplierID: "sup-1", DefaultWindowHours: 8,
	})
	svc := &Service{returnPolicies: store, now: func() time.Time {
		return time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	}}
	completedAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	o := &Order{OrderID: "ord-1", SupplierID: "sup-1", WarehouseID: "wh-1"}
	if err := svc.ApplyClaimWindowSnapshot(context.Background(), o, completedAt); err != nil {
		t.Fatal(err)
	}
	if o.ClaimWindowHours != 8 || o.ClaimWindowPolicySource != ClaimWindowSourceSupplier {
		t.Fatalf("snapshot: %+v", o)
	}
	if o.ClaimWindowEndsAt == nil || !o.ClaimWindowEndsAt.Equal(completedAt.Add(8*time.Hour)) {
		t.Fatalf("ends_at=%v", o.ClaimWindowEndsAt)
	}
	firstEnds := *o.ClaimWindowEndsAt
	_ = store.UpsertSupplierReturnPolicy(context.Background(), SupplierReturnPolicy{
		SupplierID: "sup-1", DefaultWindowHours: 72,
	})
	if err := svc.ApplyClaimWindowSnapshot(context.Background(), o, completedAt); err != nil {
		t.Fatal(err)
	}
	if o.ClaimWindowHours != 8 || !o.ClaimWindowEndsAt.Equal(firstEnds) {
		t.Fatalf("idempotent overwrite: hours=%d ends=%v", o.ClaimWindowHours, o.ClaimWindowEndsAt)
	}
}
