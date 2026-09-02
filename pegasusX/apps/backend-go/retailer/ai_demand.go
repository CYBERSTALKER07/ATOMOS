package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type DemandPredictionRequest struct {
	SKU                  string `json:"sku"`
	HistoricalDailySales []int  `json:"historical_daily_sales"`
	CurrentStock         int64  `json:"current_stock"`
	LeadTimeDays         int    `json:"lead_time_days"`
}

type DemandPredictionResponse struct {
	SKU                 string  `json:"sku"`
	OptimalReorderQty   int64   `json:"optimal_reorder_qty"`
	PredictedDailyBurn  float64 `json:"predicted_daily_burn"`
}

// PredictDemand calls the Python optimizer sidecar to compute reorder quantities.
func PredictDemand(ctx context.Context, req DemandPredictionRequest) (*DemandPredictionResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8000/predict-demand", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DemandPredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
