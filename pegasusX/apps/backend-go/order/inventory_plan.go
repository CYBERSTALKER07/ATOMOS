package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	outOfStockPolicyReject          = "REJECT"
	outOfStockPolicyAcceptBackorder = "ACCEPT_BACKORDER"
)

func resolveOutOfStockPolicy(warehouseDefault, productOverride string) string {
	override := strings.ToUpper(strings.TrimSpace(productOverride))
	switch override {
	case outOfStockPolicyReject, outOfStockPolicyAcceptBackorder:
		return override
	}
	def := strings.ToUpper(strings.TrimSpace(warehouseDefault))
	if def == outOfStockPolicyAcceptBackorder {
		return outOfStockPolicyAcceptBackorder
	}
	return outOfStockPolicyReject
}

// InventoryCheckoutError is returned when checkout cannot proceed under warehouse policy.
type InventoryCheckoutError struct {
	Code          string            `json:"code"`
	Message       string            `json:"message"`
	OOSItems      []string          `json:"oos_items,omitempty"`
	Shortfall     map[string]int64  `json:"shortfall,omitempty"`
	RejectedSKUs  []string          `json:"rejected_skus,omitempty"`
}

func (e *InventoryCheckoutError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// InventoryPlan splits requested lines into fulfillable vs backorder quantities.
type InventoryPlan struct {
	Fulfillable      []LineItem
	Backorder        []LineItem
	BackorderCount   int
	Warnings         []StockWarning
}

// StockWarning is surfaced to retailers when backorder policy allows checkout.
type StockWarning struct {
	SKU              string `json:"sku"`
	Requested        int64  `json:"requested"`
	Available        int64  `json:"available"`
	BackorderQty     int64  `json:"backorder_qty"`
	AcceptsBackorder bool   `json:"accepts_backorder"`
}

// PlanInventoryCheckout reads warehouse + SKU policy and available stock.
// reservationCredit adds qty already reserved by an order being edited (pre-order line changes).
func PlanInventoryCheckout(
	ctx context.Context,
	client *spanner.Client,
	supplierID, warehouseID string,
	items []LineItem,
	warehousePolicyOverride string,
) (InventoryPlan, error) {
	return PlanInventoryCheckoutWithCredit(ctx, client, supplierID, warehouseID, items, warehousePolicyOverride, nil)
}

func PlanInventoryCheckoutWithCredit(
	ctx context.Context,
	client *spanner.Client,
	supplierID, warehouseID string,
	items []LineItem,
	warehousePolicyOverride string,
	reservationCredit map[string]int64,
) (InventoryPlan, error) {
	plan := InventoryPlan{}
	if client == nil || warehouseID == "" || len(items) == 0 {
		plan.Fulfillable = append([]LineItem(nil), items...)
		return plan, nil
	}

	warehouseDefault, err := loadWarehouseDefaultPolicy(ctx, client, warehouseID)
	if err != nil {
		return plan, err
	}
	if override := strings.TrimSpace(warehousePolicyOverride); override != "" {
		warehouseDefault = resolveOutOfStockPolicy(override, "")
	}

	type skuState struct {
		available int64
		policy    string
		found     bool
	}
	states := make(map[string]*skuState, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		if _, ok := states[sku]; !ok {
			states[sku] = &skuState{policy: resolveOutOfStockPolicy(warehouseDefault, "")}
		}
	}

	keys := make([]spanner.KeySet, 0, len(states))
	for sku := range states {
		keys = append(keys, spanner.Key{supplierID, warehouseID, sku})
	}
	iter := client.Single().Read(ctx, "SupplierInventoryV2", spanner.KeySets(keys...),
		[]string{"ProductId", "QuantityOnHand", "QuantityReserved", "OutOfStockPolicy"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return plan, fmt.Errorf("read inventory: %w", err)
		}
		var productID string
		var qoh, qr int64
		var policy spanner.NullString
		if err := row.Columns(&productID, &qoh, &qr, &policy); err != nil {
			return plan, fmt.Errorf("decode inventory: %w", err)
		}
		avail := qoh - qr
		if credit := reservationCredit[productID]; credit > 0 {
			avail += credit
		}
		if avail < 0 {
			avail = 0
		}
		st := states[productID]
		if st == nil {
			st = &skuState{}
			states[productID] = st
		}
		st.available = avail
		st.found = true
		st.policy = resolveOutOfStockPolicy(warehouseDefault, policy.StringVal)
	}

	planStates := make(map[string]skuPlanState, len(states))
	for sku, st := range states {
		if st == nil {
			continue
		}
		avail := st.available
		if !st.found && reservationCredit != nil {
			if credit := reservationCredit[sku]; credit > 0 {
				avail = credit
			}
		}
		planStates[sku] = skuPlanState{available: avail, policy: st.policy}
	}
	return buildInventoryPlan(warehouseDefault, items, planStates)
}

// buildInventoryPlan splits lines using pre-resolved per-SKU availability and policy (unit-test seam).
func buildInventoryPlan(warehouseDefault string, items []LineItem, states map[string]skuPlanState) (InventoryPlan, error) {
	plan := InventoryPlan{}
	rejected := make([]string, 0)
	shortfall := make(map[string]int64)
	oosAll := make([]string, 0)

	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		st, ok := states[sku]
		if !ok {
			st = skuPlanState{policy: resolveOutOfStockPolicy(warehouseDefault, "")}
		}
		available := st.available
		requested := item.Quantity

		if available >= requested {
			plan.Fulfillable = append(plan.Fulfillable, item)
			st.available -= requested
			states[sku] = st
			continue
		}

		short := requested - available
		if available > 0 {
			fill := item
			fill.Quantity = available
			plan.Fulfillable = append(plan.Fulfillable, fill)
			st.available = 0
			states[sku] = st
		}

		if st.policy == outOfStockPolicyAcceptBackorder {
			bo := item
			bo.Quantity = short
			plan.Backorder = append(plan.Backorder, bo)
			plan.BackorderCount++
			plan.Warnings = append(plan.Warnings, StockWarning{
				SKU:              sku,
				Requested:        requested,
				Available:        available,
				BackorderQty:     short,
				AcceptsBackorder: true,
			})
			shortfall[sku] = short
			continue
		}

		rejected = append(rejected, sku)
		shortfall[sku] = short
		if available <= 0 {
			oosAll = append(oosAll, sku)
		}
	}

	if len(plan.Fulfillable) == 0 && len(rejected) > 0 {
		return plan, &InventoryCheckoutError{
			Code:         "ALL_ITEMS_OUT_OF_STOCK",
			Message:      "inventory_exhausted",
			OOSItems:     oosAll,
			Shortfall:    shortfall,
			RejectedSKUs: rejected,
		}
	}
	if len(rejected) > 0 {
		return plan, &InventoryCheckoutError{
			Code:         "PARTIAL_OUT_OF_STOCK_REJECTED",
			Message:      "inventory_exhausted",
			OOSItems:     rejected,
			Shortfall:    shortfall,
			RejectedSKUs: rejected,
		}
	}
	return plan, nil
}

