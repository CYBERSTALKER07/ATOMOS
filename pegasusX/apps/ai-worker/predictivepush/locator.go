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

// FindNearestWarehousesBatch resolves nearest warehouses for multiple (retailerID, supplierID) pairs in 2 batched queries.
func (l *Locator) FindNearestWarehousesBatch(ctx context.Context, events []*DemandEvent) (map[string]*Warehouse, error) {
	out := make(map[string]*Warehouse, len(events))
	if l == nil || l.spannerClient == nil || len(events) == 0 {
		return out, nil
	}
	ridSet := make(map[string]struct{}, len(events))
	sidSet := make(map[string]struct{}, len(events))
	rids := make([]string, 0, len(events))
	sids := make([]string, 0, len(events))
	for _, ev := range events {
		if ev == nil {
			continue
		}
		if ev.RetailerId != "" {
			if _, ok := ridSet[ev.RetailerId]; !ok {
				ridSet[ev.RetailerId] = struct{}{}
				rids = append(rids, ev.RetailerId)
			}
		}
		if ev.SupplierId != "" {
			if _, ok := sidSet[ev.SupplierId]; !ok {
				sidSet[ev.SupplierId] = struct{}{}
				sids = append(sids, ev.SupplierId)
			}
		}
	}
	if len(rids) == 0 || len(sids) == 0 {
		return out, nil
	}

	ro := l.spannerClient.ReadOnlyTransaction()
	defer ro.Close()

	type point struct{ lat, lng float64 }
	retCoords := make(map[string]point, len(rids))
	rIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT RetailerId, Lat, Lng FROM Retailers WHERE RetailerId IN UNNEST(@rids)`,
		Params: map[string]any{"rids": rids},
	})
	for {
		row, err := rIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			rIter.Stop()
			return nil, err
		}
		var rid string
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&rid, &lat, &lng); err != nil {
			rIter.Stop()
			return nil, err
		}
		if lat.Valid && lng.Valid {
			retCoords[rid] = point{lat: lat.Float64, lng: lng.Float64}
		}
	}
	rIter.Stop()

	whBySupplier := make(map[string][]Warehouse, len(sids))
	wIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT SupplierId, WarehouseId, PrimaryFactoryId, Lat, Lng FROM Warehouses WHERE SupplierId IN UNNEST(@sids)`,
		Params: map[string]any{"sids": sids},
	})
	for {
		row, err := wIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			wIter.Stop()
			return nil, err
		}
		var sid, wid string
		var pf spanner.NullString
		var wLat, wLng spanner.NullFloat64
		if err := row.Columns(&sid, &wid, &pf, &wLat, &wLng); err != nil {
			wIter.Stop()
			return nil, err
		}
		if wLat.Valid && wLng.Valid {
			whBySupplier[sid] = append(whBySupplier[sid], Warehouse{
				WarehouseId:      wid,
				PrimaryFactoryId: pf.StringVal,
				Lat:              wLat.Float64,
				Lng:              wLng.Float64,
			})
		}
	}
	wIter.Stop()

	for _, ev := range events {
		if ev == nil || ev.RetailerId == "" || ev.SupplierId == "" {
			continue
		}
		key := ev.RetailerId + "|" + ev.SupplierId
		if _, done := out[key]; done {
			continue
		}
		rc, hasR := retCoords[ev.RetailerId]
		if !hasR {
			out[key] = nil
			continue
		}
		var nearest *Warehouse
		minDist := math.MaxFloat64
		for _, w := range whBySupplier[ev.SupplierId] {
			d := haversineDistance(rc.lat, rc.lng, w.Lat, w.Lng)
			if d < minDist {
				minDist = d
				wCopy := w
				nearest = &wCopy
			}
		}
		out[key] = nearest
	}
	return out, nil
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
