package proximity

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// CoverageStore loads matching candidates from Spanner.
type CoverageStore struct {
	Client *spanner.Client
}

// ListWarehouses returns active warehouses plus coverage cells for one supplier.
func (s *CoverageStore) ListWarehouses(ctx context.Context, supplierID string) ([]WarehouseCandidate, error) {
	if s == nil || s.Client == nil {
		return nil, fmt.Errorf("coverage store: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required")
	}
	cells, err := s.loadCoverageCells(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	iter := s.Client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, COALESCE(CountryCode, ''), COALESCE(H3Cell, ''),
		             Lat, Lng, IsActive, COALESCE(IsOnShift, true),
		             COALESCE(DefaultOutOfStockPolicy, 'REJECT'),
		             COALESCE(ShowStockCountsToRetailers, false)
		      FROM Warehouses
		      WHERE SupplierId = @supplierId AND IsActive = true`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	out := make([]WarehouseCandidate, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query warehouses: %w", err)
		}
		var id, country, h3 string
		var lat, lng spanner.NullFloat64
		var active, onShift, showCounts bool
		var policy string
		if err := row.Columns(&id, &country, &h3, &lat, &lng, &active, &onShift, &policy, &showCounts); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}
		out = append(out, WarehouseCandidate{
			WarehouseID:             id,
			CountryCode:             country,
			H3Cell:                  h3,
			Lat:                     lat.Float64,
			Lng:                     lng.Float64,
			CoverageCells:           cells[id],
			IsActive:                active,
			IsOnShift:               onShift,
			DefaultOutOfStockPolicy: policy,
			ShowStockCounts:         showCounts,
		})
	}
	return out, nil
}

// ListFactories returns factories for one supplier.
func (s *CoverageStore) ListFactories(ctx context.Context, supplierID string) ([]FactoryCandidate, error) {
	if s == nil || s.Client == nil {
		return nil, fmt.Errorf("coverage store: nil client")
	}
	iter := s.Client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FactoryId, COALESCE(CountryCode, ''), Lat, Lng, IsActive
		      FROM Factories WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": strings.TrimSpace(supplierID)},
	})
	defer iter.Stop()
	out := make([]FactoryCandidate, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query factories: %w", err)
		}
		var id, country string
		var lat, lng spanner.NullFloat64
		var active bool
		if err := row.Columns(&id, &country, &lat, &lng, &active); err != nil {
			continue
		}
		out = append(out, FactoryCandidate{
			FactoryID:   id,
			CountryCode: country,
			Lat:         lat.Float64,
			Lng:         lng.Float64,
			IsActive:    active,
		})
	}
	return out, nil
}

// ListSupplyLanes returns active lanes into one warehouse.
func (s *CoverageStore) ListSupplyLanes(ctx context.Context, supplierID, warehouseID string) ([]SupplyLane, error) {
	if s == nil || s.Client == nil {
		return nil, fmt.Errorf("coverage store: nil client")
	}
	iter := s.Client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FactoryId, Priority, IsActive
		      FROM SupplyLanes
		      WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{
			"sid": strings.TrimSpace(supplierID),
			"wid": strings.TrimSpace(warehouseID),
		},
	})
	defer iter.Stop()
	out := make([]SupplyLane, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query supply lanes: %w", err)
		}
		var factoryID string
		var priority int64
		var active bool
		if err := row.Columns(&factoryID, &priority, &active); err != nil {
			continue
		}
		out = append(out, SupplyLane{FactoryID: factoryID, Priority: priority, IsActive: active})
	}
	return out, nil
}

// LoadStore reads the retailer org row and optional active location.
func (s *CoverageStore) LoadStore(ctx context.Context, retailerID, activeLocationID string) (StorePoint, error) {
	if s == nil || s.Client == nil {
		return StorePoint{}, fmt.Errorf("coverage store: nil client")
	}
	retailerID = strings.TrimSpace(retailerID)
	base := StorePoint{RetailerID: retailerID}
	row, err := s.Client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID},
		[]string{"Lat", "Lng", "CountryCode", "RegionId"})
	if err == nil {
		var lat, lng spanner.NullFloat64
		var country, region spanner.NullString
		if err := row.Columns(&lat, &lng, &country, &region); err == nil {
			if lat.Valid {
				base.Lat = lat.Float64
			}
			if lng.Valid {
				base.Lng = lng.Float64
			}
			if country.Valid {
				base.CountryCode = country.StringVal
			}
			if region.Valid {
				base.RegionID = region.StringVal
			}
		}
	}
	overlay := StorePoint{}
	if locID := strings.TrimSpace(activeLocationID); locID != "" {
		loc, locErr := s.loadLocation(ctx, locID)
		if locErr == nil && (loc.RetailerID == "" || loc.RetailerID == retailerID) {
			overlay = loc
		}
	}
	return MergeStorePin(base, overlay, 0, 0), nil
}

func (s *CoverageStore) loadLocation(ctx context.Context, locationID string) (StorePoint, error) {
	row, err := s.Client.Single().ReadRow(ctx, "RetailerLocations", spanner.Key{locationID},
		[]string{"RetailerId", "Lat", "Lng", "H3Cell", "CountryCode"})
	if err != nil {
		return StorePoint{}, err
	}
	var retailerID string
	var lat, lng spanner.NullFloat64
	var h3, country spanner.NullString
	if err := row.Columns(&retailerID, &lat, &lng, &h3, &country); err != nil {
		return StorePoint{}, err
	}
	out := StorePoint{LocationID: locationID, RetailerID: retailerID}
	if lat.Valid {
		out.Lat = lat.Float64
	}
	if lng.Valid {
		out.Lng = lng.Float64
	}
	if h3.Valid {
		out.H3Cell = h3.StringVal
	}
	if country.Valid {
		out.CountryCode = country.StringVal
	}
	return out, nil
}

func (s *CoverageStore) loadCoverageCells(ctx context.Context, supplierID string) (map[string][]string, error) {
	iter := s.Client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId, H3Cell FROM WarehouseCoverageCells WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	out := map[string][]string{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query coverage cells: %w", err)
		}
		var wid, cell string
		if err := row.Columns(&wid, &cell); err != nil {
			continue
		}
		out[wid] = append(out[wid], cell)
	}
	return out, nil
}

// ListPins returns supplier service pins used as matching overrides.
func (s *CoverageStore) ListPins(ctx context.Context, supplierID string) ([]ServicePin, error) {
	if s == nil || s.Client == nil {
		return nil, fmt.Errorf("coverage store: nil client")
	}
	iter := s.Client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, TargetType, TargetId, Priority
		      FROM ServicePins WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": strings.TrimSpace(supplierID)},
	})
	defer iter.Stop()
	out := make([]ServicePin, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query service pins: %w", err)
		}
		var warehouseID, targetType, targetID string
		var priority int64
		if err := row.Columns(&warehouseID, &targetType, &targetID, &priority); err != nil {
			continue
		}
		out = append(out, ServicePin{
			WarehouseID: warehouseID,
			TargetType:  targetType,
			TargetID:    targetID,
			Priority:    priority,
		})
	}
	return out, nil
}
