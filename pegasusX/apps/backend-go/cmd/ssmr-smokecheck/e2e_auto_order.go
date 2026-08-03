package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// runAutoOrderDraftE2E posts a draft auto-order run when the retailer has settings.
// Prints PX_E2E_AUTO_ORDER_DRAFT_OK or SKIPPED.
func runAutoOrderDraftE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodPost,
		base+"/v1/retailer/settings/auto-order/run?mode=draft", nil, retailerToken, "auto-order-draft")
	if err != nil {
		return fmt.Errorf("auto-order run: %w", err)
	}
	if status == http.StatusNotFound || status == http.StatusForbidden {
		fmt.Println("PX_E2E_AUTO_ORDER_DRAFT_SKIPPED")
		return nil
	}
	if status != http.StatusOK {
		return fmt.Errorf("auto-order run status %d body %s", status, string(body))
	}
	var run struct {
		Status string `json:"status"`
		Mode   string `json:"mode"`
	}
	_ = json.Unmarshal(body, &run)
	if run.Mode != "" && run.Mode != "draft" {
		return fmt.Errorf("auto-order mode %q want draft", run.Mode)
	}
	// List runs for audit surface
	status, body, _, err = clientDo(ctx, client, http.MethodGet,
		base+"/v1/retailer/settings/auto-order/runs?limit=5", nil, retailerToken, "")
	if err != nil {
		return fmt.Errorf("auto-order runs: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("auto-order runs status %d", status)
	}
	fmt.Println("PX_E2E_AUTO_ORDER_DRAFT_OK")
	return nil
}
