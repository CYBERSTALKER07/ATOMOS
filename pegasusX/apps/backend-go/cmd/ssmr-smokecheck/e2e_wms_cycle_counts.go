package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runWMSCycleCountsE2E exercises cycle-count create, submit variance, and adjust-approve.
// Only skips when WMS_CYCLE_COUNTS_ENABLED is explicitly disabled (409 disabled); fails on errors.
func runWMSCycleCountsE2E(ctx context.Context, client *http.Client, base, cookie string, _ *bootstrap.Config) error {
	whID := demoWarehouseID()

	status, listBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/cycle-counts?warehouse_id="+whID, nil, cookie, "")
	if err != nil {
		return fmt.Errorf("cycle-counts list: %w", err)
	}
	if status == http.StatusConflict {
		var errBody struct {
			Error string `json:"error"`
		}
		if decErr := json.Unmarshal(listBody, &errBody); decErr != nil {
			return fmt.Errorf("cycle-counts list 409 decode: %w", decErr)
		}
		if strings.Contains(errBody.Error, "disabled") {
			fmt.Println("PX_E2E_WMS_CYCLE_COUNT_SKIPPED")
			fmt.Println("PX_E2E_WMS_ADJUST_APPLY_SKIPPED")
			return nil
		}
		return fmt.Errorf("cycle-counts list conflict: %s", string(listBody))
	}
	if status >= 400 {
		return fmt.Errorf("cycle-counts list status=%d body=%s", status, string(listBody))
	}
	var listed struct {
		Counts             []any `json:"counts"`
		CycleCountsEnabled bool  `json:"cycle_counts_enabled"`
	}
	if decErr := json.Unmarshal(listBody, &listed); decErr != nil {
		return fmt.Errorf("cycle-counts list decode: %w", decErr)
	}

	productID := strings.TrimSpace(envOr("SSMR_SMOKE_PRODUCT_ID", "sku-cola-1.5l"))
	locationID := fmt.Sprintf("ssmr-cc-bin-%d", time.Now().Unix()%100000)
	lotID := ""
	exp := int64(3)
	counted := int64(1)

	// Inspect available lots so adjustment apply operates against a real lot with unreserved headroom.
	lotsStatus, lotsBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/lots?warehouse_id="+whID+"&status=AVAILABLE", nil, cookie, "")
	if err != nil {
		return fmt.Errorf("lots list for cycle count: %w", err)
	}
	if lotsStatus >= 400 {
		return fmt.Errorf("lots list status=%d body=%s", lotsStatus, string(lotsBody))
	}
	var lotsResp struct {
		LotsEnabled bool `json:"lots_enabled"`
		Lots        []struct {
			LotID            string `json:"lot_id"`
			ProductID        string `json:"product_id"`
			LocationID       string `json:"location_id"`
			QuantityOnHand   int64  `json:"quantity_on_hand"`
			QuantityReserved int64  `json:"quantity_reserved"`
			Status           string `json:"status"`
		} `json:"lots"`
	}
	if decErr := json.Unmarshal(lotsBody, &lotsResp); decErr != nil {
		return fmt.Errorf("lots list decode: %w", decErr)
	}
	for _, l := range lotsResp.Lots {
		if strings.EqualFold(strings.TrimSpace(l.Status), "AVAILABLE") && l.QuantityOnHand-l.QuantityReserved >= 2 {
			lotID = l.LotID
			productID = l.ProductID
			if strings.TrimSpace(l.LocationID) != "" {
				locationID = l.LocationID
			}
			exp = l.QuantityOnHand
			counted = l.QuantityOnHand - 1
			break
		}
	}

	createPayload := map[string]any{
		"location_id":  locationID,
		"product_id":   productID,
		"expected_qty": exp,
	}
	if lotID != "" {
		createPayload["lot_id"] = lotID
	}
	createBody, err := json.Marshal(createPayload)
	if err != nil {
		return fmt.Errorf("marshal cycle-count create: %w", err)
	}
	status, createResp, _, err := clientPost(ctx, client,
		base+"/v1/warehouse/ops/cycle-counts?warehouse_id="+whID,
		createBody, cookie, "ssmr-cycle-count-"+fmt.Sprintf("%d", time.Now().Unix()))
	if err != nil {
		return fmt.Errorf("cycle-count create: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("cycle-count create status=%d body=%s", status, string(createResp))
	}
	var created struct {
		CountID string `json:"count_id"`
		Status  string `json:"status"`
	}
	if decErr := json.Unmarshal(createResp, &created); decErr != nil {
		return fmt.Errorf("cycle-count create decode: %w", decErr)
	}
	if strings.TrimSpace(created.CountID) == "" {
		return fmt.Errorf("cycle-count create missing count_id body=%s", string(createResp))
	}
	fmt.Println("PX_E2E_WMS_CYCLE_COUNT_OK")

	subBody, err := json.Marshal(map[string]any{
		"counted_qty": counted,
		"reason_code": "SSMR_SMOKE_VARIANCE",
	})
	if err != nil {
		return fmt.Errorf("marshal cycle-count submit: %w", err)
	}
	status, subResp, _, err := clientPost(ctx, client,
		base+"/v1/warehouse/ops/cycle-counts/"+created.CountID+"/submit?warehouse_id="+whID,
		subBody, cookie, "ssmr-cycle-submit-"+created.CountID)
	if err != nil {
		return fmt.Errorf("cycle-count submit: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("cycle-count submit status=%d body=%s", status, string(subResp))
	}

	status, adjBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/inventory-adjustments?warehouse_id="+whID+"&status=PENDING", nil, cookie, "")
	if err != nil {
		return fmt.Errorf("inventory-adjustments list: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("inventory-adjustments list status=%d body=%s", status, string(adjBody))
	}
	var adj struct {
		Adjustments []struct {
			AdjustmentID string `json:"adjustment_id"`
			CountID      string `json:"count_id"`
			Status       string `json:"status"`
		} `json:"adjustments"`
	}
	if decErr := json.Unmarshal(adjBody, &adj); decErr != nil {
		return fmt.Errorf("inventory-adjustments decode: %w", decErr)
	}
	aid := ""
	for _, a := range adj.Adjustments {
		if a.CountID == created.CountID {
			aid = a.AdjustmentID
			break
		}
	}
	if aid == "" {
		return fmt.Errorf("no PENDING adjustment created for count %s body=%s", created.CountID, string(adjBody))
	}

	status, appResp, _, err := clientPost(ctx, client,
		base+"/v1/warehouse/ops/inventory-adjustments/"+aid+"/approve?warehouse_id="+whID,
		[]byte("{}"), cookie, "ssmr-adj-approve-"+aid)
	if err != nil {
		return fmt.Errorf("inventory-adjustment approve: %w", err)
	}
	if status >= 400 {
		// When WMS_LOTS_ENABLED is off and no StockLots exist in sandbox, lot adjustment cannot apply.
		if lotID == "" && !lotsResp.LotsEnabled && status == http.StatusUnprocessableEntity && strings.Contains(string(appResp), "no_available_lot_for_adjustment") {
			fmt.Println("PX_E2E_WMS_ADJUST_APPLY_SKIPPED")
			return nil
		}
		return fmt.Errorf("inventory-adjustment approve status=%d body=%s", status, string(appResp))
	}
	var approved struct {
		AdjustmentID string `json:"adjustment_id"`
		Status       string `json:"status"`
	}
	if decErr := json.Unmarshal(appResp, &approved); decErr != nil {
		return fmt.Errorf("inventory-adjustment approve decode: %w", decErr)
	}
	if !strings.EqualFold(approved.Status, "APPROVED") {
		return fmt.Errorf("inventory-adjustment approve expected APPROVED got %q body=%s", approved.Status, string(appResp))
	}
	fmt.Println("PX_E2E_WMS_ADJUST_APPLY_OK")
	return nil
}
