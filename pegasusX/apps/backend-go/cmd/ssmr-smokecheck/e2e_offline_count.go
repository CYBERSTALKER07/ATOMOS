package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runOfflineCountE2E exercises Wave C3.3 versioned count commit when OFFLINE_COUNT_ENABLED=true.
// When flag is off, prints PX_E2E_OFFLINE_COUNT_CONFLICT_SKIPPED.
func runOfflineCountE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, retailerToken string) error {
	if !envTruthy("OFFLINE_COUNT_ENABLED") {
		fmt.Println("PX_E2E_OFFLINE_COUNT_CONFLICT_SKIPPED")
		return nil
	}

	// Ensure stock module reachable
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/stock", nil, retailerToken, "")
	if err != nil {
		return fmt.Errorf("stock list: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("stock list status %d body %s", status, string(body))
	}

	var stockList struct {
		Items []struct {
			LocationID string `json:"location_id"`
			StockBin   string `json:"stock_bin"`
			Sku        string `json:"sku"`
		} `json:"items"`
	}
	_ = json.Unmarshal(body, &stockList)
	locID := ""
	sku := ""
	if len(stockList.Items) > 0 {
		locID = stockList.Items[0].LocationID
		sku = stockList.Items[0].Sku
	}
	if locID == "" {
		locID = envOr("SSMR_STOCK_LOCATION_ID", "")
	}
	if sku == "" {
		sku = envOr("SSMR_STOCK_SKU", "SKU-A")
	}
	if locID == "" {
		fmt.Println("PX_E2E_OFFLINE_COUNT_CONFLICT_SKIPPED")
		return nil
	}

	status, body, _, err = clientDo(ctx, client, http.MethodGet,
		base+"/v1/retailer/stock/counts/version?location_id="+locID+"&stock_bin=BACKROOM", nil, retailerToken, "")
	if err != nil {
		return fmt.Errorf("count version: %w", err)
	}
	if status == http.StatusNotFound {
		fmt.Println("PX_E2E_OFFLINE_COUNT_CONFLICT_SKIPPED")
		return nil
	}
	if status != http.StatusOK {
		return fmt.Errorf("count version status %d body %s", status, string(body))
	}
	var verResp struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(body, &verResp); err != nil {
		return fmt.Errorf("parse version: %w", err)
	}

	// Stale commit → 409
	stalePayload, _ := json.Marshal(map[string]any{
		"location_id":  locID,
		"stock_bin":    "BACKROOM",
		"base_version": verResp.Version - 1,
		"force":        false,
		"lines":        []map[string]any{{"sku_id": sku, "counted_qty": 1}},
	})
	status, body, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/stock/counts/commit",
		stalePayload, retailerToken, "offline-count-stale")
	if err != nil {
		return fmt.Errorf("stale commit: %w", err)
	}
	if status != http.StatusConflict {
		return fmt.Errorf("stale commit status %d want 409 body %s", status, string(body))
	}
	if !strings.Contains(string(body), "COUNT_VERSION_CONFLICT") {
		return fmt.Errorf("stale commit missing conflict code: %s", string(body))
	}
	fmt.Println("PX_E2E_OFFLINE_COUNT_CONFLICT_OK")
	return nil
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(envOr(key, "")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// issueRetailerOwnerJWT is defined in e2e_multi_org.go; referenced when needed from shared helpers.

var _ = auth.RoleRetailer
