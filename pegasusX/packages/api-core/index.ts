export { reconnectDelayMs, parseRetryAfterSeconds, retryAfterSecondsFromResponse } from "./reconnect";
export type { ReconnectBackoffOptions } from "./reconnect";
export {
  driverDeliverKey,
  driverOffloadKey,
  driverCompleteKey,
  driverCollectCashKey,
  driverFiscalRetryKey,
  adminForceCompleteKey,
  driverAvailabilityKey,
  retailerCheckoutKey,
  retailerUnifiedCheckoutKey,
  retailerCardCheckoutKey,
  retailerCashCheckoutKey,
  retailerPosSaleKey,
  retailerSupplierAddKey,
  retailerSupplierRemoveKey,
  supplierDispatchKey,
  warehouseDispatchKey,
  warehouseOrderDelayKey,
  warehouseOrderRejectKey,
  warehouseOrderOverflowKey,
  warehouseDispatchLockAcquireKey,
  warehouseDispatchLockReleaseKey,
  payloadStartLoadingKey,
  payloadSupplierStartLoadingKey,
  payloadSealKey,
  payloadOrderSealKey,
  payloadInjectKey,
  payloadSupplierInjectKey,
  payloadSealCompletedKey,
  payloadRecommendReassignKey,
  payloadFleetReassignKey,
  payloadApplyReassignKey,
  payloadSupplierSealManifestKey,
  supplierManifestSealKey,
  supplierManifestSealAllKey,
  payloadSealAllKey,
  supplierManifestStartLoadingKey,
  supplierManifestInjectKey,
  supplierVetOrderKey,
  supplierImportCreateKey,
  supplierImportIngestKey,
  supplierImportApproveKey,
  supplierImportApplyKey,
  supplierOrgMemberCreateKey,
  supplierOrgMemberUpdateKey,
  supplierOrgMemberDeactivateKey,
  supplierFleetDriverCreateKey,
  supplierFleetVehicleCreateKey,
  supplierChargebackKey,
  supplierChargebackReversalKey,
  supplierTopologyPutKey,
  supplierReplenishmentTriggerKey,
  supplierBroadcastKey,
  supplierPaymentBypassKey,
  supplierApproveEarlyCompleteKey,
  supplierNegotiationResolveKey,
  driverRequestEarlyCompleteKey,
  driverRouteReorderKey,
  driverConfirmPaymentBypassKey,
  driverBypassOffloadKey,
  driverReportShopClosedKey,
  driverProximityUnlockKey,
  driverPartialOffloadKey,
  supplierShopClosedResolveKey,
  supplierResolveReturnKey,
  supplierProfileUpdateKey,
  supplierConfigureKey,
  supplierBusinessSetupKey,
  supplierPricingRulePatchKey,
  supplierInventoryAdjustKey,
  supplierRetailerPriceOverrideCreateKey,
  supplierRetailerPriceOverrideDeleteKey,
  supplierPromotionCreateKey,
  supplierPromotionUpdateKey,
  supplierPromotionDeactivateKey,
  driverSupplyTransferArriveKey,
  warehouseCreateSupplyRequestKey,
  warehouseSupplyRequestTransitionKey,
  driverDepartKey,
  driverReturnCompleteKey,
  driverSyncBatchKey,
  driverMarkArrivedKey,
  driverSplitPaymentKey,
  driverCreditDeliveryKey,
  driverMissingItemsKey,
  payloadMissingItemsKey,
  driverReportDamageKey,
  driverNegotiateKey,
  retailerOrderCreateKey,
  retailerConfirmCashKey,
  retailerCancelKey,
  retailerRequestCancelKey,
  retailerShopClosedResponseKey,
  retailerConfirmPreorderKey,
  retailerAcceptDeliveryProposalKey,
  retailerRejectDeliveryProposalKey,
  retailerRejectPreorderKey,
  warehouseOrderProposeDeliveryKey,
  retailerConfirmAIKey,
  retailerRejectAIKey,
  retailerEditPreorderKey,
  retailerSetupKey,
  retailerProfileUpdateKey,
  adminOrderAssignKey,
  adminOrderStatusPatchKey,
  warehouseEmergencyTransferKey,
  warehouseForceReceiveKey,
  warehouseReceiveTransferKey,
  warehouseCreateDriverKey,
  warehouseCreateStaffKey,
  warehouseCreateVehicleKey,
  warehouseAdjustInventoryKey,
  warehouseAssignDriverVehicleKey,
  warehouseUpdateVehicleKey,
  warehouseInventoryPolicyKey,
  warehouseReplenishmentInsightActionKey,
  warehouseDispatchSettingsKey,
  warehouseOpsSettingsKey,
  supplierReturnPolicyPutKey,
  warehouseReturnPolicyPutKey,
  warehouseOpsLocationKey,
  warehouseBroadcastKey,
  warehouseBroadcastTemplateCreateKey,
  warehouseBroadcastTemplateDeleteKey,
  factoryManifestStartLoadingKey,
  factoryManifestSealKey,
  factoryManifestDispatchKey,
  factoryManifestCompleteKey,
  factoryBatchDispatchKey,
  factoryManifestRebalanceKey,
  factoryManifestCancelTransferKey,
  factoryManifestCancelKey,
  factoryTransferCreateKey,
  factoryTransferTransitionKey,
  factorySupplyRequestTransitionKey,
  factorySupplyRequestAcceptKey,
  factorySupplyRequestQCKey,
  supplierPlanningKillSwitchKey,
  supplierPlanningPullMatrixKey,
  supplierPlanningPredictivePushKey,
  supplierLoyaltyProgramPatchKey,
  supplierPayoutGenerateKey,
  supplierNetworkModePutKey,
  factoryOpsLocationKey,
  warehouseInboundScanKey,
  warehouseInboundConfirmKey,
  payloadInboundScanKey,
  payloadInboundConfirmKey,
  payloadManifestExceptionKey,
  claimFileKey,
  claimApproveKey,
  claimRejectKey,
  migrationProjectCreateKey,
  migrationBundleCreateKey,
  migrationBundleUploadedKey,
  migrationBundleSortKey,
  migrationMappingConfirmKey,
  migrationDryRunKey,
  migrationSessionApproveKey,
  migrationSessionApplyKey,
  migrationSessionRollbackKey,
  migrationOpenOrdersReleaseKey,
  migrationCutoverKey,
  joinRequestDecideKey,
  retailerTaxIdentityKey,
  retailerArAcknowledgeKey,
  retailerArDisputeKey,
} from "./idempotency";

