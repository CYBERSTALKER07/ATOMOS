package dispatch

import (
	"fmt"

	"github.com/uber/h3-go/v4"
)

// H3CellLookup groups orders by H3 resolution-7 cell for binpack consolidation.
func H3CellLookup(lat, lng float64) string {
	cell, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, H3DispatchResolution)
	if err != nil {
		return fmt.Sprintf("%.4f,%.4f", lat, lng)
	}
	return cell.String()
}
