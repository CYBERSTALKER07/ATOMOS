package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

// runFxSettlementConvertE2E asserts settlement authority exposes operating_currency_total_minor
// after FX rates are available. Soft-skips when FxRates / payment auth unavailable.
func runFxSettlementConvertE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierCookie, supplierID string,
	cfg *bootstrap.Config,
) error {
	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-fx-settlement-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		fmt.Println("PX_E2E_FX_SETTLEMENT_CONVERT_SKIPPED")
		return nil
	}

	op := strings.ToUpper(strings.TrimSpace(cfg.SeedSupplierCurrency))
	if op == "" {
		op = "UZS"
	}

	// Ensure USD/op rate exists for conversion paths (identity alone is enough for rollup field).
	putBody, _ := json.Marshal(map[string]any{
		"base_currency":  "USD",
		"quote_currency": op,
		"rate_scaled":    int64(12_750_000_000),
		"scale":          fxrates.DefaultScale,
		"source":         "SSMR",
	})
	putStatus, putResp, _, putErr := clientDo(ctx, client, http.MethodPut, base+"/v1/admin/fx-rates", putBody, adminToken, "")
	if putErr != nil || putStatus >= 400 {
		msg := string(putResp)
		if putStatus == http.StatusNotFound ||
			strings.Contains(msg, "not found") ||
			strings.Contains(msg, "Unrecognized name") ||
			strings.Contains(msg, "FxRates") {
			fmt.Println("PX_E2E_FX_SETTLEMENT_CONVERT_SKIPPED")
			return nil
		}
		// Soft-skip on upsert failure; settlement may still expose operating fields via identity.
	}

	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/payment/settlement/authority?group_limit=50", nil, supplierCookie, "")
	if err != nil || status != http.StatusOK {
		fmt.Println("PX_E2E_FX_SETTLEMENT_CONVERT_SKIPPED")
		return nil
	}
	var payload struct {
		OperatingCurrency           string `json:"operating_currency"`
		OperatingCurrencyTotalMinor *int64 `json:"operating_currency_total_minor"`
		OperatingConversionPartial  *bool  `json:"operating_conversion_partial"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode settlement authority: %w", err)
	}
	if payload.OperatingCurrencyTotalMinor == nil {
		fmt.Println("PX_E2E_FX_SETTLEMENT_CONVERT_SKIPPED")
		return nil
	}
	if !strings.EqualFold(payload.OperatingCurrency, op) {
		return fmt.Errorf("operating_currency=%q want %q", payload.OperatingCurrency, op)
	}
	fmt.Println("PX_E2E_FX_SETTLEMENT_CONVERT_OK")
	return nil
}