export {
  CELL_API_URLS,
  homeCellFromJwt,
  isDevApiBootstrap,
  pinApiBaseUrl,
  trimApiBase,
  wsUrlFromApi,
} from "./cell-api";
export type { PinApiBaseUrlOptions } from "./cell-api";
export {
  cacheAuthSession,
  fetchAuthSession,
  fiscalReceiptLabel,
  displayPackCurrency,
  formatPackMoney,
  mapInitialViewState,
  moneyCurrency,
  packAllowsPsp,
  packCurrency,
  packMapCenter,
  selectablePackPsps,
  readCachedAuthSession,
  sessionMapCenter,
  sessionPackCurrency,
} from "./market-pack";
export type { AuthSession, MarketPack, PackMapCenter } from "./market-pack";
export {
  SESSION_RECONCILE_ENDPOINTS,
  reconcileSession,
} from "./session-reconcile";
export type { SessionReconcileRole, SessionReconcileEndpoint, SessionReconcileOptions, SessionReconcileResult } from "./session-reconcile";
export { dispatchBackpressureFromResponse, etagFromResponse } from "./conditional-get";
export type { ConditionalGet } from "./conditional-get";


export { ApiError } from "./src/client-base";
export type { ApiClientConfig, RequestOptions } from "./src/client-base";
export { ApiClient } from "./src/client-warehouse-partner";
export * from "./src/foundry";
export * from "./src/client-migration";

import type {
  NegotiationPendingResponse,
  NegotiationResolveRequest,
  NegotiationResolveResponse,
  RetailerTrackingResponse,
} from "@pegasusx/types";

export type {
  NegotiationPendingResponse,
  NegotiationResolveRequest,
  NegotiationResolveResponse,
  RetailerTrackingResponse,
};

/**
 * Role-row contract delegate for supplier pending negotiations.
 */
export async function getSupplierNegotiationsPending(
  client: import("./src/client-warehouse-partner").ApiClient,
  params?: { limit?: number; offset?: number },
): Promise<NegotiationPendingResponse> {
  return client.getSupplierNegotiationsPending(params);
}

/**
 * Role-row contract delegate for resolving supplier negotiations.
 */
export async function resolveSupplierNegotiation(
  client: import("./src/client-warehouse-partner").ApiClient,
  request: NegotiationResolveRequest,
  idempotencyKey: string,
): Promise<NegotiationResolveResponse> {
  return client.resolveSupplierNegotiation(request, idempotencyKey);
}

/**
 * Role-row contract delegate for retailer tracking.
 */
export async function getRetailerTracking(
  client: import("./src/client-warehouse-partner").ApiClient,
): Promise<RetailerTrackingResponse> {
  return client.getRetailerTracking();
}
