package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func runDriverEdgesContractE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID string) error {
	driverID := envOr("SSMR_SMOKE_DRIVER_ID", "ssmr-driver-1")
	driverToken, err := auth.Issue(auth.Claims{
		Subject:    driverID,
		Role:       auth.RoleDriver,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}
	// Must not return 501 — contract is mounted on order.Service.
	payload := map[string]any{"route_id": "route-ssmr-1", "order_sequence": []string{"ord-missing"}}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/fleet/route/reorder", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+driverToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotImplemented {
		return fmt.Errorf("fleet route reorder still returns 501")
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fleet route reorder unavailable: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func runNegotiationE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, supplierCookie, orderID string) error {
	driverID := envOr("SSMR_SMOKE_DRIVER_ID", "ssmr-driver-1")
	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-supplier-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue admin jwt: %w", err)
	}

	assignBody, _ := json.Marshal(map[string]any{
		"driver_id": driverID,
		"route_id":  "route-ssmr-negotiate",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "admin-order-assign:"+orderID+":"+driverID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("negotiation assign status %d body %s", status, string(respBody))
	}
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, "admin-order-status:"+orderID+":"+next)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("negotiation order status %s: %d body %s", next, status, string(respBody))
		}
	}

	driverToken, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"),
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}

	negotiateBody, _ := json.Marshal(map[string]any{
		"order_id": orderID,
		"items": []map[string]any{
			{"sku_id": "SSMR-SKU-1", "original_qty": 2, "proposed_qty": 1},
		},
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/negotiate", negotiateBody, driverToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("negotiate propose status %d body %s", status, string(respBody))
	}
	var proposeResp struct {
		ProposalID string `json:"proposal_id"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &proposeResp); err != nil {
		return err
	}
	if proposeResp.ProposalID == "" || proposeResp.Status != "PENDING" {
		return fmt.Errorf("unexpected negotiate propose response: %s", string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/negotiations/pending", nil, supplierCookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("negotiations pending status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), proposeResp.ProposalID) {
		return fmt.Errorf("pending list missing proposal %s: %s", proposeResp.ProposalID, string(respBody))
	}

	resolveBody, _ := json.Marshal(map[string]string{
		"proposal_id": proposeResp.ProposalID,
		"action":      "APPROVE",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/negotiate/resolve", resolveBody, supplierCookie, "supplier-negotiate-resolve:"+proposeResp.ProposalID+":APPROVE")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("negotiate resolve status %d body %s", status, string(respBody))
	}
	var resolveResp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &resolveResp); err != nil {
		return err
	}
	if resolveResp.Status != "APPROVE" {
		return fmt.Errorf("unexpected negotiate resolve status %q body %s", resolveResp.Status, string(respBody))
	}
	return nil
}

func runShopClosedE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, retailerToken, orderID, supplierCookie string) error {
	driverID := envOr("SSMR_SMOKE_DRIVER_ID", "ssmr-driver-1")
	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-supplier-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue admin jwt: %w", err)
	}

	assignBody, _ := json.Marshal(map[string]any{
		"driver_id": driverID,
		"route_id":  "route-ssmr-shop-closed",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "admin-order-assign:"+orderID+":"+driverID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("assign order status %d body %s", status, string(respBody))
	}
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, "admin-order-status:"+orderID+":"+next)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("order status %s: %d body %s", next, status, string(respBody))
		}
	}

	driverToken, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"),
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}
	arriveBody, _ := json.Marshal(map[string]string{"order_id": orderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/arrive", arriveBody, driverToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("delivery arrive status %d body %s", status, string(respBody))
	}

	shopBody, _ := json.Marshal(map[string]any{
		"order_id":  orderID,
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/shop-closed", shopBody, driverToken, "driver-report-shop-closed:"+driverID+":"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("shop closed report status %d body %s", status, string(respBody))
	}
	inboxAuth := strings.TrimSpace(supplierCookie)
	if inboxAuth == "" {
		inboxAuth = adminToken
	}
	if err := assertInboxContainsEvent(ctx, client, base, inboxAuth, events.EventShopClosed); err != nil {
		return fmt.Errorf("shop closed inbox: %w", err)
	}

	responseBody, _ := json.Marshal(map[string]string{
		"order_id": orderID,
		"response": "OPEN_NOW",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/retailer/shop-closed-response", responseBody, retailerToken, "shop-closed-response:"+orderID+":OPEN_NOW")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("shop closed response status %d body %s", status, string(respBody))
	}
	return nil
}

func assertDriverManifestGate(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, driverID, manifestID string, wantCleared bool) error {
	if strings.TrimSpace(driverID) == "" {
		driverID = envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
	}
	token, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"),
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}
	if token == "" {
		return fmt.Errorf("empty driver token")
	}

	url := base + "/v1/driver/manifest-gate?manifest_id=" + manifestID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, url, nil, token, "")
	if err != nil {
		return err
	}
	var gate struct {
		Cleared bool   `json:"cleared"`
		Allowed bool   `json:"allowed"`
		State   string `json:"state"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(respBody, &gate)

	if wantCleared {
		if status != http.StatusOK {
			return fmt.Errorf("expected 200 cleared gate, got %d body %s", status, string(respBody))
		}
		if !gate.Cleared || !gate.Allowed {
			return fmt.Errorf("expected cleared gate, got %#v", gate)
		}
		return nil
	}
	if status != http.StatusForbidden {
		return fmt.Errorf("expected 403 awaiting seal, got %d body %s", status, string(respBody))
	}
	if gate.Cleared || gate.Error != "AWAITING_PAYLOAD_SEAL" {
		return fmt.Errorf("expected awaiting seal, got %#v", gate)
	}
	return nil
}

func assertDriverManifestDetail(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, driverID, manifestID, wantState string) error {
	if strings.TrimSpace(driverID) == "" {
		driverID = envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
	}
	token, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"),
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}

	url := base + "/v1/driver/manifest?manifest_id=" + manifestID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, url, nil, token, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200 manifest detail, got %d body %s", status, string(respBody))
	}
	var body struct {
		ManifestID string `json:"manifest_id"`
		StopCount  int    `json:"stop_count"`
		Manifest   struct {
			State string `json:"state"`
		} `json:"manifest"`
		Transfers []any `json:"transfers"`
	}
	if err := json.Unmarshal(respBody, &body); err != nil {
		return err
	}
	if body.ManifestID != manifestID {
		return fmt.Errorf("expected manifest_id %s, got %s", manifestID, body.ManifestID)
	}
	if wantState == "" {
		wantState = "SEALED"
	}
	if body.Manifest.State != wantState {
		return fmt.Errorf("expected %s manifest, got %q", wantState, body.Manifest.State)
	}
	if body.StopCount < 1 || len(body.Transfers) < 1 {
		return fmt.Errorf("expected stops/transfers on detail, got stop=%d transfers=%d", body.StopCount, len(body.Transfers))
	}
	return nil
}

