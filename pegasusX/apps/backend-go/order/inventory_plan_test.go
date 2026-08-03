package order

import (
	"errors"
	"testing"
)

func TestResolveOutOfStockPolicy(t *testing.T) {
	if got := resolveOutOfStockPolicy("REJECT", "ACCEPT_BACKORDER"); got != outOfStockPolicyAcceptBackorder {
		t.Fatalf("product override expected ACCEPT_BACKORDER, got %s", got)
	}
	if got := resolveOutOfStockPolicy("ACCEPT_BACKORDER", ""); got != outOfStockPolicyAcceptBackorder {
		t.Fatalf("warehouse default expected ACCEPT_BACKORDER, got %s", got)
	}
	if got := resolveOutOfStockPolicy("ACCEPT_BACKORDER", "REJECT"); got != outOfStockPolicyReject {
		t.Fatalf("product reject override expected REJECT, got %s", got)
	}
}

func TestBuildInventoryPlanRejectPartialShort(t *testing.T) {
	states := map[string]skuPlanState{
		"sku-a": {available: 5, policy: outOfStockPolicyReject},
	}
	items := []LineItem{{SKU: "sku-a", Quantity: 10, UnitPrice: 100}}
	_, err := buildInventoryPlan(outOfStockPolicyReject, items, states)
	if err == nil {
		t.Fatal("expected error for partial short under REJECT")
	}
	ice, ok := err.(*InventoryCheckoutError)
	if !ok {
		t.Fatalf("expected InventoryCheckoutError, got %T", err)
	}
	if ice.Code != "PARTIAL_OUT_OF_STOCK_REJECTED" {
		t.Fatalf("code=%s want PARTIAL_OUT_OF_STOCK_REJECTED", ice.Code)
	}
}

func TestBuildInventoryPlanAcceptBackorderSplit(t *testing.T) {
	states := map[string]skuPlanState{
		"sku-a": {available: 3, policy: outOfStockPolicyAcceptBackorder},
	}
	items := []LineItem{{SKU: "sku-a", Quantity: 10, UnitPrice: 100}}
	plan, err := buildInventoryPlan(outOfStockPolicyAcceptBackorder, items, states)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Fulfillable) != 1 || plan.Fulfillable[0].Quantity != 3 {
		t.Fatalf("fulfillable=%v want qty 3", plan.Fulfillable)
	}
	if len(plan.Backorder) != 1 || plan.Backorder[0].Quantity != 7 {
		t.Fatalf("backorder=%v want qty 7", plan.Backorder)
	}
	if plan.BackorderCount != 1 {
		t.Fatalf("backorder_count=%d want 1", plan.BackorderCount)
	}
}

func TestBuildInventoryPlanProductRejectOverridesWarehouseAccept(t *testing.T) {
	states := map[string]skuPlanState{
		"sku-a": {available: 0, policy: outOfStockPolicyReject},
	}
	items := []LineItem{{SKU: "sku-a", Quantity: 2, UnitPrice: 50}}
	_, err := buildInventoryPlan(outOfStockPolicyAcceptBackorder, items, states)
	if err == nil {
		t.Fatal("expected reject when SKU policy is REJECT")
	}
	var ice *InventoryCheckoutError
	if !errors.As(err, &ice) {
		t.Fatalf("expected InventoryCheckoutError, got %T", err)
	}
	if ice.Code != "ALL_ITEMS_OUT_OF_STOCK" {
		t.Fatalf("code=%s want ALL_ITEMS_OUT_OF_STOCK", ice.Code)
	}
}

func TestBuildInventoryPlanInheritWarehouseDefault(t *testing.T) {
	states := map[string]skuPlanState{
		"sku-a": {available: 0, policy: resolveOutOfStockPolicy(outOfStockPolicyAcceptBackorder, "")},
	}
	items := []LineItem{{SKU: "sku-a", Quantity: 4, UnitPrice: 25}}
	plan, err := buildInventoryPlan(outOfStockPolicyAcceptBackorder, items, states)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Backorder) != 1 || plan.Backorder[0].Quantity != 4 {
		t.Fatalf("backorder=%v want full qty 4", plan.Backorder)
	}
}
