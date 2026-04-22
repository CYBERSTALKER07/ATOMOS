package proximity

import (
	"math"
	"sync"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CASE B: HEXAGONAL BORDER — Precision & Determinism Tests
//
// Inputs are coordinates placed EXACTLY on the shared edge of two H3 cells.
// LookupCell and gridCellID must be deterministic: always the same ID, no
// "oscillating" dispatch assignments.
// ═══════════════════════════════════════════════════════════════════════════════

// stepLat is the grid spacing in degrees for latitude.
var testStepLat = H3Res7EdgeKm / 111.0

// TestLookupCell_LatBoundary places a point exactly on a latitude snap boundary
// and verifies determinism across 10,000 iterations.
func TestLookupCell_LatBoundary(t *testing.T) {
	// Exact boundary: a multiple of stepLat from the origin
	borderLat := math.Round(41.30/testStepLat) * testStepLat // snaps exactly to grid line
	lng := 69.24

	first := LookupCell(borderLat, lng)
	if first == "" {
		t.Fatal("LookupCell returned empty string")
	}

	for i := 0; i < 10_000; i++ {
		got := LookupCell(borderLat, lng)
		if got != first {
			t.Fatalf("iteration %d: LookupCell(%f, %f) = %q, want %q — determinism violation",
				i, borderLat, lng, got, first)
		}
	}
}

// TestLookupCell_LngBoundary places a point exactly on a longitude snap boundary.
func TestLookupCell_LngBoundary(t *testing.T) {
	lat := 41.30
	// Snap latitude first (same as gridCellID does)
	snappedLat := math.Round(lat/testStepLat) * testStepLat
	cosLat := math.Cos(degreesToRadians(snappedLat))
	if cosLat < 1e-10 {
		cosLat = 1e-10
	}
	stepLng := H3Res7EdgeKm / (111.0 * cosLat)

	borderLng := math.Round(69.24/stepLng) * stepLng // exact lng boundary

	first := LookupCell(lat, borderLng)
	if first == "" {
		t.Fatal("LookupCell returned empty string")
	}

	for i := 0; i < 10_000; i++ {
		got := LookupCell(lat, borderLng)
		if got != first {
			t.Fatalf("iteration %d: LookupCell(%f, %f) = %q, want %q — determinism violation",
				i, lat, borderLng, got, first)
		}
	}
}

// TestLookupCell_DualBoundary tests a point on BOTH lat and lng snap boundaries
// simultaneously (corner of four cells).
func TestLookupCell_DualBoundary(t *testing.T) {
	borderLat := math.Round(41.30/testStepLat) * testStepLat
	snappedLat := borderLat
	cosLat := math.Cos(degreesToRadians(snappedLat))
	if cosLat < 1e-10 {
		cosLat = 1e-10
	}
	stepLng := H3Res7EdgeKm / (111.0 * cosLat)
	borderLng := math.Round(69.24/stepLng) * stepLng

	first := LookupCell(borderLat, borderLng)
	for i := 0; i < 10_000; i++ {
		got := LookupCell(borderLat, borderLng)
		if got != first {
			t.Fatalf("iteration %d: corner point oscillated: %q vs %q", i, got, first)
		}
	}
}

// TestLookupCell_RoundTrip ensures that a cell ID decoded back to its
// geographic center and re-encoded yields the same cell — the core H3
// idempotency invariant.
func TestLookupCell_RoundTrip(t *testing.T) {
	points := [][2]float64{
		{41.30, 69.24},
		{39.65, 66.96},
		{0.0, 0.0},
		{-33.87, 151.21}, // Sydney
		{89.99, 0.0},     // near pole
	}
	for _, p := range points {
		id := LookupCell(p[0], p[1])
		cLat, cLng, ok := CellToLatLng(id)
		if !ok {
			t.Errorf("LookupCell(%f,%f)=%q decoded to invalid cell", p[0], p[1], id)
			continue
		}
		again := LookupCell(cLat, cLng)
		if again != id {
			t.Errorf("round-trip mismatch for (%f,%f): %q → center (%f,%f) → %q",
				p[0], p[1], id, cLat, cLng, again)
		}
	}
}

// TestComputeGridCoverage_BorderPointIncluded verifies that a border point's
// cell is present in the coverage set when that point is within the radius.
func TestComputeGridCoverage_BorderPointIncluded(t *testing.T) {
	centerLat := 41.30
	centerLng := 69.24
	radiusKm := 10.0

	borderLat := math.Round(centerLat/testStepLat) * testStepLat
	// Only test if the border point is actually inside the radius
	dist := HaversineKm(centerLat, centerLng, borderLat, centerLng)
	if dist > radiusKm {
		t.Skipf("border point %.4f km from center, outside %.1f km radius", dist, radiusKm)
	}

	coverage := ComputeGridCoverage(centerLat, centerLng, radiusKm)
	borderCell := LookupCell(borderLat, centerLng)

	coverageSet := make(map[string]bool, len(coverage))
	for _, c := range coverage {
		coverageSet[c] = true
	}

	if !coverageSet[borderCell] {
		t.Errorf("border cell %q not found in coverage set (%d cells)", borderCell, len(coverage))
	}
}

// TestLookupCell_ConcurrentDeterminism runs LookupCell from 100 goroutines
// on the same border coordinate. All must produce identical results.
func TestLookupCell_ConcurrentDeterminism(t *testing.T) {
	borderLat := math.Round(41.30/testStepLat) * testStepLat
	lng := 69.24
	expected := LookupCell(borderLat, lng)

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	errCh := make(chan string, goroutines)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				got := LookupCell(borderLat, lng)
				if got != expected {
					errCh <- got
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	if bad, ok := <-errCh; ok {
		t.Fatalf("concurrent determinism violation: got %q, want %q", bad, expected)
	}
}

// TestLookupCell_MicroPerturbation verifies that adding/subtracting a tiny
// epsilon to a grid boundary still snaps to the same cell (no float instability).
func TestLookupCell_MicroPerturbation(t *testing.T) {
	borderLat := math.Round(41.30/testStepLat) * testStepLat
	lng := 69.24

	base := LookupCell(borderLat, lng)

	epsilons := []float64{1e-15, 1e-14, 1e-13, 1e-12}
	for _, eps := range epsilons {
		plus := LookupCell(borderLat+eps, lng)
		minus := LookupCell(borderLat-eps, lng)
		if plus != base {
			t.Errorf("epsilon +%e shifted cell: %q → %q", eps, base, plus)
		}
		if minus != base {
			t.Errorf("epsilon -%e shifted cell: %q → %q", eps, base, minus)
		}
	}
}
