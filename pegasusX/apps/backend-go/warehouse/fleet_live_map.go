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
	OrderCount            int64                         `json:"order_count,omitempty"`
	LoadingStartedAt      string                        `json:"loading_started_at,omitempty"`
	DeliverySummary       string                        `json:"delivery_summary,omitempty"`
	RouteGeometry         *routing.RouteGeometry        `json:"route_geometry,omitempty"`
	DriverLocation        *WarehouseFleetDriverLocation `json:"driver_location,omitempty"`
	LiveLocationAvailable bool                          `json:"live_location_available"`
	LocationStale         bool                          `json:"location_stale"`
}

// WarehouseFleetLiveMapResponse is GET /v1/warehouse/ops/fleet/live-map.
type WarehouseFleetLiveMapResponse struct {
	Routes       []WarehouseFleetLiveRoute `json:"routes"`
	YardManifests []WarehouseYardManifest  `json:"yard_manifests,omitempty"`
	WarehouseID  string                    `json:"warehouse_id"`
	FetchedAt    string                    `json:"fetched_at"`
}

// WarehouseYardManifest is a LOADING manifest on the yard radar layer.
type WarehouseYardManifest struct {
	ManifestID       string `json:"manifest_id"`
	DriverName       string `json:"driver_name,omitempty"`
	OrderCount       int64  `json:"order_count"`
	LoadingStartedAt string `json:"loading_started_at,omitempty"`
	DeliverySummary  string `json:"delivery_summary,omitempty"`
	ManifestState    string `json:"manifest_state"`
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
	routes, yard, err := s.listWarehouseFleetLiveRoutes(r.Context(), whID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "warehouse fleet live map failed", "err", err, "warehouse_id", whID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_live_map_failed"})
		return
	}
	writeJSON(w, http.StatusOK, WarehouseFleetLiveMapResponse{
		Routes:        routes,
		YardManifests: yard,
		WarehouseID:   whID,
		FetchedAt:     s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) listWarehouseFleetLiveRoutes(ctx context.Context, warehouseID string) ([]WarehouseFleetLiveRoute, []WarehouseYardManifest, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return nil, nil, nil
	}
	if s.spannerClient == nil {
		return nil, nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT m.ManifestId, m.RouteId, m.DriverId, m.State,
		             m.EncodedRoutePolyline, m.RouteGeometrySource, m.StopCount,
		             m.LoadingStartedAt, d.Name
		      FROM SupplierTruckManifests m
		      LEFT JOIN Drivers d ON d.DriverId = m.DriverId
		      WHERE m.WarehouseId = @wh
		        AND m.State IN ('SEALED', 'DISPATCHED', 'LOADING')
		      ORDER BY m.UpdatedAt DESC
		      LIMIT 40`,
		Params: map[string]any{"wh": warehouseID},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]WarehouseFleetLiveRoute, 0, 8)
	yard := make([]WarehouseYardManifest, 0, 8)
	seenDrivers := make(map[string]struct{}, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("warehouse fleet live map query: %w", err)
		}
		var manifestID, routeID, driverID, state, driverName string
		var encoded spanner.NullString
		var source spanner.NullString
		var stopCount int64
		var loadingStarted spanner.NullTime
		if err := row.Columns(&manifestID, &routeID, &driverID, &state, &encoded, &source, &stopCount, &loadingStarted, &driverName); err != nil {
			return nil, nil, fmt.Errorf("warehouse fleet live map scan: %w", err)
		}
		if strings.EqualFold(state, "LOADING") {
			yardEntry := WarehouseYardManifest{
				ManifestID:    manifestID,
				DriverName:    strings.TrimSpace(driverName),
				OrderCount:    stopCount,
				DeliverySummary: fmt.Sprintf("%d stops loading", stopCount),
				ManifestState: state,
			}
			if loadingStarted.Valid {
				yardEntry.LoadingStartedAt = loadingStarted.Time.UTC().Format(time.RFC3339Nano)
			}
			yard = append(yard, yardEntry)
			continue
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
			OrderCount:    stopCount,
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
	return rows, yard, nil
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
