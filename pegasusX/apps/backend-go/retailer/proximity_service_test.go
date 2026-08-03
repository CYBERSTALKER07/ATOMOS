package retailer

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePerimeterStore struct {
	sets map[string]map[string]struct{}
}

func newFakePerimeterStore() *fakePerimeterStore {
	return &fakePerimeterStore{sets: make(map[string]map[string]struct{})}
}

func (f *fakePerimeterStore) ReplaceSet(_ context.Context, key string, members []string, _ time.Duration) error {
	set := make(map[string]struct{}, len(members))
	for _, member := range members {
		set[member] = struct{}{}
	}
	f.sets[key] = set
	return nil
}

func (f *fakePerimeterStore) SIsMember(_ context.Context, key string, member string) (bool, error) {
	set := f.sets[key]
	if set == nil {
		return false, nil
	}
	_, ok := set[member]
	return ok, nil
}

func (f *fakePerimeterStore) Exists(_ context.Context, key string) (bool, error) {
	set := f.sets[key]
	return len(set) > 0, nil
}

func TestRetailerProximity_PrecomputeAndMembership(t *testing.T) {
	store := newFakePerimeterStore()
	svc := NewRetailerProximityService(store, RetailerProximityConfig{Resolution: 9})

	snapshot, err := svc.PrecomputeDeliveryZoneForCenter(context.Background(), 41.2995, 69.2401, 10)
	if err != nil {
		t.Fatalf("precompute delivery zone: %v", err)
	}
	if snapshot.Cells == 0 {
		t.Fatalf("expected non-empty perimeter cells")
	}
	if snapshot.CompactedCells == 0 {
		t.Fatalf("expected non-empty compacted perimeter cells")
	}

	cell, err := svc.CellForCoordinate(41.2995, 69.2401)
	if err != nil {
		t.Fatalf("derive center cell: %v", err)
	}
	if err := svc.IsRetailerInZone(context.Background(), cell); err != nil {
		t.Fatalf("expected center cell inside perimeter, got: %v", err)
	}
}

func TestRetailerProximity_MissingPerimeterFailsClosed(t *testing.T) {
	store := newFakePerimeterStore()
	svc := NewRetailerProximityService(store, RetailerProximityConfig{Resolution: 9})

	cell, err := svc.CellForCoordinate(41.2995, 69.2401)
	if err != nil {
		t.Fatalf("derive cell: %v", err)
	}
	err = svc.IsRetailerInZone(context.Background(), cell)
	if !errors.Is(err, ErrZoneMiss) {
		t.Fatalf("expected ErrZoneMiss, got: %v", err)
	}
}
