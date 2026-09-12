package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

// inventoryAdapter bridges inventory.Service → supplier.InventoryServicer.
type inventoryAdapter struct {
	svc *inventory.Service
}

func (a *inventoryAdapter) ListBySupplier(ctx context.Context, supplierID string) ([]supplier.InventoryLevelView, error) {
	levels, err := a.svc.ListBySupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	views := make([]supplier.InventoryLevelView, len(levels))
	for i, l := range levels {
		views[i] = supplier.InventoryLevelView{
			InventoryID:      l.InventoryID,
			ProductID:        l.ProductID,
			WarehouseID:      l.WarehouseID,
			SupplierID:       l.SupplierID,
			QuantityOnHand:   l.QuantityOnHand,
			QuantityReserved: l.QuantityReserved,
			ReorderThreshold: l.ReorderThreshold,
			Version:          l.Version,
		}
	}
	return views, nil
}

func (a *inventoryAdapter) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	return a.svc.AdjustStock(ctx, inventoryID, delta, expectedVersion)
}

func (a *inventoryAdapter) FindByWarehouseProduct(ctx context.Context, warehouseID, productID string) (string, bool, error) {
	level, err := a.svc.FindByWarehouseProduct(ctx, warehouseID, productID)
	if err != nil {
		return "", false, err
	}
	if level == nil {
		return "", false, nil
	}
	return level.InventoryID, true, nil
}

func (a *inventoryAdapter) UpsertLevel(ctx context.Context, level supplier.InventoryLevelUpsert) error {
	return a.svc.Upsert(ctx, inventory.Level{
		InventoryID:      level.InventoryID,
		ProductID:        level.ProductID,
		WarehouseID:      level.WarehouseID,
		SupplierID:       level.SupplierID,
		QuantityOnHand:   level.QuantityOnHand,
		QuantityReserved: level.QuantityReserved,
		ReorderThreshold: level.ReorderThreshold,
		Version:          level.Version,
	})
}

// notificationReaderAdapter bridges notifications.Service to the
// retailer.NotificationReader and driver.DriverNotificationReader interfaces.
type notificationReaderAdapter struct {
	svc *notifications.Service
}

func (a *notificationReaderAdapter) ListForRecipient(ctx context.Context, recipientID string, limit, offset int) ([]any, error) {
	if a == nil || a.svc == nil {
		return []any{}, nil
	}
	notifs, err := a.svc.ListForRecipient(ctx, recipientID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(notifs))
	for i, n := range notifs {
		out[i] = n
	}
	return out, nil
}

func (a *notificationReaderAdapter) MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.MarkRead(ctx, recipientID, notificationIDs)
}

func (a *notificationReaderAdapter) MarkAllRead(ctx context.Context, recipientID string) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.MarkAllRead(ctx, recipientID)
}

func (a *notificationReaderAdapter) UnreadCount(ctx context.Context, recipientID string) (int64, error) {
	if a == nil || a.svc == nil {
		return 0, nil
	}
	return a.svc.UnreadCount(ctx, recipientID)
}

// CreateNotification implements retailer.NotificationWriter for variance alerts.
func (a *notificationReaderAdapter) CreateNotification(ctx context.Context, recipientID, recipientRole, eventType, title, body, deepLink string) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.CreateNotification(ctx, recipientID, recipientRole, eventType, title, body, deepLink)
}

type runtimeSeedRepository struct {
	client *spanner.Client
}

func (r *runtimeSeedRepository) UpsertSupplier(ctx context.Context, s seed.Supplier) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("runtime seed repository: nil client")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt, err := existingSeedSupplierCreatedAt(ctx, txn, s.SupplierID)
		if err != nil {
			return err
		}
		mutation := spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":   s.SupplierID,
			"Name":         s.Name,
			"CountryCode":  s.CountryCode,
			"Currency":     s.Currency,
			"IsConfigured": false,
			"CreatedAt":    createdAt,
			"UpdatedAt":    spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		return fmt.Errorf("upsert seed supplier %s: %w", s.SupplierID, err)
	}
	return nil
}

func existingSeedSupplierCreatedAt(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string) (time.Time, error) {
	row, err := txn.ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"CreatedAt"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return time.Now().UTC(), nil
		}
		return time.Time{}, fmt.Errorf("read seed supplier %s: %w", supplierID, err)
	}
	var createdAt time.Time
	if err := row.Columns(&createdAt); err != nil {
		return time.Time{}, fmt.Errorf("decode seed supplier %s created_at: %w", supplierID, err)
	}
	return createdAt, nil
}
