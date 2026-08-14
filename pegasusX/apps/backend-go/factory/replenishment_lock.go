package factory

import (
	"fmt"
	"time"
)

const replenishmentLockTTL = 10 * time.Minute

// LockResult is the outcome of a SKU+factory advisory lock attempt.
type LockResult struct {
	Acquired     bool
	LockKey      string
	WarehouseID  string
	Priority     float64
	HeldBy       string
	HeldPriority float64
}

type lockSnapshot struct {
	AcquiredBy string
	Priority   float64
	ExpiresAt  time.Time
	Present    bool
}

func replenishmentLockKey(skuID, factoryID string) string {
	return fmt.Sprintf("SKU:%s:FACTORY:%s", skuID, factoryID)
}

// DecideLockAcquire applies 10-minute TTL and velocity-priority preemption.
func DecideLockAcquire(now time.Time, warehouseID string, velocity float64, held lockSnapshot) LockResult {
	result := LockResult{
		WarehouseID: warehouseID,
		Priority:    velocity,
	}
	if !held.Present {
		result.Acquired = true
		return result
	}
	if now.Before(held.ExpiresAt) && held.AcquiredBy != warehouseID {
		if velocity > held.Priority {
			result.Acquired = true
			return result
		}
		result.HeldBy = held.AcquiredBy
		result.HeldPriority = held.Priority
		return result
	}
	result.Acquired = true
	return result
}
