package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func e2eTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("SSMR_E2E_TIMEOUT_SEC")); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 3 * time.Minute
}

// runE2ECheck exercises supplier topology, retailer registration, order create,
// and tracking against a live backend (SSMR stack).
func runE2ECheck(ctx context.Context, cfg *bootstrap.Config) error {
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
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue retailer jwt: %w", err)
	}

	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("order create: %w", err)
	}

	if err := assertRetailerTracking(ctx, client, base, retailerToken, orderID); err != nil {
		return fmt.Errorf("retailer tracking: %w", err)
	}

	sessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, orderID, cfg)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("global-pay webhook: %w", err)
	}
	if err := runWarehouseDispatchPreview(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse dispatch preview: %w", err)
	}
	if err := runWarehouseDispatchLock(ctx, client, base, cookie, orderID); err != nil {
		return fmt.Errorf("warehouse dispatch lock: %w", err)
	}
	if err := runWarehouseOrderMutationE2E(ctx, client, base, cookie, orderID); err != nil {
		return fmt.Errorf("warehouse order mutation: %w", err)
	}
	shopClosedOrderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("shop closed order create: %w", err)
	}
	shopClosedSessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, shopClosedOrderID, cfg)
	if err != nil {
		return fmt.Errorf("shop closed checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, shopClosedSessionID, shopClosedOrderID); err != nil {
		return fmt.Errorf("shop closed webhook: %w", err)
	}
	if err := runShopClosedE2E(ctx, client, base, cfg, supplierID, retailerToken, shopClosedOrderID); err != nil {
		return fmt.Errorf("shop closed e2e: %w", err)
	}
	if err := runWarehouseTransferActionsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse transfer actions: %w", err)
	}
	if err := assertSupplierPortalAPIs(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("supplier portal apis: %w", err)
	}
	if err := runFactoryOps(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("factory ops: %w", err)
	}
	if err := postDriverTelemetry(ctx, client, base, cfg, supplierID); err != nil {
		return fmt.Errorf("driver telemetry: %w", err)
	}
	if err := runPayloaderE2E(ctx, client, base, cfg, supplierID); err != nil {
		return fmt.Errorf("payloader e2e: %w", err)
	}
	if negotiateOrderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell); err != nil {
		return fmt.Errorf("negotiation order create: %w", err)
	} else if err := runNegotiationE2E(ctx, client, base, cfg, supplierID, cookie, negotiateOrderID); err != nil {
		return fmt.Errorf("negotiation e2e: %w", err)
	}

	if err := runClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("client policy e2e: %w", err)
	}
	if err := runCatalogCategorySuppliersE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("catalog category suppliers: %w", err)
	}
	if err := runDeviceTokenE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("device token: %w", err)
	}
	if err := runDriverEdgesContractE2E(ctx, client, base, cfg, supplierID); err != nil {
		return fmt.Errorf("driver edges contract: %w", err)
	}

	fmt.Println("PX_E2E_ORDER_OK")
	fmt.Println("PX_E2E_PAYMENT_OK")
	fmt.Println("PX_E2E_WAREHOUSE_OK")
	fmt.Println("PX_E2E_FACTORY_OK")
	fmt.Println("PX_E2E_DELIVERY_OK")
	fmt.Println("PX_E2E_TELEMETRY_OK")
	fmt.Println("PX_E2E_PAYLOAD_OK")
	fmt.Println("PX_E2E_SHOP_CLOSED_OK")
	fmt.Println("PX_E2E_NEGOTIATION_OK")
	fmt.Println("PX_E2E_CATALOG_OK")
	fmt.Println("PX_E2E_DEVICE_TOKEN_OK")
	fmt.Println("PX_E2E_DRIVER_EDGES_OK")
	return nil
}

func runCatalogCategorySuppliersE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/catalog/categories/ssmr-category/suppliers", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("category suppliers status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rows []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return fmt.Errorf("decode category suppliers: %w", err)
	}
	return nil
}

func runDeviceTokenE2E(ctx context.Context, client *http.Client, base, actorToken string) error {
	payload := map[string]string{"token": "ssmr-fcm-test-token", "platform": "android"}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/user/device-token", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+actorToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("device token status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

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
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("negotiation assign status %d body %s", status, string(respBody))
	}
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, "")
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
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/negotiate/resolve", resolveBody, supplierCookie, "ssmr-negotiate-"+proposeResp.ProposalID)
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

func runShopClosedE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, retailerToken, orderID string) error {
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
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("assign order status %d body %s", status, string(respBody))
	}
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, "")
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
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/shop-closed", shopBody, driverToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("shop closed report status %d body %s", status, string(respBody))
	}

	responseBody, _ := json.Marshal(map[string]string{
		"order_id": orderID,
		"response": "OPEN_NOW",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/retailer/shop-closed-response", responseBody, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("shop closed response status %d body %s", status, string(respBody))
	}
	return nil
}

