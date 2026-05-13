package order

import "strings"

const (
	SettlementTargetGlobalSupplier = "GLOBAL_SUPPLIER"
	SettlementTargetLocalWarehouse = "LOCAL_WAREHOUSE"
	SettlementTargetMixedWarehouse = "MIXED_WAREHOUSE"
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
