package factory

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	"backend-go/auth"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Factory-Warehouse Recommendation Engine ──────────────────────────────────
//
// Recommends which warehouses a new factory should serve, based on Haversine
// distance (shortest great-circle path). For multi-factory suppliers, the
// "optimal assignments" endpoint computes a partition where each warehouse is
// served by its geographically nearest factory —  the industry-standard
// "greedy nearest-neighbor" assignment used by major logistics platforms.

type WarehouseRecommendation struct {
	WarehouseId string  `json:"warehouse_id"`
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DistanceKm  float64 `json:"distance_km"`
	Rank        int     `json:"rank"`
	IsAssigned  bool    `json:"is_assigned"` // true if already assigned to this factory
	AssignedTo  string  `json:"assigned_to"` // factory_id currently assigned (empty if none)
}

type FactoryWarehouseAssignment struct {
	WarehouseId string  `json:"warehouse_id"`
	FactoryId   string  `json:"factory_id"`
	FactoryName string  `json:"factory_name"`
	DistanceKm  float64 `json:"distance_km"`
}

// HandleRecommendWarehouses returns warehouses sorted by distance to a factory location.
// GET /v1/supplier/factories/recommend-warehouses?factory_lat=X&factory_lng=Y[&factory_id=Z]
func HandleRecommendWarehouses(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		latStr := r.URL.Query().Get("factory_lat")
		lngStr := r.URL.Query().Get("factory_lng")
		if latStr == "" || lngStr == "" {
			http.Error(w, `{"error":"factory_lat and factory_lng query parameters required"}`, http.StatusBadRequest)
			return
		}
		factoryLat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid factory_lat"}`, http.StatusBadRequest)
			return
		}
		factoryLng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid factory_lng"}`, http.StatusBadRequest)
			return
		}

		// Optional: factory_id to mark current assignments
		factoryID := r.URL.Query().Get("factory_id")

		warehouses, err := fetchSupplierWarehouses(r.Context(), spannerClient, claims.ResolveSupplierID())
		if err != nil {
			log.Printf("[FACTORY-RECOMMEND] fetch warehouses error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		recommendations := make([]WarehouseRecommendation, 0, len(warehouses))
		for _, wh := range warehouses {
			dist := proximity.HaversineKm(factoryLat, factoryLng, wh.Lat, wh.Lng)
			rec := WarehouseRecommendation{
				WarehouseId: wh.WarehouseId,
				Name:        wh.Name,
				Address:     wh.Address,
				Lat:         wh.Lat,
				Lng:         wh.Lng,
				DistanceKm:  math.Round(dist*100) / 100,
				IsAssigned:  factoryID != "" && wh.PrimaryFactoryId == factoryID,
				AssignedTo:  wh.PrimaryFactoryId,
			}
			recommendations = append(recommendations, rec)
		}

		sort.Slice(recommendations, func(i, j int) bool {
			return recommendations[i].DistanceKm < recommendations[j].DistanceKm
		})
		for i := range recommendations {
			recommendations[i].Rank = i + 1
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"recommendations": recommendations,
			"total":           len(recommendations),
		})
	}
}

// HandleOptimalAssignments computes the globally-optimal factory→warehouse partition
// for all active factories and warehouses under the supplier. Each warehouse is
// assigned to its nearest factory (greedy nearest-neighbor).
// GET /v1/supplier/factories/optimal-assignments
func HandleOptimalAssignments(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		factories, err := fetchSupplierFactoriesForRecommend(r.Context(), spannerClient, claims.ResolveSupplierID())
		if err != nil {
			log.Printf("[FACTORY-OPTIMAL] fetch factories error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		warehouses, err := fetchSupplierWarehouses(r.Context(), spannerClient, claims.ResolveSupplierID())
		if err != nil {
			log.Printf("[FACTORY-OPTIMAL] fetch warehouses error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		assignments := computeOptimalAssignments(factories, warehouses)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"assignments": assignments,
			"total":       len(assignments),
		})
	}
}

// ── Internal types and helpers ────────────────────────────────────────────────

type warehouseForRecommend struct {
	WarehouseId      string
	Name             string
	Address          string
	Lat              float64
	Lng              float64
	PrimaryFactoryId string
}

type factoryForRecommend struct {
	FactoryId string
	Name      string
	Lat       float64
	Lng       float64
}

func fetchSupplierWarehouses(ctx context.Context, client *spanner.Client, supplierId string) ([]warehouseForRecommend, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, COALESCE(Address, ''),
		             IFNULL(Lat, 0), IFNULL(Lng, 0),
		             COALESCE(PrimaryFactoryId, '')
		      FROM Warehouses
		      WHERE SupplierId = @sid AND IsActive = TRUE
		      ORDER BY Name ASC`,
		Params: map[string]interface{}{"sid": supplierId},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []warehouseForRecommend
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var wh warehouseForRecommend
		if err := row.Columns(&wh.WarehouseId, &wh.Name, &wh.Address, &wh.Lat, &wh.Lng, &wh.PrimaryFactoryId); err != nil {
			return nil, err
		}
		result = append(result, wh)
	}
	return result, nil
}

func fetchSupplierFactoriesForRecommend(ctx context.Context, client *spanner.Client, supplierId string) ([]factoryForRecommend, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, Name, IFNULL(Lat, 0), IFNULL(Lng, 0)
		      FROM Factories
		      WHERE SupplierId = @sid AND IsActive = TRUE
		      ORDER BY Name ASC`,
		Params: map[string]interface{}{"sid": supplierId},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []factoryForRecommend
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var f factoryForRecommend
		if err := row.Columns(&f.FactoryId, &f.Name, &f.Lat, &f.Lng); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, nil
}

// computeOptimalAssignments uses greedy nearest-neighbor: for each warehouse,
// assign it to the geographically nearest factory. This is the standard approach
// used by large logistics companies (FedEx, DHL, Amazon) for facility linking.
func computeOptimalAssignments(factories []factoryForRecommend, warehouses []warehouseForRecommend) []FactoryWarehouseAssignment {
	if len(factories) == 0 || len(warehouses) == 0 {
		return []FactoryWarehouseAssignment{}
	}

	assignments := make([]FactoryWarehouseAssignment, 0, len(warehouses))

	for _, wh := range warehouses {
		bestDist := math.MaxFloat64
		bestFactory := factories[0]

		for _, f := range factories {
			dist := proximity.HaversineKm(f.Lat, f.Lng, wh.Lat, wh.Lng)
			if dist < bestDist {
				bestDist = dist
				bestFactory = f
			}
		}

		assignments = append(assignments, FactoryWarehouseAssignment{
			WarehouseId: wh.WarehouseId,
			FactoryId:   bestFactory.FactoryId,
			FactoryName: bestFactory.Name,
			DistanceKm:  math.Round(bestDist*100) / 100,
		})
	}

	// Sort by factory grouping then distance within each group
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].FactoryId != assignments[j].FactoryId {
			return assignments[i].FactoryId < assignments[j].FactoryId
		}
		return assignments[i].DistanceKm < assignments[j].DistanceKm
	})

	return assignments
}
