package order

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func snapshotServicePromiseInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) (*spanner.Mutation, error) {
	if o == nil || o.OrderID == "" || o.SupplierID == "" {
		return nil, nil
	}
	row, err := txn.ReadRow(ctx, "SupplierServicePolicies", spanner.Key{o.SupplierID}, []string{
		"LeadTimeDays", "SameDayCutoffTime", "NextDayCutoffTime", "MinOrderMinor", "Currency", "FillRateGuaranteeBps",
	})
	leadTimeDays := int64(1)
	fillRateBps := int64(9500)
	minOrderMinor := int64(0)
	currency := o.Currency
	if currency == "" {
		currency = "UZS"
	}
	if err == nil {
		var leadTime, fillRate, minOrder spanner.NullInt64
		var sameDay, nextDay, curr spanner.NullString
		_ = row.Columns(&leadTime, &sameDay, &nextDay, &minOrder, &curr, &fillRate)
		if leadTime.Valid && leadTime.Int64 > 0 {
			leadTimeDays = leadTime.Int64
		}
		if fillRate.Valid && fillRate.Int64 > 0 {
			fillRateBps = fillRate.Int64
		}
		if minOrder.Valid {
			minOrderMinor = minOrder.Int64
		}
		if curr.Valid && curr.StringVal != "" {
			currency = curr.StringVal
		}
	}

	promiseType := "NEXT_DAY"
	guaranteedDelivery := o.CreatedAt.Add(time.Duration(leadTimeDays*24) * time.Hour)
	if o.RequestedDeliveryDate != nil && !o.RequestedDeliveryDate.IsZero() {
		promiseType = "SCHEDULED"
		guaranteedDelivery = *o.RequestedDeliveryDate
	}
	slaHours := leadTimeDays * 24
	if promiseType == "SCHEDULED" {
		slaHours = int64(guaranteedDelivery.Sub(o.CreatedAt).Hours())
		if slaHours < 0 {
			slaHours = 0
		}
	}

	mut := spanner.InsertMap("OrderServicePromiseSnapshots", map[string]any{
		"OrderId":                o.OrderID,
		"SupplierId":             o.SupplierID,
		"RetailerId":             o.RetailerID,
		"WarehouseId":            o.WarehouseID,
		"PromiseType":            promiseType,
		"GuaranteedDeliveryDate": guaranteedDelivery,
		"FillRateTargetBps":      fillRateBps,
		"MinOrderMinor":          minOrderMinor,
		"Currency":               currency,
		"SLAHours":               slaHours,
		"Status":                 "PENDING",
		"CreatedAt":              spanner.CommitTimestamp,
		"UpdatedAt":              spanner.CommitTimestamp,
	})
	return mut, nil
}
