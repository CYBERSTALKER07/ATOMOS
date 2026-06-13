package manifest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"google.golang.org/api/iterator"
)

type manifestRouteInputs struct {
	routeID   string
	stopCount int
	waypoints []routing.LatLng
}

// PersistRouteGeometryForManifest computes and stores the route overlay at seal time.
func (s *Store) PersistRouteGeometryForManifest(ctx context.Context, manifestID, source string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return fmt.Errorf("manifest store: empty manifest id")
	}
	if source == "" {
		source = "manifest_sealed"
	}

	inputs, err := s.loadManifestRouteInputs(ctx, manifestID)
	if err != nil {
		return fmt.Errorf("persist manifest route geometry %s: %w", manifestID, err)
	}
	geometry := s.buildRouteGeometry(ctx, inputs)
	if geometry.EncodedPolyline == "" {
		return nil
	}
	return s.writeRouteGeometry(ctx, manifestID, geometry, source)
}

// PersistRouteGeometryForDriverRoute recomputes and stores geometry after stop reorder.
func (s *Store) PersistRouteGeometryForDriverRoute(ctx context.Context, driverID, routeID, source string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	driverID = strings.TrimSpace(driverID)
	routeID = strings.TrimSpace(routeID)
	if driverID == "" || routeID == "" {
		return fmt.Errorf("manifest store: driver and route id required")
	}
	if source == "" {
		source = "route_reordered"
	}

	inputs, err := s.loadDriverRouteInputs(ctx, driverID, routeID)
	if err != nil {
		return fmt.Errorf("persist route geometry %s: %w", routeID, err)
	}
	geometry := s.buildRouteGeometry(ctx, inputs)
	if geometry.EncodedPolyline == "" {
		return nil
	}
	manifestID, err := s.latestManifestIDForRouteRead(ctx, driverID, routeID)
	if err != nil {
		return err
	}
	if manifestID == "" {
		return nil
	}
	return s.writeRouteGeometry(ctx, manifestID, geometry, source)
}

func (s *Store) buildRouteGeometry(ctx context.Context, inputs manifestRouteInputs) routing.RouteGeometry {
	var geometry routing.RouteGeometry
	if s.geometryBuilder != nil {
		geometry = s.geometryBuilder.Build(ctx, inputs.routeID, inputs.waypoints)
	} else {
		geometry = routing.BuildDenseRouteGeometry(inputs.routeID, inputs.waypoints)
	}
	if inputs.stopCount > 0 {
		geometry.StopCount = inputs.stopCount
	}
	return geometry
}

func (s *Store) loadManifestRouteInputs(ctx context.Context, manifestID string) (manifestRouteInputs, error) {
	row, err := s.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).
		ReadRow(ctx, "SupplierTruckManifests", spanner.Key{manifestID},
			[]string{"RouteId", "StopCount"})
	if err != nil {
		return manifestRouteInputs{}, fmt.Errorf("read manifest: %w", err)
	}
	var routeID spanner.NullString
	var stopCount int64
	if err := row.Columns(&routeID, &stopCount); err != nil {
		return manifestRouteInputs{}, err
	}

	waypoints, err := routing.WaypointsForManifest(ctx, s.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), manifestID)
	if err != nil {
		return manifestRouteInputs{}, err
	}
	return manifestRouteInputs{
		routeID:   routeIDString(routeID),
		stopCount: int(stopCount),
		waypoints: waypoints,
	}, nil
}

func (s *Store) loadDriverRouteInputs(ctx context.Context, driverID, routeID string) (manifestRouteInputs, error) {
	waypoints, err := routing.WaypointsForDriverRoute(ctx, s.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), driverID, routeID)
	if err != nil {
		return manifestRouteInputs{}, err
	}
	return manifestRouteInputs{
		routeID:   routeID,
		waypoints: waypoints,
		stopCount: countStops(waypoints),
	}, nil
}

func (s *Store) latestManifestIDForRouteRead(ctx context.Context, driverID, routeID string) (string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId
		      FROM SupplierTruckManifests
		      WHERE DriverId = @did AND RouteId = @rid
		      ORDER BY UpdatedAt DESC
		      LIMIT 1`,
		Params: map[string]any{"did": driverID, "rid": routeID},
	}
	iter := s.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lookup manifest for route: %w", err)
	}
	var manifestID string
	if err := row.Columns(&manifestID); err != nil {
		return "", err
	}
	return manifestID, nil
}

func (s *Store) writeRouteGeometry(ctx context.Context, manifestID string, geometry routing.RouteGeometry, source string) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("SupplierTruckManifests", map[string]any{
				"ManifestId":           manifestID,
				"EncodedRoutePolyline": geometry.EncodedPolyline,
				"RouteGeometrySource":  source,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func countStops(waypoints []routing.LatLng) int {
	if len(waypoints) <= 1 {
		return len(waypoints)
	}
	return len(waypoints) - 1
}

func routeIDString(routeID spanner.NullString) string {
	if routeID.Valid {
		return routeID.StringVal
	}
	return ""
}
