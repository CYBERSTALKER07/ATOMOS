package dispatch

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestBinPackHierarchical_MergesCells(t *testing.T) {
	t.Setenv("DISPATCH_HIERARCHICAL_H3", "true")
	t.Setenv("DISPATCH_HIERARCHICAL_MIN_ORDERS", "4")
	fleet := []AvailableDriver{
		{DriverID: "d1", MaxVolumeVU: 200},
		{DriverID: "d2", MaxVolumeVU: 200},
	}
	orders := []DispatchableOrder{
		{OrderID: "o1", RetailerID: "r1", VolumeVU: 10, Lat: 41.30, Lng: 69.20},
		{OrderID: "o2", RetailerID: "r2", VolumeVU: 10, Lat: 41.31, Lng: 69.21},
		{OrderID: "o3", RetailerID: "r3", VolumeVU: 10, Lat: 41.50, Lng: 69.50},
		{OrderID: "o4", RetailerID: "r4", VolumeVU: 10, Lat: 41.51, Lng: 69.51},
	}
	res := BinPackHierarchical(orders, fleet, H3CellLookup, BinPackOptions{SkipLocalSearch: true})
	if res == nil || len(res.Routes) == 0 {
		t.Fatalf("expected hierarchical routes")
	}
	assigned := 0
	for _, r := range res.Routes {
		assigned += len(r.Orders)
	}
	if assigned+len(res.Orphans) != 4 {
		t.Fatalf("assigned=%d orphans=%d", assigned, len(res.Orphans))
	}
}

func TestBuildSpatialSubproblems_2000Outlets40Vans(t *testing.T) {
	rng := rand.New(rand.NewSource(20261113))
	const (
		outletCount = 2000
		vanCount    = 40
	)
	orders := make([]DispatchableOrder, outletCount)
	totalVol := 0.0
	for i := 0; i < outletCount; i++ {
		vu := 1.0 + rng.Float64()*2.0
		totalVol += vu
		orders[i] = DispatchableOrder{
			OrderID:      fmt.Sprintf("ord-%04d", i),
			RetailerID:   fmt.Sprintf("ret-%04d", i),
			RetailerName: fmt.Sprintf("Shop %d", i),
			Lat:          41.22 + rng.Float64()*0.18, // single Tashkent metro cluster (old H3 res 5 collapsed all!)
			Lng:          69.15 + rng.Float64()*0.20,
			VolumeVU:     vu,
		}
	}
	fleet := make([]AvailableDriver, vanCount)
	for i := 0; i < vanCount; i++ {
		fleet[i] = AvailableDriver{
			DriverID:    fmt.Sprintf("drv-%02d", i),
			VehicleID:   fmt.Sprintf("veh-%02d", i),
			StartLat:    41.311,
			StartLng:    69.240,
			MaxVolumeVU: (totalVol / float64(vanCount)) * 1.4,
		}
	}

	plan := BuildSpatialSubproblems(orders, fleet, SpatialPartitionOptions{
		DepotLat:             41.311,
		DepotLng:             69.240,
		MaxStopsPerPartition: DefaultPartitionTargetStops,
		MaxStopsPerRoute:     60,
		TetrisBuffer:         TetrisBuffer,
	})
	if len(plan.DeferredOverflow) != 0 {
		t.Fatalf("unexpected deferred overflow: %d", len(plan.DeferredOverflow))
	}
	if len(plan.Subproblems) < 8 || len(plan.Subproblems) > vanCount {
		t.Fatalf("expected 8..%d spatial subproblems for 2000 stops, got %d", vanCount, len(plan.Subproblems))
	}

	seenOrders := make(map[string]bool, outletCount)
	seenDrivers := make(map[string]bool, vanCount)
	for _, sub := range plan.Subproblems {
		if len(sub.Orders) == 0 || len(sub.Orders) > MaxPartitionStops {
			t.Fatalf("subproblem %d has %d stops (must be 1..%d)", sub.Index, len(sub.Orders), MaxPartitionStops)
		}
		if len(sub.Fleet) == 0 {
			t.Fatalf("subproblem %d got zero vans", sub.Index)
		}
		for _, o := range sub.Orders {
			if seenOrders[o.OrderID] {
				t.Fatalf("duplicate order %s across partitions", o.OrderID)
			}
			seenOrders[o.OrderID] = true
		}
		for _, d := range sub.Fleet {
			if seenDrivers[d.DriverID] {
				t.Fatalf("duplicate driver %s across partitions", d.DriverID)
			}
			seenDrivers[d.DriverID] = true
		}
	}
	if len(seenOrders) != outletCount {
		t.Fatalf("partitioned orders=%d, want %d", len(seenOrders), outletCount)
	}
	if len(seenDrivers) != vanCount {
		t.Fatalf("allocated vans=%d, want all %d proportionally allocated", len(seenDrivers), vanCount)
	}
}

