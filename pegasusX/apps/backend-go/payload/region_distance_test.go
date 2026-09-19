package payload

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
)

type mockLocationStore struct {
	loc telemetry.DriverLocation
}

func (m *mockLocationStore) GetDriverLocation(ctx context.Context, driverID string) (telemetry.DriverLocation, bool, error) {
	if m.loc.DriverID == driverID {
		return m.loc, true, nil
	}
	return telemetry.DriverLocation{}, false, nil
}

func TestResolveRegionCode_Dynamic(t *testing.T) {
	svc := &Service{}

	t.Run("Default UZ fallback", func(t *testing.T) {
		t.Setenv("DEFAULT_MARKET_CODE", "UZ")
		t.Setenv("HOME_CELL", "cell-uz")
		got := svc.resolveRegionCode("wh-1")
		if got != "UZ-TAS" {
			t.Fatalf("expected UZ-TAS, got %q", got)
		}
	})

	t.Run("Custom Region Code via env", func(t *testing.T) {
		t.Setenv("REGION_CODE", "KZ-ALA")
		got := svc.resolveRegionCode("wh-1")
		if got != "KZ-ALA" {
			t.Fatalf("expected KZ-ALA, got %q", got)
		}
	})

	t.Run("Custom Home Cell via env", func(t *testing.T) {
		t.Setenv("REGION_CODE", "")
		t.Setenv("DEFAULT_MARKET_CODE", "KZ")
		t.Setenv("HOME_CELL", "cell-ast")
		got := svc.resolveRegionCode("wh-1")
		if got != "KZ-AST" {
			t.Fatalf("expected KZ-AST, got %q", got)
		}
	})
}

func TestBuildTruckRecommendationsLocked_HaversineDistance(t *testing.T) {
	mockLoc := &mockLocationStore{
		loc: telemetry.DriverLocation{
			DriverID:   "drv-100",
			SupplierID: "sup-1",
			Lat:        41.3111,
			Lng:        69.2797,
			ReportedAt: time.Now().UTC(),
			ReceivedAt: time.Now().UTC(),
		},
	}

	svc := &Service{
		locations: mockLoc,
		manifests: []ManifestRow{
			{
				ManifestID:  "mf-100",
				VehicleID:   "veh-100",
				DriverID:    "drv-100",
				MaxVolumeVU: 100,
			},
		},
	}

	order := OrderRow{
		OrderID: "ord-1",
		Lat:     41.3200,
		Lng:     69.2850,
	}

	recs := []ReassignRecommendation{
		{
			OrderID:    "ord-1",
			ToRoute:    "mf-100",
			ToDriverID: "drv-100",
			Score:      95,
			Reason:     "optimal_route",
		},
	}

	truckRecs := buildTruckRecommendationsLocked(svc, order, recs)
	if len(truckRecs) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(truckRecs))
	}

	dist := truckRecs[0].DistanceKm
	if dist <= 0 || dist > 5.0 {
		t.Fatalf("unexpected distance: %f", dist)
	}
}
