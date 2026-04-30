// Package routing — 04:00 AM cron that fires the Field General optimizer.
// Register RunDailyCron in main.go:  go routing.RunDailyCron(ctx, spannerClient, mapsAPIKey, depotLatLng)
package routing

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	// cronHour is the local hour (0-23) at which the optimizer fires.
	// Set to 4 → 04:00 Tashkent time.
	cronHour   = 4
	cronMinute = 0
)

// StartCron blocks forever, sleeping until the next 04:00 window, then
// running OptimizeDriverRoute for every driver who has LOADED orders.
// The fallbackDepot is used only when a driver has no WarehouseId or
// the warehouse has no coordinates (backward compat with Phantom Node).
// Designed to be launched as a goroutine: go routing.StartCron(ctx, ...)
func StartCron(ctx context.Context, spannerClient *spanner.Client, apiKey, fallbackDepot string) {
	log.Println("[FieldGeneral:Cron] Scheduler armed — fires daily at 04:00")
	for {
		nextRun := nextCronTime()
		log.Printf("[FieldGeneral:Cron] Next optimization window: %s", nextRun.Format(time.RFC3339))

		select {
		case <-ctx.Done():
			log.Println("[FieldGeneral:Cron] Scheduler shutdown signal received")
			return
		case <-time.After(time.Until(nextRun)):
			log.Println("[FieldGeneral:Cron] FIRING — Field General waking up")
			runOptimizationPass(ctx, spannerClient, apiKey, fallbackDepot)
		}
	}
}

// driverWarehouse is a driver with their resolved warehouse depot.
type driverWarehouse struct {
	DriverID    string
	WarehouseID string
}

// runOptimizationPass groups drivers by warehouse, resolves each warehouse's
// depot coordinates, and optimizes routes per-warehouse.
func runOptimizationPass(ctx context.Context, client *spanner.Client, apiKey, fallbackDepot string) {
	drivers, err := GetActiveDriversWithWarehouse(ctx, client)
	if err != nil {
		log.Printf("[FieldGeneral:Cron] ERROR fetching active drivers: %v", err)
		return
	}
	if len(drivers) == 0 {
		log.Println("[FieldGeneral:Cron] No drivers with LOADED orders — pass complete")
		return
	}
	log.Printf("[FieldGeneral:Cron] Found %d driver(s) with LOADED orders", len(drivers))

	// Group by warehouse
	warehouseDrivers := make(map[string][]string) // warehouseID → []driverID
	noWarehouse := []string{}                     // drivers without warehouse assignment
	for _, dw := range drivers {
		if dw.WarehouseID == "" {
			noWarehouse = append(noWarehouse, dw.DriverID)
		} else {
			warehouseDrivers[dw.WarehouseID] = append(warehouseDrivers[dw.WarehouseID], dw.DriverID)
		}
	}

	// Resolve warehouse depots
	depotCache := make(map[string]string) // warehouseID → "lat,lng"
	for whID := range warehouseDrivers {
		depot, err := resolveWarehouseDepot(ctx, client, whID)
		if err != nil || depot == "" {
			log.Printf("[FieldGeneral:Cron] WARN: no depot for warehouse %s — using fallback", whID)
			depot = fallbackDepot
		}
		depotCache[whID] = depot
	}

	// Optimize each warehouse group
	for whID, driverIDs := range warehouseDrivers {
		depot := depotCache[whID]
		for _, driverID := range driverIDs {
			optimizeDriverWithDepot(ctx, client, apiKey, depot, driverID)
		}
	}

	// Optimize un-warehoused drivers with fallback depot
	for _, driverID := range noWarehouse {
		optimizeDriverWithDepot(ctx, client, apiKey, fallbackDepot, driverID)
	}

	log.Println("[FieldGeneral:Cron] Optimization pass complete")
}

func optimizeDriverWithDepot(ctx context.Context, client *spanner.Client, apiKey, depot, driverID string) {
	orders, err := GetLoadedOrdersForDriver(ctx, client, driverID)
	if err != nil {
		log.Printf("[FieldGeneral:Cron] ERROR fetching orders for driver=%s: %v", driverID, err)
		return
	}
	if len(orders) == 0 {
		return
	}
	if err := OptimizeDriverRoute(ctx, client, apiKey, depot, orders); err != nil {
		log.Printf("[FieldGeneral:Cron] ERROR optimizing driver=%s: %v", driverID, err)
		return
	}
	log.Printf("[FieldGeneral:Cron] Driver %s optimized (%d stops)", driverID, len(orders))
}

// resolveWarehouseDepot fetches lat/lng from the Warehouses table and returns "lat,lng".
func resolveWarehouseDepot(ctx context.Context, client *spanner.Client, warehouseID string) (string, error) {
	row, err := client.Single().ReadRow(ctx, "Warehouses",
		spanner.Key{warehouseID},
		[]string{"Lat", "Lng"})
	if err != nil {
		return "", fmt.Errorf("warehouse %s not found: %w", warehouseID, err)
	}

	var lat, lng spanner.NullFloat64
	if err := row.Columns(&lat, &lng); err != nil {
		return "", err
	}
	if !lat.Valid || !lng.Valid {
		return "", fmt.Errorf("warehouse %s has no coordinates", warehouseID)
	}
	return fmt.Sprintf("%f,%f", lat.Float64, lng.Float64), nil
}

// GetActiveDriversWithWarehouse returns drivers with LOADED/DISPATCHED orders
// along with their WarehouseId for per-warehouse depot resolution.
func GetActiveDriversWithWarehouse(ctx context.Context, client *spanner.Client) ([]driverWarehouse, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT o.DriverId, COALESCE(d.WarehouseId, '') AS WarehouseId
		      FROM Orders o
		      LEFT JOIN Drivers d ON o.DriverId = d.DriverId
		      WHERE o.State IN ('LOADED', 'DISPATCHED')`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []driverWarehouse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("routing: driver+warehouse query failed: %w", err)
		}
		var dw driverWarehouse
		if err := row.Columns(&dw.DriverID, &dw.WarehouseID); err != nil {
			return nil, fmt.Errorf("routing: row parse failed: %w", err)
		}
		results = append(results, dw)
	}
	return results, nil
}

// nextCronTime returns the next 04:00:00 wall-clock moment.
func nextCronTime() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), cronHour, cronMinute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
