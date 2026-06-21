package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func replayGlobalPayWebhook(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, sessionID, orderID string) error {
	secret := envOr("GLOBAL_PAY_WEBHOOK_SECRET", "dev-global-pay-secret")
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("Paycom:"+secret))
	body, _ := json.Marshal(map[string]any{
		"session_id":     sessionID,
		"transaction_id": "ssmr-tx-" + orderID,
		"status":         "CAPTURED",
		"order_id":       orderID,
		"amount_minor":   int64(100000),
		"currency":       cfg.SeedSupplierCurrency,
	})
	idem := "ssmr-webhook-" + orderID
	status, respBody, _, err := clientPostAuthorized(ctx, client, base+"/v1/webhooks/global-pay", body, authHeader, idem)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("webhook status %d body %s", status, string(respBody))
	}
	status2, respBody2, _, err := clientPostAuthorized(ctx, client, base+"/v1/webhooks/global-pay", body, authHeader, idem)
	if err != nil {
		return err
	}
	if status2 != http.StatusOK {
		return fmt.Errorf("webhook replay status %d body %s", status2, string(respBody2))
	}
	return nil
}

func runPaymentSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 45 * time.Second}
	if _, err := clientGet(ctx, client, base+"/v1/health"); err != nil {
		return fmt.Errorf("health: %w", err)
	}
	supplierID, cookie, err := ensureSupplierSession(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("supplier session: %w", err)
	}
	if err := putSupplierTopology(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier topology: %w", err)
	}
	retailerID, h3Cell, err := registerRetailer(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("retailer register: %w", err)
	}
	retailerToken, err := auth.Issue(auth.Claims{
		Subject:    retailerID,
		Role:       auth.RoleRetailer,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
	if err != nil {
		return fmt.Errorf("issue retailer jwt: %w", err)
	}
	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("order create: %w", err)
	}
	sessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, orderID, cfg)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("webhook: %w", err)
	}
	fmt.Println("PX_E2E_PAYMENT_OK")
	return nil
}

// runShopClosedSmokeCheck exercises driver shop-closed report and retailer response.
