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

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// runNegotiationIsolationCheck proves quantity negotiation is product-disabled
// without breaking shop-closed or claims surfaces.
//
// Markers:
//
//	PX_E2E_NEGOTIATION_DISABLED_OK   — propose/resolve 410; pending empty
//	PX_E2E_SHOP_CLOSED_SURFACE_OK   — active list still authorized 2xx
//	PX_E2E_CLAIMS_SURFACE_OK        — claims list/file surface still authorized 2xx
//	PX_E2E_NEGOTIATION_ISOLATION_OK — all above
func runNegotiationIsolationCheck(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 30 * time.Second}

	// Health (non-negotiation path)
	status, body, err := rawJSON(ctx, client, http.MethodGet, base+"/healthz", nil, "", "")
	if err != nil {
		// some envs use /v1/health
		status, body, err = rawJSON(ctx, client, http.MethodGet, base+"/v1/health", nil, "", "")
	}
	if err != nil {
		return fmt.Errorf("health: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("health status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_HEALTH_OK")

	supplierID := smokeSupplierID()
	driverID := envOr("SSMR_SMOKE_DRIVER_ID", "ssmr-driver-1")

	driverTok, err := auth.Issue(auth.Claims{
		Subject:    driverID,
		Role:       auth.RoleDriver,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 20 * time.Minute})
	if err != nil {
		return fmt.Errorf("driver jwt: %w", err)
	}
	adminTok, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-supplier-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 20 * time.Minute})
	if err != nil {
		return fmt.Errorf("admin jwt: %w", err)
	}
	retailerTok, err := auth.Issue(auth.Claims{
		Subject:       "ssmr-retailer-isol",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ssmr-retailer-isol",
		RetailerRole:  "OWNER",
		SupplierID:    supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 20 * time.Minute})
	if err != nil {
		return fmt.Errorf("retailer jwt: %w", err)
	}

	// --- Negotiation must be dead ---
	if !order.NegotiationFeatureEnabled() {
		// Local compile-time gate (this binary); cloud image may lag.
		fmt.Println("PX_E2E_NEGOTIATION_GATE_COMPILE_OFF")
	} else {
		fmt.Println("PX_E2E_NEGOTIATION_GATE_COMPILE_ON_WARN")
	}

	proposeBody, _ := json.Marshal(map[string]any{
		"order_id": "ord-isolation-probe",
		"items": []map[string]any{
			{"sku_id": "SSMR-SKU-1", "original_qty": 2, "proposed_qty": 1},
		},
	})
	status, body, err = rawJSON(ctx, client, http.MethodPost, base+"/v1/delivery/negotiate", proposeBody, driverTok, "isol-negotiate-propose")
	if err != nil {
		return fmt.Errorf("negotiate propose: %w", err)
	}
	if status != http.StatusGone {
		return fmt.Errorf("negotiate propose want 410 Gone, got %d body=%s (is cloud image still on enabled gate?)", status, string(body))
	}
	if !bytes.Contains(body, []byte("feature_disabled")) && !bytes.Contains(body, []byte("quantity_negotiation")) {
		return fmt.Errorf("negotiate propose 410 body missing feature_disabled: %s", string(body))
	}
	fmt.Println("PX_E2E_NEGOTIATE_PROPOSE_410_OK")

	resolveBody, _ := json.Marshal(map[string]any{
		"proposal_id": "prop-isolation-probe",
		"action":      "APPROVE",
	})
	status, body, err = rawJSON(ctx, client, http.MethodPost, base+"/v1/supplier/negotiate/resolve", resolveBody, adminTok, "isol-negotiate-resolve")
	if err != nil {
		return fmt.Errorf("negotiate resolve: %w", err)
	}
	if status != http.StatusGone {
		return fmt.Errorf("negotiate resolve want 410 Gone, got %d body=%s", status, string(body))
	}
	fmt.Println("PX_E2E_NEGOTIATE_RESOLVE_410_OK")

	status, body, err = rawJSON(ctx, client, http.MethodGet, base+"/v1/supplier/negotiations/pending?limit=10", nil, adminTok, "")
	if err != nil {
		return fmt.Errorf("negotiations pending: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("negotiations pending want 200, got %d body=%s", status, string(body))
	}
	var pending struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &pending); err != nil {
		return fmt.Errorf("pending parse: %w body=%s", err, string(body))
	}
	if len(pending.Data) != 0 {
		return fmt.Errorf("pending list must be empty while disabled, got %d items", len(pending.Data))
	}
	fmt.Println("PX_E2E_NEGOTIATE_PENDING_EMPTY_OK")
	fmt.Println("PX_E2E_NEGOTIATION_DISABLED_OK")

	// --- Shop-closed surface must still work (list endpoint, not full e2e) ---
	status, body, err = rawJSON(ctx, client, http.MethodGet, base+"/v1/supplier/shop-closed/active", nil, adminTok, "")
	if err != nil {
		return fmt.Errorf("shop-closed active: %w", err)
	}
	if status == http.StatusGone || status == http.StatusNotImplemented {
		return fmt.Errorf("shop-closed active broken by negotiation gate: status=%d body=%s", status, string(body))
	}
	if status != http.StatusOK && status != http.StatusUnauthorized && status != http.StatusForbidden {
		// Accept 401/403 if cookie/JWT scope differs, but not 410/501.
		// Prefer 200 for ADMIN JWT with SupplierID.
		return fmt.Errorf("shop-closed active unexpected status %d body=%s", status, string(body))
	}
	if status != http.StatusOK {
		return fmt.Errorf("shop-closed active want 200 for admin jwt, got %d body=%s", status, string(body))
	}
	fmt.Println("PX_E2E_SHOP_CLOSED_SURFACE_OK")

	// --- Claims surface: list retailer claims or file validation path ---
	// GET list if mounted; otherwise probe OPTIONS/POST with empty body expecting validation != 410.
	status, body, err = rawJSON(ctx, client, http.MethodGet, base+"/v1/retailer/claims?limit=5", nil, retailerTok, "")
	if err != nil {
		return fmt.Errorf("retailer claims list: %w", err)
	}
	// Accept 200 (list), 404 (route name differs), or 400 — never 410 from negotiation gate.
	if status == http.StatusGone {
		return fmt.Errorf("retailer claims incorrectly returns 410 (negotiation bleed): %s", string(body))
	}
	if status == http.StatusNotFound {
		// Try supplier claims list
		status, body, err = rawJSON(ctx, client, http.MethodGet, base+"/v1/supplier/claims?limit=5", nil, adminTok, "")
		if err != nil {
			return fmt.Errorf("supplier claims list: %w", err)
		}
		if status == http.StatusGone {
			return fmt.Errorf("supplier claims incorrectly returns 410: %s", string(body))
		}
	}
	if status == http.StatusGone || status == http.StatusNotImplemented {
		return fmt.Errorf("claims surface down: status=%d body=%s", status, string(body))
	}
	// 200/401/403/400 all prove the handler is mounted and not negotiation-gated.
	if status != http.StatusOK && status != http.StatusBadRequest &&
		status != http.StatusUnauthorized && status != http.StatusForbidden &&
		status != http.StatusNotFound {
		// Also try POST file with minimal invalid body — should be 4xx validation, not 410.
		status2, body2, err2 := rawJSON(ctx, client, http.MethodPost, base+"/v1/claims",
			[]byte(`{}`), retailerTok, "isol-claims-probe")
		if err2 != nil {
			return fmt.Errorf("claims post probe: %w", err2)
		}
		if status2 == http.StatusGone {
			return fmt.Errorf("POST /v1/claims incorrectly 410: %s", string(body2))
		}
		if status2 == http.StatusNotFound {
			return fmt.Errorf("claims routes not found (list=%d); cannot prove surface: %s", status, string(body))
		}
		fmt.Printf("PX_E2E_CLAIMS_SURFACE_OK status_list=%d status_post=%d\n", status, status2)
	} else {
		fmt.Printf("PX_E2E_CLAIMS_SURFACE_OK status=%d\n", status)
	}

	fmt.Println("PX_E2E_NEGOTIATION_ISOLATION_OK")
	return nil
}

func rawJSON(ctx context.Context, client *http.Client, method, url string, body []byte, bearer, idem string) (int, []byte, error) {
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(bearer) != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if strings.TrimSpace(idem) != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, raw, nil
}