func runPayloaderE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID string) error {
	loginBody, _ := json.Marshal(map[string]string{
		"phone": envOr("PAYLOAD_DEMO_PHONE", "+998901110022"),
		"pin":   envOr("PAYLOAD_DEMO_PIN", "33333333"),
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/payloader/login", loginBody, "", "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader login status %d body %s", status, string(respBody))
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &login); err != nil {
		return err
	}
	if login.Token == "" {
		return fmt.Errorf("payloader login missing token")
	}
	token := login.Token

	if status, _, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/trucks", nil, token, ""); err != nil {
		return fmt.Errorf("payloader trucks: %w", err)
	} else if status != http.StatusOK {
		return fmt.Errorf("payloader trucks status %d", status)
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/manifests?state=DRAFT&truck_id=veh_payload_1", nil, token, "")
	if err != nil {
		return fmt.Errorf("supplier manifests: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader manifests status %d body %s", status, string(respBody))
	}
	var manifests struct {
		Manifests []struct {
			ManifestID string `json:"manifest_id"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(respBody, &manifests); err != nil {
		return err
	}
	if len(manifests.Manifests) == 0 {
		return fmt.Errorf("payloader manifests empty")
	}
	manifestID := manifests.Manifests[0].ManifestID

	status, _, _, err = clientPost(ctx, client, base+"/v1/payloader/manifests/"+manifestID+"/start-loading", nil, token, "ssmr-start-"+manifestID)
	if err != nil {
		return fmt.Errorf("start-loading: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("start-loading status %d", status)
	}

	if err := assertDriverManifestGate(ctx, client, base, cfg, supplierID, manifestID, false); err != nil {
		return fmt.Errorf("driver manifest-gate pre-seal: %w", err)
	}

	if err := runPayloaderReassignE2E(ctx, client, base, token); err != nil {
		return err
	}
	fmt.Println("PX_E2E_PAYLOAD_REASSIGN_OK")

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/orders?vehicle_id=veh_payload_1&state=LOADED", nil, token, "")
	if err != nil {
		return fmt.Errorf("payloader orders: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader orders status %d body %s", status, string(respBody))
	}

	sealBody, _ := json.Marshal(map[string]any{
		"order_id":         "ord_payload_1",
		"terminal_id":      "veh_payload_1",
		"manifest_cleared": true,
	})
	status, _, _, err = clientPost(ctx, client, base+"/v1/payload/seal", sealBody, token, "ssmr-seal-ord_payload_1")
	if err != nil {
		return fmt.Errorf("payload seal order: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payload seal order status %d", status)
	}

	status, _, _, err = clientPost(ctx, client, base+"/v1/payloader/manifests/"+manifestID+"/seal", nil, token, "ssmr-seal-manifest-"+manifestID)
	if err != nil {
		return fmt.Errorf("manifest seal: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("manifest seal status %d", status)
	}
	fmt.Println("PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK")

	if err := assertDriverManifestGate(ctx, client, base, cfg, supplierID, manifestID, true); err != nil {
		return fmt.Errorf("driver manifest-gate post-seal: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DRIVER_GATE_OK")

	if err := assertDriverManifestDetail(ctx, client, base, cfg, supplierID, manifestID); err != nil {
		return fmt.Errorf("driver manifest detail: %w", err)
	}

	reassignBody, _ := json.Marshal(map[string]any{
		"order_ids":    []string{"ord_payload_1"},
		"new_route_id": "drv_payload_2",
	})
	status, _, _, err = clientPost(ctx, client, base+"/v1/fleet/reassign", reassignBody, token, "ssmr-fleet-reassign")
	if err != nil {
		return fmt.Errorf("fleet reassign: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("fleet reassign status %d", status)
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/user/notifications?limit=10", nil, token, "")
	if err != nil {
		return fmt.Errorf("payloader notifications: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader notifications status %d body %s", status, string(respBody))
	}
	if err := runDeviceTokenE2E(ctx, client, base, token); err != nil {
		return fmt.Errorf("payloader device token: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DEVICE_TOKEN_OK")
	return nil
}

func runPayloaderReassignE2E(ctx context.Context, client *http.Client, base, token string) error {
	recommendBody, _ := json.Marshal(map[string]string{"order_id": "ord_payload_2"})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/payloader/recommend-reassign", recommendBody, token, "ssmr-recommend-reassign")
	if err != nil {
		return fmt.Errorf("recommend-reassign: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("recommend-reassign status %d body %s", status, string(respBody))
	}
	var recommend struct {
		OrderID         string `json:"order_id"`
		Recommendations []struct {
			DriverID  string  `json:"driver_id"`
			VehicleID string  `json:"vehicle_id"`
			Score     float64 `json:"score"`
		} `json:"recommendations"`
	}
	if err := json.Unmarshal(respBody, &recommend); err != nil {
		return err
	}
	if recommend.OrderID != "ord_payload_2" || len(recommend.Recommendations) == 0 {
		return fmt.Errorf("expected recommendations for ord_payload_2, got %#v", recommend)
	}
	targetDriver := recommend.Recommendations[0].DriverID
	if targetDriver == "" {
		targetDriver = "drv_payload_2"
	}

	applyBody, _ := json.Marshal(map[string]any{
		"order_id":       "ord_payload_2",
		"to_manifest_id": "mf_payload_2",
		"to_driver_id":   targetDriver,
		"reason":         "ssmr_balance",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/payloader/reassign-order", applyBody, token, "ssmr-reassign-order")
	if err != nil {
		return fmt.Errorf("reassign-order: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("reassign-order status %d body %s", status, string(respBody))
	}
	var applied struct {
		Status       string `json:"status"`
		ToManifestID string `json:"to_manifest_id"`
	}
	if err := json.Unmarshal(respBody, &applied); err != nil {
		return err
	}
	if applied.Status != "order_reassigned" {
		return fmt.Errorf("unexpected reassign status %q body %s", applied.Status, string(respBody))
	}
	if applied.ToManifestID != "mf_payload_2" {
		return fmt.Errorf("expected to_manifest_id mf_payload_2, got %q", applied.ToManifestID)
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/manifest-exceptions?limit=5", nil, token, "")
	if err != nil {
		return fmt.Errorf("manifest-exceptions: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("manifest-exceptions status %d body %s", status, string(respBody))
	}
	return nil
}

func assertDriverManifestGate(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, manifestID string, wantCleared bool) error {
	driverID := envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
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

func assertDriverManifestDetail(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, manifestID string) error {
	driverID := envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
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
	if body.Manifest.State != "SEALED" {
		return fmt.Errorf("expected SEALED manifest, got %q", body.Manifest.State)
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

func assertSupplierPortalAPIs(ctx context.Context, client *http.Client, base, cookie string) error {
	checks := []string{
		base + "/v1/supplier/dashboard",
		base + "/v1/supplier/profile",
		base + "/v1/supplier/inventory",
		base + "/v1/supplier/earnings",
		base + "/v1/supplier/pricing/rules",
	}
	for _, url := range checks {
		status, body, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("GET %s status %d body %s", url, status, string(body))
		}
	}
	return nil
}

func runWarehouseDispatchLock(ctx context.Context, client *http.Client, base, cookie, orderID string) error {
	body, _ := json.Marshal(map[string]string{
		"entity_type": "ORDER",
		"entity_id":   orderID,
		"reason":      "ssmr-smoke-lock",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/dispatch-lock", body, cookie, "ssmr-lock-"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("dispatch lock acquire status %d body %s", status, string(respBody))
	}
	var lock struct {
		LockID string `json:"lock_id"`
	}
	_ = json.Unmarshal(respBody, &lock)
	if lock.LockID == "" {
		return fmt.Errorf("dispatch lock missing lock_id")
	}
	releaseURL := base + "/v1/warehouse/dispatch-lock?lock_id=" + lock.LockID
	status, respBody, _, err = clientDo(ctx, client, http.MethodDelete, releaseURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch lock release status %d body %s", status, string(respBody))
	}
	return nil
}

func runFactoryOps(ctx context.Context, client *http.Client, base, cookie string) error {
	if err := runFactoryInsightsE2E(ctx, client, base); err != nil {
		return err
	}
	if err := runFactoryManifestLifecycleE2E(ctx, client, base, cookie); err != nil {
		return err
	}
	if err := runFactorySupplyRequestE2E(ctx, client, base, cookie); err != nil {
		return err
	}
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifests status %d body %s", status, string(respBody))
	}
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/dispatch", []byte(`{"reason":"ssmr-smoke-a"}`), cookie, "ssmr-factory-dispatch-a")
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("factory dispatch status %d body %s", status, string(respBody))
	}
	var dispatchA struct {
		ManifestID string `json:"manifest_id"`
	}
	if err := json.Unmarshal(respBody, &dispatchA); err != nil {
		return fmt.Errorf("decode factory dispatch a: %w", err)
	}
	if dispatchA.ManifestID == "" {
		return fmt.Errorf("factory dispatch a missing manifest_id: %s", string(respBody))
	}
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/dispatch", []byte(`{"reason":"ssmr-smoke-b"}`), cookie, "ssmr-factory-dispatch-b")
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("factory dispatch b status %d body %s", status, string(respBody))
	}
	var dispatchB struct {
		ManifestID string `json:"manifest_id"`
	}
	if err := json.Unmarshal(respBody, &dispatchB); err != nil {
		return fmt.Errorf("decode factory dispatch b: %w", err)
	}
	if dispatchB.ManifestID == "" {
		return fmt.Errorf("factory dispatch b missing manifest_id: %s", string(respBody))
	}
	if err := runFactoryPayloadOverrideE2E(ctx, client, base, cookie, dispatchA.ManifestID, dispatchB.ManifestID); err != nil {
		return err
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifest-exceptions", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest-exceptions status %d body %s", status, string(respBody))
	}
	var exceptionsResp struct {
		Exceptions []json.RawMessage `json:"exceptions"`
	}
	if err := json.Unmarshal(respBody, &exceptionsResp); err != nil {
		return fmt.Errorf("decode factory manifest-exceptions: %w", err)
	}
	if len(exceptionsResp.Exceptions) == 0 {
		return fmt.Errorf("factory manifest-exceptions empty: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_MANIFEST_EXCEPTIONS_OK")

	createBody, _ := json.Marshal(map[string]any{
		"total_vu":   int64(32),
		"order_id":   "ssmr-factory-transfer",
		"driver_id":  "drv_factory_1",
		"vehicle_id": "veh_factory_1",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/transfers/create", createBody, cookie, "ssmr-factory-transfer-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("factory transfer create status %d body %s", status, string(respBody))
	}
	var createdTransfer struct {
		TransferID string `json:"transfer_id"`
	}
	if err := json.Unmarshal(respBody, &createdTransfer); err != nil {
		return fmt.Errorf("decode factory transfer create: %w", err)
	}
	if createdTransfer.TransferID == "" {
		return fmt.Errorf("factory transfer create missing transfer_id: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_TRANSFER_CREATE_OK")
	if err := runFactoryTransferTransitionE2E(ctx, client, base, cookie, createdTransfer.TransferID); err != nil {
		return err
	}
	if err := runFactoryLoadingBayE2E(ctx, client, base, cookie, createdTransfer.TransferID); err != nil {
		return err
	}
	return nil
}

func runFactoryInsightsE2E(ctx context.Context, client *http.Client, base string) error {
	phone := envOr("FACTORY_DEMO_PHONE", "+998901000099")
	pin := envOr("FACTORY_DEMO_PIN", "1234")
	loginBody, _ := json.Marshal(map[string]string{"phone": phone, "pin": pin})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/factory/login", loginBody, "", "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory login status %d body %s", status, string(respBody))
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return fmt.Errorf("decode factory login: %w", err)
	}
	if loginResp.Token == "" {
		return fmt.Errorf("factory login missing token: %s", string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/warehouse/replenishment/insights", nil, loginResp.Token, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory insights status %d body %s", status, string(respBody))
	}
	var insightsResp struct {
		Insights []json.RawMessage `json:"insights"`
	}
	if err := json.Unmarshal(respBody, &insightsResp); err != nil {
		return fmt.Errorf("decode factory insights: %w", err)
	}
	if len(insightsResp.Insights) == 0 {
		return fmt.Errorf("factory insights empty: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_INSIGHTS_OK")
	return nil
}

func runFactoryManifestLifecycleE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	const manifestID = "mf_factory_1"
	transitions := []struct {
		action   string
		wantFrom string
		wantTo   string
	}{
		{action: "start-loading", wantFrom: "DRAFT", wantTo: "LOADING"},
		{action: "seal", wantFrom: "LOADING", wantTo: "SEALED"},
		{action: "dispatch", wantFrom: "SEALED", wantTo: "DISPATCHED"},
		{action: "complete", wantFrom: "DISPATCHED", wantTo: "COMPLETED"},
	}

	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests/"+manifestID, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest detail status %d body %s", status, string(respBody))
	}
	var detailResp struct {
		Manifest struct {
			State string `json:"state"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory manifest detail: %w", err)
	}
	state := strings.ToUpper(strings.TrimSpace(detailResp.Manifest.State))
	if state == "" {
		return fmt.Errorf("factory manifest detail missing state: %s", string(respBody))
	}

	startIdx := 0
	for i, step := range transitions {
		if state == step.wantFrom {
			startIdx = i
			break
		}
		if state == step.wantTo && i < len(transitions)-1 {
			startIdx = i + 1
			break
		}
	}
	if state == transitions[len(transitions)-1].wantTo {
		fmt.Println("PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK")
		return nil
	}

	for _, step := range transitions[startIdx:] {
		body := []byte(`{"reason":"ssmr-smoke"}`)
		url := base + "/v1/factory/manifests/" + manifestID + "/" + step.action
		status, respBody, _, err = clientPost(ctx, client, url, body, cookie, "ssmr-factory-"+step.action+"-"+manifestID)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("factory manifest %s status %d body %s", step.action, status, string(respBody))
		}
		var transitionResp struct {
			State string `json:"state"`
		}
		if err := json.Unmarshal(respBody, &transitionResp); err != nil {
			return fmt.Errorf("decode factory manifest %s: %w", step.action, err)
		}
		if strings.ToUpper(strings.TrimSpace(transitionResp.State)) != step.wantTo {
			return fmt.Errorf("factory manifest %s expected state %s got %s body %s", step.action, step.wantTo, transitionResp.State, string(respBody))
		}
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/factory/staff/stf_factory_1", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory staff detail status %d body %s", status, string(respBody))
	}
	var staffResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &staffResp); err != nil {
		return fmt.Errorf("decode factory staff detail: %w", err)
	}
	if staffResp.ID != "stf_factory_1" {
		return fmt.Errorf("factory staff detail unexpected id %q body %s", staffResp.ID, string(respBody))
	}

	fmt.Println("PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK")
	return nil
}

