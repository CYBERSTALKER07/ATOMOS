package order

import (
	"context"
	"fmt"
	"math"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerWarehouseResolver resolves the nearest covering warehouse for a supplier.
type SpannerWarehouseResolver struct {
	client *spanner.Client
}

// NewSpannerWarehouseResolver builds a Spanner-backed warehouse resolver.
func NewSpannerWarehouseResolver(client *spanner.Client) *SpannerWarehouseResolver {
	return &SpannerWarehouseResolver{client: client}
}

// ResolveNearestWarehouseID returns the closest warehouse that covers the retailer.
// Hybrid: no coverage cells → whole warehouse country; cells set → H3 membership.
func (r *SpannerWarehouseResolver) ResolveNearestWarehouseID(
	ctx context.Context,
	supplierID string,
	retailerLat float64,
	retailerLng float64,
	retailerCountry string,
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

	cellsByWH, err := r.loadCoverageCells(ctx, supplierID)
	if err != nil {
		return "", err
	}
	retailerCell := coverageH3Cell(retailerLat, retailerLng)

	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Lat, Lng, COALESCE(CountryCode, '')
		      FROM Warehouses
		      WHERE SupplierId = @supplierId
		        AND IsActive = true
		        AND COALESCE(IsOnShift, true) = true`,
		Params: map[string]any{"supplierId": supplierID},
	}
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
		var warehouseID, country string
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&warehouseID, &lat, &lng, &country); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}
		if !WarehouseCoversRetailer(country, cellsByWH[warehouseID], retailerCountry, retailerCell) {
			continue
		}
		distance := haversineKm(retailerLat, retailerLng, lat.Float64, lng.Float64)
		if isCloserWarehouse(distance, warehouseID, coveredDist, coveredID) {
			coveredDist = distance
			coveredID = warehouseID
		}
	}
	return coveredID, nil
}

func (r *SpannerWarehouseResolver) loadCoverageCells(ctx context.Context, supplierID string) (map[string][]string, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
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
	const earthKm = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

func (s *Service) lookupRetailerCountry(ctx context.Context, retailerID string) string {
	if s == nil || s.spannerClient == nil {
		return ""
	}
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return ""
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"CountryCode"})
	if err != nil {
		return ""
	}
	var code spanner.NullString
	if err := row.Column(0, &code); err != nil || !code.Valid {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(code.StringVal))
}
