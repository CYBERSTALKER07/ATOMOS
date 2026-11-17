package whatif_test

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
)

func TestEvaluatePipeline_KeepsNewestCostLayerPerSKU(t *testing.T) {
	out := whatif.EvaluatePipeline(whatif.SimulationDataset{
		Forecasts: []whatif.ForecastRecord{
			{SKU: "SKU-1", ProductID: "SKU-1", LocationID: "wh-1", IsWinner: true, P50Milli: 10_000}, // 10 units
		},
		Inventory: []whatif.InventoryRecord{
			{WarehouseID: "wh-1", ProductID: "SKU-1", QuantityOnHand: 0, QuantityReserved: 0, ReorderThreshold: 0},
		},
		Products: []whatif.ProductPriceRecord{
			{ProductID: "SKU-1", Name: "Tea", PriceMinor: 15_000, Currency: "UZS"},
		},
		// Ordered newest-first (ReceivedAt DESC): 8,000 must NOT be overwritten by older 5,000.
		CostLayers: []whatif.CostLayerRecord{
			{ProductID: "SKU-1", UnitCostMinor: 8_000},
			{ProductID: "SKU-1", UnitCostMinor: 5_000},
		},
	}, whatif.SimulateInput{HorizonDays: 30, Currency: "UZS"})

	if out.ReplenishmentCostMinor == 0 {
		t.Fatalf("expected non-zero ReplenishmentCostMinor, got %+v", out)
	}
	// 10 shortfall units * 8,000 newest layer = 80,000 (older layer would yield 50,000).
	if out.ReplenishmentCostMinor != 80_000 {
		t.Fatalf("ReplenishmentCostMinor=%d, want 80000 (newest cost layer)", out.ReplenishmentCostMinor)
	}
}
