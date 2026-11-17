package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ErrDemandSolverNotConfigured is returned when no AI demand sidecar URL is configured.
var ErrDemandSolverNotConfigured = errors.New("ai demand solver is not configured (set AI_DEMAND_SOLVER_URL, DISPATCH_OPTIMIZER_URL, OPTIMIZER_SIDECAR_URL, or OPTIMIZER_BASE_URL)")

type DemandPredictionRequest struct {
	SKU                  string `json:"sku"`
	HistoricalDailySales []int  `json:"historical_daily_sales"`
	CurrentStock         int64  `json:"current_stock"`
	LeadTimeDays         int    `json:"lead_time_days"`
}

type DemandPredictionResponse struct {
	SKU                string  `json:"sku"`
	OptimalReorderQty  int64   `json:"optimal_reorder_qty"`
	PredictedDailyBurn float64 `json:"predicted_daily_burn"`
}

var demandSolverHTTPClient = &http.Client{Timeout: 5 * time.Second}

func resolveDemandSolverBaseURL() (string, error) {
	for _, envKey := range []string{"AI_DEMAND_SOLVER_URL", "DISPATCH_OPTIMIZER_URL", "OPTIMIZER_SIDECAR_URL"} {
		if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
			return val, nil
		}
	}
	if val := strings.TrimSpace(os.Getenv("OPTIMIZER_BASE_URL")); val != "" && !strings.Contains(val, ":8082") {
		return val, nil
	}
	// Explicit non-production test/local opt-in only; fail closed in all other cases.
	allowLocal := strings.EqualFold(strings.TrimSpace(os.Getenv("AI_DEMAND_SIDECAR_ALLOW_LOCALHOST")), "true")
	if allowLocal && !auth.IsProduction() && !auth.IsStaging() {
		return (&url.URL{Scheme: "http", Host: net.JoinHostPort("localhost", "8000")}).String(), nil
	}
	return "", ErrDemandSolverNotConfigured
}

// PredictDemand calls the Python optimizer sidecar to compute reorder quantities.
func PredictDemand(ctx context.Context, req DemandPredictionRequest) (*DemandPredictionResponse, error) {
	baseURL, err := resolveDemandSolverBaseURL()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/predict-demand"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(os.Getenv("INTERNAL_API_KEY")); apiKey != "" {
		httpReq.Header.Set("X-Internal-Api-Key", apiKey)
	}

	resp, err := demandSolverHTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("demand prediction solver returned HTTP %d", resp.StatusCode)
	}

	var result DemandPredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
