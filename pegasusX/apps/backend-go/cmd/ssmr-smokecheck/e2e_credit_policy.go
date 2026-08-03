package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// runCreditPolicyE2E exercises irreversible enable, terms, self-serve disable 403, admin disable.
func runCreditPolicyE2E(ctx context.Context, client *http.Client, base, supplierCookie, supplierID, retailerID, retailerToken string) error {
	if !envTruthy("CREDIT_POLICY_V2_ENABLED") {
		fmt.Println("PX_E2E_CREDIT_POLICY_SKIPPED")
		return nil
	}

	// Enable without ack → 400
	status, body, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/credit-program",
		[]byte(`{"warning_ack":false}`), supplierCookie, "credit-prog-no-ack")
	if err != nil {
		return fmt.Errorf("enable program no-ack: %w", err)
	}
	if status != http.StatusBadRequest || !strings.Contains(string(body), "warning_ack_required") {
		return fmt.Errorf("expected warning_ack_required, got %d %s", status, string(body))
	}

	enableBody, _ := json.Marshal(map[string]any{
		"warning_ack":                true,
		"warning_ack_at":             "2026-08-04T00:00:00Z",
		"global_terms_days":          14,
		"global_default_limit_minor": 50_000_000,
	})
	status, body, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/credit-program",
		enableBody, supplierCookie, "credit-prog-enable")
	if err != nil {
		return fmt.Errorf("enable program: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("enable program status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_CREDIT_ENABLE_IRREVERSIBLE_OK")

	relBody, _ := json.Marshal(map[string]any{
		"warning_ack":         true,
		"warning_ack_at":      "2026-08-04T00:00:00Z",
		"terms_days":          14,
		"credit_limit_minor":  25_000_000,
		"use_global_defaults": false,
	})
	status, body, _, err = clientDo(ctx, client, http.MethodPost,
		base+"/v1/supplier/credit-relationships/"+retailerID+"/enable",
		relBody, supplierCookie, "credit-rel-enable")
	if err != nil {
		return fmt.Errorf("enable relationship: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("enable relationship status %d body %s", status, string(body))
	}

	// Self-serve disable → 403
	status, body, _, err = clientDo(ctx, client, http.MethodPost,
		base+"/v1/supplier/credit-relationships/"+retailerID+"/disable",
		[]byte(`{}`), supplierCookie, "credit-rel-disable")
	if err != nil {
		return fmt.Errorf("self-serve disable: %w", err)
	}
	if status != http.StatusForbidden || !strings.Contains(string(body), "credit_disable_requires_support") {
		return fmt.Errorf("expected credit_disable_requires_support, got %d %s", status, string(body))
	}

	// Terms patch
	termsBody, _ := json.Marshal(map[string]any{"terms_days": 21})
	status, body, _, err = clientDo(ctx, client, http.MethodPatch,
		base+"/v1/supplier/credit-relationships/"+retailerID+"/terms",
		termsBody, supplierCookie, "credit-rel-terms")
	if err != nil {
		return fmt.Errorf("patch terms: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("patch terms status %d body %s", status, string(body))
	}
	if !strings.Contains(string(body), `"terms_days":21`) && !strings.Contains(string(body), `"terms_days": 21`) {
		// JSON encoder may omit spaces; check numeric presence
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		if int(parsed["terms_days"].(float64)) != 21 {
			return fmt.Errorf("terms not patched: %s", string(body))
		}
	}
	fmt.Println("PX_E2E_CREDIT_TERMS_DUE_OK")

	// Retailer visibility
	status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/credit-relationships",
		nil, retailerToken, "")
	if err != nil {
		return fmt.Errorf("retailer relationships: %w", err)
	}
	if status != http.StatusOK || !strings.Contains(string(body), "relationships") {
		return fmt.Errorf("retailer relationships status %d body %s", status, string(body))
	}
	if !strings.Contains(string(body), supplierID) {
		return fmt.Errorf("retailer relationships missing supplier: %s", string(body))
	}

	// Admin disable
	adminBody, _ := json.Marshal(map[string]any{
		"ticket_id": "SSMR-CREDIT-1",
		"reason":    "e2e admin disable",
	})
	status, body, _, err = clientDo(ctx, client, http.MethodPost,
		base+"/v1/admin/credit-relationships/"+supplierID+"/"+retailerID+"/disable",
		adminBody, supplierCookie, "credit-admin-disable")
	if err != nil {
		return fmt.Errorf("admin disable: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("admin disable status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_CREDIT_ADMIN_DISABLE_OK")

	// Re-enable after admin disable (new ack)
	status, body, _, err = clientDo(ctx, client, http.MethodPost,
		base+"/v1/supplier/credit-relationships/"+retailerID+"/enable",
		relBody, supplierCookie, "credit-rel-reenable")
	if err != nil {
		return fmt.Errorf("re-enable: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("re-enable status %d body %s", status, string(body))
	}

	return nil
}
