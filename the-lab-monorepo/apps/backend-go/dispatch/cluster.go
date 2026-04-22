package dispatch

import "math"

// ─── Spatial Clustering Constants ────────────────────────────────────────────

const (
	kMeansMaxIter              = 50
	kMeansConvergenceThreshold = 0.0001 // degrees
)

// KMeansCluster partitions orders into K spatial clusters using Lloyd's algorithm.
// If K >= len(orders), each order gets its own cluster.
func KMeansCluster(orders []GeoOrder, K int) [][]GeoOrder {
	n := len(orders)
	if n == 0 {
		return nil
	}
	if K <= 0 {
		K = 1
	}
	if K > n {
		K = n
	}

	// Deterministic seeding: K evenly-spaced orders.
	centroids := make([][2]float64, K)
	for i := 0; i < K; i++ {
		idx := i * n / K
		centroids[i] = [2]float64{orders[idx].Lat, orders[idx].Lng}
	}

	assignments := make([]int, n)

	for iter := 0; iter < kMeansMaxIter; iter++ {
		// Assign each order to nearest centroid.
		for i, o := range orders {
			bestC := 0
			bestDist := math.MaxFloat64
			for c := 0; c < K; c++ {
				d := HaversineKm(o.Lat, o.Lng, centroids[c][0], centroids[c][1])
				if d < bestDist {
					bestDist = d
					bestC = c
				}
			}
			assignments[i] = bestC
		}

		// Recompute centroids.
		newCentroids := make([][2]float64, K)
		counts := make([]int, K)
		for i, o := range orders {
			c := assignments[i]
			newCentroids[c][0] += o.Lat
			newCentroids[c][1] += o.Lng
			counts[c]++
		}

		converged := true
		for c := 0; c < K; c++ {
			if counts[c] > 0 {
				newCentroids[c][0] /= float64(counts[c])
				newCentroids[c][1] /= float64(counts[c])
			} else {
				newCentroids[c] = centroids[c]
			}
			dx := newCentroids[c][0] - centroids[c][0]
			dy := newCentroids[c][1] - centroids[c][1]
			if math.Sqrt(dx*dx+dy*dy) > kMeansConvergenceThreshold {
				converged = false
			}
		}
		centroids = newCentroids

		if converged {
			break
		}
	}

	clusters := make([][]GeoOrder, K)
	for i := 0; i < K; i++ {
		clusters[i] = []GeoOrder{}
	}
	for i, o := range orders {
		clusters[assignments[i]] = append(clusters[assignments[i]], o)
	}
	return clusters
}
