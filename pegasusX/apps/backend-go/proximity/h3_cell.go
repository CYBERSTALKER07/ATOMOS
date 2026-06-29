package proximity

// H3CellFromLatLng returns the center H3 cell string at resolution 9, or empty when coords are unset/invalid.
func H3CellFromLatLng(lat, lng float64) string {
	if lat == 0 && lng == 0 {
		return ""
	}
	cells, err := CellsInRadius(lat, lng, 9, 0)
	if err != nil || len(cells) == 0 {
		return ""
	}
	return cells[0]
}
