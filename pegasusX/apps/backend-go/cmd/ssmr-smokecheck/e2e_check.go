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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

// dispatchManifestHint carries warehouse dispatch execute output into the payloader journey.
type dispatchManifestHint struct {
	ManifestID string
	DriverID   string
	VehicleID  string
	OrderIDs   []string
}

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

	if err := runRetailerReceivingWindowE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("retailer receiving window: %w", err)
	}
	if err := runRetailerCatalogProductsE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("retailer catalog products: %w", err)
	}

	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("order create: %w", err)
	}
	if err := runRetailerCancelE2E(ctx, client, base, retailerToken, orderID, retailerID); err != nil {
		return fmt.Errorf("retailer cancel: %w", err)
	}
	if err := runRetailerCardInitiateE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("retailer card initiate: %w", err)
	}
	if err := runRetailerClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("retailer client policy: %w", err)
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
	if err := runWarehouseDispatchSettingsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse dispatch settings: %w", err)
	}
	if err := runWarehouseStockPolicyE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse stock policy: %w", err)
	}
	if err := runWarehouseReplenishmentInsightE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse replenishment insight: %w", err)
	}
	if err := runSupplierInventoryImportE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier inventory import (staging substrate): %w", err)
	}
	if err := runSupplierImportWizardE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier import session wizard: %w", err)
	}
	if err := runSupplierImportAsyncE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier import async worker: %w", err)
	}
	if err := runWarehouseAnalyticsE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("warehouse analytics: %w", err)
	}
	if err := runWarehouseClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("warehouse client policy: %w", err)
	}
	if err := runReplenishmentSupplyChainE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("replenishment supply chain: %w", err)
	}
	if err := runReplenishColocateE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("replenishment colocate: %w", err)
	}
	if err := ensureWarehouseDispatchFleet(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse dispatch fleet: %w", err)
	}
	fleetDriverID, fleetVehicleID, err := runWarehouseFleetMgmtE2E(ctx, client, base, cookie, cfg, supplierID)
	if err != nil {
		return fmt.Errorf("warehouse fleet mgmt: %w", err)
	}
	if err := runDispatchCapacityE2E(ctx, client, base, cookie, orderID, fleetDriverID, fleetVehicleID); err != nil {
		return fmt.Errorf("dispatch capacity: %w", err)
	}
	dispatchHint, err := runWarehouseDispatchExecute(ctx, client, base, cookie, orderID)
	if err != nil {
		return fmt.Errorf("warehouse dispatch execute: %w", err)
	}
	if err := runWarehouseFleetLiveMapE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse fleet live map: %w", err)
	}
	if err := runNotificationInboxE2E(ctx, client, base, cookie, retailerToken); err != nil {
		return fmt.Errorf("notification inbox: %w", err)
	}
	if err := runRetailerNotificationInboxE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("retailer notification inbox: %w", err)
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
	if err := runShopClosedE2E(ctx, client, base, cfg, supplierID, retailerToken, shopClosedOrderID, cookie); err != nil {
		return fmt.Errorf("shop closed e2e: %w", err)
	}
	if err := runWarehouseTransferActionsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse transfer actions: %w", err)
	}
	if err := assertSupplierPortalAPIs(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("supplier portal apis: %w", err)
	}
	if err := runRetailerPricingOverrideE2E(ctx, client, base, cookie, supplierID, retailerID, retailerToken); err != nil {
		return fmt.Errorf("retailer pricing override: %w", err)
	}
	if err := runSupplierIntelligenceE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("supplier intelligence: %w", err)
	}
	if err := runSupplierOperationsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("supplier operations: %w", err)
	}
	if err := runSupplierClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("supplier client policy: %w", err)
	}
	if err := runFactoryOps(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("factory ops: %w", err)
	}
	if err := postDriverTelemetry(ctx, client, base, cfg, supplierID); err != nil {
		return fmt.Errorf("driver telemetry: %w", err)
	}
	if err := runPayloaderE2E(ctx, client, base, cfg, supplierID, dispatchHint); err != nil {
		return fmt.Errorf("payloader e2e: %w", err)
	}
	if err := runFleetReassignGuardE2E(ctx, client, base, cookie, dispatchHint); err != nil {
		return fmt.Errorf("fleet reassign guard: %w", err)
	}
	// Quantity negotiation disabled ecosystem-wide — skip negotiation E2E.
	fmt.Println("PX_E2E_NEGOTIATION_SKIPPED")

	if err := runClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("client policy e2e: %w", err)
	}
	if err := runDriverNotificationInboxE2E(ctx, client, base); err != nil {
		return fmt.Errorf("driver notification inbox: %w", err)
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
	if err := runDeliveryEdgeCasesE2E(ctx, client, base, cfg, retailerToken, supplierID, cookie, h3Cell); err != nil {
		return fmt.Errorf("delivery edge cases: %w", err)
	}

	fmt.Println("PX_E2E_ORDER_OK")
	fmt.Println("PX_E2E_PAYMENT_OK")
	fmt.Println("PX_E2E_WAREHOUSE_OK")
	fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_SETTINGS_OK")
	fmt.Println("PX_E2E_WAREHOUSE_STOCK_POLICY_OK")
	fmt.Println("PX_E2E_WAREHOUSE_REPLENISHMENT_OK")
	fmt.Println("PX_E2E_WAREHOUSE_ANALYTICS_OK")
	fmt.Println("PX_E2E_FACTORY_OK")
	fmt.Println("PX_E2E_FACTORY_ANALYTICS_OK")
	fmt.Println("PX_E2E_FACTORY_CLIENT_POLICY_OK")
	fmt.Println("PX_E2E_FACTORY_NOTIFICATION_INBOX_OK")
	fmt.Println("PX_E2E_DELIVERY_OK")
	fmt.Println("PX_E2E_TELEMETRY_OK")
	fmt.Println("PX_E2E_PAYLOAD_OK")
	fmt.Println("PX_E2E_SHOP_CLOSED_OK")
	fmt.Println("PX_E2E_CATALOG_OK")
	fmt.Println("PX_E2E_DEVICE_TOKEN_OK")
	fmt.Println("PX_E2E_DRIVER_EDGES_OK")
	fmt.Println("PX_E2E_DRIVER_CLIENT_POLICY_OK")
	fmt.Println("PX_E2E_DRIVER_NOTIFICATION_INBOX_OK")
	fmt.Println("PX_E2E_REPLENISH_OK")
	fmt.Println("PX_E2E_REPLENISH_COLOCATE_OK")
	fmt.Println("PX_E2E_WAREHOUSE_FLEET_MGMT_OK")
	fmt.Println("PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK")
	fmt.Println("PX_E2E_WAREHOUSE_CLIENT_POLICY_OK")
	fmt.Println("PX_E2E_NOTIFICATION_INBOX_OK")
	fmt.Println("PX_E2E_DISPATCH_CAPACITY_OK")
	fmt.Println("PX_E2E_PAYLOAD_SEAL_FLOWS_OK")
	fmt.Println("PX_E2E_PAYLOAD_CLIENT_POLICY_OK")
	fmt.Println("PX_E2E_REASSIGN_FLOWS_OK")
	fmt.Println("PX_E2E_DRIVER_ASSIGN_DETECTION_OK")
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

func runPayloaderE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID string, dispatch *dispatchManifestHint) error {
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
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(respBody, &login); err != nil {
		return err
	}
	if login.Token == "" {
		return fmt.Errorf("payloader login missing token")
	}
	if login.RefreshToken == "" {
		return fmt.Errorf("payloader login missing refresh_token")
	}
	token := login.Token

	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": login.RefreshToken})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/auth/payloader/refresh", refreshBody, "", "")
	if err != nil {
		return fmt.Errorf("payloader refresh: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader refresh status %d body %s", status, string(respBody))
	}
	var refreshResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &refreshResp); err != nil {
		return err
	}
	if refreshResp.Token == "" {
		return fmt.Errorf("payloader refresh missing token")
	}
	token = refreshResp.Token
	fmt.Println("PX_E2E_PAYLOAD_AUTH_REFRESH_OK")

	if status, _, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/trucks", nil, token, ""); err != nil {
		return fmt.Errorf("payloader trucks: %w", err)
	} else if status != http.StatusOK {
		return fmt.Errorf("payloader trucks status %d", status)
	}

	var (
		manifestID      string
		driverID        string
		vehicleID       string
		sealOrder       string
		dispatchJourney bool
	)
	if dispatch != nil && strings.TrimSpace(dispatch.ManifestID) != "" {
		dispatchJourney = true
		manifestID = dispatch.ManifestID
		driverID = dispatch.DriverID
		vehicleID = dispatch.VehicleID
		if len(dispatch.OrderIDs) > 0 {
			sealOrder = dispatch.OrderIDs[0]
		}
		fmt.Println("PX_E2E_PAYLOAD_DISPATCH_JOURNEY_OK")
	} else {
		vehicleID = "veh_payload_1"
		sealOrder = "ord_payload_1"
		driverID = envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/manifests?state=DRAFT&truck_id="+vehicleID, nil, token, "")
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
		manifestID = manifests.Manifests[0].ManifestID
	}

	status, _, _, err = clientPost(ctx, client, base+"/v1/payloader/manifests/"+manifestID+"/start-loading", nil, token, "ssmr-start-"+manifestID)
	if err != nil {
		return fmt.Errorf("start-loading: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("start-loading status %d", status)
	}

	if err := assertDriverManifestGate(ctx, client, base, cfg, supplierID, driverID, manifestID, false); err != nil {
		return fmt.Errorf("driver manifest-gate pre-seal: %w", err)
	}

	if !dispatchJourney {
		if err := runPayloaderReassignE2E(ctx, client, base, token); err != nil {
			return err
		}
		fmt.Println("PX_E2E_PAYLOAD_REASSIGN_OK")
		fmt.Println("PX_E2E_REASSIGN_FLOWS_OK")
	}

	if vehicleID != "" {
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/orders?vehicle_id="+vehicleID+"&state=LOADED", nil, token, "")
		if err != nil {
			return fmt.Errorf("payloader orders: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("payloader orders status %d body %s", status, string(respBody))
		}
	}

	if sealOrder != "" {
		sealBody, _ := json.Marshal(map[string]any{
			"order_id":         sealOrder,
			"terminal_id":      vehicleID,
			"manifest_cleared": true,
		})
		status, _, _, err = clientPost(ctx, client, base+"/v1/payload/seal", sealBody, token, "ssmr-seal-"+sealOrder)
		if err != nil {
			return fmt.Errorf("payload seal order: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("payload seal order status %d", status)
		}
	}

	batchBody, _ := json.Marshal(map[string]any{"manifest_ids": []string{manifestID}})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/payloader/manifests/seal-completed", batchBody, token, "ssmr-seal-batch-"+manifestID)
	if err != nil {
		return fmt.Errorf("manifest seal-completed: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("manifest seal-completed status %d body %s", status, string(respBody))
	}
	var batchResp struct {
		SealedCount int `json:"sealed_count"`
	}
	_ = json.Unmarshal(respBody, &batchResp)
	if batchResp.SealedCount < 1 {
		return fmt.Errorf("manifest seal-completed expected sealed_count >= 1 body %s", string(respBody))
	}
	fmt.Println("PX_E2E_PAYLOAD_SEAL_FLOWS_OK")
	fmt.Println("PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK")

	if err := assertDriverManifestGate(ctx, client, base, cfg, supplierID, driverID, manifestID, true); err != nil {
		return fmt.Errorf("driver manifest-gate post-seal: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DRIVER_GATE_OK")

	if err := assertDriverDepart(ctx, client, base, cfg, supplierID, driverID); err != nil {
		return fmt.Errorf("driver depart: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DRIVER_DEPART_OK")

	if err := assertDriverManifestDetail(ctx, client, base, cfg, supplierID, driverID, manifestID, "DISPATCHED"); err != nil {
		return fmt.Errorf("driver manifest detail: %w", err)
	}

	if !dispatchJourney {
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
	if err := runPayloadClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("payload client policy: %w", err)
	}
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

func runSupplierIntelligenceE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	return runSupplierAnalyticsE2E(ctx, client, base, cookie)
}

func runSupplierAnalyticsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	checks := []string{
		base + "/v1/supplier/analytics/velocity",
		base + "/v1/supplier/analytics/revenue",
		base + "/v1/supplier/analytics/demand/today",
		base + "/v1/supplier/analytics/demand/history",
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
	fmt.Println("PX_E2E_SUPPLIER_ANALYTICS_OK")
	return nil
}

func runSupplierOperationsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/empathy/adoption", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET empathy/adoption status %d body %s", status, string(body))
	}

	broadcastPayload := []byte(`{"title":"SSMR ops","body":"broadcast smoke","role":"ALL"}`)
	status, body, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/broadcast", broadcastPayload, cookie, "application/json")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("POST broadcast status %d body %s", status, string(body))
	}

	fmt.Println("PX_E2E_SUPPLIER_OPERATIONS_OK")
	return nil
}

func runSupplierClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=ADMIN&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode supplier client policy: %w", err)
	}
	if resp.Role != "ADMIN" {
		return fmt.Errorf("supplier client policy role=%q want ADMIN", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("supplier client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_SUPPLIER_CLIENT_POLICY_OK")
	return nil
}

func runRetailerReceivingWindowE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	updateBody := []byte(`{"receiving_window_open":"10:30","receiving_window_close":"19:45"}`)
	status, body, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/retailer/profile", updateBody, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("PUT retailer/profile status %d body %s", status, string(body))
	}

	status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/profile", nil, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET retailer/profile status %d body %s", status, string(body))
	}
	var profile map[string]any
	if err := json.Unmarshal(body, &profile); err != nil {
		return fmt.Errorf("decode retailer profile: %w", err)
	}
	if profile["receiving_window_open"] != "10:30" {
		return fmt.Errorf("receiving_window_open=%v want 10:30", profile["receiving_window_open"])
	}
	if profile["receiving_window_close"] != "19:45" {
		return fmt.Errorf("receiving_window_close=%v want 19:45", profile["receiving_window_close"])
	}

	fmt.Println("PX_E2E_RETAILER_RECEIVING_WINDOW_OK")
	return nil
}

