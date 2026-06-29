package warehouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (s *Service) loadWarehouseManifestsFromSpanner(ctx context.Context, warehouseID string) ([]portalManifest, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return []portalManifest{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT m.ManifestId, COALESCE(d.Name, m.DriverId), COALESCE(v.LicensePlate, m.TruckId),
		             m.StopCount, m.CreatedAt
		      FROM SupplierTruckManifests m
		      LEFT JOIN Drivers d ON d.DriverId = m.DriverId
		      LEFT JOIN Vehicles v ON v.VehicleId = m.TruckId
		      WHERE m.WarehouseId = @wh
		      ORDER BY m.UpdatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"wh": warehouseID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]portalManifest, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse manifests query: %w", err)
		}
		var manifestID, driverName, vehicleLabel string
		var stopCount int64
		var createdAt time.Time
		if err := row.Columns(&manifestID, &driverName, &vehicleLabel, &stopCount, &createdAt); err != nil {
			return nil, fmt.Errorf("warehouse manifests scan: %w", err)
		}
		out = append(out, portalManifest{
			ManifestID:   manifestID,
			DriverName:   strings.TrimSpace(driverName),
			VehicleLabel: strings.TrimSpace(vehicleLabel),
			StopCount:    int(stopCount),
			CreatedAt:    createdAt.UTC().Format(time.RFC3339Nano),
		})
	}
}
