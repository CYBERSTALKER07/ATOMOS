package proximity

import (
	"testing"

	h3 "github.com/uber/h3-go/v4"
)

// ─── LookupCell / ComputeGridCoverage coherence ────────────────────────────

// TestLookupCell_InsideCoverage verifies that LookupCell for any point inside
// a warehouse's coverage circle produces a cell ID present in the coverage set.
func TestLookupCell_InsideCoverage(t *testing.T) {
	tests := []struct {
		name      string
		centerLat float64
		centerLng float64
		radiusKm  float64
		pointLat  float64
		pointLng  float64
	}{
		{"Tashkent center", 41.2995, 69.2401, 10, 41.2995, 69.2401},
		{"Tashkent north edge", 41.2995, 69.2401, 10, 41.38, 69.2401},
		{"Tashkent east edge", 41.2995, 69.2401, 10, 41.2995, 69.36},
		{"Tashkent diagonal", 41.2995, 69.2401, 10, 41.35, 69.30},
		{"Samarkand 5km radius", 39.6542, 66.9597, 5, 39.68, 66.97},
		{"Bukhara 15km radius", 39.7681, 64.4556, 15, 39.85, 64.50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := HaversineKm(tt.centerLat, tt.centerLng, tt.pointLat, tt.pointLng)
			if dist > tt.radiusKm {
				t.Skipf("test point is %.2f km away, outside %.1f km radius", dist, tt.radiusKm)
			}
			coverage := ComputeGridCoverage(tt.centerLat, tt.centerLng, tt.radiusKm)
			pointCell := LookupCell(tt.pointLat, tt.pointLng)

			coverageSet := make(map[string]bool, len(coverage))
			for _, c := range coverage {
				coverageSet[c] = true
			}
			if !coverageSet[pointCell] {
				t.Errorf("LookupCell(%f,%f)=%q not in coverage (%d cells)",
					tt.pointLat, tt.pointLng, pointCell, len(coverage))
			}
		})
	}
}

// TestLookupCell_Deterministic verifies the same input always produces the
// same cell ID.
func TestLookupCell_Deterministic(t *testing.T) {
	lat, lng := 41.311234, 69.279876
	first := LookupCell(lat, lng)
	for i := 0; i < 1000; i++ {
		if got := LookupCell(lat, lng); got != first {
			t.Fatalf("iteration %d: got %q, want %q", i, got, first)
		}
	}
}

// TestLookupCell_SnapConsistency verifies that two points inside the same
// H3 cell produce the same cell ID.
func TestLookupCell_SnapConsistency(t *testing.T) {
	// Two points within ~50m — guaranteed same res-7 hex (~1.22 km edge).
	a := LookupCell(41.3000, 69.2400)
	b := LookupCell(41.3003, 69.2403)
	if a != b {
		t.Errorf("nearby points produced different cells: %q vs %q", a, b)
	}
}

// TestLookupCell_DifferentCells verifies that distant points produce different
// cell IDs.
func TestLookupCell_DifferentCells(t *testing.T) {
	a := LookupCell(41.30, 69.24)
	b := LookupCell(41.50, 69.50)
	if a == b {
		t.Errorf("distant points produced the same cell: %q", a)
	}
}

// TestLookupCell_ValidH3 verifies the returned cell ID is a valid H3 res-7 cell.
func TestLookupCell_ValidH3(t *testing.T) {
	id := LookupCell(41.30, 69.24)
	cell := h3.CellFromString(id)
	if !cell.IsValid() {
		t.Fatalf("LookupCell returned invalid H3 cell: %q", id)
	}
	res := cell.Resolution()
	if res != H3Resolution {
		t.Errorf("LookupCell returned resolution %d, want %d", res, H3Resolution)
	}
}

func TestComputeGridCoverage_NotEmpty(t *testing.T) {
	if len(ComputeGridCoverage(41.30, 69.24, 5.0)) == 0 {
		t.Fatal("coverage is empty")
	}
}

func TestComputeGridCoverage_ContainsCenter(t *testing.T) {
	lat, lng := 41.30, 69.24
	cells := ComputeGridCoverage(lat, lng, 5.0)
	center := LookupCell(lat, lng)
	for _, c := range cells {
		if c == center {
			return
		}
	}
	t.Errorf("center cell %q not found in coverage of %d cells", center, len(cells))
}

func TestComputeGridCoverage_NoDuplicates(t *testing.T) {
	cells := ComputeGridCoverage(41.30, 69.24, 10.0)
	seen := make(map[string]bool, len(cells))
	for _, c := range cells {
		if seen[c] {
			t.Errorf("duplicate cell: %s", c)
		}
		seen[c] = true
	}
}

// TestComputeGridCoverage_AllWithinRadius verifies all generated cells have
// centers within radius (+ a one-edge tolerance for hex center offsets).
func TestComputeGridCoverage_AllWithinRadius(t *testing.T) {
	lat, lng, radius := 41.30, 69.24, 10.0
	cells := ComputeGridCoverage(lat, lng, radius)
	maxDist := radius + H3Res7EdgeKm
	for _, c := range cells {
		cLat, cLng, ok := CellToLatLng(c)
		if !ok {
			t.Errorf("failed to decode cell %q", c)
			continue
		}
		if d := HaversineKm(lat, lng, cLat, cLng); d > maxDist {
			t.Errorf("cell %q center (%.4f,%.4f) is %.2f km from center — exceeds %.2f km",
				c, cLat, cLng, d, maxDist)
		}
	}
}

// ─── Haversine ──────────────────────────────────────────────────────────────

func TestHaversineKm_SamePoint(t *testing.T) {
	if d := HaversineKm(41.30, 69.24, 41.30, 69.24); d != 0 {
		t.Errorf("same point distance = %f, want 0", d)
	}
}

func TestHaversineKm_KnownDistance(t *testing.T) {
	// Tashkent to Samarkand ≈ 270 km
	d := HaversineKm(41.2995, 69.2401, 39.6542, 66.9597)
	if d < 250 || d > 290 {
		t.Errorf("Tashkent→Samarkand = %.1f km, expected ~270 km", d)
	}
}

func TestIsWithinRadius(t *testing.T) {
	if !IsWithinRadius(41.30, 69.24, 41.31, 69.25, 5.0) {
		t.Error("expected nearby points to be within 5km")
	}
	if IsWithinRadius(41.30, 69.24, 42.30, 70.24, 5.0) {
		t.Error("expected distant points to NOT be within 5km")
	}
}

// ─── Pole guard ─────────────────────────────────────────────────────────────

func TestLookupCell_NearPole(t *testing.T) {
	if id := LookupCell(89.99, 0.0); id == "" {
		t.Error("pole cell ID is empty")
	}
}

func TestComputeGridCoverage_NearPole(t *testing.T) {
	if len(ComputeGridCoverage(89.5, 0.0, 5.0)) == 0 {
		t.Error("pole coverage is empty")
	}
}