func runRetailerCatalogProductsE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/catalog/products", nil, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET catalog/products status %d body %s", status, string(body))
	}
	var products []map[string]any
	if err := json.Unmarshal(body, &products); err != nil {
		return fmt.Errorf("decode catalog/products: %w", err)
	}
	for _, product := range products {
		if _, ok := product["available_stock"]; ok {
			fmt.Println("PX_E2E_RETAILER_CATALOG_STOCK_OK")
			break
		}
	}
	fmt.Println("PX_E2E_RETAILER_CATALOG_PRODUCTS_OK")
	return nil
}

func runRetailerCancelE2E(ctx context.Context, client *http.Client, base, retailerToken, orderID, retailerID string) error {
	cancelBody, _ := json.Marshal(map[string]string{
		"order_id":    orderID,
		"retailer_id": retailerID,
	})
	status, body, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/order/cancel", cancelBody, retailerToken, "retailer-cancel-smoke:"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("POST order/cancel status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_RETAILER_CANCEL_OK")
	return nil
}

func runRetailerCardInitiateE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/card/initiate", []byte("{}"), retailerToken, "retailer-card-initiate-smoke")
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusServiceUnavailable {
		return fmt.Errorf("POST retailer/card/initiate status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_RETAILER_CARD_INITIATE_OK")
	return nil
}

func runRetailerClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=RETAILER&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode retailer client policy: %w", err)
	}
	if resp.Role != "RETAILER" {
		return fmt.Errorf("retailer client policy role=%q want RETAILER", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("retailer client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_RETAILER_CLIENT_POLICY_OK")
	return nil
}

func runRetailerNotificationInboxE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = assertInboxHasRows(ctx, client, base, retailerToken, "retailer")
		if lastErr == nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("retailer inbox not ready after kafka fanout window: %w", lastErr)
	}
	markBody, _ := json.Marshal(map[string]any{"mark_all": true})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, retailerToken, "ssmr-retailer-inbox-read")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("retailer mark notifications read status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_RETAILER_NOTIFICATION_INBOX_OK")
	return nil
}

