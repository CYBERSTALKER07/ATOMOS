package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

// ReceivingWindowReader loads retailer receiving windows for the in-memory
// order repository dev path. Production Spanner reads windows inside the
// CreateOrder RW transaction.
type ReceivingWindowReader interface {
	GetReceivingWindows(ctx context.Context, retailerID string) (open, close string, err error)
}

// SnapshotReceivingWindowsOnOrder canonicalizes retailer windows and writes them
// onto the order aggregate. Empty retailer windows yield empty order fields.
func SnapshotReceivingWindowsOnOrder(o *Order, retailerOpen, retailerClose string) error {
	if o == nil {
		return fmt.Errorf("snapshot receiving windows: nil order")
	}
	open, err := proximity.ValidateReceivingWindow(retailerOpen)
	if err != nil {
		return fmt.Errorf("snapshot receiving_window_open: %w", err)
	}
	closeWindow, err := proximity.ValidateReceivingWindow(retailerClose)
	if err != nil {
		return fmt.Errorf("snapshot receiving_window_close: %w", err)
	}
	o.ReceivingWindowOpen = open
	o.ReceivingWindowClose = closeWindow
	return nil
}

func snapshotReceivingWindowsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error {
	if o == nil {
		return fmt.Errorf("snapshot receiving windows: nil order")
	}
	retailerID := strings.TrimSpace(o.RetailerID)
	if retailerID == "" {
		return fmt.Errorf("snapshot receiving windows: retailer_id required")
	}
	row, err := txn.ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{
		"ReceivingWindowOpen",
		"ReceivingWindowClose",
	})
	if err != nil {
		return fmt.Errorf("read retailer %s receiving windows: %w", retailerID, err)
	}
	var open, closeWindow spanner.NullString
	if err := row.Columns(&open, &closeWindow); err != nil {
		return fmt.Errorf("scan retailer %s receiving windows: %w", retailerID, err)
	}
	return SnapshotReceivingWindowsOnOrder(o, open.StringVal, closeWindow.StringVal)
}

func snapshotWarehousePolicyInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error {
	if o == nil {
		return fmt.Errorf("snapshot warehouse policy: nil order")
	}
	warehouseID := strings.TrimSpace(o.WarehouseID)
	if warehouseID == "" {
		return fmt.Errorf("snapshot warehouse policy: warehouse_id required")
	}
	row, err := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"OperatingSchedule"})
	if err != nil {
		return fmt.Errorf("read warehouse %s operating schedule: %w", warehouseID, err)
	}
	var scheduleRaw spanner.NullJSON
	if err := row.Columns(&scheduleRaw); err != nil {
		return fmt.Errorf("scan warehouse %s operating schedule: %w", warehouseID, err)
	}
	if scheduleRaw.Valid && scheduleRaw.Value != nil {
		str, ok := scheduleRaw.Value.(string)
		if ok && str != "" {
			sched := ParseOperatingSchedule([]byte(str))
			o.Timezone = sched.Timezone
		}
	}
	return nil
}
