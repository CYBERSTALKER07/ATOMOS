package routing

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type spannerQueryer interface {
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

// WaypointsForDriverRoute resolves depot + ordered stops for a driver's active route.
func WaypointsForDriverRoute(ctx context.Context, client spannerQueryer, driverID, routeID string) ([]LatLng, error) {
	driverID = strings.TrimSpace(driverID)
	routeID = strings.TrimSpace(routeID)
	if driverID == "" || routeID == "" {
		return nil, nil
	}

	waypoints := make([]LatLng, 0, 16)
	depotStmt := spanner.Statement{
		SQL: `SELECT w.Lat, w.Lng
		      FROM Orders o
		      JOIN Warehouses w ON w.WarehouseId = o.WarehouseId
		      WHERE o.DriverId = @did AND o.RouteId = @rid
		        AND w.Lat IS NOT NULL AND w.Lng IS NOT NULL
		      LIMIT 1`,
		Params: map[string]any{"did": driverID, "rid": routeID},
	}
	depotIter := client.Query(ctx, depotStmt)
	depotRow, depotErr := depotIter.Next()
	depotIter.Stop()
	if depotErr == nil {
		var depotLat, depotLng float64
		if err := depotRow.Columns(&depotLat, &depotLng); err == nil {
			waypoints = append(waypoints, LatLng{Lat: depotLat, Lng: depotLng})
		}
	}

	stopStmt := spanner.Statement{
		SQL: `SELECT o.Lat, o.Lng
		      FROM Orders o
		      LEFT JOIN ManifestOrders mo ON mo.ManifestId = o.ManifestId AND mo.OrderId = o.OrderId
		      WHERE o.DriverId = @did AND o.RouteId = @rid
		        AND o.Status NOT IN ('COMPLETED', 'CANCELLED')
		        AND o.Lat IS NOT NULL AND o.Lng IS NOT NULL
		      ORDER BY COALESCE(mo.SequenceIndex, 999999) ASC, o.CreatedAt ASC`,
		Params: map[string]any{"did": driverID, "rid": routeID},
	}
	stopIter := client.Query(ctx, stopStmt)
	defer stopIter.Stop()
	for {
		row, err := stopIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("route stop query: %w", err)
		}
		var lat, lng float64
		if err := row.Columns(&lat, &lng); err != nil {
			return nil, fmt.Errorf("route stop scan: %w", err)
		}
		waypoints = append(waypoints, LatLng{Lat: lat, Lng: lng})
	}
	return waypoints, nil
}

// WaypointsForManifest resolves depot + ordered stops for one supplier truck manifest.
func WaypointsForManifest(ctx context.Context, client spannerQueryer, manifestID string) ([]LatLng, error) {
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return nil, nil
	}

	waypoints := make([]LatLng, 0, 16)
	depotStmt := spanner.Statement{
		SQL: `SELECT w.Lat, w.Lng
		      FROM SupplierTruckManifests m
		      JOIN Warehouses w ON w.WarehouseId = m.WarehouseId
		      WHERE m.ManifestId = @mid
		        AND w.Lat IS NOT NULL AND w.Lng IS NOT NULL
		      LIMIT 1`,
		Params: map[string]any{"mid": manifestID},
	}
	depotIter := client.Query(ctx, depotStmt)
	depotRow, depotErr := depotIter.Next()
	depotIter.Stop()
	if depotErr == nil {
		var depotLat, depotLng float64
		if err := depotRow.Columns(&depotLat, &depotLng); err == nil {
			waypoints = append(waypoints, LatLng{Lat: depotLat, Lng: depotLng})
		}
	}

	stopStmt := spanner.Statement{
		SQL: `SELECT o.Lat, o.Lng
		      FROM ManifestOrders mo
		      JOIN Orders o ON o.OrderId = mo.OrderId
		      WHERE mo.ManifestId = @mid
		        AND o.Lat IS NOT NULL AND o.Lng IS NOT NULL
		      ORDER BY mo.SequenceIndex ASC, o.CreatedAt ASC`,
		Params: map[string]any{"mid": manifestID},
	}
	stopIter := client.Query(ctx, stopStmt)
	defer stopIter.Stop()
	for {
		row, err := stopIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("manifest stop query: %w", err)
		}
		var lat, lng float64
		if err := row.Columns(&lat, &lng); err != nil {
			return nil, fmt.Errorf("manifest stop scan: %w", err)
		}
		waypoints = append(waypoints, LatLng{Lat: lat, Lng: lng})
	}
	return waypoints, nil
}
