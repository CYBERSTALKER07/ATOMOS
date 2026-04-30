package dispatch

import (
	"context"

	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Dispatch Constants ─────────────────────────────────────────────────────

const (
	// TetrisBuffer is the volumetric safety margin applied to truck capacity.
	TetrisBuffer = 0.95

	// MaxDetourRadius is the maximum km detour for outlier sweep.
	MaxDetourRadius = 10.0

	// MaxWaypointsPerManifest is the Google Maps waypoint ceiling.
	MaxWaypointsPerManifest = 25

	// VUDivisor converts LWH (cm) to volumetric units: L×W×H / 5000.
	VUDivisor = 5000.0

	// DefaultUnitVolumeVU is the per-unit volume assumption used by INSERT
	// paths and the backfill binary while SupplierProducts.VolumePerUnit is
	// not yet a column. When that column lands, callers should compute volume
	// as Σ(quantity × VolumePerUnit) instead of Σ(quantity × DefaultUnitVolumeVU).
	DefaultUnitVolumeVU = 1.0
)

// ─── Driver Status State Machine ────────────────────────────────────────────

const (
	DriverStatusAvailable   = "AVAILABLE"
	DriverStatusIdle        = "IDLE"
	DriverStatusLoading     = "LOADING"
	DriverStatusReady       = "READY"
	DriverStatusInTransit   = "IN_TRANSIT"
	DriverStatusReturning   = "RETURNING"
	DriverStatusMaintenance = "MAINTENANCE"
)

// IsDispatchable returns true if the driver status allows new route assignment.
func IsDispatchable(status string, isOffline bool) bool {
	if isOffline {
		return false
	}
	switch status {
	case DriverStatusIdle, DriverStatusAvailable, DriverStatusReturning:
		return true
	default:
		return false
	}
}

// CalculateVU converts physical dimensions (cm) to Volumetric Units.
func CalculateVU(lengthCM, widthCM, heightCM float64) float64 {
	return (lengthCM * widthCM * heightCM) / VUDivisor
}

// ─── Service ────────────────────────────────────────────────────────────────

// Service encapsulates auto-dispatch, bin-packing, and routing logic
// for all operational scopes (Supplier, Warehouse, Factory).
type Service struct {
	Spanner *spanner.Client
	Cache   *cache.Cache
}

// NewService creates a new Dispatch Service.
func NewService(spannerClient *spanner.Client, c *cache.Cache) *Service {
	return &Service{
		Spanner: spannerClient,
		Cache:   c,
	}
}

// IsFreezeLocked checks if a dispatch lock is active for the given scope.
func (s *Service) IsFreezeLocked(ctx context.Context, warehouseID, factoryID string) (bool, error) {
	sql := `SELECT 1 FROM DispatchLocks
	        WHERE UnlockedAt IS NULL
	          AND (WarehouseId = @wid OR FactoryId = @fid)
	        LIMIT 1`
	params := map[string]interface{}{"wid": warehouseID, "fid": factoryID}
	stmt := spanner.Statement{SQL: sql, Params: params}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
