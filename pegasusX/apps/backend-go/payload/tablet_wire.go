package payload

import "strings"

func dispatchCodeForOrder(orderID string) string {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return "DSP-UNKNOWN"
	}
	if len(orderID) > 8 {
		return "DSP-" + strings.ToUpper(orderID[len(orderID)-8:])
	}
	return "DSP-" + strings.ToUpper(orderID)
}

func plateForVehicleLocked(s *Service, vehicleID string) string {
	for i := range s.trucks {
		if s.trucks[i].VehicleID == vehicleID {
			return s.trucks[i].PlateNo
		}
	}
	return vehicleID
}

// buildTruckRecommendationsLocked converts candidates; caller must hold s.mu.
func buildTruckRecommendationsLocked(s *Service, order OrderRow, recs []ReassignRecommendation) []truckRecommendationWire {
	out := make([]truckRecommendationWire, 0, len(recs))
	for i := range recs {
		rec := recs[i]
		var manifest *ManifestRow
		for j := range s.manifests {
			if routeIDForManifest(s.manifests[j]) == rec.ToRoute {
				manifest = &s.manifests[j]
				break
			}
		}
		w := truckRecommendationWire{
			DriverID:       rec.ToDriverID,
			DriverName:     rec.ToDriverID,
			Score:          rec.Score,
			Recommendation: rec.Reason,
			DistanceKm:     1.2,
		}
		if manifest != nil {
			w.VehicleID = manifest.VehicleID
			w.LicensePlate = plateForVehicleLocked(s, manifest.VehicleID)
			w.VehicleClass = "TRUCK"
			w.MaxVolumeVU = float64(manifest.MaxVolumeVU)
			w.UsedVolumeVU = float64(manifest.TotalVolumeVU)
			w.FreeVolumeVU = float64(manifest.MaxVolumeVU - manifest.TotalVolumeVU)
			w.OrderCount = manifest.StopCount
			w.TruckStatus = manifest.State
			if w.DriverID == "" {
				w.DriverID = manifest.DriverID
			}
			if w.DriverName == "" {
				w.DriverName = manifest.DriverID
			}
		}
		if w.DriverID == "" {
			w.DriverID = rec.ToRoute
		}
		_ = order
		out = append(out, w)
	}
	return out
}

type truckRecommendationWire struct {
	DriverID       string  `json:"driver_id"`
	DriverName     string  `json:"driver_name"`
	VehicleID      string  `json:"vehicle_id"`
	VehicleClass   string  `json:"vehicle_class"`
	LicensePlate   string  `json:"license_plate"`
	MaxVolumeVU    float64 `json:"max_volume_vu"`
	UsedVolumeVU   float64 `json:"used_volume_vu"`
	FreeVolumeVU   float64 `json:"free_volume_vu"`
	DistanceKm     float64 `json:"distance_km"`
	OrderCount     int     `json:"order_count"`
	TruckStatus    string  `json:"truck_status"`
	Score          float64 `json:"score"`
	Recommendation string  `json:"recommendation"`
}
