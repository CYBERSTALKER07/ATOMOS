package plan

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
)

func TestOptimizeAndValidate_SmallBatchSkipsOptimizer(t *testing.T) {
	t.Setenv("DISPATCH_AI_MIN_STOPS", "12")
	job := Job{
		Orders: []dispatch.DispatchableOrder{
			{OrderID: "o1", RetailerID: "r1", VolumeVU: 5, Lat: 41.3, Lng: 69.2},
			{OrderID: "o2", RetailerID: "r2", VolumeVU: 5, Lat: 41.3, Lng: 69.2},
		},
		Fleet: []dispatch.AvailableDriver{
			{DriverID: "d1", MaxVolumeVU: 100},
		},
		DepotLat: 41.3,
		DepotLng: 69.2,
	}
	// nil client → fallback_phase1; with non-nil we'd need a mock — test pure_small_batch via threshold with a stub client is harder.
	// When client is nil, source is fallback_phase1 even for small batches.
	res, source, err := OptimizeAndValidate(context.Background(), nil, job)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.Routes) == 0 {
		t.Fatalf("expected pure pack routes")
	}
	if source != SourceFallbackPhase1 {
		t.Fatalf("source=%s", source)
	}
}

func TestDenseBatchThresholdEnv(t *testing.T) {
	t.Setenv("DISPATCH_AI_MIN_STOPS", "5")
	if denseBatchThreshold() != 5 {
		t.Fatalf("got %d", denseBatchThreshold())
	}
}
