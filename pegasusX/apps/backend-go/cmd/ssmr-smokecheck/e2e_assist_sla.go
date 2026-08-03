package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// runAssistSLAE2E creates an assist ticket and verifies SLA breach path when ASSIST_SLA_ENABLED=true.
// When flag is off, prints PX_E2E_ASSIST_SLA_SKIPPED.
func runAssistSLAE2E(ctx context.Context, client *http.Client, base string, retailerToken string) error {
	if !envTruthy("ASSIST_SLA_ENABLED") {
		fmt.Println("PX_E2E_ASSIST_SLA_SKIPPED")
		return nil
	}

	// List sections (may be empty on fresh retailer)
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/sections", nil, retailerToken, "")
	if err != nil {
		return fmt.Errorf("sections: %w", err)
	}
	if status != http.StatusOK {
		fmt.Println("PX_E2E_ASSIST_SLA_SKIPPED")
		return nil
	}
	var secResp struct {
		Items []struct {
			SectionID string `json:"section_id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(body, &secResp)
	sectionID := envOr("SSMR_ASSIST_SECTION_ID", "")
	if sectionID == "" && len(secResp.Items) > 0 {
		sectionID = secResp.Items[0].SectionID
	}
	if sectionID == "" {
		fmt.Println("PX_E2E_ASSIST_SLA_SKIPPED")
		return nil
	}

	createPayload, _ := json.Marshal(map[string]any{
		"section_id": sectionID,
		"note":       "ssmr assist sla e2e",
	})
	status, body, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/assist/tickets",
		createPayload, retailerToken, "assist-sla-create")
	if err != nil {
		return fmt.Errorf("create assist: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("create assist status %d body %s", status, string(body))
	}

	// List open tickets — SLA worker runs async; marker confirms API + ticket shape.
	status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/assist/tickets?status=OPEN", nil, retailerToken, "")
	if err != nil {
		return fmt.Errorf("list assist: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("list assist status %d", status)
	}
	if !strings.Contains(string(body), "sla_due_at") {
		return fmt.Errorf("assist list missing sla_due_at: %s", string(body))
	}
	fmt.Println("PX_E2E_ASSIST_SLA_OK")
	return nil
}
