package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func TestDriverLocation_IsLive(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	// Missing driver ID
	locNoDriver := DriverLocation{
		SupplierID: "sup-1",
		ReceivedAt: now.Add(-5 * time.Second),
	}
	if locNoDriver.IsLive(now) {
		t.Fatal("expected IsLive=false when DriverID is empty")
	}

	// Missing supplier ID
	locNoSupplier := DriverLocation{
		DriverID:   "drv-1",
		ReceivedAt: now.Add(-5 * time.Second),
	}
	if locNoSupplier.IsLive(now) {
		t.Fatal("expected IsLive=false when SupplierID is empty")
	}

	// Zero ReceivedAt
	locNoTime := DriverLocation{
		DriverID:   "drv-1",
		SupplierID: "sup-1",
	}
	if locNoTime.IsLive(now) {
		t.Fatal("expected IsLive=false when ReceivedAt is zero")
	}

	// Fresh location within default 30s
	locFresh := DriverLocation{
		DriverID:   "drv-1",
		SupplierID: "sup-1",
		ReceivedAt: now.Add(-10 * time.Second),
	}
	if !locFresh.IsLive(now) {
		t.Fatal("expected IsLive=true for 10s old location (default stale 30s)")
	}

	// Stale location past 30s
	locStale := DriverLocation{
		DriverID:   "drv-1",
		SupplierID: "sup-1",
		ReceivedAt: now.Add(-35 * time.Second),
	}
	if locStale.IsLive(now) {
		t.Fatal("expected IsLive=false for 35s old location")
	}

	// Custom StaleAfterSeconds (e.g. 60s)
	locCustom := DriverLocation{
		DriverID:          "drv-1",
		SupplierID:        "sup-1",
		ReceivedAt:        now.Add(-45 * time.Second),
		StaleAfterSeconds: 60,
	}
	if !locCustom.IsLive(now) {
		t.Fatal("expected IsLive=true for 45s old location when StaleAfterSeconds=60")
	}

	// Clock skew into future (ReceivedAt > now)
	locFuture := DriverLocation{
		DriverID:   "drv-1",
		SupplierID: "sup-1",
		ReceivedAt: now.Add(5 * time.Second),
	}
	if !locFuture.IsLive(now) {
		t.Fatal("expected IsLive=true for forward clock skew")
	}
}

