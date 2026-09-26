package factory

import (
	"context"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
)

func TestFactoryTransfersToOrders_UsesWarehouseAsRetailerGroup(t *testing.T) {
	rows := factoryTransfersToOrders([]factoryDispatchTransfer{
		{TransferID: "tr-1", WarehouseID: "wh-a", WarehouseName: "North", VolumeVU: 12, Lat: 41.3, Lng: 69.2},
		{TransferID: "tr-2", WarehouseID: "wh-a", WarehouseName: "North", VolumeVU: 8, Lat: 41.3, Lng: 69.2},
	})
	if len(rows) != 2 {
		t.Fatalf("want 2 geo stops, got %d", len(rows))
	}
	if rows[0].OrderID != "tr-1" || rows[0].RetailerID != "wh-a" {
		t.Fatalf("transfer must map to OrderID=transfer, RetailerID=warehouse, got %+v", rows[0])
	}
	if rows[1].WarehouseID != "wh-a" {
		t.Fatalf("warehouse group key missing: %+v", rows[1])
	}
}

func TestFactoryBuildManualAssignment_AndCapacityBuffer(t *testing.T) {
	rows := []dispatch.DispatchableOrder{
		{OrderID: "tr-1", RetailerID: "wh-a", VolumeVU: 80, Lat: 41, Lng: 69},
		{OrderID: "tr-2", RetailerID: "wh-a", VolumeVU: 80, Lat: 41, Lng: 69},
	}
	assignment := factoryBuildManualAssignment(rows, []FactoryDispatchRoute{{
		DriverID:    "drv-1",
		TransferIDs: []string{"tr-1", "tr-2"},
	}}, map[string]float64{"drv-1": 100})
	if assignment == nil || len(assignment.Routes) != 1 {
		t.Fatalf("manual assignment: %+v", assignment)
	}
	warnings := factoryCapacityFromDispatch(assignment, map[string]float64{"drv-1": 100}, rows)
	if len(warnings) == 0 {
		t.Fatal("160 VU on 100*0.95 truck must warn")
	}
	if warnings[0].DriverID != "drv-1" {
		t.Fatalf("warning=%+v", warnings[0])
	}
}

func TestFactoryDispatchFingerprint_StableAndSensitive(t *testing.T) {
	rows := []dispatch.DispatchableOrder{
		{OrderID: "tr-1", VolumeVU: 10},
		{OrderID: "tr-2", VolumeVU: 20},
	}
	fleet := []dispatch.FleetDriverInput{{DriverID: "d1", VehicleID: "v1", MaxVolumeVU: 150, IsActive: true}}
	a := factoryDispatchFingerprint(rows, fleet, nil)
	b := factoryDispatchFingerprint(rows, fleet, nil)
	if a == "" || a != b {
		t.Fatalf("fingerprint must be stable, got %q %q", a, b)
	}
	rows[0].VolumeVU = 11
	c := factoryDispatchFingerprint(rows, fleet, nil)
	if c == a {
		t.Fatal("volume change must bust fingerprint")
	}
}

func TestExecuteFactoryDispatch_NilSpannerUnavailable(t *testing.T) {
	svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})
	_, err := svc.ExecuteFactoryDispatch(context.Background(), FactoryDispatchRequest{
		FactoryID:  "fac-1",
		SupplierID: "sup-1",
		Mode:       "AUTO",
	})
	if err == nil || !strings.Contains(err.Error(), "dispatch_unavailable") {
		t.Fatalf("want dispatch_unavailable, got %v", err)
	}
}

func TestEmptyFactoryDispatchResult_NeverInvent(t *testing.T) {
	out := emptyFactoryDispatchResult("fac-1", "sup-1", DispatchAlgoPickN)
	if out.CreatedManifestCount != 0 || out.ManifestsCreated != 0 || out.ManifestID != "" {
		t.Fatalf("empty result invented manifests: %+v", out)
	}
	if out.OptimizerClass != OptimizerHeuristic {
		t.Fatalf("honesty class: %+v", out)
	}
}
