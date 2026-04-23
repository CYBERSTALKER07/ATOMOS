package main

import (
	"fmt"
	"math/rand"
)

type SimRetailer struct {
	ID          string
	Name        string
	Phone       string
	ShopName    string
	Lat         float64
	Lng         float64
	LocationWKT string
	H3Cell      string
}

// GenerateGeoClusters generates a list of retailers spread across the Tashkent bounding box
func GenerateGeoClusters(count int, rng *rand.Rand) []SimRetailer {
	retailers := make([]SimRetailer, 0, count)

	// Case A: Dense (Yunusabad approx)
	denseLat, denseLng := 41.3653, 69.2882
	// Case B: Sparse Multi-Cluster
	c1Lat, c1Lng := 41.2750, 69.2000 // Chilanzar
	c2Lat, c2Lng := 41.3300, 69.3200 // Mirzo Ulugbek

	// Case C: Outlier (Chirchiq)
	outlierLat, outlierLng := 41.4658, 69.5815

	generateCluster := func(clusterSize int, centerLat, centerLng, radius float64, prefix string) {
		for i := 0; i < clusterSize; i++ {
			rLat := centerLat + (rng.Float64()*2-1)*radius
			rLng := centerLng + (rng.Float64()*2-1)*radius

			id := fmt.Sprintf("RET-SIM-%s-%03d", prefix, i+1)
			retailers = append(retailers, SimRetailer{
				ID:          id,
				Name:        "Sim Owner " + id,
				Phone:       fmt.Sprintf("+99891%07d", rng.Intn(10000000)),
				ShopName:    "Shop " + id,
				Lat:         rLat,
				Lng:         rLng,
				LocationWKT: fmt.Sprintf("POINT(%f %f)", rLng, rLat),
				H3Cell:      "872830828ffffff", // simplified dummy placeholder matching H3 shape
			})
		}
	}

	// 1. Dense group: 50%
	generateCluster(count/2, denseLat, denseLng, 0.005, "DNS")

	// 2. Sparse groups: 20% each
	generateCluster(count/5, c1Lat, c1Lng, 0.01, "SPR1")
	generateCluster(count/5, c2Lat, c2Lng, 0.01, "SPR2")

	// 3. Outlier and exact numbers
	remaining := count - len(retailers) - 1
	if remaining > 0 {
		generateCluster(remaining, denseLat, denseLng, 0.005, "DNS-EXTRA")
	}

	// The single extreme outlier
	outlierID := "RET-SIM-OUTLIER"
	retailers = append(retailers, SimRetailer{
		ID:          outlierID,
		Name:        "Far Away Retailer",
		Phone:       "+998999999999",
		ShopName:    "Outlier Shop",
		Lat:         outlierLat,
		Lng:         outlierLng,
		LocationWKT: fmt.Sprintf("POINT(%f %f)", outlierLng, outlierLat),
		H3Cell:      "872830828ffffff",
	})

	return retailers
}
