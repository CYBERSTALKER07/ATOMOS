package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func runPlan90E2E(ctx context.Context, client *http.Client, base, cookie string) error {
	if err := runMEIONetworkE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("mei network: %w", err)
	}
	if err := runTouchlessReplenishE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("touchless replenish: %w", err)
	}
	if err := runControlTowerOverrideE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("control tower override: %w", err)
	}
	if err := runDemandBaselineE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("demand baseline: %w", err)
	}
	if err := runScenarioSandboxE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("scenario sandbox: %w", err)
	}
	if err := runKnowledgeGraphE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("knowledge graph: %w", err)
	}
	fmt.Println("PX_E2E_MEIO_NETWORK_OK")
	fmt.Println("PX_E2E_TOUCHLESS_REPLENISH_OK")
	fmt.Println("PX_E2E_CONTROL_TOWER_OVERRIDE_OK")
	fmt.Println("PX_E2E_DEMAND_BASELINE_OK")
	fmt.Println("PX_E2E_SCENARIO_SANDBOX_OK")
	fmt.Println("PX_E2E_KG_READ_OK")
	return nil
}

func runMEIONetworkE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/meio/network-summary", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("mei network summary status %d body %s", status, string(body))
	}
	var summary struct {
		WarehousesScanned int `json:"warehouses_scanned"`
	}
	if err := json.Unmarshal(body, &summary); err != nil {
		return err
	}
	return nil
}

func runTouchlessReplenishE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/replenishment/policies", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("replenishment policies status %d body %s", status, string(body))
	}
	return nil
}

func runControlTowerOverrideE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	polygon := map[string]any{
		"type": "Polygon",
		"coordinates": [][][]float64{
			{{69.0, 41.0}, {69.05, 41.0}, {69.05, 41.05}, {69.0, 41.05}, {69.0, 41.0}},
		},
	}
	createBody, _ := json.Marshal(map[string]any{
		"action":           "REROUTE",
		"ttl_seconds":      600,
		"polygon_geojson":  polygon,
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/supplier/control-tower/zone-overrides", createBody, cookie, "ssmr-ct-override")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("zone override create status %d body %s", status, string(respBody))
	}
	status, listBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/control-tower/zone-overrides", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("zone override list status %d body %s", status, string(listBody))
	}
	return nil
}

func runDemandBaselineE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/analytics/demand/today", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNotFound {
		return fmt.Errorf("demand today status %d body %s", status, string(body))
	}
	return nil
}

func runScenarioSandboxE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	reqBody, _ := json.Marshal(map[string]any{
		"factory_downtime_hours": 8,
		"demand_delta_pct":       10,
	})
	status, body, _, err := clientPost(ctx, client, base+"/v1/supplier/planning/scenarios/run", reqBody, cookie, "ssmr-scenario")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("scenario run status %d body %s", status, string(body))
	}
	var result struct {
		ScenarioID string `json:"scenario_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.ScenarioID == "" {
		return fmt.Errorf("scenario missing id: %s", string(body))
	}
	return nil
}

func runKnowledgeGraphE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/knowledge-graph", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("knowledge graph status %d body %s", status, string(body))
	}
	var kg struct {
		Nodes []any `json:"nodes"`
		Edges []any `json:"edges"`
	}
	if err := json.Unmarshal(body, &kg); err != nil {
		return err
	}
	if len(kg.Nodes) == 0 {
		return fmt.Errorf("knowledge graph empty nodes")
	}
	return nil
}
