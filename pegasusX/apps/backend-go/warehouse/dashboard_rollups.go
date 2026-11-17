package warehouse

import (
	"sort"
	"strings"
)

var warehouseOrderStatusFunnel = []string{
	"PENDING", "SCHEDULED", "AUTO_ACCEPTED", "BACKORDERED",
	"LOADED", "IN_TRANSIT", "DELAYED",
	"ARRIVED", "ARRIVED_SHOP_CLOSED",
	"AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", "DELIVERED_ON_CREDIT",
	"FISCALIZING", "FISCAL_FAILED", "RECONCILIATION_REQUIRED",
	"COMPLETED", "CANCELLED",
}

var warehouseTruckDutyStatuses = []string{
	"AVAILABLE",
	"IN_TRANSIT",
	"RETURNING_TO_WAREHOUSE",
	"OFF_SHIFT",
	"UNASSIGNED",
	"VEHICLE_INACTIVE",
	"UNAVAILABLE",
	"INACTIVE",
}

func emptyWarehouseOrderStatusCounts() map[string]int {
	out := make(map[string]int, len(warehouseOrderStatusFunnel))
	for _, key := range warehouseOrderStatusFunnel {
		out[key] = 0
	}
	return out
}

func emptyWarehouseTruckDutyCounts() map[string]int {
	out := make(map[string]int, len(warehouseTruckDutyStatuses))
	for _, key := range warehouseTruckDutyStatuses {
		out[key] = 0
	}
	return out
}

func canonicalizeWarehouseOrderStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "DISPATCHED":
		return "LOADED"
	case "EN_ROUTE":
		return "IN_TRANSIT"
	case "ARRIVING":
		return "ARRIVED"
	case "SHOP_CLOSED_PENDING":
		return "ARRIVED_SHOP_CLOSED"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func canonicalizeWarehouseTruckDuty(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "RETURNING", "RETURNING_TO_WH":
		return "RETURNING_TO_WAREHOUSE"
	case "IDLE":
		return "AVAILABLE"
	case "FULL", "NEEDS_RESCUE":
		return "UNAVAILABLE"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func classifyWarehouseTruckDuty(d PortalDriver) string {
	if !d.IsActive {
		return "INACTIVE"
	}
	if !d.OnShift {
		return driverOffShiftTruckStatus(d.UnavailableReason)
	}
	if strings.TrimSpace(d.VehicleID) == "" {
		return "UNASSIGNED"
	}
	if !d.VehicleIsActive {
		return "VEHICLE_INACTIVE"
	}
	key := canonicalizeWarehouseTruckDuty(d.TruckStatus)
	if _, ok := emptyWarehouseTruckDutyCounts()[key]; ok && key != "" {
		return key
	}
	if strings.TrimSpace(d.UnavailableReason) != "" {
		return "UNAVAILABLE"
	}
	return "AVAILABLE"
}

func countWarehouseOrdersByStatus(orders []OrderRow) (counts map[string]int, active, pending int64) {
	counts = emptyWarehouseOrderStatusCounts()
	for _, order := range orders {
		status := canonicalizeWarehouseOrderStatus(order.Status)
		if _, ok := counts[status]; ok {
			counts[status]++
		}
		switch status {
		case "PENDING", "LOADED", "AUTO_ACCEPTED":
			pending++
			active++
		case "IN_TRANSIT", "ARRIVED", "DELAYED", "ARRIVED_SHOP_CLOSED":
			active++
		}
	}
	return counts, active, pending
}

func countWarehouseTruckDuty(drivers []PortalDriver) map[string]int {
	counts := emptyWarehouseTruckDutyCounts()
	for _, driver := range drivers {
		key := classifyWarehouseTruckDuty(driver)
		if _, ok := counts[key]; ok {
			counts[key]++
		}
	}
	return counts
}

func fleetStatusFromDuty(duty map[string]int) []map[string]any {
	out := make([]map[string]any, 0, len(warehouseTruckDutyStatuses))
	for _, status := range warehouseTruckDutyStatuses {
		out = append(out, map[string]any{"status": status, "count": duty[status]})
	}
	return out
}

func collectHoldReasons(drivers []PortalDriver, vehicles []PortalVehicle) []map[string]any {
	counts := map[string]int{}
	add := func(code string) {
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" || code == "IN_TRANSIT" || code == "AVAILABLE" || code == "OFF_SHIFT" {
			return
		}
		counts[code]++
	}
	for _, driver := range drivers {
		add(driver.UnavailableReason)
		add(driver.VehicleUnavailableReason)
	}
	for _, vehicle := range vehicles {
		add(vehicle.UnavailableReason)
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"code": key, "count": counts[key]})
	}
	return out
}
