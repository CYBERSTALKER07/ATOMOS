package order

import "strings"

const (
	SettlementTargetGlobalSupplier = "GLOBAL_SUPPLIER"
	SettlementTargetLocalWarehouse = "LOCAL_WAREHOUSE"
	SettlementTargetMixedWarehouse = "MIXED_WAREHOUSE"
	PayoutModeHQSupplier           = "HQ_SUPPLIER"
	PayoutModeWarehouseLocal       = "WAREHOUSE_LOCAL"
	PayoutOwnerTypeSupplier        = "SUPPLIER"
	PayoutOwnerTypeWarehouse       = "WAREHOUSE"
	FeePolicyVersionLegacyCheckout = "LEGACY_PLATFORM_FEE_BPS_V1"
	FeeTierLegacyFlat              = "LEGACY_FLAT_BPS"
	FeePolicyVersionMixed          = "MIXED"
)

func settlementTargetForWarehouseID(warehouseID string) string {
	if strings.TrimSpace(warehouseID) == "" {
		return SettlementTargetGlobalSupplier
	}
	return SettlementTargetLocalWarehouse
}

func settlementTargetForWarehouseIDs(warehouseIDs []string) string {
	if len(warehouseIDs) == 0 {
		return SettlementTargetGlobalSupplier
	}

	unique := make(map[string]struct{}, len(warehouseIDs))
	sawEmpty := false
	for _, warehouseID := range warehouseIDs {
		trimmed := strings.TrimSpace(warehouseID)
		if trimmed == "" {
			sawEmpty = true
			continue
		}
		unique[trimmed] = struct{}{}
	}

	if len(unique) == 0 {
		return SettlementTargetGlobalSupplier
	}
	if len(unique) == 1 && !sawEmpty {
		return SettlementTargetLocalWarehouse
	}
	return SettlementTargetMixedWarehouse
}

func normalizePayoutMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case PayoutModeWarehouseLocal:
		return PayoutModeWarehouseLocal
	default:
		return PayoutModeHQSupplier
	}
}

func normalizeFeePolicyVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return FeePolicyVersionLegacyCheckout
	}
	return trimmed
}
