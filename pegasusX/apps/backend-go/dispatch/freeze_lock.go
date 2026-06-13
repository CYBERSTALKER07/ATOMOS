package dispatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

const manualDispatchLockType = "MANUAL_DISPATCH"

// ErrDispatchLocked is returned when a warehouse dispatch freeze lock is already active.
var ErrDispatchLocked = errors.New("warehouse dispatch locked")

// FreezeLock is one active WarehouseDispatchLocks row.
type FreezeLock struct {
	LockID     string
	EntityType string
	EntityID   string
	Reason     string
}

type freezeLockTxnBuffer struct {
	events []outbox.Event
}

func (b *freezeLockTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// LoadFreezeLocks returns active locks for a warehouse node.
func LoadFreezeLocks(ctx context.Context, client *spanner.Client, warehouseID string) (map[string]FreezeLock, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if client == nil || warehouseID == "" {
		return map[string]FreezeLock{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT LockId, EntityType, EntityId, Reason
		      FROM WarehouseDispatchLocks@{FORCE_INDEX=Idx_WarehouseDispatchLocks_ByWarehouse}
		      WHERE WarehouseId = @warehouseId`,
		Params: map[string]any{"warehouseId": warehouseID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	locks := make(map[string]FreezeLock)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return locks, nil
		}
		if err != nil {
			return nil, fmt.Errorf("load freeze locks: %w", err)
		}
		var lock FreezeLock
		if err := row.Columns(&lock.LockID, &lock.EntityType, &lock.EntityID, &lock.Reason); err != nil {
			return nil, fmt.Errorf("scan freeze lock: %w", err)
		}
		locks[lock.LockID] = lock
	}
}

// IsWarehouseDispatchFrozen reports whether manual dispatch is blocked for the warehouse.
func IsWarehouseDispatchFrozen(locks map[string]FreezeLock, warehouseID string) (bool, string) {
	warehouseID = strings.TrimSpace(warehouseID)
	for _, lock := range locks {
		entityType := strings.ToUpper(strings.TrimSpace(lock.EntityType))
		entityID := strings.TrimSpace(lock.EntityID)
		if entityType == "WAREHOUSE" && (entityID == warehouseID || entityID == "warehouse-scope") {
			return true, "warehouse_dispatch_locked"
		}
	}
	return false, ""
}

// FilterFreezeLockedOrders removes orders with an active ORDER-scoped freeze lock.
func FilterFreezeLockedOrders(locks map[string]FreezeLock, rows []DispatchableOrder) []DispatchableOrder {
	if len(locks) == 0 || len(rows) == 0 {
		return rows
	}
	blocked := make(map[string]struct{})
	for _, lock := range locks {
		if strings.EqualFold(lock.EntityType, "ORDER") {
			blocked[strings.TrimSpace(lock.EntityID)] = struct{}{}
		}
	}
	if len(blocked) == 0 {
		return rows
	}
	filtered := make([]DispatchableOrder, 0, len(rows))
	for _, row := range rows {
		if _, skip := blocked[row.OrderID]; skip {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

// FilterFreezeLockedDriverIDs removes driver IDs with an active DRIVER-scoped freeze lock.
func FilterFreezeLockedDriverIDs(locks map[string]FreezeLock, driverIDs []string) []string {
	if len(locks) == 0 || len(driverIDs) == 0 {
		return driverIDs
	}
	blocked := make(map[string]struct{})
	for _, lock := range locks {
		if strings.EqualFold(lock.EntityType, "DRIVER") {
			blocked[strings.TrimSpace(lock.EntityID)] = struct{}{}
		}
	}
	if len(blocked) == 0 {
		return driverIDs
	}
	filtered := make([]string, 0, len(driverIDs))
	for _, driverID := range driverIDs {
		if _, skip := blocked[strings.TrimSpace(driverID)]; skip {
			continue
		}
		filtered = append(filtered, driverID)
	}
	return filtered
}

// AcquireManualDispatchLock inserts a WAREHOUSE-scoped freeze lock for supplier dispatch execute.
func AcquireManualDispatchLock(ctx context.Context, client *spanner.Client, warehouseID, supplierID, lockedBy string, now time.Time) (string, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	supplierID = strings.TrimSpace(supplierID)
	if client == nil || warehouseID == "" || supplierID == "" {
		return "", fmt.Errorf("acquire dispatch lock: warehouse_id and supplier_id required")
	}
	lockID := uuid.NewString()
	timestamp := now.UTC().Format(time.RFC3339Nano)
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if frozen, err := warehouseFrozenInTxn(ctx, txn, warehouseID); err != nil {
			return err
		} else if frozen {
			return ErrDispatchLocked
		}
		buf := &freezeLockTxnBuffer{}
		if err := emitFreezeLockAcquire(ctx, buf, lockID, supplierID, warehouseID, lockedBy, timestamp); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("WarehouseDispatchLocks", map[string]any{
				"LockId":      lockID,
				"WarehouseId": warehouseID,
				"EntityType":  "WAREHOUSE",
				"EntityId":    warehouseID,
				"Reason":      encodeFreezeLockReason(manualDispatchLockType, "supplier_dispatch_execute"),
				"CreatedAt":   spanner.CommitTimestamp,
				"UpdatedAt":   spanner.CommitTimestamp,
			}),
		}
		mutations = append(mutations, freezeLockOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return "", err
	}
	return lockID, nil
}

// ReleaseManualDispatchLock deletes a supplier dispatch execute freeze lock.
func ReleaseManualDispatchLock(ctx context.Context, client *spanner.Client, warehouseID, lockID, supplierID, lockedBy string, now time.Time) error {
	warehouseID = strings.TrimSpace(warehouseID)
	lockID = strings.TrimSpace(lockID)
	supplierID = strings.TrimSpace(supplierID)
	if client == nil || warehouseID == "" || lockID == "" {
		return nil
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &freezeLockTxnBuffer{}
		if err := emitFreezeLockRelease(ctx, buf, lockID, supplierID, warehouseID, lockedBy, timestamp); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{
			spanner.Delete("WarehouseDispatchLocks", spanner.Key{lockID}),
		}
		mutations = append(mutations, freezeLockOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}

func warehouseFrozenInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT EntityType, EntityId
		      FROM WarehouseDispatchLocks@{FORCE_INDEX=Idx_WarehouseDispatchLocks_ByWarehouse}
		      WHERE WarehouseId = @warehouseId`,
		Params: map[string]any{"warehouseId": warehouseID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read dispatch locks in txn: %w", err)
		}
		var entityType, entityID string
		if err := row.Columns(&entityType, &entityID); err != nil {
			return false, fmt.Errorf("scan dispatch lock in txn: %w", err)
		}
		entityType = strings.ToUpper(strings.TrimSpace(entityType))
		entityID = strings.TrimSpace(entityID)
		if entityType == "WAREHOUSE" && (entityID == warehouseID || entityID == "warehouse-scope") {
			return true, nil
		}
	}
}

func encodeFreezeLockReason(lockType, reason string) string {
	lt := strings.TrimSpace(lockType)
	if lt == "" {
		lt = manualDispatchLockType
	}
	r := strings.TrimSpace(reason)
	if r == "" {
		return "lock_type:" + lt
	}
	return "lock_type:" + lt + "|" + r
}

func emitFreezeLockAcquire(ctx context.Context, buf outbox.TxnBuffer, lockID, supplierID, warehouseID, lockedBy, timestamp string) error {
	warehouseEvent := events.WarehouseEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventWarehouseDispatchLockChanged, Timestamp: timestamp},
		LockID:      lockID,
		WarehouseID: warehouseID,
		SupplierID:  supplierID,
		Status:      "ACTIVE",
		Action:      "ACQUIRED",
		RequestID:   warehouseID,
	}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, lockID, events.TopicMain, warehouseEvent); err != nil {
		return err
	}
	freeze := events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockAcquired, Timestamp: timestamp},
		LockID:      lockID,
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		LockType:    manualDispatchLockType,
		LockedBy:    lockedBy,
	}
	if err := outbox.EmitJSON(ctx, buf, "DispatchLock", lockID, events.TopicFreezeLocks, freeze); err != nil {
		return err
	}
	return outbox.EmitJSON(ctx, buf, "DispatchLock", lockID, events.TopicMain, freeze)
}

