package factory

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"google.golang.org/api/iterator"
)

// FactoryFleetDriverLocation is the factory-safe driver last-location projection.
type FactoryFleetDriverLocation struct {
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

// FactoryFleetLiveRoute is one active factory manifest on the fleet map.
type FactoryFleetLiveRoute struct {
	ManifestID            string                      `json:"manifest_id"`
	RouteID               string                      `json:"route_id,omitempty"`
	DriverID              string                      `json:"driver_id"`
	DriverName            string                      `json:"driver_name,omitempty"`
	ManifestState         string                      `json:"manifest_state"`
	OrderCount            int64                       `json:"order_count,omitempty"`
	LoadingStartedAt      string                      `json:"loading_started_at,omitempty"`
	DeliverySummary       string                      `json:"delivery_summary,omitempty"`
	DriverLocation        *FactoryFleetDriverLocation `json:"driver_location,omitempty"`
	LiveLocationAvailable bool                        `json:"live_location_available"`
	LocationStale         bool                        `json:"location_stale"`
}

// FactoryYardManifest is a LOADING manifest on the yard layer.
type FactoryYardManifest struct {
	ManifestID       string `json:"manifest_id"`
	DriverName       string `json:"driver_name,omitempty"`
	OrderCount       int64  `json:"order_count"`
	LoadingStartedAt string `json:"loading_started_at,omitempty"`
	DeliverySummary  string `json:"delivery_summary,omitempty"`
	ManifestState    string `json:"manifest_state"`
}

// FactoryFleetLiveMapResponse is GET /v1/factory/fleet/live-map.
type FactoryFleetLiveMapResponse struct {
	Routes        []FactoryFleetLiveRoute `json:"routes"`
	YardManifests []FactoryYardManifest   `json:"yard_manifests,omitempty"`
	FactoryID     string                  `json:"factory_id"`
	FetchedAt     string                  `json:"fetched_at"`
}

// HandleFactoryFleetLiveMap serves GET /v1/factory/fleet/live-map (driver pins; geometry deferred).
func (s *Service) HandleFactoryFleetLiveMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	factoryID, ok := s.scopedFactoryID(r)
	if !ok || factoryID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "factory_id_required"})
		return
	}
	routes, yard, err := s.listFactoryFleetLiveRoutes(r.Context(), factoryID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "factory fleet live map failed", "err", err, "factory_id", factoryID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fleet_live_map_failed"})
		return
	}
	writeJSON(w, http.StatusOK, FactoryFleetLiveMapResponse{
		Routes:        routes,
		YardManifests: yard,
		FactoryID:     factoryID,
		FetchedAt:     s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) listFactoryFleetLiveRoutes(ctx context.Context, factoryID string) ([]FactoryFleetLiveRoute, []FactoryYardManifest, error) {
	factoryID = strings.TrimSpace(factoryID)
	if factoryID == "" {
		return nil, nil, nil
	}
	if s.spannerClient == nil {
		return nil, nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT m.ManifestId, IFNULL(m.DriverId, ''), m.State, IFNULL(m.StopCount, 0),
		             m.LoadingStartedAt, IFNULL(d.Name, '')
		      FROM FactoryTruckManifests m
		      LEFT JOIN Drivers d ON d.DriverId = m.DriverId
		      WHERE m.FactoryId = @fid
		        AND m.State IN ('SEALED', 'DISPATCHED', 'LOADING')
		      ORDER BY m.UpdatedAt DESC
		      LIMIT 40`,
		Params: map[string]any{"fid": factoryID},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]FactoryFleetLiveRoute, 0, 8)
	yard := make([]FactoryYardManifest, 0, 8)
	seenDrivers := make(map[string]struct{}, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("factory fleet live map query: %w", err)
		}
		var manifestID, driverID, state, driverName string
		var stopCount int64
		var loadingStarted spanner.NullTime
		if err := row.Columns(&manifestID, &driverID, &state, &stopCount, &loadingStarted, &driverName); err != nil {
			return nil, nil, fmt.Errorf("factory fleet live map scan: %w", err)
		}
		if strings.EqualFold(state, "LOADING") {
			yardEntry := FactoryYardManifest{
				ManifestID:      manifestID,
				DriverName:      strings.TrimSpace(driverName),
				OrderCount:      stopCount,
				DeliverySummary: fmt.Sprintf("%d stops loading", stopCount),
				ManifestState:   state,
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
		liveRoute := FactoryFleetLiveRoute{
			ManifestID:      manifestID,
			RouteID:         "route_" + manifestID,
			DriverID:        driverID,
			DriverName:      strings.TrimSpace(driverName),
			ManifestState:   state,
			OrderCount:      stopCount,
			DeliverySummary: fmt.Sprintf("%d stops", stopCount),
		}
		if loadingStarted.Valid {
			liveRoute.LoadingStartedAt = loadingStarted.Time.UTC().Format(time.RFC3339Nano)
		}
		s.attachFactoryDriverLocation(ctx, &liveRoute)
		rows = append(rows, liveRoute)
	}
	return rows, yard, nil
}

func (s *Service) attachFactoryDriverLocation(ctx context.Context, route *FactoryFleetLiveRoute) {
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
	route.DriverLocation = factoryDriverLocationFromTelemetry(location)
	route.LiveLocationAvailable = location.IsLive(now)
	route.LocationStale = !location.IsLive(now)
}

func factoryDriverLocationFromTelemetry(location telemetry.DriverLocation) *FactoryFleetDriverLocation {
	return &FactoryFleetDriverLocation{
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
