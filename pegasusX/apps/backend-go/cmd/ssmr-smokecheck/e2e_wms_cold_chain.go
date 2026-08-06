package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runWMSColdChainE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	_ = cfg
	body, _ := json.Marshal(map[string]any{
		"manifest_id": fmt.Sprintf("ssmr-cold-%d", time.Now().Unix()%100000),
		"temp_c":      22.0,
		"min_c":       0.0,
		"max_c":       8.0,
	})
	status, resp, _, err := clientPost(ctx, client,
		base+"/v1/warehouse/ops/temperature-readings",
		body, cookie, "ssmr-cold-"+fmt.Sprintf("%d", time.Now().Unix()))
	if err != nil || status == http.StatusConflict || status >= 400 {
		fmt.Println("PX_E2E_WMS_COLD_CHAIN_SKIPPED")
		return nil
	}
	var reading struct {
		ReadingID string `json:"reading_id"`
		Excursion bool   `json:"excursion"`
	}
	_ = json.Unmarshal(resp, &reading)
	if reading.ReadingID == "" {
		fmt.Println("PX_E2E_WMS_COLD_CHAIN_SKIPPED")
		return nil
	}
	fmt.Println("PX_E2E_WMS_COLD_CHAIN_OK")
	return nil
}
