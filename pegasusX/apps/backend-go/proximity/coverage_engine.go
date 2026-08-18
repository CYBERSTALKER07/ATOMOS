package proximity

import (
	"errors"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/uber/h3-go/v4"
)

var (
	// ErrZoneMiss is no covering warehouse in the same pack country.
	ErrZoneMiss = errors.New("zone_miss")
	// ErrFactoryUnassigned is no same-country factory for a warehouse.
	ErrFactoryUnassigned = errors.New("factory_unassigned")
)

const (
	PinTargetLocation = "LOCATION"
	PinTargetRetailer = "RETAILER"
	PinTargetRegion   = "REGION"
	PinTargetCity     = "CITY"

	CoverageModeCountryClosest = "COUNTRY_CLOSEST"
	CoverageModeCityCells      = "CITY_CELLS"
	CoverageModePinned         = "PINNED"
)

// StorePoint is the retailer store used for matching (active location or org row).
type StorePoint struct {
	LocationID  string
	RetailerID  string
	RegionID    string
	CityName    string
	CountryCode string
	H3Cell      string
	Lat         float64
	Lng         float64
}

// ServicePin is a supplier override: this warehouse serves this target.
type ServicePin struct {
	WarehouseID string `json:"warehouse_id,omitempty"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Priority    int64  `json:"priority"`
}

// WarehouseCandidate is one active warehouse the engine may assign.
type WarehouseCandidate struct {
	WarehouseID             string
	CountryCode             string
	H3Cell                  string
	Lat                     float64
	Lng                     float64
	CoverageCells           []string
	IsActive                bool
	IsOnShift               bool
	DefaultOutOfStockPolicy string
	ShowStockCounts         bool
}

// FactoryCandidate is one factory the engine may assign to a warehouse.
type FactoryCandidate struct {
	FactoryID   string
	CountryCode string
	Lat         float64
	Lng         float64
	IsActive    bool
}

// SupplyLane is a factory→warehouse priority edge.
type SupplyLane struct {
	FactoryID string
	Priority  int64
	IsActive  bool
}

// CellInCoverage is true when the retailer H3 cell is the stored cell or a child of it.
func CellInCoverage(retailerCell string, stored []string) bool {
	retailerCell = strings.TrimSpace(retailerCell)
	if retailerCell == "" || len(stored) == 0 {
		return false
	}
	r := h3.Cell(h3.IndexFromString(retailerCell))
	if !r.IsValid() {
		for _, s := range stored {
			if strings.TrimSpace(s) == retailerCell {
				return true
			}
		}
		return false
	}
	for _, raw := range stored {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if s == retailerCell {
			return true
		}
		sc := h3.Cell(h3.IndexFromString(s))
		if !sc.IsValid() {
			continue
		}
		if r == sc {
			return true
		}
		res := sc.Resolution()
		if res < 0 || res > 15 {
			continue
		}
		parent, err := r.Parent(res)
		if err == nil && parent == sc {
			return true
		}
	}
	return false
}

// CoversRetailer is GS-L0 hybrid cover: empty country is incomplete;
// countries must match; cells set → H3 membership; no cells → whole country.
func CoversRetailer(warehouseCountry string, coverageCells []string, retailerCountry, retailerCell string) bool {
	whC := auth.NormalizeCountryCode(warehouseCountry)
	rtC := auth.NormalizeCountryCode(retailerCountry)
	if whC == "" || rtC == "" {
		return false
	}
	if whC != rtC {
		return false
	}
	if len(coverageCells) > 0 {
		return CellInCoverage(retailerCell, coverageCells)
	}
	return true
}

// MergeStorePin applies active-location overlay, then body lat/lng refine.
// Body coordinates cannot change country.
func MergeStorePin(base, overlay StorePoint, bodyLat, bodyLng float64) StorePoint {
	out := base
	if strings.TrimSpace(overlay.LocationID) != "" {
		out.LocationID = overlay.LocationID
		if overlay.Lat != 0 || overlay.Lng != 0 {
			out.Lat = overlay.Lat
			out.Lng = overlay.Lng
		}
		if overlay.CountryCode != "" {
			out.CountryCode = overlay.CountryCode
		}
		if overlay.H3Cell != "" {
			out.H3Cell = overlay.H3Cell
		}
		if overlay.RegionID != "" {
			out.RegionID = overlay.RegionID
		}
		if overlay.CityName != "" {
			out.CityName = overlay.CityName
		}
	}
	if bodyLat != 0 || bodyLng != 0 {
		out.Lat = bodyLat
		out.Lng = bodyLng
	}
	if out.H3Cell == "" && (out.Lat != 0 || out.Lng != 0) {
		out.H3Cell = MatchingH3Cell(out.Lat, out.Lng)
	}
	return out
}

// ResolveServingWarehouse is the single matching function (GS-L2/L3).
func ResolveServingWarehouse(packCountry string, store StorePoint, warehouses []WarehouseCandidate, pins []ServicePin) (string, error) {
	packCountry = auth.NormalizeCountryCode(packCountry)
	storeCountry := auth.NormalizeCountryCode(store.CountryCode)
	if packCountry == "" || storeCountry == "" {
		return "", auth.ErrGeographyIncomplete
	}
	if storeCountry != packCountry {
		return "", auth.ErrCrossMarketDeferred
	}
	if store.Lat == 0 && store.Lng == 0 {
		return "", auth.ErrGeographyIncomplete
	}
	cell := strings.TrimSpace(store.H3Cell)
	if cell == "" {
		cell = MatchingH3Cell(store.Lat, store.Lng)
	}

	byID := warehouseIndex(warehouses)
	if id := resolvePinnedWarehouse(store, byID, pins, packCountry); id != "" {
		return id, nil
	}

	bestID := ""
	bestDist := 0.0
	for _, wh := range warehouses {
		if !warehouseEligible(wh, packCountry) {
			continue
		}
		if !CoversRetailer(wh.CountryCode, wh.CoverageCells, storeCountry, cell) {
			continue
		}
		dist := HaversineDistance(store.Lat, store.Lng, wh.Lat, wh.Lng)
		if closerCandidate(dist, wh.WarehouseID, bestDist, bestID) {
			bestDist = dist
			bestID = wh.WarehouseID
		}
	}
	if bestID == "" {
		return "", ErrZoneMiss
	}
	return bestID, nil
}

// EffectiveCoverageMode is the GET-coverage honesty label for one warehouse.
func EffectiveCoverageMode(warehouse WarehouseCandidate, pins []ServicePin) string {
	for _, pin := range pins {
		if strings.TrimSpace(pin.WarehouseID) == warehouse.WarehouseID {
			return CoverageModePinned
		}
	}
	if len(warehouse.CoverageCells) > 0 {
		return CoverageModeCityCells
	}
	return CoverageModeCountryClosest
}

func resolvePinnedWarehouse(store StorePoint, byID map[string]WarehouseCandidate, pins []ServicePin, packCountry string) string {
	if id := bestPinnedWarehouse(store.LocationID, PinTargetLocation, store, byID, pins, packCountry); id != "" {
		return id
	}
	if id := bestPinnedWarehouse(store.RetailerID, PinTargetRetailer, store, byID, pins, packCountry); id != "" {
		return id
	}
	if id := bestPinnedWarehouse(store.RegionID, PinTargetRegion, store, byID, pins, packCountry); id != "" {
		return id
	}
	if id := bestPinnedWarehouse(store.CityName, PinTargetCity, store, byID, pins, packCountry); id != "" {
		return id
	}
	return ""
}

func bestPinnedWarehouse(
	targetID, targetType string,
	store StorePoint,
	byID map[string]WarehouseCandidate,
	pins []ServicePin,
	packCountry string,
) string {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return ""
	}
	bestID := ""
	bestPri := int64(-1 << 62)
	bestDist := 0.0
	for _, pin := range pins {
		if strings.ToUpper(strings.TrimSpace(pin.TargetType)) != targetType {
			continue
		}
		if strings.TrimSpace(pin.TargetID) != targetID {
			continue
		}
		wh, ok := byID[strings.TrimSpace(pin.WarehouseID)]
		if !ok || !warehouseEligible(wh, packCountry) {
			continue
		}
		dist := HaversineDistance(store.Lat, store.Lng, wh.Lat, wh.Lng)
		if bestID == "" || pin.Priority > bestPri ||
			(pin.Priority == bestPri && closerCandidate(dist, wh.WarehouseID, bestDist, bestID)) {
			bestID = wh.WarehouseID
			bestPri = pin.Priority
			bestDist = dist
		}
	}
	return bestID
}

func warehouseIndex(warehouses []WarehouseCandidate) map[string]WarehouseCandidate {
	out := make(map[string]WarehouseCandidate, len(warehouses))
	for _, wh := range warehouses {
		id := strings.TrimSpace(wh.WarehouseID)
		if id == "" {
			continue
		}
		out[id] = wh
	}
	return out
}

func warehouseEligible(wh WarehouseCandidate, packCountry string) bool {
	if !wh.IsActive || !wh.IsOnShift {
		return false
	}
	whCountry := auth.NormalizeCountryCode(wh.CountryCode)
	return whCountry != "" && whCountry == packCountry
}

// ResolveSupplyFactory is the single factory matcher (GS-L2).
func ResolveSupplyFactory(
	warehouseCountry string,
	warehouseLat, warehouseLng float64,
	primaryFactoryID string,
	lanes []SupplyLane,
	factories []FactoryCandidate,
) (string, error) {
	warehouseCountry = auth.NormalizeCountryCode(warehouseCountry)
	if warehouseCountry == "" {
		return "", auth.ErrGeographyIncomplete
	}
	byID := make(map[string]FactoryCandidate, len(factories))
	for _, f := range factories {
		id := strings.TrimSpace(f.FactoryID)
		if id == "" {
			continue
		}
		byID[id] = f
	}

	if primary := strings.TrimSpace(primaryFactoryID); primary != "" {
		if f, ok := byID[primary]; ok && factoryEligible(f, warehouseCountry) {
			return primary, nil
		}
	}

	bestLaneID := ""
	bestLanePri := int64(-1 << 62)
	bestLaneDist := 0.0
	for _, lane := range lanes {
		if !lane.IsActive {
			continue
		}
		f, ok := byID[strings.TrimSpace(lane.FactoryID)]
		if !ok || !factoryEligible(f, warehouseCountry) {
			continue
		}
		dist := HaversineDistance(warehouseLat, warehouseLng, f.Lat, f.Lng)
		if bestLaneID == "" || lane.Priority > bestLanePri ||
			(lane.Priority == bestLanePri && closerCandidate(dist, f.FactoryID, bestLaneDist, bestLaneID)) {
			bestLaneID = f.FactoryID
			bestLanePri = lane.Priority
			bestLaneDist = dist
		}
	}
	if bestLaneID != "" {
		return bestLaneID, nil
	}

	bestID := ""
	bestDist := 0.0
	for _, f := range factories {
		if !factoryEligible(f, warehouseCountry) {
			continue
		}
		dist := HaversineDistance(warehouseLat, warehouseLng, f.Lat, f.Lng)
		if closerCandidate(dist, f.FactoryID, bestDist, bestID) {
			bestDist = dist
			bestID = f.FactoryID
		}
	}
	if bestID == "" {
		return "", ErrFactoryUnassigned
	}
	return bestID, nil
}

// PerimeterCells is the Redis-publish set: explicit coverage cells only.
// Whole-country warehouses (empty cells) are not a finite SISMEMBER set.
// Create must not SISMEMBER this until a publisher writes exactly this union.
func PerimeterCells(warehouses []WarehouseCandidate) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, wh := range warehouses {
		if !wh.IsActive || !wh.IsOnShift {
			continue
		}
		for _, cell := range wh.CoverageCells {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			if _, ok := seen[cell]; ok {
				continue
			}
			seen[cell] = struct{}{}
			out = append(out, cell)
		}
	}
	return out
}

func factoryEligible(f FactoryCandidate, warehouseCountry string) bool {
	if !f.IsActive {
		return false
	}
	return auth.NormalizeCountryCode(f.CountryCode) == warehouseCountry
}

func closerCandidate(distance float64, id string, bestDistance float64, bestID string) bool {
	if bestID == "" {
		return true
	}
	if distance < bestDistance {
		return true
	}
	if distance > bestDistance {
		return false
	}
	return id < bestID
}
