package dispatch

import "math"

// HaversineKm returns the great-circle distance in kilometres between two points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// ClusterCentroid returns the arithmetic mean [lat, lng] of a set of orders.
func ClusterCentroid(orders []GeoOrder) [2]float64 {
	if len(orders) == 0 {
		return [2]float64{0, 0}
	}
	var sumLat, sumLng float64
	for _, o := range orders {
		sumLat += o.Lat
		sumLng += o.Lng
	}
	n := float64(len(orders))
	return [2]float64{sumLat / n, sumLng / n}
}

// NearestNeighborSort reorders stops using a greedy nearest-neighbour
// traversal starting from originLat/originLng. Used by factory dispatch
// for route sequencing from the factory gate.
func NearestNeighborSort(orders []GeoOrder, originLat, originLng float64) []GeoOrder {
	if len(orders) <= 1 {
		return orders
	}
	sorted := make([]GeoOrder, 0, len(orders))
	remaining := make([]GeoOrder, len(orders))
	copy(remaining, orders)

	curLat, curLng := originLat, originLng
	for len(remaining) > 0 {
		bestIdx := 0
		bestDist := math.MaxFloat64
		for i, o := range remaining {
			d := HaversineKm(curLat, curLng, o.Lat, o.Lng)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		chosen := remaining[bestIdx]
		sorted = append(sorted, chosen)
		curLat, curLng = chosen.Lat, chosen.Lng
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}
	return sorted
}
