package proximity

import (
	"math"
	"sort"

	h3 "github.com/uber/h3-go/v4"
)

// ─── Spatial Recommendation Engine ─────────────────────────────────────────────
//
// Calculates warehouse-to-retailer suitability scores and generates Voronoi-style
// territory proposals. Used by the admin portal's "Map Recommendation" surface
// to let suppliers visualize and approve optimal warehouse-retailer assignments.
//
// Scoring formula (non-linear penalty above 70% load):
//   if load ≤ 0.70:  score = distance × (1 + load)
//   if load >  0.70:  score = distance × (1 + load²)
//
// Lower score = better candidate. An overloaded warehouse is penalized
// aggressively once it crosses the 70% threshold.

// WarehouseGeo holds the spatial + load data needed for territory recommendations.
type WarehouseGeo struct {
	WarehouseId      string
	Lat              float64
	Lng              float64
	CoverageRadiusKm float64
	LoadPercent      float64 // 0.0–1.0 from Redis queue depth / max capacity
	H3Indexes        []string
}

// CellAssignment maps an H3 cell to a warehouse with its suitability score.
type CellAssignment struct {
	CellID      string  `json:"cell_id"`
	WarehouseId string  `json:"warehouse_id"`
	Score       float64 `json:"score"`
	DistanceKm  float64 `json:"distance_km"`
	LoadPercent float64 `json:"load_percent"`
}

// TerritoryProposal is the output of GenerateNaturalTerritories — a complete
// Voronoi-style assignment of cells to warehouses.
type TerritoryProposal struct {
	Assignments map[string][]CellAssignment `json:"assignments"` // warehouseId → cells
	Unassigned  []string                    `json:"unassigned"`  // cells too far from any warehouse
}

// loadPenalty returns the multiplier for a given load fraction.
// Below 70%: linear penalty  →  (1 + load)
// Above 70%: quadratic penalty → (1 + load²)
// This makes overloaded warehouses dramatically less attractive once they
// cross the threshold — a 95% warehouse is penalized 1.9025× vs 1.95× linear.
func loadPenalty(load float64) float64 {
	if load > 0.70 {
		return 1.0 + math.Pow(load, 2)
	}
	return 1.0 + load
}

// CalculateSuitability scores how well a warehouse can serve a given retailer cell.
// Uses Haversine distance + non-linear load penalty.
//
//	if load ≤ 0.70:  score = distanceKm × (1 + load)
//	if load >  0.70:  score = distanceKm × (1 + load²)
//
// Lower is better. A warehouse at 95% load with 3km distance scores:
//
//	3.0 × (1 + 0.95²) = 3.0 × 1.9025 = 5.71
//
// while a warehouse at 40% load at 5km scores:
//
//	5.0 × (1 + 0.40)  = 5.0 × 1.40   = 7.00
func CalculateSuitability(retailerLat, retailerLng float64, wh WarehouseGeo) float64 {
	dist := HaversineKm(retailerLat, retailerLng, wh.Lat, wh.Lng)
	if dist < 0.01 {
		dist = 0.01 // floor to prevent zero-score ties
	}
	return dist * loadPenalty(wh.LoadPercent)
}

// ScoreWarehouse is the H3-grid-native scorer. Instead of Haversine it uses
// H3GridDistance (ring count × edge length) for O(1) cost estimation, then
// applies the same non-linear load penalty.
//
// Use this when both retailer and warehouse positions are available as cell IDs.
func ScoreWarehouse(retailerH3 string, wh WarehouseGeo) float64 {
	whCell := LookupCell(wh.Lat, wh.Lng)
	rings := H3GridDistance(retailerH3, whCell)
	if rings == math.MaxInt32 {
		return math.MaxFloat64 // unparseable — worst score
	}
	distKm := float64(rings) * H3Res7EdgeKm
	if distKm < 0.01 {
		distKm = 0.01
	}
	return distKm * loadPenalty(wh.LoadPercent)
}

