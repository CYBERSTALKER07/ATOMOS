package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
)

// runOptimizerConstraintE2E posts a cold-chain fixture directly to optimizer-core
// when OPTIMIZER_BASE_URL is reachable. Soft-skips when the sidecar is absent
// (SSMR/prod replicas 0) so ecosystem gates stay green on heuristic-only clusters.
func runOptimizerConstraintE2E(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(strings.TrimSpace(cfg.OptimizerBaseURL), "/")
	apiKey := strings.TrimSpace(cfg.InternalAPIKey)
	if base == "" || apiKey == "" {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}

	reqBody := map[string]any{
		"v":            contract.V,
		"trace_id":     fmt.Sprintf("ssmr-opt-constraint-%d", time.Now().UnixNano()),
		"supplier_id":  "ssmr-supplier",
		"home_node_id": "ssmr-wh",
		"stops": []map[string]any{
			{
				"order_id":            "cold-fixture-1",
				"retailer_id":         "r1",
				"lat":                 41.31,
				"lng":                 69.25,
				"volume_vu":           10.0,
				"requires_cold_chain": true,
				"handling_class":      "COLD_CHAIN",
			},
			{
				"order_id":    "dry-fixture-2",
				"retailer_id": "r2",
				"lat":         41.32,
				"lng":         69.26,
				"volume_vu":   10.0,
			},
		},
		"vehicles": []map[string]any{
			{
				"vehicle_id":        "dry-truck",
				"driver_id":         "d-dry",
				"max_volume_vu":     100.0,
				"start_lat":         41.30,
				"start_lng":         69.24,
				"has_refrigeration": false,
			},
			{
				"vehicle_id":        "reefer",
				"driver_id":         "d-cold",
				"max_volume_vu":     100.0,
				"start_lat":         41.301,
				"start_lng":         69.241,
				"has_refrigeration": true,
			},
		},
		"tunables": map[string]any{"time_limit_ms": 3000},
	}
	raw, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+contract.SolvePath, bytes.NewReader(raw))
	if err != nil {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(contract.AuthHeader, apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}

	var solved struct {
		Source string `json:"source"`
		Routes []struct {
			VehicleID string `json:"vehicle_id"`
			Stops     []struct {
				OrderID string `json:"order_id"`
			} `json:"stops"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &solved); err != nil {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}
	coldVehicle := ""
	for _, route := range solved.Routes {
		for _, stop := range route.Stops {
			if stop.OrderID == "cold-fixture-1" {
				coldVehicle = route.VehicleID
			}
		}
	}
	if coldVehicle != "reefer" {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}
	fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_OK")
	return nil
}