func emitFreezeLockRelease(ctx context.Context, buf outbox.TxnBuffer, lockID, supplierID, warehouseID, lockedBy, timestamp string) error {
	warehouseEvent := events.WarehouseEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventWarehouseDispatchLockChanged, Timestamp: timestamp},
		LockID:      lockID,
		WarehouseID: warehouseID,
		SupplierID:  supplierID,
		Status:      "RELEASED",
		Action:      "RELEASED",
		RequestID:   warehouseID,
	}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, lockID, events.TopicMain, warehouseEvent); err != nil {
		return err
	}
	freeze := events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockReleased, Timestamp: timestamp},
		LockID:      lockID,
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		LockType:    manualDispatchLockType,
		LockedBy:    lockedBy,
	}
	if err := outbox.EmitJSON(ctx, buf, "DispatchLock", lockID, events.TopicFreezeLocks, freeze); err != nil {
		return err
	}
	return outbox.EmitJSON(ctx, buf, "DispatchLock", lockID, events.TopicMain, freeze)
}

func freezeLockOutboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, event := range eventsList {
		row := map[string]any{
			"EventId":       event.EventID,
			"AggregateType": event.AggregateType,
			"AggregateId":   event.AggregateID,
			"TopicName":     event.TopicName,
			"Payload":       event.Payload,
			"CreatedAt":     event.CreatedAt.UTC(),
			"PublishedAt":   nil,
		}
		if event.PublishedAt != nil {
			row["PublishedAt"] = event.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return mutations
}