type skuPlanState struct {
	available int64
	policy    string
}

func loadWarehouseDefaultPolicy(ctx context.Context, client *spanner.Client, warehouseID string) (string, error) {
	row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"DefaultOutOfStockPolicy"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return outOfStockPolicyReject, nil
		}
		return "", fmt.Errorf("load warehouse policy: %w", err)
	}
	var policy spanner.NullString
	if err := row.Columns(&policy); err != nil {
		return "", err
	}
	return resolveOutOfStockPolicy(policy.StringVal, ""), nil
}

// MarshalInventoryCheckoutError writes a JSON body for HTTP responses.
func MarshalInventoryCheckoutError(err error) ([]byte, bool) {
	var ice *InventoryCheckoutError
	if !errors.As(err, &ice) {
		var wrapped *InventoryCheckoutError
		if errors.As(err, &wrapped) {
			ice = wrapped
		} else {
			return nil, false
		}
	}
	payload := map[string]any{
		"error":   ice.Message,
		"code":    ice.Code,
		"message": ice.Message,
	}
	if len(ice.OOSItems) > 0 {
		payload["oos_items"] = ice.OOSItems
	}
	if len(ice.Shortfall) > 0 {
		payload["shortfall"] = ice.Shortfall
	}
	raw, _ := json.Marshal(payload)
	return raw, true
}
