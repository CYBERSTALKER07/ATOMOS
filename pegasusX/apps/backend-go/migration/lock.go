package migration

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// DefaultWarehouseLockTTL caps how long an active apply lease holds (supplier_id, warehouse_id)
// before an unrenewed crashed worker lock becomes eligible for reclaim.
const DefaultWarehouseLockTTL = 15 * time.Minute

var (
	// ErrWarehouseLockBusy indicates another active session holds the warehouse apply lock.
	ErrWarehouseLockBusy = errors.New("warehouse_lock_busy")
	// ErrWarehouseLockNotHeld indicates the session tried to renew or release a lock it does not own.
	ErrWarehouseLockNotHeld = errors.New("warehouse_lock_not_held")
)

func canAcquireLock(existing *WarehouseLock, sessionID string, now time.Time) bool {
	if existing == nil {
		return true
	}
	if strings.TrimSpace(existing.SessionID) == strings.TrimSpace(sessionID) {
		return true
	}
	return !existing.ExpiresAt.After(now)
}

func buildWarehouseLockBusyError(existing WarehouseLock) error {
	return fmt.Errorf("%w: warehouse %s locked by session %s until %s",
		ErrWarehouseLockBusy,
		existing.WarehouseID,
		existing.SessionID,
		existing.ExpiresAt.UTC().Format(time.RFC3339),
	)
}
