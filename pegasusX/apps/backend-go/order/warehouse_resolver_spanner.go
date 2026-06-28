package order

import (
	"context"
	"fmt"
	"math"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"google.golang.org/api/iterator"
)

// SpannerWarehouseResolver resolves the nearest active warehouse for a supplier
// based on retailer coordinates.
type SpannerWarehouseResolver struct {
	client *spanner.Client
}

// NewSpannerWarehouseResolver builds a Spanner-backed warehouse resolver.
func NewSpannerWarehouseResolver(client *spanner.Client) *SpannerWarehouseResolver {
	return &SpannerWarehouseResolver{client: client}
}

// ResolveNearestWarehouseID returns the closest warehouse id for the supplier.
// It only returns warehouses where distance <= coverage radius.
// When no active on-shift warehouse covers the coordinate, it returns empty,
// allowing callers to fail closed with a zone-miss contract.
func (r *SpannerWarehouseResolver) ResolveNearestWarehouseID(
	ctx context.Context,
	supplierID string,
	retailerLat float64,
	retailerLng float64,
) (string, error) {
	if r == nil || r.client == nil {
		return "", fmt.Errorf("spanner warehouse resolver: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return "", fmt.Errorf("supplier_id required")
	}
	if retailerLat == 0 && retailerLng == 0 {
		return "", nil
	}

	neighborCells, _ := proximity.CellsInRadius(retailerLat, retailerLng, 9, proximity.DefaultNeighborK())
	if len(neighborCells) > 0 {
		if id, err := r.resolveFromH3Candidates(ctx, supplierID, retailerLat, retailerLng, neighborCells); err != nil {
			return "", err
		} else if id != "" {
			return id, nil
		}
	}

	return r.resolveFromFullScan(ctx, supplierID, retailerLat, retailerLng)
}

func (r *SpannerWarehouseResolver) resolveFromH3Candidates(
	ctx context.Context,
	supplierID string,
	retailerLat, retailerLng float64,
	neighborCells []string,
) (string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Lat, Lng, CoverageRadiusKm
		      FROM Warehouses
		      WHERE SupplierId = @supplierId
		        AND IsActive = true
		        AND COALESCE(IsOnShift, true) = true
		        AND H3Cell IN UNNEST(@cells)`,
		Params: map[string]any{
			"supplierId": supplierID,
			"cells":      neighborCells,
		},
	}
	return r.scanWarehouseCandidates(ctx, stmt, retailerLat, retailerLng)
}

func (r *SpannerWarehouseResolver) resolveFromFullScan(
	ctx context.Context,
	supplierID string,
	retailerLat, retailerLng float64,
) (string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Lat, Lng, CoverageRadiusKm
		      FROM Warehouses
		      WHERE SupplierId = @supplierId
		        AND IsActive = true
		        AND COALESCE(IsOnShift, true) = true`,
		Params: map[string]any{"supplierId": supplierID},
	}
	return r.scanWarehouseCandidates(ctx, stmt, retailerLat, retailerLng)
}

func (r *SpannerWarehouseResolver) scanWarehouseCandidates(
	ctx context.Context,
	stmt spanner.Statement,
	retailerLat, retailerLng float64,
) (string, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	coveredID := ""
	coveredDist := math.MaxFloat64

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", fmt.Errorf("query warehouses: %w", err)
		}

		var warehouseID string
		var lat, lng, radius spanner.NullFloat64
		if err := row.Columns(&warehouseID, &lat, &lng, &radius); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}

		distance := haversineKm(retailerLat, retailerLng, lat.Float64, lng.Float64)

		effectiveRadius := math.MaxFloat64
		if radius.Valid && radius.Float64 > 0 {
			effectiveRadius = radius.Float64
		}
		if distance <= effectiveRadius && isCloserWarehouse(distance, warehouseID, coveredDist, coveredID) {
			coveredDist = distance
			coveredID = warehouseID
		}
	}

	if coveredID != "" {
		return coveredID, nil
	}
	return "", nil
}

func isCloserWarehouse(distance float64, warehouseID string, bestDistance float64, bestID string) bool {
	if distance < bestDistance {
		return true
	}
	if distance > bestDistance {
		return false
	}
	if bestID == "" {
		return true
	}
	return warehouseID < bestID
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}