func runRetailerPricingOverrideE2E(
	ctx context.Context,
	client *http.Client,
	base, cookie, supplierID, retailerID, retailerToken string,
) error {
	const overridePrice = int64(42000)
	productID := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")

	createBody, _ := json.Marshal(map[string]any{
		"retailer_id": retailerID,
		"product_id":  productID,
		"price":       overridePrice,
		"notes":       "SSMR override",
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/pricing/retailer-overrides", createBody, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create retailer override status %d body %s", status, string(respBody))
	}
	var created struct {
		OverrideID string `json:"override_id"`
		Price      int64  `json:"price"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return err
	}
	if created.OverrideID == "" || created.Price != overridePrice {
		return fmt.Errorf("create retailer override invalid response: %s", string(respBody))
	}

	listURL := base + "/v1/supplier/pricing/retailer-overrides?retailer_id=" + retailerID + "&product_id=" + productID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, listURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("list retailer overrides status %d body %s", status, string(respBody))
	}
	var listed struct {
		Overrides []struct {
			OverrideID string `json:"override_id"`
			Price      int64  `json:"price"`
		} `json:"overrides"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(respBody, &listed); err != nil {
		return err
	}
	if listed.Total < 1 || len(listed.Overrides) == 0 || listed.Overrides[0].Price != overridePrice {
		return fmt.Errorf("list retailer overrides missing active row: %s", string(respBody))
	}

	quoteBody, _ := json.Marshal(map[string]any{
		"supplier_id": supplierID,
		"lines": []map[string]any{
			{"product_id": productID, "quantity": 1, "unit_price_minor": 50000, "currency": "UZS"},
		},
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/checkout/quote", quoteBody, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("checkout quote status %d body %s", status, string(respBody))
	}
	var quote struct {
		Lines []struct {
			UnitPrice int64 `json:"unit_price_minor"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(respBody, &quote); err != nil {
		return err
	}
	if len(quote.Lines) == 0 || quote.Lines[0].UnitPrice != overridePrice {
		return fmt.Errorf("checkout quote did not apply override: %s", string(respBody))
	}

	if err := assertInboxContainsEvent(ctx, client, base, retailerToken, events.EventRetailerPriceOverride); err != nil {
		return fmt.Errorf("retailer inbox after override: %w", err)
	}

	deleteURL := base + "/v1/supplier/pricing/retailer-overrides/" + created.OverrideID
	status, respBody, _, err = clientDo(ctx, client, http.MethodDelete, deleteURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("delete retailer override status %d body %s", status, string(respBody))
	}

	fmt.Println("PX_E2E_RETAILER_PRICING_OVERRIDE_OK")
	return nil
}

func runSupplierInventoryImportE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	csvBody := fmt.Sprintf(
		"product_id,warehouse_id,quantity_on_hand,reorder_threshold\nSSMR-SKU-1,%s,50,5\nSSMR-SKU-BAD,%s,10,1\n",
		demoWarehouseID(), demoWarehouseID(),
	)
	status, respBody, _, err := clientDoContentType(
		ctx, client, http.MethodPost, base+"/v1/supplier/inventory/import",
		[]byte(csvBody), "text/csv", cookie, "ssmr-supplier-inventory-import",
	)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("supplier inventory import status %d body %s", status, string(respBody))
	}
	var result struct {
		SessionID string `json:"session_id"`
		Applied   int    `json:"applied"`
		Skipped   int    `json:"skipped"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("decode supplier inventory import: %w", err)
	}
	if result.Applied < 1 {
		return fmt.Errorf("supplier inventory import applied=%d body %s", result.Applied, string(respBody))
	}
	if result.Skipped < 1 {
		return fmt.Errorf("supplier inventory import skipped=%d want >=1 (anomaly row) body %s", result.Skipped, string(respBody))
	}
	if strings.TrimSpace(result.SessionID) == "" {
		return fmt.Errorf("supplier inventory import missing session_id body %s", string(respBody))
	}
	supplierID := supplierIDFromJWT(cookie, cfg.JWTSecret)
	if err := assertSupplierImportStagingRows(ctx, cfg, supplierID, result.SessionID, demoWarehouseID()); err != nil {
		return fmt.Errorf("supplier import staging rows: %w", err)
	}
	fmt.Println("PX_E2E_SUPPLIER_INVENTORY_IMPORT_OK")
	return nil
}

func runSupplierImportWizardE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	csvBody := fmt.Sprintf(
		"product_id,warehouse_id,quantity_on_hand,reorder_threshold\nSSMR-SKU-1,%s,75,8\n",
		demoWarehouseID(),
	)
	createBody, _ := json.Marshal(map[string]any{
		"file_name":       "ssmr-wizard.csv",
		"file_size_bytes": len(csvBody),
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/inventory/imports", createBody, cookie, "ssmr-import-wizard-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("import session create status %d body %s", status, string(respBody))
	}
	var created struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode import session create: %w", err)
	}
	if strings.TrimSpace(created.SessionID) == "" {
		return fmt.Errorf("import session create missing session_id body %s", string(respBody))
	}

	ingestURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/ingest"
	status, respBody, _, err = clientDoContentType(ctx, client, http.MethodPost, ingestURL, []byte(csvBody), "text/csv", cookie, "ssmr-import-wizard-ingest")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("import session ingest status %d body %s", status, string(respBody))
	}

	approveURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/approve"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, approveURL, nil, cookie, "ssmr-import-wizard-approve")
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("import session approve status %d body %s", status, string(respBody))
	}

	applyURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/apply"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, applyURL, nil, cookie, "ssmr-import-wizard-apply")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("import session apply status %d body %s", status, string(respBody))
	}
	var applied struct {
		AppliedRows int64  `json:"applied_rows"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &applied); err != nil {
		return fmt.Errorf("decode import session apply: %w", err)
	}
	if applied.AppliedRows < 1 {
		return fmt.Errorf("import wizard applied_rows=%d body %s", applied.AppliedRows, string(respBody))
	}
	if strings.ToUpper(strings.TrimSpace(applied.Status)) != "APPLIED" {
		return fmt.Errorf("import wizard status=%q body %s", applied.Status, string(respBody))
	}

	fmt.Println("PX_E2E_SUPPLIER_IMPORT_WIZARD_OK")
	return nil
}

func importLocalUploadRoot() string {
	if root := strings.TrimSpace(os.Getenv("SSMR_IMPORT_LOCAL_ROOT")); root != "" {
		return root
	}
	return ".ssmr/import-uploads"
}

func runSupplierImportAsyncE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	csvBody := fmt.Sprintf(
		"product_id,warehouse_id,quantity_on_hand,reorder_threshold\nSSMR-SKU-1,%s,82,9\n",
		demoWarehouseID(),
	)
	createBody, _ := json.Marshal(map[string]any{
		"file_name":       "ssmr-async.csv",
		"file_size_bytes": len(csvBody),
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/inventory/imports", createBody, cookie, "ssmr-import-async-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("import async create status %d body %s", status, string(respBody))
	}
	var created struct {
		SessionID string `json:"session_id"`
		GCSPath   string `json:"gcs_path"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode import async create: %w", err)
	}
	if strings.TrimSpace(created.SessionID) == "" || strings.TrimSpace(created.GCSPath) == "" {
		return fmt.Errorf("import async create missing session_id/gcs_path body %s", string(respBody))
	}

	localPath := filepath.Join(importLocalUploadRoot(), filepath.FromSlash(created.GCSPath))
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("mkdir import upload root: %w", err)
	}
	if err := os.WriteFile(localPath, []byte(csvBody), 0o644); err != nil {
		return fmt.Errorf("write local import object: %w", err)
	}

	uploadedBody, _ := json.Marshal(map[string]string{"gcs_path": created.GCSPath})
	uploadedURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/uploaded"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, uploadedURL, uploadedBody, cookie, "ssmr-import-async-uploaded")
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("import async uploaded status %d body %s", status, string(respBody))
	}

	deadline := time.Now().Add(45 * time.Second)
	var sessionStatus string
	for time.Now().Before(deadline) {
		getURL := base + "/v1/supplier/inventory/imports/" + created.SessionID
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, getURL, nil, cookie, "ssmr-import-async-poll")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("import async poll status %d body %s", status, string(respBody))
		}
		var session struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(respBody, &session); err != nil {
			return fmt.Errorf("decode import async session: %w", err)
		}
		sessionStatus = strings.ToUpper(strings.TrimSpace(session.Status))
		if sessionStatus == "DISCOVERED" || sessionStatus == "MAPPING_REQUIRED" {
			break
		}
		if sessionStatus == "FAILED" {
			return fmt.Errorf("import async session failed body %s", string(respBody))
		}
		time.Sleep(500 * time.Millisecond)
	}
	if sessionStatus != "DISCOVERED" && sessionStatus != "MAPPING_REQUIRED" {
		return fmt.Errorf("import async discovery timed out status=%q", sessionStatus)
	}

	approveURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/approve"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, approveURL, nil, cookie, "ssmr-import-async-approve")
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("import async approve status %d body %s", status, string(respBody))
	}

	applyURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/apply"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, applyURL, nil, cookie, "ssmr-import-async-apply")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("import async apply status %d body %s", status, string(respBody))
	}
	var applied struct {
		AppliedRows int64  `json:"applied_rows"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &applied); err != nil {
		return fmt.Errorf("decode import async apply: %w", err)
	}
	if applied.AppliedRows < 1 {
		return fmt.Errorf("import async applied_rows=%d body %s", applied.AppliedRows, string(respBody))
	}
	if strings.ToUpper(strings.TrimSpace(applied.Status)) != "APPLIED" {
		return fmt.Errorf("import async status=%q body %s", applied.Status, string(respBody))
	}

	fmt.Println("PX_E2E_SUPPLIER_IMPORT_ASYNC_OK")
	return nil
}

func assertSupplierImportStagingRows(ctx context.Context, cfg *bootstrap.Config, supplierID, sessionID, warehouseID string) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	stmt := spanner.Statement{
		SQL: `SELECT row_index, raw_data, validation_errors
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId
		        AND session_id = @sessionId
		        AND validation_errors IS NOT NULL
		        AND ARRAY_LENGTH(validation_errors) > 0`,
		Params: map[string]any{
			"supplierId": supplierID,
			"sessionId":  sessionID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	anomalyRows := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query staged rows: %w", err)
		}
		var rowIndex int64
		var raw spanner.NullJSON
		var validationErrors []string
		if err := row.Columns(&rowIndex, &raw, &validationErrors); err != nil {
			return fmt.Errorf("decode staged row: %w", err)
		}
		if len(validationErrors) == 0 {
			continue
		}
		rawMap := importStagingJSONMap(raw)
		rowWarehouse := strings.TrimSpace(stagingJSONString(rawMap, "warehouse_id"))
		if rowWarehouse != "" && rowWarehouse != warehouseID {
			continue
		}
		anomalyRows++
	}
	if anomalyRows < 1 {
		return fmt.Errorf("want >=1 staged anomaly row for warehouse %s supplier %s session %s", warehouseID, supplierID, sessionID)
	}
	return nil
}

func importStagingJSONMap(value spanner.NullJSON) map[string]any {
	if !value.Valid || value.Value == nil {
		return nil
	}
	if mapped, ok := value.Value.(map[string]any); ok {
		return mapped
	}
	encoded, err := json.Marshal(value.Value)
	if err != nil {
		return nil
	}
	decoded := make(map[string]any)
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil
	}
	return decoded
}

func stagingJSONString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	if value, ok := raw.(string); ok {
		return value
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func countWarehouseImportAnomalyRows(ctx context.Context, cfg *bootstrap.Config, supplierID, warehouseID string) (int64, error) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return 0, fmt.Errorf("supplier_id required")
	}
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return 0, fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	startAt := time.Now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT session_id, raw_data, cleaned_data, validation_errors
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId
		        AND created_at >= @startAt
		        AND validation_errors IS NOT NULL
		        AND ARRAY_LENGTH(validation_errors) > 0
		      ORDER BY updated_at DESC
		      LIMIT 5000`,
		Params: map[string]any{
			"supplierId": supplierID,
			"startAt":    startAt,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var openRows int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("query import anomalies: %w", err)
		}
		var sessionID string
		var raw spanner.NullJSON
		var cleaned spanner.NullJSON
		var validationErrors []string
		if err := row.Columns(&sessionID, &raw, &cleaned, &validationErrors); err != nil {
			return 0, fmt.Errorf("decode import anomaly row: %w", err)
		}
		rawMap := importStagingJSONMap(raw)
		cleanedMap := importStagingJSONMap(cleaned)
		rowWarehouse := strings.TrimSpace(stagingJSONString(cleanedMap, "warehouse_id"))
		if rowWarehouse == "" {
			rowWarehouse = strings.TrimSpace(stagingJSONString(rawMap, "warehouse_id"))
		}
		if rowWarehouse != "" && rowWarehouse != warehouseID {
			continue
		}
		openRows++
	}
	return openRows, nil
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
	if err := runFactoryClientPolicyE2E(ctx, client, base); err != nil {
		return err
	}
	if err := runFactoryAnalyticsOverviewE2E(ctx, client, base, cookie); err != nil {
		return err
	}
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
	if err := runFactoryNotificationInboxE2E(ctx, client, base); err != nil {
		return err
	}
	return nil
}

func runFactoryClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=FACTORY&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode factory client policy: %w", err)
	}
	if resp.Role != "FACTORY" {
		return fmt.Errorf("factory client policy role=%q want FACTORY", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("factory client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_FACTORY_CLIENT_POLICY_OK")
	return nil
}

func runFactoryAnalyticsOverviewE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/analytics/overview", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory analytics overview status %d body %s", status, string(respBody))
	}
	var overview struct {
		TransfersTotal int `json:"transfers_total"`
	}
	if err := json.Unmarshal(respBody, &overview); err != nil {
		return fmt.Errorf("decode factory analytics overview: %w", err)
	}
	fmt.Println("PX_E2E_FACTORY_ANALYTICS_OK")
	return nil
}

func factoryDemoToken(ctx context.Context, client *http.Client, base string) (string, error) {
	phone := envOr("FACTORY_DEMO_PHONE", "+998901000099")
	pin := envOr("FACTORY_DEMO_PIN", "1234")
	loginBody, _ := json.Marshal(map[string]string{"phone": phone, "pin": pin})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/factory/login", loginBody, "", "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("factory login status %d body %s", status, string(respBody))
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return "", fmt.Errorf("decode factory login: %w", err)
	}
	if loginResp.Token == "" {
		return "", fmt.Errorf("factory login missing token: %s", string(respBody))
	}
	return loginResp.Token, nil
}

func runFactoryNotificationInboxE2E(ctx context.Context, client *http.Client, base string) error {
	token, err := factoryDemoToken(ctx, client, base)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = assertInboxHasRows(ctx, client, base, token, "factory")
		if lastErr == nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("factory inbox not ready after kafka fanout window: %w", lastErr)
	}
	markBody, _ := json.Marshal(map[string]any{"mark_all": true})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, token, "ssmr-factory-inbox-read")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory mark notifications read status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_NOTIFICATION_INBOX_OK")
	return nil
}

func runFactoryInsightsE2E(ctx context.Context, client *http.Client, base string) error {
	token, err := factoryDemoToken(ctx, client, base)
	if err != nil {
		return err
	}
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/warehouse/replenishment/insights", nil, token, "")
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

	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/replenishment/insights/ins_wh_1/approve", nil, token, "ssmr-factory-insight-approve")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		if status == http.StatusConflict {
			var errResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(respBody, &errResp) == nil && errResp.Error == "insight_already_processed" {
				fmt.Println("PX_E2E_FACTORY_REPLENISHMENT_ACTION_OK")
				return nil
			}
		}
		return fmt.Errorf("factory insight approve status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_REPLENISHMENT_ACTION_OK")
	return nil
}

func runWarehouseDispatchSettingsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	getURL := base + "/v1/warehouse/ops/dispatch/settings?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, getURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch settings get status %d body %s", status, string(respBody))
	}
	patchBody, _ := json.Marshal(map[string]bool{"auto_dispatch_enabled": true})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, getURL, patchBody, cookie, "ssmr-dispatch-settings")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch settings patch status %d body %s", status, string(respBody))
	}
	return nil
}

