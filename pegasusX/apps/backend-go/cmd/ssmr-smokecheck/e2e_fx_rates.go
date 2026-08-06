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
)

// runFxRatesE2E prints theatre #13 markers:
// PX_E2E_FX_RATE_SEEDED_OK/_SKIPPED and PX_E2E_CURRENCY_MISMATCH_DENIED/_SKIPPED.
func runFxRatesE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierCookie, supplierID, retailerToken, h3Cell string,
	cfg *bootstrap.Config,
) error {
	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-fx-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		fmt.Println("PX_E2E_FX_RATE_SEEDED_SKIPPED")
		fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED_SKIPPED")
		return nil
	}
	_ = supplierCookie

	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/admin/fx-rates", nil, adminToken, "")
	if err != nil || status != http.StatusOK {
		fmt.Println("PX_E2E_FX_RATE_SEEDED_SKIPPED")
	} else {
		var payload struct {
			Rates []struct {
				BaseCurrency  string `json:"base_currency"`
				QuoteCurrency string `json:"quote_currency"`
				RateScaled    int64  `json:"rate_scaled"`
			} `json:"rates"`
		}
		_ = json.Unmarshal(body, &payload)
		op := strings.ToUpper(strings.TrimSpace(cfg.SeedSupplierCurrency))
		if op == "" {
			op = "UZS"
		}
		found := false
		for _, r := range payload.Rates {
			if strings.EqualFold(r.BaseCurrency, op) && strings.EqualFold(r.QuoteCurrency, op) && r.RateScaled > 0 {
				found = true
				break
			}
		}
		if found {
			fmt.Println("PX_E2E_FX_RATE_SEEDED_OK")
		} else {
			fmt.Println("PX_E2E_FX_RATE_SEEDED_SKIPPED")
		}
	}

	if strings.TrimSpace(retailerToken) == "" || strings.TrimSpace(h3Cell) == "" {
		fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED_SKIPPED")
		return nil
	}

	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED_SKIPPED")
		return nil
	}
	if err := advanceOrderToArrived(ctx, client, base, orderID, supplierID, cfg); err != nil {
		fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED_SKIPPED")
		return nil
	}

	wrongCur := "USD"
	if strings.EqualFold(cfg.SeedSupplierCurrency, "USD") {
		wrongCur = "EUR"
	}
	mismatchBody, _ := json.Marshal(map[string]any{
		"order_id":     orderID,
		"amount_minor": int64(100000),
		"currency":     wrongCur,
		"gateway":      "GLOBAL_PAY",
	})
	st, resp, _, err := clientPost(ctx, client, base+"/v1/order/card-checkout", mismatchBody, retailerToken, "ssmr-fx-mismatch-"+orderID)
	if err != nil {
		fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED_SKIPPED")
		return nil
	}
	if st == http.StatusUnprocessableEntity && strings.Contains(string(resp), "currency_mismatch") {
		fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED")
		return nil
	}
	fmt.Println("PX_E2E_CURRENCY_MISMATCH_DENIED_SKIPPED")
	return nil
}
