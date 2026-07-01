package dispatch

import (
	"encoding/json"
	"math"
)

type geoJSONPolygon struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// PointInGeoJSONPolygon returns true when lat/lng falls inside a GeoJSON Polygon.
func PointInGeoJSONPolygon(geoJSON string, lat, lng float64) bool {
	var poly geoJSONPolygon
	if err := json.Unmarshal([]byte(geoJSON), &poly); err != nil {
		return false
	}
	if len(poly.Coordinates) == 0 || len(poly.Coordinates[0]) < 3 {
		return false
	}
	ring := poly.Coordinates[0]
	inside := false
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		intersect := ((yi > lat) != (yj > lat)) &&
			(lng < (xj-xi)*(lat-yi)/(yj-yi+1e-12)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

// ZoneOverride describes an active control-tower polygon action.
type ZoneOverride struct {
	OverrideID string
	SupplierID string
	WarehouseID string
	Action     string
	Polygon    string
}

// ApplyZoneOverrides filters dispatchable orders based on active overrides.
// FREEZE_DISPATCH removes orders inside the polygon; REROUTE/PRIORITY_BOOST are advisory metadata.
func ApplyZoneOverrides(orders []DispatchableOrder, overrides []ZoneOverride) ([]DispatchableOrder, []map[string]any) {
	if len(overrides) == 0 {
		return orders, nil
	}
	out := make([]DispatchableOrder, 0, len(orders))
	var meta []map[string]any
	for _, o := range orders {
		skip := false
		for _, ov := range overrides {
			if ov.WarehouseID != "" && ov.WarehouseID != o.WarehouseID {
				continue
			}
			if !PointInGeoJSONPolygon(ov.Polygon, o.Lat, o.Lng) {
				continue
			}
			meta = append(meta, map[string]any{
				"order_id":    o.OrderID,
				"override_id": ov.OverrideID,
				"action":      ov.Action,
			})
			if ov.Action == "FREEZE_DISPATCH" {
				skip = true
			}
		}
		if !skip {
			out = append(out, o)
		}
	}
	return out, meta
}

// PriorityBoostPenalty returns a negative cost adjustment for orders inside boost zones.
func PriorityBoostPenalty(lat, lng float64, overrides []ZoneOverride) float64 {
	penalty := 0.0
	for _, ov := range overrides {
		if ov.Action != "PRIORITY_BOOST" {
			continue
		}
		if PointInGeoJSONPolygon(ov.Polygon, lat, lng) {
			penalty -= 1000
		}
	}
	return math.Max(penalty, -5000)
}