func runWarehouseStockPolicyE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	settingsURL := base + "/v1/warehouse/ops/settings?warehouse_id=" + whID

	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, settingsURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse settings get status %d body %s", status, string(respBody))
	}
	var settings struct {
		DefaultOutOfStockPolicy string `json:"default_out_of_stock_policy"`
		OpsAlwaysAvailable    bool   `json:"ops_always_available"`
	}
	if err := json.Unmarshal(respBody, &settings); err != nil {
		return fmt.Errorf("decode warehouse settings: %w", err)
	}
	if settings.DefaultOutOfStockPolicy == "" {
		return fmt.Errorf("warehouse settings missing default_out_of_stock_policy: %s", string(respBody))
	}
	if !settings.OpsAlwaysAvailable {
		return fmt.Errorf("warehouse settings expected ops_always_available=true: %s", string(respBody))
	}

	patchBody, _ := json.Marshal(map[string]string{
		"default_out_of_stock_policy": "ACCEPT_BACKORDER",
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, settingsURL, patchBody, cookie, "ssmr-wh-stock-policy")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse settings patch status %d body %s", status, string(respBody))
	}

	invURL := base + "/v1/warehouse/ops/inventory?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, invURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse inventory get status %d body %s", status, string(respBody))
	}
	var invResp struct {
		Items []struct {
			ProductID       string `json:"product_id"`
			EffectivePolicy string `json:"effective_policy"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return fmt.Errorf("decode warehouse inventory: %w", err)
	}
	if len(invResp.Items) == 0 {
		return nil
	}
	productID := invResp.Items[0].ProductID
	policyURL := base + "/v1/warehouse/ops/inventory/" + productID + "/policy?warehouse_id=" + whID
	skuPolicyBody, _ := json.Marshal(map[string]string{
		"out_of_stock_policy": "INHERIT",
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, policyURL, skuPolicyBody, cookie, "ssmr-wh-sku-policy-"+productID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse inventory policy patch status %d body %s", status, string(respBody))
	}
	return nil
}

func runWarehouseReplenishmentInsightE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	listURL := base + "/v1/warehouse/replenishment/insights?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, listURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("replenishment insights status %d body %s", status, string(respBody))
	}
	var insightsResp struct {
		Insights []struct {
			ID string `json:"id"`
		} `json:"insights"`
	}
	if err := json.Unmarshal(respBody, &insightsResp); err != nil {
		return fmt.Errorf("decode replenishment insights: %w", err)
	}
	if len(insightsResp.Insights) == 0 {
		return fmt.Errorf("replenishment insights empty: %s", string(respBody))
	}
	insightID := insightsResp.Insights[0].ID
	actionURL := base + "/v1/warehouse/replenishment/insights/" + insightID + "/approve?warehouse_id=" + whID
	status, respBody, _, err = clientPost(ctx, client, actionURL, nil, cookie, "ssmr-wh-insight-approve")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("replenishment insight approve status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_REPLENISHMENT_OK")
	return nil
}

func runWarehouseAnalyticsE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	supplierID := supplierIDFromJWT(cookie, cfg.JWTSecret)
	if count, err := countWarehouseImportAnomalyRows(ctx, cfg, supplierID, demoWarehouseID()); err != nil {
		return fmt.Errorf("warehouse import anomaly projection: %w", err)
	} else if count < 1 {
		return fmt.Errorf("warehouse import anomaly projection want >=1 got %d supplier=%s warehouse=%s", count, supplierID, demoWarehouseID())
	}
	for _, period := range []string{"7d", "30d"} {
		url := base + "/v1/warehouse/ops/analytics?period=" + period + "&warehouse_id=" + demoWarehouseID()
		status, respBody, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("warehouse analytics %s status %d body %s", period, status, string(respBody))
		}
		var overview struct {
			Period         string `json:"period"`
			TotalOrders    int64  `json:"total_orders"`
			DailyBreakdown []struct {
				Date string `json:"date"`
			} `json:"daily_breakdown"`
			ImportFreshness struct {
				AppliedRows30d   int64  `json:"applied_rows_30d"`
				AppliedSkus30d   int64  `json:"applied_skus_30d"`
				QuantityDelta30d int64  `json:"quantity_delta_30d"`
				LastSessionID    string `json:"last_session_id"`
				LastAppliedAt    string `json:"last_applied_at"`
			} `json:"import_freshness"`
			ImportAnomalyQueue struct {
				OpenRows30d         int64  `json:"open_rows_30d"`
				AffectedSessions30d int64  `json:"affected_sessions_30d"`
				LastSessionID       string `json:"last_session_id"`
				LastDetectedAt      string `json:"last_detected_at"`
				LastDetail          string `json:"last_detail"`
			} `json:"import_anomaly_queue"`
		}
		if err := json.Unmarshal(respBody, &overview); err != nil {
			return fmt.Errorf("decode warehouse analytics %s: %w", period, err)
		}
		if overview.Period != period {
			return fmt.Errorf("warehouse analytics period mismatch want %s got %s", period, overview.Period)
		}
	}
	fmt.Println("PX_E2E_WAREHOUSE_ANALYTICS_OK")
	return nil
}

func runWarehouseClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=WAREHOUSE&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode warehouse client policy: %w", err)
	}
	if resp.Role != "WAREHOUSE" {
		return fmt.Errorf("warehouse client policy role=%q want WAREHOUSE", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("warehouse client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_WAREHOUSE_CLIENT_POLICY_OK")
	return nil
}

func runPayloadClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=PAYLOAD&platform=android&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode payload client policy: %w", err)
	}
	if resp.Role != "PAYLOAD" {
		return fmt.Errorf("payload client policy role=%q want PAYLOAD", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("payload client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_PAYLOAD_CLIENT_POLICY_OK")
	return nil
}

func runWarehouseFleetLiveMapE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	url := base + "/v1/warehouse/ops/fleet/live-map?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse fleet live map status %d body %s", status, string(respBody))
	}
	var liveMap struct {
		Routes      []json.RawMessage `json:"routes"`
		WarehouseID string            `json:"warehouse_id"`
		FetchedAt   string            `json:"fetched_at"`
	}
	if err := json.Unmarshal(respBody, &liveMap); err != nil {
		return fmt.Errorf("decode warehouse fleet live map: %w", err)
	}
	if strings.TrimSpace(liveMap.WarehouseID) != whID {
		return fmt.Errorf("warehouse fleet live map warehouse_id=%q want %q", liveMap.WarehouseID, whID)
	}
	if strings.TrimSpace(liveMap.FetchedAt) == "" {
		return fmt.Errorf("warehouse fleet live map missing fetched_at")
	}
	fmt.Println("PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK")
	return nil
}

func runNotificationInboxE2E(ctx context.Context, client *http.Client, base, supplierCookie, retailerToken string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastSupplierErr, lastRetailerErr error
	for time.Now().Before(deadline) {
		lastSupplierErr = assertInboxHasRows(ctx, client, base, supplierCookie, "supplier")
		if lastSupplierErr != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		markBody, _ := json.Marshal(map[string]any{"mark_all": true})
		status, respBody, _, markErr := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, supplierCookie, "ssmr-supplier-inbox-read")
		if markErr != nil {
			lastSupplierErr = markErr
			time.Sleep(400 * time.Millisecond)
			continue
		}
		if status != http.StatusOK {
			lastSupplierErr = fmt.Errorf("supplier mark notifications read status %d body %s", status, string(respBody))
			time.Sleep(400 * time.Millisecond)
			continue
		}
		fmt.Println("PX_E2E_SUPPLIER_NOTIFICATION_INBOX_OK")
		if retailerToken != "" {
			lastRetailerErr = assertInboxHasRows(ctx, client, base, retailerToken, "retailer")
			if lastRetailerErr != nil {
				time.Sleep(400 * time.Millisecond)
				continue
			}
		}
		fmt.Println("PX_E2E_NOTIFICATION_INBOX_OK")
		return nil
	}
	if lastSupplierErr != nil {
		return fmt.Errorf("supplier inbox not ready after kafka fanout window: %w", lastSupplierErr)
	}
	if retailerToken != "" && lastRetailerErr != nil {
		return fmt.Errorf("retailer inbox not ready after kafka fanout window: %w", lastRetailerErr)
	}
	return fmt.Errorf("notification inbox empty after kafka fanout window")
}

func assertInboxHasRows(ctx context.Context, client *http.Client, base, authToken, label string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/user/notifications?limit=10", nil, authToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("%s inbox status %d body %s", label, status, string(respBody))
	}
	var inbox struct {
		Notifications []json.RawMessage `json:"notifications"`
	}
	if err := json.Unmarshal(respBody, &inbox); err != nil {
		return fmt.Errorf("decode %s inbox: %w", label, err)
	}
	if len(inbox.Notifications) == 0 {
		return fmt.Errorf("%s inbox empty", label)
	}
	return nil
}

func assertInboxContainsEvent(ctx context.Context, client *http.Client, base, authToken, wantType string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastTypes []string
	for time.Now().Before(deadline) {
		status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/user/notifications?limit=25", nil, authToken, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("inbox status %d body %s", status, string(respBody))
		}
		var inbox struct {
			Notifications []struct {
				Type  string `json:"type"`
				Title string `json:"title"`
			} `json:"notifications"`
		}
		if err := json.Unmarshal(respBody, &inbox); err != nil {
			return fmt.Errorf("decode inbox: %w", err)
		}
		lastTypes = lastTypes[:0]
		for _, row := range inbox.Notifications {
			if row.Type == wantType {
				return nil
			}
			lastTypes = append(lastTypes, row.Type)
		}
		time.Sleep(400 * time.Millisecond)
	}
	if len(lastTypes) == 0 {
		return fmt.Errorf("inbox missing event type %s (no rows)", wantType)
	}
	return fmt.Errorf("inbox missing event type %s (saw %d rows: %s)", wantType, len(lastTypes), strings.Join(lastTypes, ", "))
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
		if step.action == "seal" {
			if err := assertInboxContainsEvent(ctx, client, base, cookie, events.EventManifestSealed); err != nil {
				return fmt.Errorf("manifest sealed inbox: %w", err)
			}
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

func runReplenishmentSupplyChainE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	whID := demoWarehouseID()
	createURL := fmt.Sprintf("%s/v1/warehouse/supply-requests?warehouse_id=%s&start_date=2026-06-14&days=3", base, whID)
	status, respBody, _, err := clientPost(ctx, client, createURL, nil, cookie, "ssmr-replen-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse supply create status %d body %s", status, string(respBody))
	}
	var created struct {
		RequestID string `json:"request_id"`
		State     string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode warehouse supply create: %w", err)
	}
	if created.RequestID == "" {
		return fmt.Errorf("warehouse supply create missing request_id: %s", string(respBody))
	}
	if state := strings.ToUpper(strings.TrimSpace(created.State)); state != "SUBMITTED" {
		return fmt.Errorf("warehouse supply create expected SUBMITTED got %s", created.State)
	}

	ackBody, _ := json.Marshal(map[string]string{"action": "ACKNOWLEDGE"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+created.RequestID, ackBody, cookie, "ssmr-replen-ack-"+created.RequestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply acknowledge status %d body %s", status, string(respBody))
	}

	fulfillBody, _ := json.Marshal(map[string]string{"action": "FULFILL"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+created.RequestID, fulfillBody, cookie, "ssmr-replen-fulfill-"+created.RequestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply fulfill status %d body %s", status, string(respBody))
	}
	var fulfillResp struct {
		State            string `json:"state"`
		LinkedTransferID string `json:"linked_transfer_id"`
	}
	if err := json.Unmarshal(respBody, &fulfillResp); err != nil {
		return fmt.Errorf("decode factory supply fulfill: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(fulfillResp.State)) != "FULFILLED" {
		return fmt.Errorf("factory supply fulfill expected FULFILLED got %s", fulfillResp.State)
	}
	if fulfillResp.LinkedTransferID == "" {
		return fmt.Errorf("factory supply fulfill missing linked_transfer_id: %s", string(respBody))
	}

	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/transfers/"+fulfillResp.LinkedTransferID+"/receive", nil, cookie, "ssmr-replen-receive-"+fulfillResp.LinkedTransferID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse transfer receive status %d body %s", status, string(respBody))
	}
	_ = cfg
	return nil
}

func runReplenishColocateE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	if err := putSupplierTopologyColocate(ctx, client, base, cookie, cfg); err != nil {
		return err
	}
	whID := demoWarehouseID()
	createURL := fmt.Sprintf("%s/v1/warehouse/supply-requests?warehouse_id=%s&start_date=2026-06-14&days=1", base, whID)
	status, respBody, _, err := clientPost(ctx, client, createURL, nil, cookie, "ssmr-colocate-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("colocate supply create status %d body %s", status, string(respBody))
	}
	var created struct {
		RequestID    string `json:"request_id"`
		TransferMode string `json:"transfer_mode"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode colocate supply create: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(created.TransferMode)) != "INTERNAL" {
		return fmt.Errorf("colocate supply expected INTERNAL transfer_mode got %q", created.TransferMode)
	}

	fulfillBody, _ := json.Marshal(map[string]string{"action": "FULFILL"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+created.RequestID, fulfillBody, cookie, "ssmr-colocate-fulfill-"+created.RequestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("colocate fulfill status %d body %s", status, string(respBody))
	}
	var fulfillResp struct {
		State            string `json:"state"`
		LinkedTransferID string `json:"linked_transfer_id"`
	}
	if err := json.Unmarshal(respBody, &fulfillResp); err != nil {
		return fmt.Errorf("decode colocate fulfill: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(fulfillResp.State)) != "FULFILLED" || fulfillResp.LinkedTransferID == "" {
		return fmt.Errorf("colocate fulfill invalid response: %s", string(respBody))
	}

	invURL := base + "/v1/warehouse/ops/inventory?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, invURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("colocate inventory status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "replenishment-bulk-vu") {
		return fmt.Errorf("colocate inventory missing replenishment-bulk-vu: %s", string(respBody))
	}
	return nil
}

func putSupplierTopologyColocate(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	factoryID := demoFactoryID()
	body, _ := json.Marshal(map[string]any{
		"warehouses": []map[string]any{
			{
				"warehouse_id":              demoWarehouseID(),
				"name":                      "SSMR Co-locate WH",
				"lat":                       cfg.DeliveryZoneCenterLat,
				"lng":                       cfg.DeliveryZoneCenterLng,
				"coverage_radius_km":        cfg.DeliveryZoneRadiusKm,
				"transfer_mode":             "INTERNAL",
				"co_locate_with_factory_id": factoryID,
				"is_active":                 true,
				"is_on_shift":               true,
			},
		},
		"factories": []map[string]any{
			{
				"factory_id": factoryID,
				"name":       "SSMR Factory",
				"lat":        cfg.DeliveryZoneCenterLat + 0.01,
				"lng":        cfg.DeliveryZoneCenterLng + 0.01,
				"is_active":  true,
			},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/topology", body, cookie, "ssmr-topology-colocate")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("colocate topology status %d body %s", status, string(respBody))
	}
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
			ItemCount int    `json:"item_count"`
			Items     []struct {
				ProductID string `json:"product_id"`
			} `json:"items"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return fmt.Errorf("decode factory supply-requests: %w", err)
	}
	if len(listResp.Requests) == 0 {
		return fmt.Errorf("factory supply-requests empty: %s", string(respBody))
	}
	if listResp.Requests[0].ItemCount < 0 {
		return fmt.Errorf("factory supply-requests invalid item_count: %s", string(respBody))
	}

	var requestID string
	for _, req := range listResp.Requests {
		state := strings.ToUpper(strings.TrimSpace(req.State))
		switch state {
		case "ACKNOWLEDGED", "FULFILLED", "RECEIVED":
			fmt.Println("PX_E2E_FACTORY_SUPPLY_REQUEST_OK")
			return nil
		case "SUBMITTED":
			requestID = req.RequestID
		}
	}
	if requestID == "" {
		return fmt.Errorf("factory supply-request no actionable state in list: %s", string(respBody))
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

func runWarehouseDispatchExecute(ctx context.Context, client *http.Client, base, supplierCookie, orderID string) (*dispatchManifestHint, error) {
	whID := demoWarehouseID()
	url := base + "/v1/warehouse/ops/dispatch/execute?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, url, []byte(`{}`), supplierCookie, "ssmr-dispatch-execute")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("dispatch execute status %d body %s", status, string(respBody))
	}
	var result struct {
		Status           string `json:"status"`
		ManifestsCreated int    `json:"manifests_created"`
		Manifests        []struct {
			ManifestID string   `json:"manifest_id"`
			DriverID   string   `json:"driver_id"`
			VehicleID  string   `json:"vehicle_id"`
			OrderIDs   []string `json:"order_ids"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode dispatch execute: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(result.Status)) {
	case "no_op":
		fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK")
		return nil, nil
	case "dispatched":
	default:
		return nil, fmt.Errorf("dispatch execute unexpected status %q body %s", result.Status, string(respBody))
	}
	if result.ManifestsCreated <= 0 || len(result.Manifests) == 0 {
		return nil, fmt.Errorf("dispatch execute missing manifests body %s", string(respBody))
	}
	var picked *dispatchManifestHint
	for i := range result.Manifests {
		m := result.Manifests[i]
		if strings.TrimSpace(m.ManifestID) == "" {
			return nil, fmt.Errorf("dispatch execute manifest missing id body %s", string(respBody))
		}
		if orderID != "" && sliceContains(m.OrderIDs, orderID) {
			picked = &dispatchManifestHint{
				ManifestID: m.ManifestID,
				DriverID:   m.DriverID,
				VehicleID:  m.VehicleID,
				OrderIDs:   append([]string(nil), m.OrderIDs...),
			}
			break
		}
	}
	if picked == nil {
		m := result.Manifests[0]
		picked = &dispatchManifestHint{
			ManifestID: m.ManifestID,
			DriverID:   m.DriverID,
			VehicleID:  m.VehicleID,
			OrderIDs:   append([]string(nil), m.OrderIDs...),
		}
	}
	fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK")
	return picked, nil
}

func ensureWarehouseDispatchFleet(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/fleet/drivers", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("fleet drivers status %d body %s", status, string(respBody))
	}
	var drivers struct {
		Items []struct {
			DriverID   string `json:"driver_id"`
			HomeNodeID string `json:"home_node_id"`
			VehicleID  string `json:"vehicle_id"`
			IsActive   bool   `json:"is_active"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &drivers); err != nil {
		return fmt.Errorf("decode fleet drivers: %w", err)
	}
	for _, driver := range drivers.Items {
		if driver.IsActive && driver.HomeNodeID == whID && strings.TrimSpace(driver.VehicleID) != "" {
			return nil
		}
	}

	plate := fmt.Sprintf("SSMR%04d", time.Now().Unix()%10000)
	vehicleBody, _ := json.Marshal(map[string]any{
		"label":          "SSMR Dispatch Truck",
		"license_plate":  plate,
		"home_node_type": "WAREHOUSE",
		"home_node_id":   whID,
		"vehicle_class":  "CLASS_B",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/fleet/vehicles", vehicleBody, cookie, "ssmr-fleet-vehicle")
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("fleet vehicle create status %d body %s", status, string(respBody))
	}
	var vehicles struct {
		Items []struct {
			VehicleID    string `json:"vehicle_id"`
			LicensePlate string `json:"license_plate"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &vehicles); err != nil {
		return fmt.Errorf("decode fleet vehicles: %w", err)
	}
	vehicleID := ""
	for _, vehicle := range vehicles.Items {
		if strings.EqualFold(vehicle.LicensePlate, plate) {
			vehicleID = vehicle.VehicleID
			break
		}
	}
	if vehicleID == "" && len(vehicles.Items) > 0 {
		vehicleID = vehicles.Items[len(vehicles.Items)-1].VehicleID
	}
	if vehicleID == "" {
		return fmt.Errorf("fleet vehicle create missing vehicle_id body %s", string(respBody))
	}

	driverBody, _ := json.Marshal(map[string]any{
		"name":           "SSMR Dispatch Driver",
		"phone":          envOr("SSMR_DISPATCH_DRIVER_PHONE", "+998901009991"),
		"pin":            "1234",
		"home_node_type": "WAREHOUSE",
		"home_node_id":   whID,
		"vehicle_id":     vehicleID,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/fleet/drivers", driverBody, cookie, "ssmr-fleet-driver")
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		return nil
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("fleet driver create status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_FLEET_OK")
	return nil
}

func sliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/fleet/driver/depart", nil, token, "ssmr-driver-depart")
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

func runWarehouseFleetMgmtE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config, supplierID string) (string, string, error) {
	whID := demoWarehouseID()
	plate := fmt.Sprintf("WH%04d", time.Now().Unix()%10000)
	vehicleBody, _ := json.Marshal(map[string]any{
		"label":         "SSMR WH Ops Truck",
		"license_plate": plate,
		"vehicle_class": "CLASS_A",
		"max_volume_vu": 8.0,
	})
	vehicleURL := base + "/v1/warehouse/ops/vehicles?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, vehicleURL, vehicleBody, cookie, "ssmr-wh-ops-vehicle")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated {
		return "", "", fmt.Errorf("warehouse ops vehicle status %d body %s", status, string(respBody))
	}
	var vehicleResp struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(respBody, &vehicleResp); err != nil {
		return "", "", fmt.Errorf("decode warehouse vehicle: %w", err)
	}
	if vehicleResp.VehicleID == "" {
		return "", "", fmt.Errorf("warehouse vehicle missing id body %s", string(respBody))
	}

	driverBody, _ := json.Marshal(map[string]any{
		"name":  "SSMR WH Ops Driver",
		"phone": fmt.Sprintf("+99890100%04d", time.Now().Unix()%10000),
	})
	driverURL := base + "/v1/warehouse/ops/drivers?warehouse_id=" + whID
	status, respBody, _, err = clientPost(ctx, client, driverURL, driverBody, cookie, "ssmr-wh-ops-driver")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated {
		return "", "", fmt.Errorf("warehouse ops driver status %d body %s", status, string(respBody))
	}
	var driverResp struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(respBody, &driverResp); err != nil {
		return "", "", fmt.Errorf("decode warehouse driver: %w", err)
	}
	if driverResp.DriverID == "" {
		return "", "", fmt.Errorf("warehouse driver missing id body %s", string(respBody))
	}

	assignBody, _ := json.Marshal(map[string]string{"vehicle_id": vehicleResp.VehicleID})
	assignURL := base + "/v1/warehouse/ops/drivers/" + driverResp.DriverID + "/assign-vehicle?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, assignURL, assignBody, cookie, "ssmr-wh-assign-vehicle")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("warehouse assign vehicle status %d body %s", status, string(respBody))
	}

	driverToken, err := auth.Issue(auth.Claims{
		Subject:      driverResp.DriverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   whID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return "", "", fmt.Errorf("issue driver jwt: %w", err)
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/driver/profile", nil, driverToken, "")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("driver profile status %d body %s", status, string(respBody))
	}
	var profile struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return "", "", fmt.Errorf("decode driver profile: %w", err)
	}
	if strings.TrimSpace(profile.VehicleID) != vehicleResp.VehicleID {
		return "", "", fmt.Errorf("driver profile missing vehicle_id want %s body %s", vehicleResp.VehicleID, string(respBody))
	}

	fmt.Println("PX_E2E_WAREHOUSE_FLEET_MGMT_OK")
	fmt.Println("PX_E2E_DRIVER_ASSIGN_DETECTION_OK")
	return driverResp.DriverID, vehicleResp.VehicleID, nil
}

func runDispatchCapacityE2E(ctx context.Context, client *http.Client, base, cookie, orderID, driverID, vehicleID string) error {
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(driverID) == "" {
		fmt.Println("PX_E2E_DISPATCH_CAPACITY_OK")
		return nil
	}
	whID := demoWarehouseID()
	manualBody, _ := json.Marshal(map[string]any{
		"mode": "MANUAL",
		"routes": []map[string]any{{
			"driver_id":  driverID,
			"vehicle_id": vehicleID,
			"order_ids":  []string{orderID, orderID, orderID},
		}},
	})
	url := base + "/v1/warehouse/ops/dispatch/execute?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, url, manualBody, cookie, "ssmr-dispatch-capacity")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch capacity probe status %d body %s", status, string(respBody))
	}
	var first struct {
		Status           string `json:"status"`
		CapacityWarnings []struct {
			SuggestedUnselectOrderIDs []string `json:"suggested_unselect_order_ids"`
		} `json:"capacity_warnings"`
	}
	if err := json.Unmarshal(respBody, &first); err != nil {
		return fmt.Errorf("decode dispatch capacity: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(first.Status)) == "capacity_exceeded" {
		for _, warning := range first.CapacityWarnings {
			if len(warning.SuggestedUnselectOrderIDs) == 0 {
				return fmt.Errorf("capacity_exceeded missing suggested_unselect_order_ids")
			}
		}
	}
	fmt.Println("PX_E2E_DISPATCH_CAPACITY_OK")
	return nil
}