func TestCacheLastLocationStore_SaveAndGet(t *testing.T) {
	t.Parallel()
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	store := NewCacheLastLocationStore(c, 5*time.Minute)

	ctx := context.Background()
	now := time.Now().UTC()

	loc := DriverLocation{
		DriverID:   "drv-101",
		SupplierID: "sup-202",
		Lat:        41.3111,
		Lng:        69.2797,
		ReportedAt: now,
	}

	if err := store.SaveDriverLocation(ctx, loc); err != nil {
		t.Fatalf("SaveDriverLocation failed: %v", err)
	}

	retrieved, found, err := store.GetDriverLocation(ctx, "drv-101")
	if err != nil {
		t.Fatalf("GetDriverLocation error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}

	if retrieved.DriverID != "drv-101" || retrieved.SupplierID != "sup-202" {
		t.Fatalf("unexpected driver/supplier: %+v", retrieved)
	}
	if retrieved.Latitude != 41.3111 || retrieved.Longitude != 69.2797 {
		t.Fatalf("expected coordinates normalized: lat=%v, lng=%v", retrieved.Latitude, retrieved.Longitude)
	}
	if retrieved.Lat != 41.3111 || retrieved.Lng != 69.2797 {
		t.Fatalf("expected Lat/Lng normalized: Lat=%v, Lng=%v", retrieved.Lat, retrieved.Lng)
	}
	if retrieved.StaleAfterSeconds != defaultStaleAfterSeconds {
		t.Fatalf("expected default StaleAfterSeconds=%d, got %d", defaultStaleAfterSeconds, retrieved.StaleAfterSeconds)
	}
}

func TestCacheLastLocationStore_OutOfOrderRejection(t *testing.T) {
	t.Parallel()
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	store := NewCacheLastLocationStore(c, 0) // verifies default TTL

	ctx := context.Background()
	baseTime := time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)

	// Save initial recent point at T0
	recentLoc := DriverLocation{
		DriverID:   "drv-replay",
		SupplierID: "sup-1",
		Lat:        41.3200,
		Lng:        69.2800,
		ReportedAt: baseTime,
		ReceivedAt: baseTime,
	}
	if err := store.SaveDriverLocation(ctx, recentLoc); err != nil {
		t.Fatalf("SaveDriverLocation failed: %v", err)
	}

	// Incoming older point from T0 - 10s (e.g. delayed UDP/cellular retry)
	olderLoc := DriverLocation{
		DriverID:   "drv-replay",
		SupplierID: "sup-1",
		Lat:        41.3000,
		Lng:        69.2600,
		ReportedAt: baseTime.Add(-10 * time.Second),
		ReceivedAt: baseTime.Add(2 * time.Second),
	}
	if err := store.SaveDriverLocation(ctx, olderLoc); err != nil {
		t.Fatalf("SaveDriverLocation for olderLoc failed: %v", err)
	}

	// The store must preserve the more recent location (41.3200, 69.2800)
	current, found, err := store.GetDriverLocation(ctx, "drv-replay")
	if err != nil || !found {
		t.Fatalf("GetDriverLocation error: %v, found: %v", err, found)
	}
	if current.Lat != 41.3200 || current.Lng != 69.2800 {
		t.Fatalf("out-of-order rejection failed: expected Lat 41.3200, got %v", current.Lat)
	}

	// Incoming newer point at T0 + 5s must overwrite
	newerLoc := DriverLocation{
		DriverID:   "drv-replay",
		SupplierID: "sup-1",
		Lat:        41.3300,
		Lng:        69.2900,
		ReportedAt: baseTime.Add(5 * time.Second),
		ReceivedAt: baseTime.Add(5 * time.Second),
	}
	if err := store.SaveDriverLocation(ctx, newerLoc); err != nil {
		t.Fatalf("SaveDriverLocation for newerLoc failed: %v", err)
	}

	updated, found, err := store.GetDriverLocation(ctx, "drv-replay")
	if err != nil || !found {
		t.Fatalf("GetDriverLocation error: %v, found: %v", err, found)
	}
	if updated.Lat != 41.3300 || updated.Lng != 69.2900 {
		t.Fatalf("expected newer location to overwrite: got Lat=%v", updated.Lat)
	}
}

func TestCacheLastLocationStore_Validation(t *testing.T) {
	t.Parallel()
	backend := cache.NewInMemoryBackend()
	c := cache.New(backend, nil)
	store := NewCacheLastLocationStore(c, time.Minute)

	ctx := context.Background()

	// Missing driver_id returns error
	err := store.SaveDriverLocation(ctx, DriverLocation{
		SupplierID: "sup-1",
		Lat:        41.0,
		Lng:        69.0,
	})
	if err == nil {
		t.Fatal("expected error for empty driver_id")
	}

	// Empty driver_id on get returns false, nil
	_, found, err := store.GetDriverLocation(ctx, "   ")
	if err != nil || found {
		t.Fatalf("expected found=false and err=nil, got found=%v, err=%v", found, err)
	}

	// Nil store safe guards
	var nilStore *CacheLastLocationStore
	if err := nilStore.SaveDriverLocation(ctx, DriverLocation{DriverID: "drv-1"}); err != nil {
		t.Fatalf("nil store save should return nil: %v", err)
	}
	_, found, err = nilStore.GetDriverLocation(ctx, "drv-1")
	if err != nil || found {
		t.Fatalf("nil store get should return false: %v", err)
	}
}
