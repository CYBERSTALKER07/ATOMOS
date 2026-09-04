package factory

import (
	"math"
	"strings"
)

type breachedSKU struct {
	SupplierID  string
	WarehouseID string
	ProductID   string
	CurrentQty  int64
	SafetyLevel int64
	Deficit     int64
	UnitVU      float64
}

type plannedTransfer struct {
	SupplierID  string
	WarehouseID string
	FactoryID   string
	Source      string
	State       string
	TotalVU     float64
	ProductIDs  []string
}

type factoryPicker func(supplierID, warehouseID, productID, mode string) (string, error)

type lockFn func(supplierID, warehouseID, productID, factoryID string) bool

func sourceToTransferSource(runSource string) string {
	// Pull-matrix engine always writes SYSTEM_THRESHOLD (cron or POST).
	// Run source (CRON/MANUAL) is audit-only on PullMatrixRuns, not transfer Source.
	_ = runSource
	return TransferSourceThreshold
}

// PlanPullTransfers groups safety-breached SKUs by SelectOptimalFactory.
// MANUAL_ONLY yields no transfers. lockAcquire false skips that SKU.
func PlanPullTransfers(mode, runSource string, breached []breachedSKU, pick factoryPicker, acquire lockFn) []plannedTransfer {
	mode = normalizeNetworkMode(mode)
	if mode == NetworkModeManualOnly {
		return nil
	}
	if acquire == nil {
		acquire = func(_, _, _, _ string) bool { return true }
	}
	type key struct{ WarehouseID, FactoryID string }
	grouped := map[key]*plannedTransfer{}
	order := make([]key, 0)
	for _, b := range breached {
		if b.Deficit <= 0 || strings.TrimSpace(b.WarehouseID) == "" {
			continue
		}
		factoryID, err := pick(b.SupplierID, b.WarehouseID, b.ProductID, mode)
		if err != nil || strings.TrimSpace(factoryID) == "" {
			continue
		}
		if !acquire(b.SupplierID, b.WarehouseID, b.ProductID, factoryID) {
			continue
		}
		k := key{WarehouseID: b.WarehouseID, FactoryID: factoryID}
		row, ok := grouped[k]
		if !ok {
			row = &plannedTransfer{
				SupplierID:  b.SupplierID,
				WarehouseID: b.WarehouseID,
				FactoryID:   factoryID,
				Source:      sourceToTransferSource(runSource),
				State:       TransferStateCreated,
			}
			grouped[k] = row
			order = append(order, k)
		}
		vu := b.UnitVU
		if vu <= 0 {
			vu = 1
		}
		row.TotalVU += float64(b.Deficit) * vu
		row.ProductIDs = append(row.ProductIDs, b.ProductID)
	}
	out := make([]plannedTransfer, 0, len(order))
	for _, k := range order {
		row := grouped[k]
		if row.TotalVU <= 0 {
			row.TotalVU = 1
		}
		out = append(out, *row)
	}
	return out
}

func safetyDeficit(current, safety int64) int64 {
	if safety <= 0 {
		return 0
	}
	d := safety - current
	if d <= 0 {
		return 0
	}
	return d
}

func ceilVU(v float64) float64 {
	if v <= 0 {
		return 1
	}
	return math.Ceil(v)
}
