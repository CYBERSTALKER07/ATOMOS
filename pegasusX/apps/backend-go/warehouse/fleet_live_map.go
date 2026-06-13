package warehouse

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"google.golang.org/api/iterator"
)

// WarehouseFleetDriverLocation is the warehouse-safe driver last-location projection.
type WarehouseFleetDriverLocation struct {
	DriverID          string   `json:"driver_id"`
	SupplierID        string   `json:"supplier_id,omitempty"`
	Lat               float64  `json:"lat"`
	Lng               float64  `json:"lng"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	Velocity          *float64 `json:"velocity,omitempty"`
	Heading           *float64 `json:"heading,omitempty"`
	ReportedAt        string   `json:"reported_at"`
	ReceivedAt        string   `json:"received_at"`
	StaleAfterSeconds int      `json:"stale_after_seconds"`
}

// WarehouseFleetLiveRoute is one active manifest route on the warehouse fleet map.
type WarehouseFleetLiveRoute struct {
	ManifestID            string                        `json:"manifest_id"`
	RouteID               string                        `json:"route_id"`
	DriverID              string                        `json:"driver_id"`
	DriverName            string                        `json:"driver_name,omitempty"`
	ManifestState         string                        `json:"manifest_state"`
	RouteGeometry         *routing.RouteGeometry        `json:"route_geometry,omitempty"`
	DriverLocation        *WarehouseFleetDriverLocation `json:"driver_location,omitempty"`
	LiveLocationAvailable bool                          `json:"live_location_available"`
	LocationStale         bool                          `json:"location_stale"`
}

// WarehouseFleetLiveMapResponse is GET /v1/warehouse/ops/fleet/live-map.
type WarehouseFleetLiveMapResponse struct {
	Routes      []WarehouseFleetLiveRoute `json:"routes"`
	WarehouseID string                    `json:"warehouse_id"`
	FetchedAt   string                    `json:"fetched_at"`
}

// HandleWarehouseFleetLiveMap serves GET /v1/warehouse/ops/fleet/live-map.
func (s *Service) HandleWarehouseFleetLiveMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	routes, err := s.listWarehouseFleetLiveRoutes(r.Context(), whID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "warehouse fleet live map failed", "err", err, "warehouse_id", whID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_live_map_failed"})
		return
	}
	writeJSON(w, http.StatusOK, WarehouseFleetLiveMapResponse{
		Routes:      routes,
		WarehouseID: whID,
		FetchedAt:   s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) listWarehouseFleetLiveRoutes(ctx context.Context, warehouseID string) ([]WarehouseFleetLiveRoute, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return nil, nil
	}
	if s.spannerClient == nil {
		return nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT m.ManifestId, m.RouteId, m.DriverId, m.State,
		             m.EncodedRoutePolyline, m.RouteGeometrySource, m.StopCount,
		             d.Name
		      FROM SupplierTruckManifests m
		      LEFT JOIN Drivers d ON d.DriverId = m.DriverId
		      WHERE m.WarehouseId = @wh
		        AND m.State IN ('SEALED', 'DISPATCHED')
		      ORDER BY m.UpdatedAt DESC
		      LIMIT 40`,
		Params: map[string]any{"wh": warehouseID},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]WarehouseFleetLiveRoute, 0, 8)
	seenDrivers := make(map[string]struct{}, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse fleet live map query: %w", err)
		}
		var manifestID, routeID, driverID, state, driverName string
		var encoded spanner.NullString
		var source spanner.NullString
		var stopCount int64
		if err := row.Columns(&manifestID, &routeID, &driverID, &state, &encoded, &source, &stopCount, &driverName); err != nil {
			return nil, fmt.Errorf("warehouse fleet live map scan: %w", err)
		}
		driverID = strings.TrimSpace(driverID)
		if driverID != "" {
			if _, dup := seenDrivers[driverID]; dup {
				continue
			}
			seenDrivers[driverID] = struct{}{}
		}
		liveRoute := WarehouseFleetLiveRoute{
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
		s.attachWarehouseDriverLocation(ctx, &liveRoute)
		rows = append(rows, liveRoute)
	}
	return rows, nil
}

func (s *Service) attachWarehouseDriverLocation(ctx context.Context, route *WarehouseFleetLiveRoute) {
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
	if strings.TrimSpace(s.supplierID) != "" && strings.TrimSpace(location.SupplierID) != s.supplierID {
		return
	}
	now := s.now()
	route.DriverLocation = warehouseDriverLocationFromTelemetry(location)
	route.LiveLocationAvailable = location.IsLive(now)
	route.LocationStale = !location.IsLive(now)
}

func warehouseDriverLocationFromTelemetry(location telemetry.DriverLocation) *WarehouseFleetDriverLocation {
	return &WarehouseFleetDriverLocation{
		DriverID:          location.DriverID,
		SupplierID:        location.SupplierID,
		Lat:               location.Lat,
		Lng:               location.Lng,
		Latitude:          location.Latitude,
		Longitude:         location.Longitude,
		Velocity:          location.Velocity,
		Heading:           location.Heading,
		ReportedAt:        location.ReportedAt.UTC().Format(time.RFC3339Nano),
		ReceivedAt:        location.ReceivedAt.UTC().Format(time.RFC3339Nano),
		StaleAfterSeconds: location.StaleAfterSeconds,
	}
}