func runFactorySupplyRequestE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/supply-requests", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply-requests status %d body %s", status, string(respBody))
	}
	var listResp struct {
		Requests []struct {
			RequestID string `json:"request_id"`
			State     string `json:"state"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return fmt.Errorf("decode factory supply-requests: %w", err)
	}
	if len(listResp.Requests) == 0 {
		return fmt.Errorf("factory supply-requests empty: %s", string(respBody))
	}
	requestID := listResp.Requests[0].RequestID
	state := strings.ToUpper(strings.TrimSpace(listResp.Requests[0].State))
	if state == "ACKNOWLEDGED" {
		fmt.Println("PX_E2E_FACTORY_SUPPLY_REQUEST_OK")
		return nil
	}
	if state != "SUBMITTED" {
		return fmt.Errorf("factory supply-request unexpected state %s", state)
	}
	patchBody, _ := json.Marshal(map[string]string{"action": "ACKNOWLEDGE"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+requestID, patchBody, cookie, "ssmr-factory-supply-"+requestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply-request transition status %d body %s", status, string(respBody))
	}
	var transitionResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &transitionResp); err != nil {
		return fmt.Errorf("decode factory supply-request transition: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(transitionResp.State)) != "ACKNOWLEDGED" {
		return fmt.Errorf("factory supply-request expected ACKNOWLEDGED got %s body %s", transitionResp.State, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_SUPPLY_REQUEST_OK")
	return nil
}

func runFactoryPayloadOverrideE2E(ctx context.Context, client *http.Client, base, cookie, manifestA, manifestB string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests/"+manifestA, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest detail %s status %d body %s", manifestA, status, string(respBody))
	}
	var detailResp struct {
		Transfers []struct {
			TransferID string `json:"transfer_id"`
		} `json:"transfers"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory manifest detail %s: %w", manifestA, err)
	}
	if len(detailResp.Transfers) == 0 {
		return fmt.Errorf("factory manifest %s has no transfers for rebalance", manifestA)
	}
	rebalanceBody, _ := json.Marshal(map[string]any{
		"source_manifest_id": manifestA,
		"target_manifest_id": manifestB,
		"transfer_ids":       []string{detailResp.Transfers[0].TransferID},
		"reason":             "ssmr-smoke",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/manifests/rebalance", rebalanceBody, cookie, "ssmr-factory-rebalance")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory payload rebalance status %d body %s", status, string(respBody))
	}
	var rebalanceResp struct {
		TransfersMoved int `json:"transfers_moved"`
	}
	if err := json.Unmarshal(respBody, &rebalanceResp); err != nil {
		return fmt.Errorf("decode factory payload rebalance: %w", err)
	}
	if rebalanceResp.TransfersMoved < 1 {
		return fmt.Errorf("factory payload rebalance moved zero transfers: %s", string(respBody))
	}
	for _, manifestID := range []string{manifestA, manifestB} {
		body := []byte(`{"reason":"ssmr-smoke"}`)
		url := base + "/v1/factory/manifests/" + manifestID + "/start-loading"
		status, respBody, _, err := clientPost(ctx, client, url, body, cookie, "ssmr-factory-loading-"+manifestID)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("factory start-loading %s status %d body %s", manifestID, status, string(respBody))
		}
	}
	fmt.Println("PX_E2E_FACTORY_PAYLOAD_OVERRIDE_OK")
	return nil
}

