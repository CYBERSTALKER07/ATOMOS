package order

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// CoverageMutations replaces city/cell rows for one warehouse.
func CoverageMutations(supplierID, warehouseID string, cities []CoverageCity, now time.Time) []*spanner.Mutation {
	wid := strings.TrimSpace(warehouseID)
	sid := strings.TrimSpace(supplierID)
	if wid == "" || sid == "" {
		return nil
	}
	muts := []*spanner.Mutation{
		spanner.Delete("WarehouseCoverageCells", spanner.Key{wid}.AsPrefix()),
		spanner.Delete("WarehouseCoverageCities", spanner.Key{wid}.AsPrefix()),
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	seenCell := map[string]struct{}{}
	for _, city := range cities {
		name := strings.TrimSpace(city.Name)
		if name == "" || (city.Lat == 0 && city.Lng == 0) {
			continue
		}
		muts = append(muts, spanner.InsertOrUpdateMap("WarehouseCoverageCities", map[string]any{
			"WarehouseId": wid,
			"CityName":    name,
			"Lat":         city.Lat,
			"Lng":         city.Lng,
			"SupplierId":  sid,
			"CreatedAt":   now,
		}))
		for _, cell := range CellsForCity(city.Lat, city.Lng) {
			if _, ok := seenCell[cell]; ok {
				continue
			}
			seenCell[cell] = struct{}{}
			muts = append(muts, spanner.InsertOrUpdateMap("WarehouseCoverageCells", map[string]any{
				"WarehouseId": wid,
				"H3Cell":      cell,
				"SupplierId":  sid,
				"CityName":    name,
				"Source":      "CITY",
				"CreatedAt":   now,
			}))
		}
	}
	return muts
}

// SupplyLaneMutations upserts factory↔warehouse edges. Empty factoryIDs is a no-op (keeps last-mile HQ without factories).
func SupplyLaneMutations(supplierID, warehouseID string, factoryIDs []string, now time.Time) []*spanner.Mutation {
	sid := strings.TrimSpace(supplierID)
	wid := strings.TrimSpace(warehouseID)
	if sid == "" || wid == "" {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var muts []*spanner.Mutation
	seen := map[string]struct{}{}
	for i, raw := range factoryIDs {
		fid := strings.TrimSpace(raw)
		if fid == "" {
			continue
		}
		if _, ok := seen[fid]; ok {
			continue
		}
		seen[fid] = struct{}{}
		muts = append(muts, spanner.InsertOrUpdateMap("SupplyLanes", map[string]any{
			"LaneId":               supplyLaneID(sid, fid, wid),
			"SupplierId":           sid,
			"FactoryId":            fid,
			"WarehouseId":          wid,
			"TransitTimeHours":     24.0,
			"DampenedTransitHours": 24.0,
			"FreightCostMinor":     int64(0),
			"CarbonScoreKg":        0.0,
			"IsActive":             true,
			"Priority":             int64(i),
			"CreatedAt":            now,
			"UpdatedAt":            now,
		}))
	}
	return muts
}

func supplyLaneID(supplierID, factoryID, warehouseID string) string {
	sum := sha256.Sum256([]byte(supplierID + "|" + factoryID + "|" + warehouseID))
	return hex.EncodeToString(sum[:16])
}
