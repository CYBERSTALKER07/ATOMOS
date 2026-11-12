package warehouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandleOpsAnalytics_AvgOrderValueIsInt64MinorUnits(t *testing.T) {
	svc := &Service{
		analyticsQuery: func(_ context.Context, _ string) (WarehouseAnalyticsCounts, error) {
			return WarehouseAnalyticsCounts{
				TotalOrders:     4,
				CompletedOrders: 3,
				CancelledOrders: 1,
				TotalRevenue:    1000,
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/analytics?warehouse_id=wh-1&period=7d", nil)
	rec := httptest.NewRecorder()
	svc.HandleOpsAnalytics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `"avg_order_value":333.`) {
		t.Fatalf("avg_order_value must be serialized as integer minor units, got %s", body)
	}

	var decoded struct {
		TotalRevenue    int64 `json:"total_revenue"`
		CompletedOrders int64 `json:"completed_orders"`
		AvgOrderValue   int64 `json:"avg_order_value"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode strict int64 response failed: %v (body=%s)", err, body)
	}
	if decoded.AvgOrderValue != 333 {
		t.Fatalf("avg_order_value=%d want 333", decoded.AvgOrderValue)
	}
}
