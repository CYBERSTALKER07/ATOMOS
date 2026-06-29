package main

import (
	"context"
	"fmt"
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
	if err := runSupplierTopologyEditE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier topology edit: %w", err)
	}
	if err := runSupplierOrgFleetE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("supplier org fleet: %w", err)
	}
	if err := runCrossRoleSupplierBroadcastWS(ctx, client, base, cookie, cfg, supplierID); err != nil {
		return fmt.Errorf("cross-role supplier broadcast ws: %w", err)
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
	if err := runCheckoutPreviewE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("checkout preview: %w", err)
	}

	cancelOrderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("order create: %w", err)
	}
	if err := assertRetailerTracking(ctx, client, base, retailerToken, cancelOrderID); err != nil {
		return fmt.Errorf("retailer tracking: %w", err)
	}
	if err := runRetailerCancelE2E(ctx, client, base, cfg, supplierID, retailerToken, cancelOrderID, retailerID); err != nil {
		return fmt.Errorf("retailer cancel: %w", err)
	}
	if err := runRetailerCardInitiateE2E(ctx, client, base, retailerToken); err != nil {
		return fmt.Errorf("retailer card initiate: %w", err)
	}
	if err := runRetailerClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("retailer client policy: %w", err)
	}

	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("order create (checkout path): %w", err)
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
	if err := runWarehouseOpsPolicyE2E(ctx, client, base, cookie, retailerToken); err != nil {
		return fmt.Errorf("warehouse ops policy: %w", err)
	}
	if err := runWarehouseReplenishmentInsightE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse replenishment insight: %w", err)
	}
	if err := runWarehouseBroadcastOpsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse broadcast ops: %w", err)
	}
	if err := runWarehouseSupplyRequestItemsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("warehouse supply request items: %w", err)
	}
	if err := runSupplierInventoryImportE2E(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier inventory import (staging substrate): %w", err)
	}
	if err := runInventoryReleaseBypassCancelE2E(ctx, client, base, cfg, supplierID, cookie, retailerToken, h3Cell); err != nil {
		return fmt.Errorf("inventory release bypass cancel paths: %w", err)
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
	if err := runWarehouseOptimizerSourceE2E(ctx, client, base, cookie, orderID); err != nil {
		return fmt.Errorf("warehouse optimizer preview: %w", err)
	}
	capacityOrderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("dispatch capacity order create: %w", err)
	}
	if err := runDispatchCapacityE2E(ctx, client, base, cookie, capacityOrderID, fleetDriverID, fleetVehicleID); err != nil {
		return fmt.Errorf("dispatch capacity: %w", err)
	}
	dispatchHint, err := runWarehouseDispatchExecuteWithWS(ctx, client, base, cookie, orderID, cfg, supplierID, fleetDriverID, fleetVehicleID)
	if err != nil {
		return fmt.Errorf("warehouse dispatch execute: %w", err)
	}
	sessionID, err := runPayAtDeliveryCheckout(ctx, client, base, retailerToken, orderID, cfg, supplierID)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("global-pay webhook: %w", err)
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
	delayOrderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("warehouse delay order create: %w", err)
	}
	if err := runWarehouseOrderMutationE2E(ctx, client, base, cookie, delayOrderID); err != nil {
		return fmt.Errorf("warehouse order mutation: %w", err)
	}
	shopClosedOrderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("shop closed order create: %w", err)
	}
	if err := runShopClosedE2E(ctx, client, base, cfg, supplierID, retailerToken, shopClosedOrderID, cookie); err != nil {
		return fmt.Errorf("shop closed e2e: %w", err)
	}
	shopClosedSessionID, err := runCardCheckoutAtDelivery(ctx, client, base, retailerToken, shopClosedOrderID, cfg)
	if err != nil {
		return fmt.Errorf("shop closed checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, shopClosedSessionID, shopClosedOrderID); err != nil {
		return fmt.Errorf("shop closed webhook: %w", err)
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
	if err := runReturnGateE2E(ctx, client, base, cfg, supplierID, cookie, retailerToken, h3Cell); err != nil {
		return fmt.Errorf("return gate: %w", err)
	}
	if err := assertDriverFirebaseOTPLogin(ctx, client, base); err != nil {
		return fmt.Errorf("driver firebase otp: %w", err)
	}
	if err := runManualPreorderE2E(ctx, client, base, retailerToken, cookie, cfg, h3Cell); err != nil {
		return fmt.Errorf("manual preorder: %w", err)
	}
	if err := runDeliveryProposalE2E(ctx, client, base, retailerToken, cookie, cfg, h3Cell); err != nil {
		return fmt.Errorf("delivery proposal: %w", err)
	}
	if err := runConcurrentStockRejectE2E(ctx, client, base, cfg, supplierID, retailerToken, cookie, h3Cell); err != nil {
		return fmt.Errorf("concurrent stock reject: %w", err)
	}
	if err := runCheckoutPolicyGraceE2E(ctx, client, base, cfg, supplierID, retailerToken, cookie, h3Cell); err != nil {
		return fmt.Errorf("checkout policy grace: %w", err)
	}
	if err := runOrderAcceptanceClosedE2E(ctx, client, base, cookie, retailerToken, cfg, h3Cell); err != nil {
		return fmt.Errorf("order acceptance closed: %w", err)
	}

	fmt.Println("PX_E2E_ORDER_OK")
	fmt.Println("PX_E2E_PAYMENT_OK")
	fmt.Println("PX_E2E_WAREHOUSE_OK")
	fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_SETTINGS_OK")
	fmt.Println("PX_E2E_WAREHOUSE_STOCK_POLICY_OK")
	fmt.Println("PX_E2E_WAREHOUSE_REPLENISHMENT_OK")
	fmt.Println("PX_E2E_WAREHOUSE_SUPPLY_REQUEST_ITEMS_OK")
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
	fmt.Println("PX_E2E_FACTORY_DRIVER_SUPPLY_ARRIVE_OK")
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
	if err := assertDomainTopicDispatchMarker(); err != nil {
		return err
	}
	return nil
}