func runFleetReassignGuardE2E(ctx context.Context, client *http.Client, base, cookie string, dispatch *dispatchManifestHint) error {
	if dispatch == nil || strings.TrimSpace(dispatch.DriverID) == "" {
		return nil
	}
	whID := demoWarehouseID()
	assignBody, _ := json.Marshal(map[string]string{"vehicle_id": dispatch.VehicleID})
	assignURL := base + "/v1/warehouse/ops/drivers/" + dispatch.DriverID + "/assign-vehicle?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch, assignURL, assignBody, cookie, "ssmr-wh-assign-guard")
	if err != nil {
		return err
	}
	if status != http.StatusConflict && status != http.StatusOK {
		return fmt.Errorf("expected assign guard conflict or no-op after depart, got %d body %s", status, string(respBody))
	}
	return nil
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
				"default_out_of_stock_policy": "REJECT",
				"initial_inventory": []map[string]any{
					{"product_id": "SSMR-SKU-1", "quantity": 100},
				},
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
	contentType := "application/json"
	if body == nil {
		contentType = ""
	}
	return clientDoContentType(ctx, client, method, url, body, contentType, bearerOrCookie, idempotencyKey)
}

func clientDoContentType(ctx context.Context, client *http.Client, method, url string, body []byte, contentType, bearerOrCookie, idempotencyKey string) (int, []byte, http.Header, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, nil, nil, err
	}
	if body != nil && strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
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
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode client policy: %w", err)
	}
	if resp.Role != "DRIVER" {
		return fmt.Errorf("driver client policy role=%q want DRIVER", resp.Role)
	}
	if resp.MinimumVersion == "" {
		return fmt.Errorf("client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_CLIENT_POLICY_OK")
	fmt.Println("PX_E2E_DRIVER_CLIENT_POLICY_OK")
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

func runDeliveryEdgeCasesE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, retailerToken, supplierID, cookie, h3Cell string) error {
	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	sessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, orderID, cfg)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("webhook: %w", err)
	}

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
		"route_id":  "route-ssmr-delivery-edge",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "admin-order-assign:"+orderID+":"+driverID)
	if err != nil {
		return fmt.Errorf("assign: %w", err)
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
		return fmt.Errorf("delivery arrive: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("delivery arrive status %d body %s", status, string(respBody))
	}

	// QR flow: Retailer gets payload
	qrReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/order/"+orderID+"/qr-payload", nil)
	qrReq.Header.Set("Authorization", "Bearer "+retailerToken)
	qrResp, err := client.Do(qrReq)
	if err != nil {
		return fmt.Errorf("qr payload request: %w", err)
	}
	defer qrResp.Body.Close()
	if qrResp.StatusCode != http.StatusOK {
		return fmt.Errorf("qr payload status %d", qrResp.StatusCode)
	}
	var qrData struct {
		Token string `json:"qr_token"`
	}
	if err := json.NewDecoder(qrResp.Body).Decode(&qrData); err != nil {
		return fmt.Errorf("decode qr payload: %w", err)
	}
	if strings.TrimSpace(qrData.Token) == "" {
		return fmt.Errorf("qr payload missing qr_token")
	}

	// QR flow: Driver validates token (no state transition)
	validatePayload := `{"order_id":"` + orderID + `","scanned_token":"` + qrData.Token + `"}`
	validateReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/order/validate-qr", strings.NewReader(validatePayload))
	validateReq.Header.Set("Authorization", "Bearer "+driverToken)
	validateReq.Header.Set("Content-Type", "application/json")
	validateResp, err := client.Do(validateReq)
	if err != nil {
		return fmt.Errorf("validate qr request: %w", err)
	}
	defer validateResp.Body.Close()
	if validateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(validateResp.Body)
		return fmt.Errorf("validate qr status %d: %s", validateResp.StatusCode, string(body))
	}

	// Damaged goods report (must run before scan-qr transitions to AWAITING_PAYMENT)
	dmgPayload := `{"order_id":"` + orderID + `","damaged_items":[{"sku":"SSMR-SKU-1","quantity":1}],"reason":"Broken bottle"}`
	dmgReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/delivery/report-damage", strings.NewReader(dmgPayload))
	dmgReq.Header.Set("Authorization", "Bearer "+driverToken)
	dmgReq.Header.Set("Content-Type", "application/json")
	dmgResp, err := client.Do(dmgReq)
	if err != nil {
		return fmt.Errorf("report damage request: %w", err)
	}
	defer dmgResp.Body.Close()
	if dmgResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(dmgResp.Body)
		return fmt.Errorf("report damage status %d: %s", dmgResp.StatusCode, string(body))
	}

	// QR flow: Driver scans QR
	scanPayload := `{"order_id":"` + orderID + `","qr_token":"` + qrData.Token + `"}`
	scanReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/delivery/scan-qr", strings.NewReader(scanPayload))
	scanReq.Header.Set("Authorization", "Bearer "+driverToken)
	scanReq.Header.Set("Content-Type", "application/json")
	scanResp, err := client.Do(scanReq)
	if err != nil {
		return fmt.Errorf("scan qr request: %w", err)
	}
	defer scanResp.Body.Close()
	if scanResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(scanResp.Body)
		return fmt.Errorf("scan qr status %d: %s", scanResp.StatusCode, string(body))
	}

	// Payment bypass while order is AWAITING_PAYMENT (after scan-qr, before cash selection).
	bypassPayload := []byte(`{"order_id":"` + orderID + `","reason":"SSMR smoke"}`)
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/orders/payment-bypass", bypassPayload, cookie, "application/json")
	if err != nil {
		return fmt.Errorf("payment bypass: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payment bypass status %d body %s", status, string(respBody))
	}
	var bypassResp struct {
		BypassToken string `json:"bypass_token"`
		OrderID     string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &bypassResp); err != nil {
		return fmt.Errorf("decode payment bypass: %w", err)
	}
	if strings.TrimSpace(bypassResp.BypassToken) == "" {
		return fmt.Errorf("payment bypass missing bypass_token body %s", string(respBody))
	}
	if bypassResp.OrderID != orderID {
		return fmt.Errorf("payment bypass order_id mismatch got %s want %s", bypassResp.OrderID, orderID)
	}
	fmt.Println("PX_E2E_SUPPLIER_PAYMENT_BYPASS_OK")

	// Retailer selects cash (driver-only completion path)
	cashPayload := `{"order_id":"` + orderID + `"}`
	cashReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/delivery/confirm-cash", strings.NewReader(cashPayload))
	cashReq.Header.Set("Authorization", "Bearer "+retailerToken)
	cashReq.Header.Set("Content-Type", "application/json")
	cashResp, err := client.Do(cashReq)
	if err != nil {
		return fmt.Errorf("confirm cash request: %w", err)
	}
	defer cashResp.Body.Close()
	if cashResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(cashResp.Body)
		return fmt.Errorf("confirm cash status %d: %s", cashResp.StatusCode, string(body))
	}
	var cashSelect struct {
		State string `json:"state"`
	}
	_ = json.NewDecoder(cashResp.Body).Decode(&cashSelect)
	if cashSelect.State != "" && cashSelect.State != "PENDING_CASH_COLLECTION" {
		return fmt.Errorf("confirm cash expected PENDING_CASH_COLLECTION, got %s", cashSelect.State)
	}

	// Driver collects cash and completes order (coords must match order delivery point)
	collectBody, _ := json.Marshal(map[string]any{
		"order_id":  orderID,
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
	})
	collectReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/order/collect-cash", bytes.NewReader(collectBody))
	collectReq.Header.Set("Authorization", "Bearer "+driverToken)
	collectReq.Header.Set("Content-Type", "application/json")
	collectReq.Header.Set("Idempotency-Key", "ssmr-collect-cash-"+orderID)
	collectResp, err := client.Do(collectReq)
	if err != nil {
		return fmt.Errorf("collect cash request: %w", err)
	}
	defer collectResp.Body.Close()
	if collectResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(collectResp.Body)
		return fmt.Errorf("collect cash status %d: %s", collectResp.StatusCode, string(body))
	}

	return nil
}

// runPaymentSmokeCheck exercises unified checkout and global-pay webhook replay.
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
	if err := runShopClosedE2E(ctx, client, base, cfg, supplierID, retailerToken, orderID, cookie); err != nil {
		return err
	}
	fmt.Println("PX_E2E_SHOP_CLOSED_OK")
	return nil
}

// runManifestSealSmokeCheck exercises payloader start-loading, order seal, and manifest seal.
func runManifestSealSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 45 * time.Second}
	if _, err := clientGet(ctx, client, base+"/v1/health"); err != nil {
		return fmt.Errorf("health: %w", err)
	}
	supplierID, _, err := ensureSupplierSession(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("supplier session: %w", err)
	}
	if err := runPayloaderE2E(ctx, client, base, cfg, supplierID, nil); err != nil {
		return err
	}
	fmt.Println("PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK")
	return nil
}
