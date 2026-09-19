package payload

import "strings"

// BoardManifestStates is the GS-U7 payload board dictionary.
// COMPLETED is not a board column. Empty truck_status is not invented as DRAFT.
var BoardManifestStates = []string{
	payloadManifestStateDraft,
	payloadManifestStateLoading,
	payloadManifestStateSealed,
	payloadManifestStateDispatched,
}

// CanonicalBoardState returns a board column key or "" (not a column).
func CanonicalBoardState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case payloadManifestStateDraft:
		return payloadManifestStateDraft
	case payloadManifestStateLoading:
		return payloadManifestStateLoading
	case payloadManifestStateSealed:
		return payloadManifestStateSealed
	case payloadManifestStateDispatched:
		return payloadManifestStateDispatched
	default:
		return ""
	}
}

// BoardColumn is one DRAFT/LOADING/SEALED/DISPATCHED group.
type BoardColumn struct {
	State  string               `json:"state"`
	Trucks []payloaderTruckWire `json:"trucks"`
}

// GroupBoardColumns always returns the four board states, even when empty.
// Trucks with empty / COMPLETED / unknown truck_status are omitted (not a 5th column).
func GroupBoardColumns(trucks []payloaderTruckWire) []BoardColumn {
	cols := make([]BoardColumn, len(BoardManifestStates))
	index := make(map[string]int, len(BoardManifestStates))
	for i, st := range BoardManifestStates {
		cols[i] = BoardColumn{State: st, Trucks: []payloaderTruckWire{}}
		index[st] = i
	}
	for i := range trucks {
		st := CanonicalBoardState(trucks[i].TruckStatus)
		if st == "" {
			continue
		}
		idx := index[st]
		cols[idx].Trucks = append(cols[idx].Trucks, trucks[i])
	}
	return cols
}

func currentBoardManifestLocked(s *Service, vehicleID string) *ManifestRow {
	vehicleID = strings.TrimSpace(vehicleID)
	if s == nil || vehicleID == "" {
		return nil
	}
	var best *ManifestRow
	for i := range s.manifests {
		m := &s.manifests[i]
		if strings.TrimSpace(m.VehicleID) != vehicleID {
			continue
		}
		if CanonicalBoardState(m.State) == "" {
			continue
		}
		if best == nil || m.UpdatedAt > best.UpdatedAt {
			best = m
		}
	}
	return best
}

func truckWireFromRowLocked(s *Service, row TruckRow) payloaderTruckWire {
	w := payloaderTruckWire{
		ID:           row.VehicleID,
		Label:        row.PlateNo,
		LicensePlate: row.PlateNo,
		VehicleClass: "TRUCK",
	}
	if m := currentBoardManifestLocked(s, row.VehicleID); m != nil {
		w.TruckStatus = CanonicalBoardState(m.State)
		w.UsedVolumeVU = m.TotalVolumeVU
		w.MaxVolumeVU = m.MaxVolumeVU
		w.StopCount = m.StopCount
	}
	return w
}
