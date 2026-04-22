package proximity

import (
	"math"

	h3 "github.com/uber/h3-go/v4"
)

// ─── H3 Hexagonal Grid — uber/h3-go/v4 ─────────────────────────────────────────
//
// Cell IDs are 15-char lowercase hex strings produced by Cell.String() —
// directly decodable by h3-js on the MapLibre frontend.
//
// Resolution 7 ≈ 5.16 km² per hex; neighbor center-to-center ≈ 2.11 km.

const (
	EarthRadiusKm = 6371.0
	H3Resolution  = 7
	// Approximate hex edge length at resolution 7 in km.
	H3Res7EdgeKm = 1.22
	// Neighbor center-to-center distance at resolution 7 (edge * sqrt(3)).
	h3Res7CenterKm = 2.11
)

// HaversineKm returns the great-circle distance in km between two lat/lng points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusKm * c
}

// IsWithinRadius checks if point (lat2, lng2) is within radiusKm of (lat1, lng1).
func IsWithinRadius(lat1, lng1, lat2, lng2, radiusKm float64) bool {
	return HaversineKm(lat1, lng1, lat2, lng2) <= radiusKm
}

// ComputeGridCoverage returns the H3 res-7 cell IDs whose centers lie within
// radiusKm of (lat, lng). Cell IDs are 15-char lowercase hex strings directly
// consumable by h3-js on the frontend and Spanner ARRAY<STRING> columns.
func ComputeGridCoverage(lat, lng, radiusKm float64) []string {
	origin, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, H3Resolution)
	if err != nil {
		return nil
	}

	k := int(math.Ceil(radiusKm/h3Res7CenterKm)) + 1
	if k < 1 {
		k = 1
	}

	disk, err := origin.GridDisk(k)
	if err != nil {
		return nil
	}

	out := make([]string, 0, len(disk))
	for _, c := range disk {
		ll, err := c.LatLng()
		if err != nil {
			continue
		}
		if HaversineKm(lat, lng, ll.Lat, ll.Lng) <= radiusKm {
			out = append(out, c.String())
		}
	}
	return out
}

// LookupCell returns the H3 res-7 cell ID containing the point.
func LookupCell(lat, lng float64) string {
	c, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, H3Resolution)
	if err != nil {
		return ""
	}
	return c.String()
}

// CellToLatLng returns the geographic center of an H3 cell ID. Returns
// (0, 0, false) if the cell ID is invalid.
func CellToLatLng(cellID string) (lat, lng float64, ok bool) {
	c := h3.CellFromString(cellID)
	if !c.IsValid() {
		return 0, 0, false
	}
	ll, err := c.LatLng()
	if err != nil {
		return 0, 0, false
	}
	return ll.Lat, ll.Lng, true
}

// GridDisk returns all H3 cell IDs within k rings of the given cell.
func GridDisk(cellID string, k int) []string {
	c := h3.CellFromString(cellID)
	if !c.IsValid() {
		return nil
	}
	disk, err := c.GridDisk(k)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(disk))
	for _, d := range disk {
		out = append(out, d.String())
	}
	return out
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}
