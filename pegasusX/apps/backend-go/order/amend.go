package order

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// SupplierReturn is one quarantine row for rejected delivery quantity.
type SupplierReturn struct {
	ReturnID    string
	SKU         string
	RejectedQty int64
	Reason      string
	DriverNotes string
	ManifestID  string
	DriverID    string
	WarehouseID string
}

// DefaultPostDeliveryAmendWindow is the concealed-damage window after COMPLETED.
const DefaultPostDeliveryAmendWindow = 48 * time.Hour

var validAmendReasons = map[string]struct{}{
	"DAMAGED":     {},
	"MISSING":     {},
	"WRONG_ITEM":  {},
	"OTHER":       {},
}

// postDeliveryAmendReasons may be applied after COMPLETED within the claim window.
var postDeliveryAmendReasons = map[string]struct{}{
	"DAMAGED": {},
	"MISSING": {}, // concealed shortage discovered after seal open
}

func orderAmendable(status Status) bool {
	switch status {
	case StatusInTransit, StatusArrived, StatusArrivedShopClosed:
		return true
	default:
		return false
	}
}

// orderPostDeliveryAmendable allows COMPLETED orders within the time window for damage/shortage.
func orderPostDeliveryAmendable(status Status, completedAt, now time.Time, reasons []string) bool {
	if status != StatusCompleted {
		return false
	}
	if len(reasons) == 0 {
		return false
	}
	for _, r := range reasons {
		r = normalizeAmendReason(r)
		if _, ok := postDeliveryAmendReasons[r]; !ok {
			return false
		}
	}
	window := postDeliveryWindowFromEnv()
	if completedAt.IsZero() {
		return true
	}
	return !now.After(completedAt.UTC().Add(window))
}

func postDeliveryWindowFromEnv() time.Duration {
	h := strings.TrimSpace(os.Getenv("CLAIM_WINDOW_HOURS"))
	if h == "" {
		return DefaultPostDeliveryAmendWindow
	}
	n, err := strconv.Atoi(h)
	if err != nil || n <= 0 {
		return DefaultPostDeliveryAmendWindow
	}
	return time.Duration(n) * time.Hour
}

func normalizeAmendReason(reason string) string {
	return strings.ToUpper(strings.TrimSpace(reason))
}

func validateAmendReason(reason, customReason string) error {
	if _, ok := validAmendReasons[reason]; !ok {
		return fmt.Errorf("invalid reason %q", reason)
	}
	if reason == "OTHER" && strings.TrimSpace(customReason) == "" {
		return errors.New("custom_reason required when reason is OTHER")
	}
	return nil
}

func resolveAmendQuantities(origQty, acceptedQty, rejectedQty int64) (int64, int64, error) {
	if acceptedQty < 0 || rejectedQty < 0 {
		return 0, 0, errors.New("quantities cannot be negative")
	}
	if rejectedQty == 0 && acceptedQty <= origQty {
		rejectedQty = origQty - acceptedQty
	}
	if acceptedQty+rejectedQty != origQty {
		return 0, 0, fmt.Errorf("accepted(%d) + rejected(%d) != original(%d)", acceptedQty, rejectedQty, origQty)
	}
	return acceptedQty, rejectedQty, nil
}

func supplierReturnNotes(reason, customReason, driverNotes string) string {
	if reason == "OTHER" {
		return strings.TrimSpace(customReason)
	}
	return strings.TrimSpace(driverNotes)
}

func orderOriginalAmountMinor(orderRecord Order) int64 {
	if orderRecord.OriginalTotalMinor > 0 {
		return orderRecord.OriginalTotalMinor
	}
	return orderRecord.TotalMinor
}
