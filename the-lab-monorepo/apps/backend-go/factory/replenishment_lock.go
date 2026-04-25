package factory

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Replenishment Lock — Distributed Advisory Lock ────────────────────────────
// When multiple warehouses simultaneously breach SafetyStockLevel for the same
// SKU at the same factory, we need a priority-based lock to prevent double-
// allocation of limited factory capacity.
//
// Priority is determined by 30-day sales velocity (higher velocity = higher
// priority = gets the factory allocation first). The loser gets routed to the
// next-best factory via SupplyLanes fallback.
//
// Locks expire after 10 minutes (configurable) to prevent deadlocks.

const lockTTL = 10 * time.Minute

// ReplenishmentLockService handles distributed replenishment lock acquisition.
type ReplenishmentLockService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

// LockResult describes the outcome of a lock attempt.
type LockResult struct {
	Acquired     bool    `json:"acquired"`
	LockKey      string  `json:"lock_key"`
	WarehouseId  string  `json:"warehouse_id"`
	Priority     float64 `json:"priority"`
	HeldBy       string  `json:"held_by,omitempty"`       // if not acquired
	HeldPriority float64 `json:"held_priority,omitempty"` // if not acquired
}

// AcquireLock attempts to acquire a replenishment lock for a warehouse on a
// specific SKU+Factory combination. Uses Spanner ReadWriteTransaction to
// guarantee atomicity.
//
// Returns: LockResult indicating success or showing who holds the lock.
func (s *ReplenishmentLockService) AcquireLock(ctx context.Context, supplierID, warehouseID, skuID, factoryID string) (*LockResult, error) {
	lockKey := fmt.Sprintf("SKU:%s:FACTORY:%s", skuID, factoryID)
	now := time.Now().UTC()

	// Calculate this warehouse's 30-day sales velocity for this SKU
	velocity, err := s.calculateSalesVelocity(ctx, supplierID, warehouseID, skuID)
	if err != nil {
		log.Printf("[REPLENISHMENT_LOCK] velocity calc error for %s/%s: %v", warehouseID, skuID, err)
		velocity = 0
	}

	var result LockResult
	result.LockKey = lockKey
	result.WarehouseId = warehouseID
	result.Priority = velocity

	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check if lock exists and is not expired
		row, err := txn.ReadRow(ctx, "ReplenishmentLocks", spanner.Key{lockKey},
			[]string{"AcquiredBy", "Priority", "ExpiresAt"})

		if err == nil {
			// Lock exists — check expiry and priority
			var heldBy string
			var heldPriority float64
			var expiresAt time.Time
			if err := row.Columns(&heldBy, &heldPriority, &expiresAt); err != nil {
				return err
			}

			if now.Before(expiresAt) && heldBy != warehouseID {
				// Lock is valid and held by someone else
				if velocity > heldPriority {
					// We have higher priority — preempt
					log.Printf("[REPLENISHMENT_LOCK] Preempting lock %s: %s (%.1f) > %s (%.1f)",
						lockKey, warehouseID, velocity, heldBy, heldPriority)
					result.Acquired = true
					if err := txn.BufferWrite([]*spanner.Mutation{
						spanner.InsertOrUpdate("ReplenishmentLocks",
							[]string{"LockKey", "AcquiredBy", "SupplierId", "Priority", "AcquiredAt", "ExpiresAt"},
							[]interface{}{lockKey, warehouseID, supplierID, velocity, spanner.CommitTimestamp, now.Add(lockTTL)},
						),
					}); err != nil {
						return err
					}
					return emitReplenishmentLockEvent(ctx, txn, lockKey, warehouseID, supplierID, velocity, "PREEMPTED", kafka.EventReplenishmentLockAcquired, now)
				}
				// Lower or equal priority — lock denied
				result.Acquired = false
				result.HeldBy = heldBy
				result.HeldPriority = heldPriority
				return nil
			}
			// Lock expired or we already hold it — (re)acquire
		}

		// No valid lock exists — acquire
		result.Acquired = true
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("ReplenishmentLocks",
				[]string{"LockKey", "AcquiredBy", "SupplierId", "Priority", "AcquiredAt", "ExpiresAt"},
				[]interface{}{lockKey, warehouseID, supplierID, velocity, spanner.CommitTimestamp, now.Add(lockTTL)},
			),
		}); err != nil {
			return err
		}
		return emitReplenishmentLockEvent(ctx, txn, lockKey, warehouseID, supplierID, velocity, "ACQUIRED", kafka.EventReplenishmentLockAcquired, now)
	})
	if err != nil {
		return nil, fmt.Errorf("lock transaction failed: %w", err)
	}

	return &result, nil
}

