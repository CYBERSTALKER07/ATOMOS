package supplier

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"google.golang.org/api/iterator"
)

// SupplierFleetLiveRoute is one active manifest route on the supplier fleet map.
type SupplierFleetLiveRoute struct {
	ManifestID            string                  `json:"manifest_id"`
	RouteID               string                  `json:"route_id"`
	DriverID              string                  `json:"driver_id"`
	DriverName            string                  `json:"driver_name,omitempty"`
	ManifestState         string                  `json:"manifest_state"`
	RouteGeometry         *routing.RouteGeometry  `json:"route_geometry,omitempty"`
	DriverLocation        *SupplierOrderLocation  `json:"driver_location,omitempty"`
	LiveLocationAvailable bool                    `json:"live_location_available"`
	LocationStale         bool                    `json:"location_stale"`
}

// SupplierFleetLiveMapResponse is GET /v1/supplier/fleet/live-map.
type SupplierFleetLiveMapResponse struct {
	Routes    []SupplierFleetLiveRoute `json:"routes"`
	FetchedAt string                   `json:"fetched_at"`
}

// HandleSupplierFleetLiveMap serves GET /v1/supplier/fleet/live-map.
func (s *Service) HandleSupplierFleetLiveMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	routes, err := s.listFleetLiveRoutes(r.Context(), sid)
	if err != nil {
		s.log.ErrorContext(r.Context(), "supplier fleet live map failed", "err", err, "supplier_id", sid)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_live_map_failed"})
		return
	}
	writeJSON(w, http.StatusOK, SupplierFleetLiveMapResponse{
		Routes:    routes,
		FetchedAt: s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) listFleetLiveRoutes(ctx context.Context, supplierID string) ([]SupplierFleetLiveRoute, error) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, nil
	}
	if s.portalSpanner == nil {
		return s.fleetLiveRoutesFromOrders(ctx, supplierID)
	}

	stmt := spanner.Statement{
		SQL: `SELECT m.ManifestId, m.RouteId, m.DriverId, m.State,
		             m.EncodedRoutePolyline, m.RouteGeometrySource, m.StopCount,
		             d.Name
		      FROM SupplierTruckManifests m
		      INNER JOIN Drivers d ON d.DriverId = m.DriverId AND d.SupplierId = m.SupplierId
		      WHERE m.SupplierId = @sid
		        AND m.State IN ('SEALED', 'DISPATCHED')
		        AND d.OnShift = true
		      ORDER BY m.UpdatedAt DESC
		      LIMIT 40`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := s.portalSpanner.Single().
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]SupplierFleetLiveRoute, 0, 8)
	seenDrivers := make(map[string]struct{}, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fleet live map query: %w", err)
		}
		var manifestID, routeID, driverID, state, driverName string
		var encoded spanner.NullString
		var source spanner.NullString
		var stopCount int64
		if err := row.Columns(&manifestID, &routeID, &driverID, &state, &encoded, &source, &stopCount, &driverName); err != nil {
			return nil, fmt.Errorf("fleet live map scan: %w", err)
		}
		driverID = strings.TrimSpace(driverID)
		if driverID != "" {
			if _, dup := seenDrivers[driverID]; dup {
				continue
			}
			seenDrivers[driverID] = struct{}{}
		}
		liveRoute := SupplierFleetLiveRoute{
			ManifestID:    manifestID,
			RouteID:       routeID,
			DriverID:      driverID,
			DriverName:    strings.TrimSpace(driverName),
			ManifestState: state,
		}
		if encoded.Valid && encoded.StringVal != "" {
			geometry, decodeErr := routing.GeometryFromStoredPolyline(
				routeID,
				encoded.StringVal,
				source.StringVal,
				int(stopCount),
			)
			if decodeErr == nil {
				liveRoute.RouteGeometry = &geometry
			}
		}
		s.attachDriverLocation(ctx, supplierID, &liveRoute)
		rows = append(rows, liveRoute)
	}
	if len(rows) == 0 {
		return s.fleetLiveRoutesFromOrders(ctx, supplierID)
	}
	return rows, nil
}

func (s *Service) fleetLiveRoutesFromOrders(ctx context.Context, supplierID string) ([]SupplierFleetLiveRoute, error) {
	orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
	if err != nil {
		return nil, err
	}
	orders = s.attachOrderLocations(ctx, supplierID, orders)
	byRoute := make(map[string]SupplierFleetLiveRoute)
	for _, order := range orders {
		status := strings.ToUpper(strings.TrimSpace(order.Status))
		if status == "COMPLETED" || status == "CANCELLED" || status == "PENDING" {
			continue
		}
		routeID := strings.TrimSpace(order.RouteID)
		driverID := strings.TrimSpace(order.DriverID)
		if routeID == "" || driverID == "" {
			continue
		}
		key := routeID + ":" + driverID
		if _, ok := byRoute[key]; ok {
			continue
		}
		row := SupplierFleetLiveRoute{
			ManifestID:    strings.TrimSpace(order.ManifestID),
			RouteID:       routeID,
			DriverID:      driverID,
			ManifestState: status,
		}
		if order.DriverLocation != nil {
			row.DriverLocation = order.DriverLocation
			row.LiveLocationAvailable = order.LiveLocationAvailable
		}
		byRoute[key] = row
	}
	rows := make([]SupplierFleetLiveRoute, 0, len(byRoute))
	for _, row := range byRoute {
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *Service) attachDriverLocation(ctx context.Context, supplierID string, route *SupplierFleetLiveRoute) {
	if s == nil || route == nil || s.locations == nil {
		return
	}
	driverID := strings.TrimSpace(route.DriverID)
	if driverID == "" {
		return
	}
	location, found, err := s.locations.GetDriverLocation(ctx, driverID)
	if err != nil || !found {
		return
	}
	now := s.now()
	if strings.TrimSpace(location.SupplierID) != supplierID {
		return
	}
	route.DriverLocation = supplierOrderLocationFromTelemetry(location)
	route.LiveLocationAvailable = location.IsLive(now)
	route.LocationStale = !location.IsLive(now)
}