func runFactoryLoadingBayE2E(ctx context.Context, client *http.Client, base, cookie, approvedTransferID string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/transfers?states=APPROVED,LOADING&limit=50", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory loading bay list status %d body %s", status, string(respBody))
	}
	var listResp struct {
		Transfers []struct {
			TransferID string `json:"transfer_id"`
			State      string `json:"state"`
		} `json:"transfers"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return fmt.Errorf("decode factory loading bay list: %w", err)
	}
	if listResp.Total < 1 || len(listResp.Transfers) == 0 {
		return fmt.Errorf("factory loading bay list empty: %s", string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/factory/transfers/"+approvedTransferID, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory transfer detail status %d body %s", status, string(respBody))
	}
	var detailResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory transfer detail: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(detailResp.State)) != "APPROVED" {
		return fmt.Errorf("factory transfer detail expected APPROVED got %s body %s", detailResp.State, string(respBody))
	}

	loadingBody, _ := json.Marshal(map[string]string{"target_state": "LOADING"})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/transfers/"+approvedTransferID+"/transition", loadingBody, cookie, "ssmr-factory-loading-bay-transition")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory loading bay transition status %d body %s", status, string(respBody))
	}

	dispatchBody, _ := json.Marshal(map[string]any{
		"transfer_ids": []string{approvedTransferID},
		"reason":       "ssmr-loading-bay-dispatch",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/dispatch", dispatchBody, cookie, "ssmr-factory-loading-bay-dispatch")
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("factory loading bay dispatch status %d body %s", status, string(respBody))
	}
	var dispatchResp struct {
		ManifestID       string `json:"manifest_id"`
		ManifestsCreated int    `json:"manifests_created"`
	}
	if err := json.Unmarshal(respBody, &dispatchResp); err != nil {
		return fmt.Errorf("decode factory loading bay dispatch: %w", err)
	}
	if dispatchResp.ManifestID == "" {
		return fmt.Errorf("factory loading bay dispatch missing manifest_id: %s", string(respBody))
	}
	if dispatchResp.ManifestsCreated < 1 {
		return fmt.Errorf("factory loading bay dispatch missing manifests_created: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_LOADING_BAY_OK")
	return nil
}

func runFactoryTransferTransitionE2E(ctx context.Context, client *http.Client, base, cookie, transferID string) error {
	transitionBody, _ := json.Marshal(map[string]string{"target_state": "APPROVED"})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/factory/transfers/"+transferID+"/transition", transitionBody, cookie, "ssmr-factory-transfer-transition-"+transferID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory transfer transition status %d body %s", status, string(respBody))
	}
	var transitionResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &transitionResp); err != nil {
		return fmt.Errorf("decode factory transfer transition: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(transitionResp.State)) != "APPROVED" {
		return fmt.Errorf("factory transfer expected APPROVED got %s body %s", transitionResp.State, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_TRANSFER_TRANSITION_OK")
	return nil
}

func runUnifiedCheckout(ctx context.Context, client *http.Client, base, retailerToken, orderID string, cfg *bootstrap.Config) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"order_id":     orderID,
		"amount_minor": int64(100000),
		"currency":     cfg.SeedSupplierCurrency,
		"gateway":      "GLOBAL_PAY",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/checkout/unified", body, retailerToken, "ssmr-checkout-"+orderID)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", fmt.Errorf("checkout status %d body %s", status, string(respBody))
	}
	var resp struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}
	if resp.SessionID == "" {
		return "", fmt.Errorf("checkout missing session_id: %s", string(respBody))
	}
	return resp.SessionID, nil
}

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

func clientPostAuthorized(ctx context.Context, client *http.Client, url string, body []byte, authorization, idempotencyKey string) (int, []byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	return resp.StatusCode, respBody, resp.Header, nil
}

func runWarehouseOrderMutationE2E(ctx context.Context, client *http.Client, base, cookie, orderID string) error {
	delayBody, _ := json.Marshal(map[string]string{"reason": "ssmr-warehouse-delay"})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/ops/orders/"+orderID+"/delay", delayBody, cookie, "ssmr-wh-order-delay")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse order delay status %d body %s", status, string(respBody))
	}
	var delayResp struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &delayResp); err != nil {
		return fmt.Errorf("decode warehouse order delay: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(delayResp.Status)) != "DELAYED" {
		return fmt.Errorf("warehouse order delay expected DELAYED got %s body %s", delayResp.Status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_ORDER_MUTATION_OK")
	return nil
}

func runWarehouseTransferActionsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	emergencyBody, _ := json.Marshal(map[string]any{
		"total_volume_vu": 18.0,
		"notes":           "ssmr-emergency-transfer",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/transfers/emergency", emergencyBody, cookie, "ssmr-wh-emergency-transfer")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse emergency transfer status %d body %s", status, string(respBody))
	}
	var emergencyResp struct {
		TransferID string `json:"transfer_id"`
		State      string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &emergencyResp); err != nil {
		return fmt.Errorf("decode warehouse emergency transfer: %w", err)
	}
	if emergencyResp.TransferID == "" || strings.ToUpper(emergencyResp.State) != "APPROVED" {
		return fmt.Errorf("warehouse emergency transfer invalid: %s", string(respBody))
	}

	forceBody, _ := json.Marshal(map[string]any{
		"total_volume_vu": 22.0,
		"notes":           "ssmr-force-receive",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/transfers/force-receive", forceBody, cookie, "ssmr-wh-force-receive")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse force receive status %d body %s", status, string(respBody))
	}
	var forceResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &forceResp); err != nil {
		return fmt.Errorf("decode warehouse force receive: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(forceResp.State)) != "RECEIVED" {
		return fmt.Errorf("warehouse force receive expected RECEIVED got %s body %s", forceResp.State, string(respBody))
	}

	const receiveID = "ssmr-wh-transfer-receive"
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/transfers/"+receiveID+"/receive", nil, cookie, "ssmr-wh-receive-transfer")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse receive transfer status %d body %s", status, string(respBody))
	}
	var receiveResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &receiveResp); err != nil {
		return fmt.Errorf("decode warehouse receive transfer: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(receiveResp.State)) != "RECEIVED" {
		return fmt.Errorf("warehouse receive transfer expected RECEIVED got %s body %s", receiveResp.State, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_TRANSFER_ACTIONS_OK")
	return nil
}

func runWarehouseDispatchPreview(ctx context.Context, client *http.Client, base, supplierCookie string) error {
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/ops/dispatch/preview", []byte(`{}`), supplierCookie, "ssmr-dispatch-preview")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch preview status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "preview_ready") {
		return fmt.Errorf("dispatch preview unexpected body %s", string(respBody))
	}
	return nil
}

func ensureSupplierSession(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) (string, string, error) {
	phone := envOr("SSMR_SMOKE_SUPPLIER_PHONE", "+998901000001")
	password := envOr("SSMR_SMOKE_SUPPLIER_PASSWORD", "SmokeTest!234")

	loginBody, _ := json.Marshal(map[string]string{
		"phone":    phone,
		"password": password,
	})
	status, respBody, hdrs, err := clientPostRetry(ctx, client, base+"/v1/auth/supplier/login", loginBody, "", "")
	if err != nil {
		return "", "", err
	}
	if status == http.StatusOK {
		return supplierSessionFromResponse(respBody, hdrs, cfg)
	}

	registerBody, _ := json.Marshal(map[string]any{
		"phone": phone,
		"account": map[string]any{
			"legalName":   envOr("SSMR_SMOKE_SUPPLIER_NAME", "SSMR Smoke Supplier"),
			"contactName": "Smoke Admin",
			"email":       "smoke-supplier@pegasusx.local",
			"password":    password,
			"country":     cfg.SeedSupplierCountry,
		},
		"location": map[string]any{
			"warehouse": map[string]any{
				"name":    "SSMR Warehouse",
				"address": "Tashkent SSMR",
				"lat":     cfg.DeliveryZoneCenterLat,
				"lng":     cfg.DeliveryZoneCenterLng,
			},
			"sameAsWarehouse": true,
		},
		"business": map[string]any{
			"taxId":             "SSMR-TAX",
			"companyRegNumber":  "SSMR-REG",
			"fleetVehicleCount": 2,
			"fleetMaxVU":        100,
			"factoryCount":      1,
		},
		"categories": []string{"GENERAL"},
	})
	status, respBody, hdrs, err = clientPostRetry(ctx, client, base+"/v1/auth/supplier/register", registerBody, "", "ssmr-supplier-register")
	if err != nil {
		return "", "", err
	}
	if status == http.StatusConflict || status == http.StatusTooManyRequests {
		status, respBody, hdrs, err = clientPostRetry(ctx, client, base+"/v1/auth/supplier/login", loginBody, "", "")
		if err != nil {
			return "", "", err
		}
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", "", fmt.Errorf("supplier session status %d body %s", status, string(respBody))
	}
	return supplierSessionFromResponse(respBody, hdrs, cfg)
}

func supplierSessionFromResponse(respBody []byte, hdrs http.Header, cfg *bootstrap.Config) (string, string, error) {
	cookie := sessionCookie(hdrs)
	if cookie == "" {
		return "", "", fmt.Errorf("supplier session missing cookie")
	}
	var resp struct {
		SupplierID string `json:"supplier_id"`
	}
	_ = json.Unmarshal(respBody, &resp)
	sid := strings.TrimSpace(resp.SupplierID)
	if sid == "" {
		sid = supplierIDFromJWT(cookie, cfg.JWTSecret)
	}
	if sid == "" {
		return "", "", fmt.Errorf("supplier session missing supplier_id")
	}
	return sid, cookie, nil
}

func demoWarehouseID() string {
	if id := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID")); id != "" {
		return id
	}
	if id := strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID")); id != "" {
		return id
	}
	return "wh-demo-1"
}

func demoFactoryID() string {
	if id := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID")); id != "" {
		return id
	}
	return "factory-demo-1"
}

func putSupplierTopology(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	body, _ := json.Marshal(map[string]any{
		"warehouses": []map[string]any{
			{
				"warehouse_id":       demoWarehouseID(),
				"name":               "SSMR Central WH",
				"lat":                cfg.DeliveryZoneCenterLat,
				"lng":                cfg.DeliveryZoneCenterLng,
				"coverage_radius_km": cfg.DeliveryZoneRadiusKm,
				"is_active":          true,
				"is_on_shift":        true,
			},
		},
		"factories": []map[string]any{
			{
				"factory_id": demoFactoryID(),
				"name":       "SSMR Factory",
				"lat":        cfg.DeliveryZoneCenterLat + 0.01,
				"lng":        cfg.DeliveryZoneCenterLng + 0.01,
				"is_active":  true,
			},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/topology", body, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("topology status %d body %s", status, string(respBody))
	}
	return nil
}

func registerRetailer(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) (string, string, error) {
	return registerRetailerWithPhone(ctx, client, base, cfg, envOr("SSMR_SMOKE_RETAILER_PHONE", "+998901000099"))
}

func registerRetailerWithPhone(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, phone string) (string, string, error) {
	body, _ := json.Marshal(map[string]any{
		"phone": phone,
		"name":  "SSMR Retailer",
		"lat":   cfg.DeliveryZoneCenterLat,
		"lng":   cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err := clientPostRetry(ctx, client, base+"/v1/auth/retailer/register", body, "", "")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", "", fmt.Errorf("retailer register status %d body %s", status, string(respBody))
	}
	var resp struct {
		RetailerID string `json:"retailer_id"`
		H3Cell     string `json:"h3_cell"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", err
	}
	if resp.RetailerID == "" || len(resp.H3Cell) != 15 {
		return "", "", fmt.Errorf("retailer register invalid response: %s", string(respBody))
	}
	return resp.RetailerID, resp.H3Cell, nil
}

