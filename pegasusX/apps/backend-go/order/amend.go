package order

import (
	"errors"
	"fmt"
	"strings"
)

// SupplierReturn is one quarantine row for rejected delivery quantity.
type SupplierReturn struct {
	ReturnID    string
	SKU         string
	RejectedQty int64
	Reason      string
	DriverNotes string
}

var validAmendReasons = map[string]struct{}{
	"DAMAGED":     {},
	"MISSING":     {},
	"WRONG_ITEM":  {},
	"OTHER":       {},
}

func orderAmendable(status Status) bool {
	switch status {
	case StatusInTransit, StatusArrived, StatusArrivedShopClosed:
		return true
	default:
		return false
	}
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
