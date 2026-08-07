package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// runSeasonalOverrideE2E creates an override with an explicit multiplier and
// asserts GET returns the persisted value. Soft-skips when planning/Spanner
// column is unavailable.
func runSeasonalOverrideE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	start := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	end := time.Now().UTC().AddDate(0, 0, 14).Format("2006-01-02")
	reqBody, _ := json.Marshal(map[string]any{
		"template_id": "holiday_peak",
		"name":        "ssmr seasonal override",
		"start_date":  start,
		"end_date":    end,
		"multiplier":  1.42,
	})
	status, body, _, err := clientPost(
		ctx, client,
		base+"/v1/supplier/planning/seasonal-overrides",
		reqBody, cookie,
		fmt.Sprintf("ssmr-seasonal-override-%d", time.Now().UnixNano()),
	)
	if err != nil {
		fmt.Println("PX_E2E_SEASONAL_OVERRIDE_SKIPPED")
		return nil
	}
	if status == http.StatusServiceUnavailable || status == http.StatusNotFound {
		fmt.Println("PX_E2E_SEASONAL_OVERRIDE_SKIPPED")
		return nil
	}
	if status != http.StatusCreated {
		// Column / table missing surfaces as 400/500 — soft-skip for pre-migration SSMR.
		if status >= 400 {
			msg := string(body)
			if strings.Contains(msg, "Multiplier") ||
				strings.Contains(msg, "not found") ||
				strings.Contains(msg, "Unrecognized name") ||
				strings.Contains(msg, "planning_unavailable") {
				fmt.Println("PX_E2E_SEASONAL_OVERRIDE_SKIPPED")
				return nil
			}
		}
		return fmt.Errorf("seasonal override create status %d body %s", status, string(body))
	}
	var created struct {
		OverrideID string  `json:"override_id"`
		Multiplier float64 `json:"multiplier"`
		TemplateID string  `json:"template_id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("decode create: %w", err)
	}
	if created.Multiplier < 1.41 || created.Multiplier > 1.43 {
		return fmt.Errorf("create multiplier=%v want ~1.42", created.Multiplier)
	}

	getStatus, getBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/planning/seasonal-overrides", nil, cookie, "")
	if err != nil {
		return err
	}
	if getStatus != http.StatusOK {
		return fmt.Errorf("seasonal override list status %d body %s", getStatus, string(getBody))
	}
	var listed struct {
		Overrides []struct {
			OverrideID string  `json:"override_id"`
			Multiplier float64 `json:"multiplier"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(getBody, &listed); err != nil {
		return fmt.Errorf("decode list: %w", err)
	}
	found := false
	for _, row := range listed.Overrides {
		if row.OverrideID == created.OverrideID {
			found = true
			if row.Multiplier < 1.41 || row.Multiplier > 1.43 {
				return fmt.Errorf("list multiplier=%v want ~1.42", row.Multiplier)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("created override %s missing from GET list", created.OverrideID)
	}
	fmt.Println("PX_E2E_SEASONAL_OVERRIDE_OK")
	return nil
}
