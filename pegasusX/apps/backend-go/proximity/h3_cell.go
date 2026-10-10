package proximity

// SettlementH3Resolution is H3 resolution 9 (~174 m edge), used for fine-grained
// doorstep settlement unlock and perimeter fence evaluations.
const SettlementH3Resolution = 9

// H3CellRes9 returns the H3 cell string at resolution 9, or empty when coords are unset/invalid.
func H3CellRes9(lat, lng float64) string {
	if lat == 0 && lng == 0 {
		return ""
	}
	cell, err := PanicSafeLatLngToCell(lat, lng, SettlementH3Resolution)
	if err != nil || !cell.IsValid() {
		return ""
	}
	return cell.String()
}

// SettlementH3Cell returns the resolution 9 H3 cell for doorstep settlement / perimeter checks,
// explicitly distinguishing fine settlement resolution from coarse matching resolution (MatchingResolution = 7).
func SettlementH3Cell(lat, lng float64) string {
	return H3CellRes9(lat, lng)
}

// H3CellFromLatLng returns the center H3 cell string at resolution 9 (legacy compatibility alias for H3CellRes9).
func H3CellFromLatLng(lat, lng float64) string {
	return H3CellRes9(lat, lng)
}

