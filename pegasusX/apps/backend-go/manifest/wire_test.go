package manifest_test

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

func TestFromPortalRow_MirrorsStatusAndTruckID(t *testing.T) {
	w := manifest.FromPortalRow(manifest.PortalRow{
		ManifestID:   "mf-1",
		Status:       "LOADING",
		OrdersCount:  3,
		DriverID:     "drv-1",
		DriverName:   "Driver One",
		VehicleID:    "veh-1",
		VehiclePlate: "01A001AA",
		TotalVu:      42,
		UpdatedAt:    "2026-01-01T00:00:00Z",
	})
	if w.State != "LOADING" || w.Status != "LOADING" {
		t.Fatalf("status/state mismatch: %#v", w)
	}
	if w.TruckID != "veh-1" || w.VehicleID != "veh-1" {
		t.Fatalf("vehicle ids: %#v", w)
	}
	if w.OrdersCount != 3 || w.StopCount != 3 {
		t.Fatalf("counts: %#v", w)
	}
}

func TestFromPayloadRow_MirrorsPortalFields(t *testing.T) {
	w := manifest.FromPayloadRow(manifest.PayloadRow{
		ManifestID:    "mf_payload_1",
		VehicleID:     "veh_payload_1",
		DriverID:      "drv_payload_1",
		State:         "DRAFT",
		TotalVolumeVU: 75,
		MaxVolumeVU:   140,
		StopCount:     2,
		VehiclePlate:  "01P111AA",
	})
	if w.Status != "DRAFT" || w.TotalVu != 75 {
		t.Fatalf("portal mirror fields: %#v", w)
	}
	if w.TruckID != "veh_payload_1" {
		t.Fatalf("truck_id: %q", w.TruckID)
	}
}
