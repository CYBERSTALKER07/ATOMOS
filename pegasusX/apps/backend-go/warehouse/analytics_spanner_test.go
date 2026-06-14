package warehouse

import (
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func TestParseAnalyticsPeriod(t *testing.T) {
	days, label := parseAnalyticsPeriod("30d")
	if days != 30 || label != "30d" {
		t.Fatalf("expected 30d, got %d %s", days, label)
	}
	days, label = parseAnalyticsPeriod("")
	if days != 7 || label != "7d" {
		t.Fatalf("expected default 7d, got %d %s", days, label)
	}
}

func TestMergeLineItemsIntoTopProducts(t *testing.T) {
	raw, err := json.Marshal([]order.LineItem{
		{SKU: "sku-1", Name: "Apples", Quantity: 2, UnitPrice: 1000},
		{SKU: "sku-1", Name: "Apples", Quantity: 3, UnitPrice: 1000},
		{Name: "Bananas", Quantity: 1, UnitPrice: 500},
	})
	if err != nil {
		t.Fatal(err)
	}
	agg := make(map[string]*analyticsTopProductAgg)
	mergeLineItemsIntoTopProducts(agg, raw)
	if len(agg) != 2 {
		t.Fatalf("expected 2 products, got %d", len(agg))
	}
	if agg["sku-1"].TotalQty != 5 || agg["sku-1"].Revenue != 5000 {
		t.Fatalf("unexpected apples aggregate: %+v", agg["sku-1"])
	}
}