// H3GridDistance returns the grid-ring distance between two H3 cells using
// the native h3 library. Returns math.MaxInt32 if either cell is invalid or
// the cells are too far apart for the library to compute a distance.
func H3GridDistance(cellA, cellB string) int {
	a := h3.CellFromString(cellA)
	b := h3.CellFromString(cellB)
	if !a.IsValid() || !b.IsValid() {
		return math.MaxInt32
	}
	d, err := h3.GridDistance(a, b)
	if err != nil {
		return math.MaxInt32
	}
	return d
}

// RankWarehouses returns all warehouses sorted by suitability for a retailer location.
// The first element is the best candidate.
func RankWarehouses(retailerLat, retailerLng float64, warehouses []WarehouseGeo) []CellAssignment {
	if len(warehouses) == 0 {
		return nil
	}

	ranked := make([]CellAssignment, len(warehouses))
	for i, wh := range warehouses {
		dist := HaversineKm(retailerLat, retailerLng, wh.Lat, wh.Lng)
		ranked[i] = CellAssignment{
			WarehouseId: wh.WarehouseId,
			Score:       CalculateSuitability(retailerLat, retailerLng, wh),
			DistanceKm:  dist,
			LoadPercent: wh.LoadPercent,
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score < ranked[j].Score
	})
	return ranked
}

// GenerateNaturalTerritories computes a Voronoi-style assignment of H3 cells to
// warehouses based on distance × load suitability. For each warehouse's coverage
// area, every candidate cell is assigned to the warehouse with the lowest
// suitability score — so cells near an overloaded warehouse may be "claimed" by
// a further but healthier neighbor.
//
// The union of all warehouse coverage circles forms the candidate cell grid.
// Cells outside all coverage radii are returned as "unassigned".
func GenerateNaturalTerritories(warehouses []WarehouseGeo) *TerritoryProposal {
	if len(warehouses) == 0 {
		return &TerritoryProposal{
			Assignments: make(map[string][]CellAssignment),
		}
	}

	// Phase 1: Generate the universe of candidate cells from all warehouse coverages
	allCells := make(map[string]struct{})
	warehouseCells := make(map[string]map[string]struct{}) // whID → cell set

	for _, wh := range warehouses {
		cells := ComputeGridCoverage(wh.Lat, wh.Lng, wh.CoverageRadiusKm)
		warehouseCells[wh.WarehouseId] = make(map[string]struct{})
		for _, c := range cells {
			allCells[c] = struct{}{}
			warehouseCells[wh.WarehouseId][c] = struct{}{}
		}
	}

	// Phase 2: For each candidate cell, find the warehouse with the best suitability
	result := &TerritoryProposal{
		Assignments: make(map[string][]CellAssignment),
	}
	// Pre-init assignment slices for all warehouses
	for _, wh := range warehouses {
		result.Assignments[wh.WarehouseId] = []CellAssignment{}
	}

	for cellID := range allCells {
		cellLat, cellLng, ok := parseCellCoords(cellID)
		if !ok {
			continue
		}

		var bestWh string
		var bestScore float64 = math.MaxFloat64
		var bestDist float64

		for _, wh := range warehouses {
			// Only consider warehouses whose coverage includes this cell
			if _, covered := warehouseCells[wh.WarehouseId][cellID]; !covered {
				continue
			}
			score := CalculateSuitability(cellLat, cellLng, wh)
			if score < bestScore {
				bestScore = score
				bestWh = wh.WarehouseId
				bestDist = HaversineKm(cellLat, cellLng, wh.Lat, wh.Lng)
			}
		}

		if bestWh == "" {
			result.Unassigned = append(result.Unassigned, cellID)
			continue
		}

		// Find the load for the winning warehouse
		var loadPct float64
		for _, wh := range warehouses {
			if wh.WarehouseId == bestWh {
				loadPct = wh.LoadPercent
				break
			}
		}

		result.Assignments[bestWh] = append(result.Assignments[bestWh], CellAssignment{
			CellID:      cellID,
			WarehouseId: bestWh,
			Score:       bestScore,
			DistanceKm:  bestDist,
			LoadPercent: loadPct,
		})
	}

	return result
}

// parseCellCoords returns the geographic center (lat, lng) of an H3 cell ID.
func parseCellCoords(cellID string) (float64, float64, bool) {
	return CellToLatLng(cellID)
}
