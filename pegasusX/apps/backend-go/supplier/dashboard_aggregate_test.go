package supplier

import (
	"testing"
	"time"
)

func TestAggregateOrderMetrics_AllFunnelKeysAndAliases(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	stamp := now.Format(time.RFC3339Nano)
	got := aggregateOrderMetrics([]SupplierOrder{
		{Status: "PENDING", UpdatedAt: stamp},
		{Status: "EN_ROUTE", UpdatedAt: stamp},
		{Status: "DISPATCHED", UpdatedAt: stamp},
		{Status: "SHOP_CLOSED_PENDING", UpdatedAt: stamp},
		{Status: "FISCAL_FAILED", UpdatedAt: stamp},
	}, now)
	for _, key := range orderStatusFunnel {
		if _, ok := got.ordersByStatus[key]; !ok {
			t.Fatalf("missing funnel key %s", key)
		}
	}
	if got.ordersByStatus["PENDING"] != 1 {
		t.Fatalf("PENDING=%d", got.ordersByStatus["PENDING"])
	}
	if got.ordersByStatus["IN_TRANSIT"] != 1 {
		t.Fatalf("EN_ROUTE should count as IN_TRANSIT, got %d", got.ordersByStatus["IN_TRANSIT"])
	}
	if got.ordersByStatus["LOADED"] != 1 {
		t.Fatalf("DISPATCHED should count as LOADED, got %d", got.ordersByStatus["LOADED"])
	}
	if got.ordersByStatus["ARRIVED_SHOP_CLOSED"] != 1 {
		t.Fatalf("SHOP_CLOSED_PENDING should count as ARRIVED_SHOP_CLOSED, got %d", got.ordersByStatus["ARRIVED_SHOP_CLOSED"])
	}
	if got.ordersByStatus["FISCAL_FAILED"] != 1 {
		t.Fatalf("FISCAL_FAILED=%d", got.ordersByStatus["FISCAL_FAILED"])
	}
	if got.ordersByStatus["COMPLETED"] != 0 {
		t.Fatalf("zero COMPLETED must stay visible, got %d", got.ordersByStatus["COMPLETED"])
	}
}

func TestFleetVuUsedFromManifests_OpenOnly(t *testing.T) {
	used := fleetVuUsedFromManifests([]SupplierManifestRow{
		{State: "SEALED", TotalVu: 12},
		{State: "COMPLETED", TotalVu: 99},
		{State: "DRAFT", TotalVolumeVU: 3},
	})
	if used != 15 {
		t.Fatalf("used=%d want 15 (open manifests only)", used)
	}
}

func TestFleetVuUsedFromManifests_EmptyIsZero(t *testing.T) {
	if used := fleetVuUsedFromManifests(nil); used != 0 {
		t.Fatalf("empty manifests must be 0, got %d", used)
	}
}
