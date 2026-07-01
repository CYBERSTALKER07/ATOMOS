package predictivepush

import (
	"context"
	"math"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type Warehouse struct {
	WarehouseId      string
	PrimaryFactoryId string
	Lat              float64
	Lng              float64
}

type Locator struct {
	spannerClient *spanner.Client
}

func NewLocator(client *spanner.Client) *Locator {
	return &Locator{
		spannerClient: client,
	}
}

// FindNearestWarehouse finds the closest warehouse to a given retailer that belongs to the same supplier.
func (l *Locator) FindNearestWarehouse(ctx context.Context, retailerId string, supplierId string) (*Warehouse, error) {
	// First get the retailer's location
	var rLat, rLng spanner.NullFloat64
	stmt := spanner.Statement{
		SQL: `SELECT Lat, Lng FROM Retailers WHERE RetailerId = @retailerId`,
		Params: map[string]interface{}{
			"retailerId": retailerId,
		},
	}
	
	iter := l.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // Retailer not found
	}
	if err != nil {
		return nil, err
	}
	if err := row.Columns(&rLat, &rLng); err != nil {
		return nil, err
	}

	if !rLat.Valid || !rLng.Valid {
		return nil, nil // No location data for retailer
	}

	// Now get all warehouses for the supplier
	wStmt := spanner.Statement{
		SQL: `SELECT WarehouseId, PrimaryFactoryId, Lat, Lng FROM Warehouses WHERE SupplierId = @supplierId`,
		Params: map[string]interface{}{
			"supplierId": supplierId,
		},
	}
	
	wIter := l.spannerClient.Single().Query(ctx, wStmt)
	defer wIter.Stop()

	var nearest *Warehouse
	var minDistance = math.MaxFloat64

	for {
		wRow, wErr := wIter.Next()
		if wErr == iterator.Done {
			break
		}
		if wErr != nil {
			return nil, wErr
		}

		var wId string
		var pfId spanner.NullString
		var wLat, wLng spanner.NullFloat64

		if err := wRow.Columns(&wId, &pfId, &wLat, &wLng); err != nil {
			return nil, err
		}

		if wLat.Valid && wLng.Valid {
			dist := haversineDistance(rLat.Float64, rLng.Float64, wLat.Float64, wLng.Float64)
			if dist < minDistance {
				minDistance = dist
				nearest = &Warehouse{
					WarehouseId:      wId,
					PrimaryFactoryId: pfId.StringVal, // might be empty, that's okay
					Lat:              wLat.Float64,
					Lng:              wLng.Float64,
				}
			}
		}
	}

	return nearest, nil
}

// haversineDistance calculates the distance between two points in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := lat1 * math.Pi / 180.0
	lon1Rad := lon1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lon2Rad := lon2 * math.Pi / 180.0

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Pow(math.Sin(dLat/2), 2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(dLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