func createOrder(ctx context.Context, client *http.Client, base, bearer string, cfg *bootstrap.Config, h3Cell string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": "SSMR-SKU-1", "quantity": 2, "unit_price_minor": 50000},
		},
		"h3_cell": h3Cell,
		"lat":     cfg.DeliveryZoneCenterLat,
		"lng":     cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/order/create", body, bearer, "")
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("order create status %d body %s", status, string(respBody))
	}
	var resp struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}
	if resp.OrderID == "" {
		return "", fmt.Errorf("order create missing order_id")
	}
	return resp.OrderID, nil
}

func assertRetailerTracking(ctx context.Context, client *http.Client, base, retailerToken, orderID string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/tracking", nil, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("tracking status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), orderID) {
		return fmt.Errorf("tracking response missing order_id %s", orderID)
	}
	return nil
}

func clientGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, url, nil, "", "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET %s status %d body %s", url, status, string(body))
	}
	return body, nil
}

func clientPost(ctx context.Context, client *http.Client, url string, body []byte, bearer, idempotencyKey string) (int, []byte, http.Header, error) {
	return clientDo(ctx, client, http.MethodPost, url, body, bearer, idempotencyKey)
}

func clientPostRetry(ctx context.Context, client *http.Client, url string, body []byte, bearer, idempotencyKey string) (int, []byte, http.Header, error) {
	return clientDoRetry(ctx, client, http.MethodPost, url, body, bearer, idempotencyKey)
}

