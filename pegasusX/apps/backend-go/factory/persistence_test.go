package factory

import (
	"testing"
	"time"
)

func TestFactoryBatchFromSnapshot_IncludesTransfers(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	snap := &PersistenceSnapshot{
		Manifests: []ManifestRow{
			{
				ManifestID:    "mf_factory_1",
				State:         manifestStateDraft,
				TransferCnt:   1,
				TotalVolumeVU: 42,
				MaxVolumeVU:   120,
				DriverID:      "drv_factory_1",
				VehicleID:     "veh_factory_1",
				CreatedAt:     now.Format(time.RFC3339Nano),
				UpdatedAt:     now.Format(time.RFC3339Nano),
			},
		},
		Transfers: []TransferRow{
			{
				TransferID: "tr_factory_1",
				OrderID:    "ord_factory_1",
				ManifestID: "mf_factory_1",
				State:      "ASSIGNED",
				TotalVU:    42,
				DriverID:   "drv_factory_1",
				VehicleID:  "veh_factory_1",
				CreatedAt:  now.Format(time.RFC3339Nano),
				UpdatedAt:  now.Format(time.RFC3339Nano),
			},
		},
	}

	batch := factoryBatchFromSnapshot("sup_demo", "fac_demo", snap)
	if len(batch.Manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(batch.Manifests))
	}
	if len(batch.Transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(batch.Transfers))
	}
	if batch.Transfers[0].TransferID != "tr_factory_1" {
		t.Fatalf("expected tr_factory_1, got %q", batch.Transfers[0].TransferID)
	}
	if batch.Transfers[0].FactoryID != "fac_demo" {
		t.Fatalf("expected factory fac_demo, got %q", batch.Transfers[0].FactoryID)
	}
	if batch.Transfers[0].SupplierID != "sup_demo" {
		t.Fatalf("expected supplier sup_demo, got %q", batch.Transfers[0].SupplierID)
	}
}

func TestTransferRowRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	original := TransferRow{
		TransferID:     "tr_factory_1",
		OrderID:        "ord_factory_1",
		ManifestID:     "mf_factory_1",
		State:          "APPROVED",
		TotalVU:        42,
		DriverID:       "drv_factory_1",
		VehicleID:      "veh_factory_1",
		ReassignDepth:  2,
		ExceptionCount: 1,
		CreatedAt:      now.Format(time.RFC3339Nano),
		UpdatedAt:      now.Format(time.RFC3339Nano),
	}
	row := factoryTransferFromRow("sup_demo", "fac_demo", original)
	restored := transferRowFromFactoryTransfer(row)
	if restored.TransferID != original.TransferID {
		t.Fatalf("transfer id drift: %q", restored.TransferID)
	}
	if restored.State != original.State {
		t.Fatalf("state drift: %q", restored.State)
	}
	if restored.TotalVU != original.TotalVU {
		t.Fatalf("total vu drift: %d", restored.TotalVU)
	}
	if restored.ReassignDepth != original.ReassignDepth {
		t.Fatalf("reassign depth drift: %d", restored.ReassignDepth)
	}
}