func postDriverTelemetry(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID string) error {
	token, err := driverBearerToken(ctx, client, base)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{
		"lat": cfg.DeliveryZoneCenterLat,
		"lng": cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/telemetry/location", body, token, "")
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("telemetry status %d body %s", status, string(respBody))
	}
	return nil
}

func driverBearerToken(ctx context.Context, client *http.Client, base string) (string, error) {
	loginBody, _ := json.Marshal(map[string]string{
		"phone": envOr("DRIVER_DEMO_PHONE", "+998901000066"),
		"pin":   envOr("DRIVER_DEMO_PIN", "1234"),
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/driver/login", loginBody, "", "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("driver login status %d body %s", status, string(respBody))
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}
	if strings.TrimSpace(resp.Token) == "" {
		return "", fmt.Errorf("driver login missing token body %s", string(respBody))
	}
	return resp.Token, nil
}

func assertDriverFirebaseOTPLogin(ctx context.Context, client *http.Client, base string) error {
	testIDToken := strings.TrimSpace(os.Getenv("DRIVER_FIREBASE_TEST_ID_TOKEN"))
	if testIDToken == "" {
		return nil
	}
	otpBody, _ := json.Marshal(map[string]string{"id_token": testIDToken})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/driver/login", otpBody, "", "")
	if err != nil {
		return fmt.Errorf("driver firebase otp login: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("driver firebase otp login status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_DRIVER_FIREBASE_OTP_OK")
	return nil
}

func assertDriverDepart(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, driverID string) error {
	if strings.TrimSpace(driverID) == "" {
		driverID = envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
	}
	token, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"),
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/fleet/driver/depart", nil, token, "ssmr-driver-depart:"+driverID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("driver depart status %d body %s", status, string(respBody))
	}
	var depart struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &depart); err != nil {
		return fmt.Errorf("decode driver depart: %w", err)
	}
	if strings.TrimSpace(depart.Status) != "departed" {
		return fmt.Errorf("driver depart unexpected status %q body %s", depart.Status, string(respBody))
	}
	return nil
}

func runDriverNotificationInboxE2E(ctx context.Context, client *http.Client, base string) error {
	token, err := driverBearerToken(ctx, client, base)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = assertInboxHasRows(ctx, client, base, token, "driver")
		if lastErr == nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("driver inbox not ready after kafka fanout window: %w", lastErr)
	}
	markBody, _ := json.Marshal(map[string]any{"mark_all": true})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, token, "ssmr-driver-inbox-read")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("driver mark notifications read status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_DRIVER_NOTIFICATION_INBOX_OK")
	return nil
}

func runShopClosedSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
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
	if err := grantRetailerCredit(ctx, client, base, cookie, retailerID, 500_000_000); err != nil {
		return fmt.Errorf("retailer credit grant: %w", err)
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
	sessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, orderID, cfg, supplierID)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("webhook: %w", err)
	}
	if err := runShopClosedE2E(ctx, client, base, cfg, supplierID, retailerToken, orderID, cookie); err != nil {
		return err
	}
	fmt.Println("PX_E2E_SHOP_CLOSED_OK")
	return nil
}

// runManifestSealSmokeCheck exercises payloader start-loading, order seal, and manifest seal.