// ReleaseLock releases a replenishment lock held by the specified warehouse.
func (s *ReplenishmentLockService) ReleaseLock(ctx context.Context, supplierID, warehouseID, skuID, factoryID string) error {
	lockKey := fmt.Sprintf("SKU:%s:FACTORY:%s", skuID, factoryID)

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "ReplenishmentLocks", spanner.Key{lockKey}, []string{"AcquiredBy"})
		if err != nil {
			return nil // Lock doesn't exist — nothing to release
		}
		var heldBy string
		if err := row.Columns(&heldBy); err != nil {
			return err
		}
		if heldBy != warehouseID {
			return nil // Not our lock
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("ReplenishmentLocks", spanner.Key{lockKey}),
		}); err != nil {
			return err
		}
		return emitReplenishmentLockEvent(ctx, txn, lockKey, warehouseID, supplierID, 0, "RELEASED", kafka.EventReplenishmentLockReleased, time.Now().UTC())
	})
	if err != nil {
		return fmt.Errorf("release lock failed: %w", err)
	}

	return nil
}

func emitReplenishmentLockEvent(ctx context.Context, txn *spanner.ReadWriteTransaction, lockKey, warehouseID, supplierID string, priority float64, action, eventType string, timestamp time.Time) error {
	evt := kafka.ReplenishmentLockEvent{
		LockKey:     lockKey,
		WarehouseId: warehouseID,
		SupplierId:  supplierID,
		Priority:    priority,
		Action:      action,
		Timestamp:   timestamp,
	}
	return outbox.EmitJSON(txn, "ReplenishmentLock", lockKey, eventType, kafka.TopicMain, evt, telemetry.TraceIDFromContext(ctx))
}

// CleanExpiredLocks removes all expired locks. Called periodically by the SLA monitor cron.
func (s *ReplenishmentLockService) CleanExpiredLocks(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	stmt := spanner.Statement{
		SQL:    `SELECT LockKey FROM ReplenishmentLocks WHERE ExpiresAt < @now`,
		Params: map[string]interface{}{"now": now},
	}

	var keys []string
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var key string
		if err := row.Columns(&key); err != nil {
			continue
		}
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return 0, nil
	}

	var mutations []*spanner.Mutation
	for _, k := range keys {
		mutations = append(mutations, spanner.Delete("ReplenishmentLocks", spanner.Key{k}))
	}

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return 0, err
	}

	return len(keys), nil
}

// calculateSalesVelocity computes the 30-day sales velocity (total units sold)
// for a SKU at a specific warehouse. Higher velocity = higher replenishment priority.
func (s *ReplenishmentLockService) calculateSalesVelocity(ctx context.Context, supplierID, warehouseID, skuID string) (float64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -30)

	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(oli.Quantity), 0) AS total_qty
		      FROM OrderLineItems oli
		      JOIN Orders o ON oli.OrderId = o.OrderId
		      WHERE o.SupplierId = @supplierID
		        AND o.WarehouseId = @warehouseID
		        AND oli.SkuId = @skuID
		        AND o.State = 'COMPLETED'
		        AND o.CompletedAt >= @cutoff`,
		Params: map[string]interface{}{
			"supplierID":  supplierID,
			"warehouseID": warehouseID,
			"skuID":       skuID,
			"cutoff":      cutoff,
		},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, err
	}

	var totalQty int64
	if err := row.Columns(&totalQty); err != nil {
		return 0, err
	}

	return float64(totalQty), nil
}
