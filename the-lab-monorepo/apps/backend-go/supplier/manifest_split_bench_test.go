package supplier

import (
	"fmt"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// MANIFEST SPLIT + K-MEANS PERFORMANCE BENCHMARKS — b.RunParallel
// ═══════════════════════════════════════════════════════════════════════════════

// BenchmarkSplitManifest_100 benchmarks splitting 100 orders into 4 chunks.
func BenchmarkSplitManifest_100(b *testing.B) {
	orders := makeBenchOrders(100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			SplitManifest("drv-bench", "trk-bench", orders, 25)
		}
	})
}

// BenchmarkSplitManifest_2000 benchmarks the massive-split scenario.
func BenchmarkSplitManifest_2000(b *testing.B) {
	orders := makeBenchOrders(2000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			SplitManifest("drv-bench", "trk-bench", orders, 25)
		}
	})
}

// BenchmarkKMeansCluster_200_K5 benchmarks the core clustering algorithm.
func BenchmarkKMeansCluster_200_K5(b *testing.B) {
	orders := makeBenchOrders(200)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Copy to avoid data races (kMeansCluster is read-only on input but safer)
			local := make([]GeoOrder, len(orders))
			copy(local, orders)
			kMeansCluster(local, 5)
		}
	})
}

// BenchmarkKMeansCluster_500_K10 benchmarks larger clustering.
func BenchmarkKMeansCluster_500_K10(b *testing.B) {
	orders := makeBenchOrders(500)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			local := make([]GeoOrder, len(orders))
			copy(local, orders)
			kMeansCluster(local, 10)
		}
	})
}

// BenchmarkAlphaIndex benchmarks the base-26 suffix encoder.
func BenchmarkAlphaIndex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		alphaIndex(i % 1000)
	}
}

// makeBenchOrders generates n GeoOrders for benchmarking.
func makeBenchOrders(n int) []GeoOrder {
	orders := make([]GeoOrder, n)
	for i := 0; i < n; i++ {
		orders[i] = GeoOrder{
			OrderID:      fmt.Sprintf("BENCH-%05d", i),
			RetailerID:   fmt.Sprintf("BRET-%03d", i%100),
			RetailerName: fmt.Sprintf("BShop-%d", i%100),
			Amount:       int64(1000 + i%500),
			Lat:          41.20 + float64(i%30)*0.015,
			Lng:          69.10 + float64(i/30)*0.015,
			Volume:       float64(1 + i%8),
		}
	}
	return orders
}
