package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
)

// AssertLaborCapacityAvailable hard-refuses dispatch when LABOR_CAPACITY_ENFORCE is
// effective and UsedCapacity + additionalStops > TotalCapacity for the zone/date (G2.C).
// Missing ZoneCapacity row: fail-closed only when enforce is on (no silent overload).
func AssertLaborCapacityAvailable(ctx context.Context, client *spanner.Client, warehouseID, supplierID, zoneH3 string, additionalStops int64, force bool) error {
	if force {
		return nil
	}
	if !EffectiveLaborCapacityEnforce(ctx, warehouseID, supplierID) {
		return nil
	}
	if client == nil {
		return fmt.Errorf("%w: spanner unavailable", ErrLaborCapacityExceeded)
	}
	zoneH3 = strings.TrimSpace(zoneH3)
	if zoneH3 == "" {
		// No zone resolved → cannot enforce; fail closed for seal-class labor.
		return fmt.Errorf("%w: zone_unknown", ErrLaborCapacityExceeded)
	}
	if additionalStops < 0 {
		additionalStops = 0
	}
	day := civil.DateOf(time.Now().UTC())
	row, err := client.Single().ReadRow(ctx, "ZoneCapacity", spanner.Key{zoneH3, day},
		[]string{"TotalCapacity", "UsedCapacity"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return fmt.Errorf("%w: no_capacity_snapshot", ErrLaborCapacityExceeded)
		}
		return fmt.Errorf("%w: %v", ErrLaborCapacityExceeded, err)
	}
	var total, used int64
	if err := row.Columns(&total, &used); err != nil {
		return fmt.Errorf("%w: %v", ErrLaborCapacityExceeded, err)
	}
	if used+additionalStops > total {
		return fmt.Errorf("%w: used=%d additional=%d total=%d zone=%s",
			ErrLaborCapacityExceeded, used, additionalStops, total, zoneH3)
	}
	return nil
}
