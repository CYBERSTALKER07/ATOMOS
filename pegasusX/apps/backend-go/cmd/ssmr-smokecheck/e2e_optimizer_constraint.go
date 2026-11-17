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

type optimizerRouteStop struct {
	OrderID string `json:"order_id"`
}

type optimizerRoute struct {
	VehicleID string               `json:"vehicle_id"`
	Stops     []optimizerRouteStop `json:"stops"`
}

// vehicleForStop returns the vehicle_id carrying orderID, or "".
func vehicleForStop(routes []optimizerRoute, orderID string) string {
	for _, route := range routes {
		for _, stop := range route.Stops {
			if stop.OrderID == orderID {
				return route.VehicleID
			}
		}
	}
	return ""
}

// runOptimizerConstraintE2E posts a hazmat fixture directly to optimizer-core
// when OPTIMIZER_BASE_URL is reachable. Soft-skips only when the sidecar is
// absent (no URL/key, dial/transport failure) so heuristic-only clusters stay
// green. Once the sidecar answers, constraint mis-assignment is a hard FAIL
// (P2-1) — never papered over as SKIPPED.
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
				"order_id":       "hazmat-fixture-1",
				"retailer_id":    "r1",
				"lat":            41.31,
				"lng":            69.25,
				"volume_vu":      10.0,
				"is_hazardous":   true,
				"handling_class": "HAZARDOUS",
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
				"vehicle_id":       "dry-truck",
				"driver_id":        "d-dry",
				"max_volume_vu":    100.0,
				"start_lat":        41.30,
				"start_lng":        69.24,
				"hazmat_certified": false,
			},
			{
				"vehicle_id":       "hazmat-truck",
				"driver_id":        "d-hazmat",
				"max_volume_vu":    100.0,
				"start_lat":        41.301,
				"start_lng":        69.241,
				"hazmat_certified": true,
			},
		},
		"tunables": map[string]any{"time_limit_ms": 3000},
	}
	raw, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+contract.SolvePath, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("PX_E2E_OPTIMIZER_CONSTRAINT_FAIL: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(contract.AuthHeader, apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		// Sidecar not reachable — soft-skip (replicas 0 / compose down).
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED")
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("PX_E2E_OPTIMIZER_CONSTRAINT_FAIL status=%d body=%s\n", resp.StatusCode, truncateForE2E(string(body), 240))
		return fmt.Errorf("optimizer constraint e2e: sidecar HTTP %d", resp.StatusCode)
	}

	var solved struct {
		Source string           `json:"source"`
		Routes []optimizerRoute `json:"routes"`
	}
	if err := json.Unmarshal(body, &solved); err != nil {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_FAIL")
		return fmt.Errorf("optimizer constraint e2e: decode response: %w", err)
	}
	hazmatVehicle := vehicleForStop(solved.Routes, "hazmat-fixture-1")
	if hazmatVehicle == "" {
		fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_FAIL")
		return fmt.Errorf("optimizer constraint e2e: hazmat-fixture-1 missing from routes (source=%s)", solved.Source)
	}
	if hazmatVehicle != "hazmat-truck" {
		// Constraint violation: hazmat stop on a non-certified vehicle — hard fail (P2-1).
		fmt.Printf("PX_E2E_OPTIMIZER_CONSTRAINT_FAIL hazmat_vehicle=%s want=hazmat-truck source=%s\n", hazmatVehicle, solved.Source)
		return fmt.Errorf("optimizer constraint e2e: hazmat stop assigned to %q (want hazmat-truck)", hazmatVehicle)
	}
	fmt.Println("PX_E2E_OPTIMIZER_CONSTRAINT_OK")
	return nil
}

func truncateForE2E(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