func TestBuildSpatialSubproblems_MultiDepotHazmatAndOverflowDeferral(t *testing.T) {
	// 240 orders across 2 depots:
	// - Tashkent (180 orders, 2 vans of max 60/part = 120 kept, 60 deferred deterministically; hazmat stop gets hazmat van)
	// - Samarkand (60 orders, 1 van of max 60/part = 60 kept)
	orders := make([]DispatchableOrder, 240)
	for i := 0; i < len(orders); i++ {
		if i < 180 {
			lat := 41.27 + float64(i%12)*0.008
			lng := 69.18 + float64(i/12)*0.008
			if i == 0 {
				lat, lng = 41.31, 69.24 // inside Tashkent depot core
			}
			orders[i] = DispatchableOrder{
				OrderID:     fmt.Sprintf("md-%03d", i),
				RetailerID:  fmt.Sprintf("r-%03d", i),
				Lat:         lat,
				Lng:         lng,
				VolumeVU:    2.0,
				IsHazardous: i == 0,
			}
		} else {
			k := i - 180
			orders[i] = DispatchableOrder{
				OrderID:    fmt.Sprintf("md-%03d", i),
				RetailerID: fmt.Sprintf("r-%03d", i),
				Lat:        39.65 + float64(k%6)*0.002,
				Lng:        66.96 + float64(k/6)*0.002,
				VolumeVU:   2.0,
			}
		}
	}
	fleet := []AvailableDriver{
		{DriverID: "tash-std", StartLat: 41.31, StartLng: 69.24, MaxVolumeVU: 180},
		{DriverID: "tash-haz", StartLat: 41.31, StartLng: 69.24, MaxVolumeVU: 180, HazmatCertified: true},
		{DriverID: "sam-1", StartLat: 39.66, StartLng: 66.97, MaxVolumeVU: 180},
	}

	plan := BuildSpatialSubproblems(orders, fleet, SpatialPartitionOptions{
		MaxStopsPerPartition: 60, // Tashkent: 2*60=120 kept (60 deferred); Samarkand: 1*60=60 kept
		MaxStopsPerRoute:     60,
	})
	kept := 0
	hazmatPaired := false
	for _, sub := range plan.Subproblems {
		kept += len(sub.Orders)
		for _, o := range sub.Orders {
			if o.IsHazardous {
				for _, d := range sub.Fleet {
					if d.HazmatCertified {
						hazmatPaired = true
					}
				}
			}
		}
	}
	if kept != 180 || len(plan.DeferredOverflow) != 60 {
		t.Fatalf("kept=%d deferred=%d, want 180 and 60", kept, len(plan.DeferredOverflow))
	}
	if !hazmatPaired {
		t.Fatalf("expected hazmat stop partition to receive hazmat-certified driver")
	}
	// Deferred overflow must be strictly deterministic (sorted by OrderID).
	for i := 1; i < len(plan.DeferredOverflow); i++ {
		if plan.DeferredOverflow[i-1].OrderID >= plan.DeferredOverflow[i].OrderID {
			t.Fatalf("deferred overflow not strictly sorted at %d", i)
		}
	}
}
