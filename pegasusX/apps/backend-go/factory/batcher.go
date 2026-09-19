package factory

import (
	"sort"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

type batchableTransfer struct {
	TransferID  string
	WarehouseID string
	VolumeVU    float64
	WhLat       float64
	WhLng       float64
}

type factoryVehicle struct {
	VehicleID   string
	DriverID    string
	MaxVolumeVU float64
}

type loadStopEntry struct {
	Sequence    int
	TransferID  string
	WarehouseID string
	VolumeVU    float64
	Instruction string
}

type packedManifest struct {
	Vehicle     factoryVehicle
	Transfers   []batchableTransfer
	UsedVU      float64
	LoadOrder   []loadStopEntry
}

// PackFFDNNLIFO is First-Fit Decreasing bin-pack, nearest-neighbor from factory origin, LIFO load.
// Does not invent transfers. Oversized leftovers are unassigned.
func PackFFDNNLIFO(originLat, originLng float64, transfers []batchableTransfer, vehicles []factoryVehicle) (manifests []packedManifest, unassigned []string) {
	if len(transfers) == 0 || len(vehicles) == 0 {
		ids := make([]string, 0, len(transfers))
		for _, t := range transfers {
			ids = append(ids, t.TransferID)
		}
		return nil, ids
	}
	trucks := append([]factoryVehicle(nil), vehicles...)
	sort.Slice(trucks, func(i, j int) bool { return trucks[i].MaxVolumeVU > trucks[j].MaxVolumeVU })
	items := append([]batchableTransfer(nil), transfers...)
	sort.Slice(items, func(i, j int) bool { return items[i].VolumeVU > items[j].VolumeVU })

	type build struct {
		vehicle   factoryVehicle
		transfers []batchableTransfer
		usedVU    float64
	}
	var builds []build
	for _, t := range items {
		placed := false
		for i := range builds {
			if builds[i].usedVU+t.VolumeVU <= builds[i].vehicle.MaxVolumeVU {
				builds[i].transfers = append(builds[i].transfers, t)
				builds[i].usedVU += t.VolumeVU
				placed = true
				break
			}
		}
		if placed {
			continue
		}
		if len(builds) < len(trucks) {
			v := trucks[len(builds)]
			if t.VolumeVU > v.MaxVolumeVU {
				unassigned = append(unassigned, t.TransferID)
				continue
			}
			builds = append(builds, build{
				vehicle:   v,
				transfers: []batchableTransfer{t},
				usedVU:    t.VolumeVU,
			})
			continue
		}
		unassigned = append(unassigned, t.TransferID)
	}

	for _, b := range builds {
		sorted := nearestNeighborSort(originLat, originLng, b.transfers)
		n := len(sorted)
		load := make([]loadStopEntry, n)
		for i, tr := range sorted {
			seq := n - i
			instruction := "Load mid"
			if seq == 1 {
				instruction = "Load first — Back of Truck"
			} else if seq == n {
				instruction = "Load last — By the Doors"
			}
			load[seq-1] = loadStopEntry{
				Sequence:    seq,
				TransferID:  tr.TransferID,
				WarehouseID: tr.WarehouseID,
				VolumeVU:    tr.VolumeVU,
				Instruction: instruction,
			}
		}
		manifests = append(manifests, packedManifest{
			Vehicle:   b.vehicle,
			Transfers: sorted,
			UsedVU:    b.usedVU,
			LoadOrder: load,
		})
	}
	return manifests, unassigned
}

func nearestNeighborSort(originLat, originLng float64, transfers []batchableTransfer) []batchableTransfer {
	remain := append([]batchableTransfer(nil), transfers...)
	out := make([]batchableTransfer, 0, len(remain))
	lat, lng := originLat, originLng
	for len(remain) > 0 {
		best := 0
		bestDist := proximity.HaversineDistance(lat, lng, remain[0].WhLat, remain[0].WhLng)
		for i := 1; i < len(remain); i++ {
			d := proximity.HaversineDistance(lat, lng, remain[i].WhLat, remain[i].WhLng)
			if d < bestDist {
				best = i
				bestDist = d
			}
		}
		next := remain[best]
		out = append(out, next)
		lat, lng = next.WhLat, next.WhLng
		remain = append(remain[:best], remain[best+1:]...)
	}
	return out
}