func clientDoRetry(ctx context.Context, client *http.Client, method, url string, body []byte, bearerOrCookie, idempotencyKey string) (int, []byte, http.Header, error) {
	var lastStatus int
	var lastBody []byte
	var lastHdrs http.Header
	for attempt := 0; attempt < 12; attempt++ {
		status, respBody, hdrs, err := clientDo(ctx, client, method, url, body, bearerOrCookie, idempotencyKey)
		if err != nil {
			return 0, nil, nil, err
		}
		if status != http.StatusTooManyRequests {
			return status, respBody, hdrs, nil
		}
		lastStatus, lastBody, lastHdrs = status, respBody, hdrs
		wait := retryAfterSeconds(hdrs, 2+attempt)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastStatus, lastBody, lastHdrs, ctx.Err()
		case <-timer.C:
		}
	}
	return lastStatus, lastBody, lastHdrs, nil
}

func retryAfterSeconds(hdrs http.Header, fallback int) time.Duration {
	if hdrs == nil {
		return time.Duration(fallback) * time.Second
	}
	raw := strings.TrimSpace(hdrs.Get("Retry-After"))
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return time.Duration(fallback) * time.Second
}

func clientDo(ctx context.Context, client *http.Client, method, url string, body []byte, bearerOrCookie, idempotencyKey string) (int, []byte, http.Header, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, nil, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if strings.HasPrefix(bearerOrCookie, auth.CookieName+"=") || strings.Contains(bearerOrCookie, "=") {
		req.Header.Set("Cookie", bearerOrCookie)
	} else if bearerOrCookie != "" {
		req.Header.Set("Authorization", "Bearer "+bearerOrCookie)
	}
	if secret := strings.TrimSpace(envOr("LOAD_BOOTSTRAP_SECRET", "")); secret != "" {
		req.Header.Set("X-PegasusX-Load-Bootstrap", secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	return resp.StatusCode, respBody, resp.Header, nil
}

func sessionCookie(hdrs http.Header) string {
	for _, c := range hdrs["Set-Cookie"] {
		if strings.HasPrefix(c, auth.CookieName+"=") {
			part := strings.SplitN(c, ";", 2)[0]
			return part
		}
	}
	return ""
}

func supplierIDFromJWT(cookieHeader, secret string) string {
	token := strings.TrimPrefix(cookieHeader, auth.CookieName+"=")
	claims, err := auth.Parse(token, secret)
	if err != nil {
		return ""
	}
	return claims.SupplierID
}

func runClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=DRIVER&platform=ios&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode client policy: %w", err)
	}
	if resp.MinimumVersion == "" {
		return fmt.Errorf("client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_CLIENT_POLICY_OK")
	return nil
}
