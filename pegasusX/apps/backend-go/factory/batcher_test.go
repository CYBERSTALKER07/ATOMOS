package factory

import "testing"

func TestPackFFDNNLIFO_EmptyQueueNoInvent(t *testing.T) {
	m, un := PackFFDNNLIFO(41, 69, nil, []factoryVehicle{{VehicleID: "v1", MaxVolumeVU: 400}})
	if len(m) != 0 || len(un) != 0 {
		t.Fatalf("empty queue must not invent manifests=%d unassigned=%d", len(m), len(un))
	}
}

func TestPackFFDNNLIFO_FFDAndLIFO(t *testing.T) {
	transfers := []batchableTransfer{
		{TransferID: "big", WarehouseID: "wh-far", VolumeVU: 200, WhLat: 41.5, WhLng: 69.5},
		{TransferID: "mid", WarehouseID: "wh-mid", VolumeVU: 80, WhLat: 41.35, WhLng: 69.3},
		{TransferID: "small", WarehouseID: "wh-near", VolumeVU: 40, WhLat: 41.32, WhLng: 69.26},
	}
	vehicles := []factoryVehicle{
		{VehicleID: "t-small", DriverID: "d1", MaxVolumeVU: 150},
		{VehicleID: "t-big", DriverID: "d2", MaxVolumeVU: 400},
	}
	manifests, unassigned := PackFFDNNLIFO(41.31, 69.24, transfers, vehicles)
	if len(unassigned) != 0 {
		t.Fatalf("unassigned=%v", unassigned)
	}
	if len(manifests) != 1 {
		for i, m := range manifests {
			t.Logf("m%d vehicle=%s used=%v n=%d ids=%v", i, m.Vehicle.VehicleID, m.UsedVU, len(m.Transfers), transferIDs(m.Transfers))
		}
		t.Fatalf("FFD should fill largest truck first, manifests=%d", len(manifests))
	}
	if manifests[0].Vehicle.VehicleID != "t-big" {
		t.Fatalf("want largest truck first, got %s", manifests[0].Vehicle.VehicleID)
	}
	if len(manifests[0].LoadOrder) != 3 {
		t.Fatalf("load order %d", len(manifests[0].LoadOrder))
	}
	// LIFO: last delivery stop loaded first (sequence 1 = back of truck = last NN stop)
	firstLoad := manifests[0].LoadOrder[0]
	if firstLoad.Sequence != 1 {
		t.Fatalf("seq=%d", firstLoad.Sequence)
	}
	lastNN := manifests[0].Transfers[len(manifests[0].Transfers)-1]
	if firstLoad.TransferID != lastNN.TransferID {
		t.Fatalf("LIFO: first load %s should be last stop %s", firstLoad.TransferID, lastNN.TransferID)
	}
}

func transferIDs(ts []batchableTransfer) []string {
	ids := make([]string, len(ts))
	for i, t := range ts {
		ids[i] = t.TransferID
	}
	return ids
}

func TestPackFFDNNLIFO_NoVehiclesAllUnassigned(t *testing.T) {
	transfers := []batchableTransfer{{TransferID: "t1", VolumeVU: 10}}
	_, un := PackFFDNNLIFO(0, 0, transfers, nil)
	if len(un) != 1 || un[0] != "t1" {
		t.Fatalf("unassigned=%v", un)
	}
}
