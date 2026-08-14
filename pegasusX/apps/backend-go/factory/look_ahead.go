package factory

import (
	"math"
	"strings"
)

const (
	LookAheadWindowDays    = 7
	SafetyStockBufferPct   = 0.15
	FactoryClassCCapacityVU = 400.0
)

type shadowDemandEntry struct {
	WarehouseID   string
	SupplierID    string
	ProductID     string
	FutureDemand  int64
	CurrentStock  int64
	ShadowDeficit int64
	UnitVU        float64
	InsightID     string
}

// ShadowDeficit is ceil(futureDemand * 1.15) - stock. Positive even when stock is "safe".
func ShadowDeficit(futureDemand, currentStock int64) int64 {
	if futureDemand <= 0 {
		return 0
	}
	buffered := int64(math.Ceil(float64(futureDemand) * (1.0 + SafetyStockBufferPct)))
	d := buffered - currentStock
	if d < 0 {
		return 0
	}
	return d
}

func lookAheadConfirmed(confirmation string) bool {
	switch strings.ToUpper(strings.TrimSpace(confirmation)) {
	case "CONFIRMED", "AUTO_CONFIRMED":
		return true
	default:
		return false
	}
}

func lookAheadOpenStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PENDING", "SCHEDULED", "AUTO_ACCEPTED", "BACKORDERED":
		return true
	default:
		return false
	}
}

// SplitClassCVolumes splits total VU into ≤400 VU chunks (Class-C).
func SplitClassCVolumes(totalVU float64) []float64 {
	if totalVU <= 0 {
		return nil
	}
	if totalVU <= FactoryClassCCapacityVU {
		return []float64{totalVU}
	}
	n := int(math.Ceil(totalVU / FactoryClassCCapacityVU))
	out := make([]float64, 0, n)
	remain := totalVU
	for remain > 0 {
		chunk := remain
		if chunk > FactoryClassCCapacityVU {
			chunk = FactoryClassCCapacityVU
		}
		out = append(out, chunk)
		remain -= chunk
	}
	return out
}

func PlanLookAheadTransfers(mode string, entries []shadowDemandEntry, pick factoryPicker, acquire lockFn) []plannedTransfer {
	mode = normalizeNetworkMode(mode)
	if mode == NetworkModeManualOnly {
		return nil
	}
	if acquire == nil {
		acquire = func(_, _, _, _ string) bool { return true }
	}
	var out []plannedTransfer
	for _, e := range entries {
		if e.ShadowDeficit <= 0 || strings.TrimSpace(e.WarehouseID) == "" {
			continue
		}
		factoryID, err := pick(e.SupplierID, e.WarehouseID, e.ProductID, mode)
		if err != nil || strings.TrimSpace(factoryID) == "" {
			continue
		}
		if !acquire(e.SupplierID, e.WarehouseID, e.ProductID, factoryID) {
			continue
		}
		vu := e.UnitVU
		if vu <= 0 {
			vu = 1
		}
		total := float64(e.ShadowDeficit) * vu
		for _, chunk := range SplitClassCVolumes(total) {
			out = append(out, plannedTransfer{
				SupplierID:  e.SupplierID,
				WarehouseID: e.WarehouseID,
				FactoryID:   factoryID,
				Source:      TransferSourceThreshold,
				State:       TransferStateCreated,
				TotalVU:     chunk,
				ProductIDs:  []string{e.ProductID},
			})
		}
	}
	return out
}
