package proximity

import "testing"

// ═══════════════════════════════════════════════════════════════════════════════
// H3 PERFORMANCE BENCHMARKS — b.RunParallel for contention profiling
// ═══════════════════════════════════════════════════════════════════════════════

// BenchmarkLookupCell measures single-cell lookup throughput.
func BenchmarkLookupCell(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		lat, lng := 41.30, 69.24
		for pb.Next() {
			LookupCell(lat, lng)
		}
	})
}

// BenchmarkComputeGridCoverage_10km measures coverage computation for a 10 km radius.
func BenchmarkComputeGridCoverage_10km(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		lat, lng := 41.30, 69.24
		for pb.Next() {
			ComputeGridCoverage(lat, lng, 10.0)
		}
	})
}

// BenchmarkComputeGridCoverage_50km measures coverage computation for a 50 km radius.
func BenchmarkComputeGridCoverage_50km(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		lat, lng := 41.30, 69.24
		for pb.Next() {
			ComputeGridCoverage(lat, lng, 50.0)
		}
	})
}

// BenchmarkComputeGridCoverage_200km stress-tests the maximum typical radius.
func BenchmarkComputeGridCoverage_200km(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		lat, lng := 41.30, 69.24
		for pb.Next() {
			ComputeGridCoverage(lat, lng, 200.0)
		}
	})
}

// BenchmarkHaversineKm measures the core distance function.
func BenchmarkHaversineKm(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			HaversineKm(41.30, 69.24, 39.65, 66.96)
		}
	})
}

// BenchmarkIsWithinRadius measures the proximity check hot path.
func BenchmarkIsWithinRadius(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			IsWithinRadius(41.30, 69.24, 41.32, 69.26, 5.0)
		}
	})
}
