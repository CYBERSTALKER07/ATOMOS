import type {
  ConfirmAIOrderRequest,
  ConfirmPreorderRequest,
  EditPreorderRequest,
  PaymentChargebackRequest,
  PaymentChargebackResponse,
  PaymentChargebackReversalRequest,
  PaymentChargebackReversalResponse,
  RejectAIOrderRequest,
  RetailerAIPredictionsResponse,
  RetailerOrderLifecycleResponse,
  PaymentLedgerQuery,
  PaymentLedgerResponse,
  ReconciliationMismatchQuery,
  ReconciliationMismatchResponse,
  RetailerActiveFulfillmentResponse,
  RetailerPendingPaymentsResponse,
  RetailerProfileResponse,
  RetailerProfileUpdateRequest,
  CreateRetailerPriceOverrideRequest,
  CreateRetailerPriceOverrideResponse,
  RetailerPriceOverridesResponse,
  RetailerSupplierPreference,
  RetailerTrackingResponse,
  SettlementAuthorityQuery,
  SettlementAuthorityResponse,
  SupplierBillingSetupRequest,
  SupplierBillingSetupResponse,
  SupplierBusinessSetupRequest,
  SupplierBusinessSetupResponse,
  SupplierConfigureRequest,
  SupplierConfigureResponse,
  SupplierDashboardResponse,
  SupplierExceptionsResponse,
  ShopClosedActiveResponse,
  ShopClosedResolveRequest,
  NegotiationPendingResponse,
  NegotiationResolveRequest,
  NegotiationResolveResponse,
  PaymentBypassRequest,
  PaymentBypassResponse,
  SupplierEmpathyAdoption,
  SupplierBroadcastRequest,
  SupplierBroadcastResponse,
  SupplierReplenishmentTriggerResponse,
  SupplierFleetOrderRow,
  SupplierFleetLiveMapResponse,
  SupplierLoginRequest,
  SupplierLoginResponse,
  SupplierManifestsResponse,
  SupplierManifestDetail,
  SupplierManifestExceptionsResponse,
  SupplierManifestInjectOrderRequest,
  SupplierManifestSealResponse,
  SupplierSupplyLanesResponse,
  SupplierAIRecommendationDecisionRequest,
  SupplierAIRecommendationDecisionResponse,
  SupplierAIRecommendationsQuery,
  SupplierActivityResponse,
  SupplierAIRecommendationsResponse,
  SupplierAnalyticsVelocityResponse,
  SupplierAnalyticsRevenueResponse,
  SupplierDemandSummaryResponse,
  SupplierDemandHistoryResponse,
  SupplierInventoryImportResult,
  SupplierImportApplyResponse,
  SupplierImportIngestResponse,
  SupplierImportMappingResponse,
  SupplierImportSession,
  SupplierImportSessionCreateResponse,
  SupplierDispatchPreview,
  SupplierDispatchExecuteRequest,
  SupplierDispatchExecuteResponse,
  SupplierOrgMemberCreateRequest,
  SupplierOrgMemberUpdateRequest,
  SupplierOrgMembersResponse,
  SupplierOrdersResponse,
  SupplierOrder,
  AssignOrderRequest,
  AssignOrderResponse,
  OrderStatusPatchRequest,
  OrderStatusPatchResponse,
  SupplierFleetDriverCreateRequest,
  SupplierFleetDriversResponse,
  SupplierFleetVehicleCreateRequest,
  SupplierFleetVehiclesResponse,
  SupplierProfile,
  SupplierProfileUpdateRequest,
  SupplierPricingRule,
  SupplierPricingRuleUpdateRequest,
  SupplierPromotion,
  SupplierPromotionUpsertRequest,
  SupplierPromotionsResponse,
  SupplierRegisterRequest,
  SupplierRegisterResponse,
  SupplierInventoryResponse,
  SupplierTopologyResponse,
  SupplierTopologyUpdateRequest,
  WarehouseDispatchLock,
  WarehouseDispatchLockAcquireRequest,
  WarehouseDispatchLockReleaseResponse,
  WarehouseDispatchLocksResponse,
  WarehouseDispatchExecuteRequest,
  WarehouseDispatchExecuteResponse,
  WarehouseDispatchPreview,
  WarehouseDemandForecastResponse,
  WarehouseEmergencyTransferRequest,
  WarehouseForceReceiveRequest,
  WarehouseInventoryResponse,
  WarehouseFleetLiveMapResponse,
  WarehouseOpsDashboardResponse,
  WarehouseOpsFinancialsResponse,
  WarehouseOrderMutationRequest,
  WarehouseOrderMutationResponse,
  WarehouseOrdersResponse,
  WarehouseDispatchSettingsPatchRequest,
  WarehouseDispatchSettingsResponse,
  WarehouseReplenishmentInsightActionResponse,
  WarehouseReplenishmentInsightsResponse,
  WarehouseSupplyRequest,
  WarehouseSupplyRequestsResponse,
  WarehouseTransferMutationResponse,
} from "@pegasusx/types";

export { reconnectDelayMs, parseRetryAfterSeconds, retryAfterSecondsFromResponse } from "./reconnect";
export type { ReconnectBackoffOptions } from "./reconnect";
export {
  driverDeliverKey,
  driverOffloadKey,
  driverCompleteKey,
  driverCollectCashKey,
  retailerCheckoutKey,
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
  supplierManifestStartLoadingKey,
  supplierManifestInjectKey,
  supplierVetOrderKey,
  supplierImportCreateKey,
  supplierImportIngestKey,
  supplierImportApproveKey,
  supplierImportApplyKey,
  supplierBroadcastKey,
  supplierPaymentBypassKey,
  supplierApproveEarlyCompleteKey,
  driverConfirmPaymentBypassKey,
  driverBypassOffloadKey,
  driverReportShopClosedKey,
  supplierShopClosedResolveKey,
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
  retailerConfirmAIKey,
  adminOrderAssignKey,
  adminOrderStatusPatchKey,
  warehouseEmergencyTransferKey,
  warehouseForceReceiveKey,
  warehouseReceiveTransferKey,
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
} from "./idempotency";
export {
  SESSION_RECONCILE_ENDPOINTS,
  reconcileSession,
} from "./session-reconcile";
export type { SessionReconcileRole, SessionReconcileEndpoint, SessionReconcileOptions, SessionReconcileResult } from "./session-reconcile";

export interface ApiClientConfig {
  baseUrl: string;
  getAuthToken?: () => string | null;
  fetchImpl?: typeof fetch;
}

interface RequestOptions {
  body?: unknown;
  rawBody?: string;
  idempotencyKey?: string;
  requiresAuth?: boolean;
  headers?: HeadersInit;
}

export class ApiError extends Error {
  public readonly status: number;
  public readonly payload: unknown;

  constructor(message: string, status: number, payload: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export class ApiClient {
  constructor(public readonly config: ApiClientConfig) {}

  async registerSupplier(request: SupplierRegisterRequest, idempotencyKey: string): Promise<SupplierRegisterResponse> {
    return this.request<SupplierRegisterResponse>("/v1/auth/supplier/register", "POST", {
      body: request,
      idempotencyKey,
      requiresAuth: false,
    });
  }

  async loginSupplier(request: SupplierLoginRequest): Promise<SupplierLoginResponse> {
    return this.request<SupplierLoginResponse>("/v1/auth/supplier/login", "POST", {
      body: request,
      requiresAuth: false,
    });
  }

  async configureSupplier(request: SupplierConfigureRequest): Promise<SupplierConfigureResponse> {
    return this.request<SupplierConfigureResponse>("/v1/supplier/configure", "POST", {
      body: request,
    });
  }

  async configureSupplierBilling(
    request: SupplierBillingSetupRequest,
    idempotencyKey: string,
  ): Promise<SupplierBillingSetupResponse> {
    return this.request<SupplierBillingSetupResponse>("/v1/supplier/billing/setup", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async setupSupplierBusiness(
    request: SupplierBusinessSetupRequest,
    idempotencyKey?: string,
  ): Promise<SupplierBusinessSetupResponse> {
    return this.request<SupplierBusinessSetupResponse>("/v1/supplier/business/setup", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierProfile(): Promise<SupplierProfile> {
    return this.request<SupplierProfile>("/v1/supplier/profile", "GET");
  }

  async updateSupplierProfile(request: SupplierProfileUpdateRequest): Promise<SupplierProfile> {
    return this.request<SupplierProfile>("/v1/supplier/profile", "PUT", { body: request });
  }

  async getSupplierTopology(): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "GET");
  }

  async updateSupplierTopology(request: SupplierTopologyUpdateRequest): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "PUT", { body: request });
  }

  async getSupplierInventory(): Promise<SupplierInventoryResponse> {
    return this.request<SupplierInventoryResponse>("/v1/supplier/inventory", "GET");
  }

  async getSupplierOrgMembers(): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>("/v1/supplier/org/members", "GET");
  }

  async createSupplierOrgMember(
    request: SupplierOrgMemberCreateRequest,
    idempotencyKey: string,
  ): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>("/v1/supplier/org/members", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async updateSupplierOrgMember(
    userId: string,
    request: SupplierOrgMemberUpdateRequest,
    idempotencyKey: string,
  ): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>(`/v1/supplier/org/members/${encodeURIComponent(userId)}`, "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  async deactivateSupplierOrgMember(
    userId: string,
    idempotencyKey: string,
  ): Promise<SupplierOrgMembersResponse> {
    return this.request<SupplierOrgMembersResponse>(`/v1/supplier/org/members/${encodeURIComponent(userId)}`, "DELETE", {
      idempotencyKey,
    });
  }

  async getSupplierFleetDrivers(): Promise<SupplierFleetDriversResponse> {
    return this.request<SupplierFleetDriversResponse>("/v1/supplier/fleet/drivers", "GET");
  }

  async createSupplierFleetDriver(
    request: SupplierFleetDriverCreateRequest,
    idempotencyKey: string,
  ): Promise<SupplierFleetDriversResponse> {
    return this.request<SupplierFleetDriversResponse>("/v1/supplier/fleet/drivers", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierFleetVehicles(): Promise<SupplierFleetVehiclesResponse> {
    return this.request<SupplierFleetVehiclesResponse>("/v1/supplier/fleet/vehicles", "GET");
  }

  async createSupplierFleetVehicle(
    request: SupplierFleetVehicleCreateRequest,
    idempotencyKey: string,
  ): Promise<SupplierFleetVehiclesResponse> {
    return this.request<SupplierFleetVehiclesResponse>("/v1/supplier/fleet/vehicles", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierPricingRule(): Promise<SupplierPricingRule> {
    return this.request<SupplierPricingRule>("/v1/supplier/pricing/rules", "GET");
  }

  async listRetailerPriceOverrides(params?: {
    retailer_id?: string;
    product_id?: string;
  }): Promise<RetailerPriceOverridesResponse> {
    const query = new URLSearchParams();
    if (params?.retailer_id) query.set("retailer_id", params.retailer_id);
    if (params?.product_id) query.set("product_id", params.product_id);
    const suffix = query.toString();
    const path = suffix
      ? `/v1/supplier/pricing/retailer-overrides?${suffix}`
      : "/v1/supplier/pricing/retailer-overrides";
    return this.request<RetailerPriceOverridesResponse>(path, "GET");
  }

  async createRetailerPriceOverride(
    request: CreateRetailerPriceOverrideRequest,
  ): Promise<CreateRetailerPriceOverrideResponse> {
    return this.request<CreateRetailerPriceOverrideResponse>(
      "/v1/supplier/pricing/retailer-overrides",
      "POST",
      { body: request },
    );
  }

  async deleteRetailerPriceOverride(overrideId: string): Promise<{ status: string; override_id: string }> {
    return this.request<{ status: string; override_id: string }>(
      `/v1/supplier/pricing/retailer-overrides/${overrideId}`,
      "DELETE",
    );
  }

  async listSupplierPromotions(): Promise<SupplierPromotionsResponse> {
    return this.request<SupplierPromotionsResponse>("/v1/supplier/promotions", "GET");
  }

  async createSupplierPromotion(
    request: SupplierPromotionUpsertRequest,
    idempotencyKey: string,
  ): Promise<SupplierPromotion> {
    return this.request<SupplierPromotion>("/v1/supplier/promotions", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async updateSupplierPromotion(
    promotionId: string,
    request: SupplierPromotionUpsertRequest,
  ): Promise<SupplierPromotion> {
    return this.request<SupplierPromotion>(`/v1/supplier/promotions/${promotionId}`, "PATCH", {
      body: request,
    });
  }

  async deactivateSupplierPromotion(promotionId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>(`/v1/supplier/promotions/${promotionId}`, "DELETE");
  }

  async getSupplierOrders(
    query: {
      limit?: number;
      offset?: number;
      status?: string;
      filter?: "ACTIVE" | "COMPLETED" | "CANCELLED" | "RETURNS";
    } = {},
  ): Promise<SupplierOrdersResponse> {
    return this.request<SupplierOrdersResponse>(appendQuery("/v1/supplier/orders", query as Record<string, unknown>), "GET");
  }

  async vetSupplierOrder(
    request: { order_id: string; decision: string; note?: string },
    idempotencyKey?: string,
  ): Promise<{ order: SupplierOrder }> {
    return this.request<{ order: SupplierOrder }>("/v1/supplier/orders/vet", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** ADMIN / WAREHOUSE_ADMIN / FACTORY_ADMIN — assign driver to order. */
  async assignOrder(
    orderId: string,
    request: AssignOrderRequest,
    idempotencyKey: string,
  ): Promise<AssignOrderResponse> {
    return this.request<AssignOrderResponse>(`/v1/orders/${encodeURIComponent(orderId)}/assign`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** ADMIN / RETAILER — patch canonical order status. */
  async patchOrderStatus(
    orderId: string,
    request: OrderStatusPatchRequest,
    idempotencyKey: string,
  ): Promise<OrderStatusPatchResponse> {
    return this.request<OrderStatusPatchResponse>(`/v1/order/${encodeURIComponent(orderId)}/status`, "PATCH", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierDispatchPreview(query: { warehouse_id?: string } = {}): Promise<SupplierDispatchPreview> {
    return this.request<SupplierDispatchPreview>(
      appendQuery("/v1/supplier/dispatch/preview", query as Record<string, unknown>),
      "GET",
    );
  }

  async createSupplierDispatchPreview(
    request: Record<string, unknown>,
  ): Promise<SupplierDispatchPreview> {
    return this.request<SupplierDispatchPreview>("/v1/supplier/dispatch/preview", "POST", {
      body: request,
    });
  }

  async executeSupplierDispatch(
    request: SupplierDispatchExecuteRequest = { mode: "AUTO" },
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<SupplierDispatchExecuteResponse> {
    return this.request<SupplierDispatchExecuteResponse>(
      appendQuery("/v1/supplier/dispatch/execute", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getSupplierActivity(query: { limit?: number } = {}): Promise<SupplierActivityResponse> {
    return this.request<SupplierActivityResponse>(
      appendQuery("/v1/supplier/activity", query as Record<string, unknown>),
      "GET",
    );
  }

  async getSupplierDashboard(): Promise<SupplierDashboardResponse> {
    return this.request<SupplierDashboardResponse>("/v1/supplier/dashboard", "GET");
  }

  async getSupplierAnalyticsVelocity(): Promise<SupplierAnalyticsVelocityResponse> {
    return this.request<SupplierAnalyticsVelocityResponse>("/v1/supplier/analytics/velocity", "GET");
  }

  async getSupplierAnalyticsRevenue(): Promise<SupplierAnalyticsRevenueResponse> {
    return this.request<SupplierAnalyticsRevenueResponse>("/v1/supplier/analytics/revenue", "GET");
  }

  async getSupplierDemandToday(): Promise<SupplierDemandSummaryResponse> {
    return this.request<SupplierDemandSummaryResponse>("/v1/supplier/analytics/demand/today", "GET");
  }

  async getSupplierDemandHistory(): Promise<SupplierDemandHistoryResponse> {
    return this.request<SupplierDemandHistoryResponse>("/v1/supplier/analytics/demand/history", "GET");
  }

  async importSupplierInventoryCSV(
    csvBody: string,
    idempotencyKey: string,
  ): Promise<SupplierInventoryImportResult> {
    return this.request<SupplierInventoryImportResult>("/v1/supplier/inventory/import", "POST", {
      rawBody: csvBody,
      idempotencyKey,
      headers: { "Content-Type": "text/csv" },
    });
  }

  async createSupplierImportSession(
    fileName: string,
    fileSizeBytes: number,
    idempotencyKey: string,
  ): Promise<SupplierImportSessionCreateResponse> {
    return this.request<SupplierImportSessionCreateResponse>("/v1/supplier/inventory/imports", "POST", {
      body: { file_name: fileName, file_size_bytes: fileSizeBytes },
      idempotencyKey,
    });
  }

  async ingestSupplierImportSession(
    sessionId: string,
    csvBody: string,
    idempotencyKey: string,
  ): Promise<SupplierImportIngestResponse> {
    return this.request<SupplierImportIngestResponse>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/ingest`,
      "POST",
      {
        rawBody: csvBody,
        idempotencyKey,
        headers: { "Content-Type": "text/csv" },
      },
    );
  }

  async getSupplierImportSession(sessionId: string): Promise<SupplierImportSession> {
    return this.request<SupplierImportSession>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}`,
      "GET",
    );
  }

  async getSupplierImportMapping(sessionId: string): Promise<SupplierImportMappingResponse> {
    return this.request<SupplierImportMappingResponse>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/mapping`,
      "GET",
    );
  }

  async approveSupplierImportSession(
    sessionId: string,
    idempotencyKey: string,
  ): Promise<{ session_id: string; status: string }> {
    return this.request(`/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/approve`, "POST", {
      idempotencyKey,
    });
  }

  async applySupplierImportSession(
    sessionId: string,
    idempotencyKey: string,
  ): Promise<SupplierImportApplyResponse> {
    return this.request<SupplierImportApplyResponse>(
      `/v1/supplier/inventory/imports/${encodeURIComponent(sessionId)}/apply`,
      "POST",
      { idempotencyKey },
    );
  }

  async updateSupplierInventory(
    request: Record<string, unknown>,
  ): Promise<void> {
    return this.request<void>("/v1/supplier/inventory", "PATCH", {
      body: request,
    });
  }

  async getSupplierInventoryAudit(): Promise<unknown> {
    return this.request<unknown>("/v1/supplier/inventory/audit", "GET");
  }

  async getSupplierManifests(): Promise<SupplierManifestsResponse> {
    return this.request<SupplierManifestsResponse>("/v1/supplier/manifests", "GET");
  }

  async getSupplierManifestDetail(manifestId: string): Promise<SupplierManifestDetail> {
    return this.request<SupplierManifestDetail>(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}`, "GET");
  }

  async startSupplierManifestLoading(manifestId: string, idempotencyKey: string): Promise<{ status?: string; manifest_id?: string; state?: string }> {
    return this.request(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}/start-loading`, "POST", {
      idempotencyKey,
    });
  }

  async injectSupplierManifestOrder(
    manifestId: string,
    request: SupplierManifestInjectOrderRequest,
    idempotencyKey: string,
  ): Promise<{ status?: string; manifest_id?: string; order_id?: string }> {
    return this.request(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}/inject-order`, "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async sealSupplierManifest(manifestId: string, idempotencyKey: string): Promise<SupplierManifestSealResponse> {
    return this.request<SupplierManifestSealResponse>(`/v1/supplier/manifests/${encodeURIComponent(manifestId)}/seal`, "POST", {
      idempotencyKey,
    });
  }

  async getSupplierManifestExceptions(params?: { escalated?: boolean }): Promise<SupplierManifestExceptionsResponse> {
    const query = new URLSearchParams();
    if (params?.escalated) query.set("escalated", "true");
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return this.request<SupplierManifestExceptionsResponse>(`/v1/supplier/manifest-exceptions${suffix}`, "GET");
  }

  async getSupplierSupplyLanes(): Promise<SupplierSupplyLanesResponse> {
    return this.request<SupplierSupplyLanesResponse>("/v1/supplier/supply-lanes", "GET");
  }

  async getSupplierExceptions(): Promise<SupplierExceptionsResponse> {
    return this.request<SupplierExceptionsResponse>("/v1/supplier/exceptions", "GET");
  }

  async getSupplierShopClosedActive(params?: { limit?: number; offset?: number }): Promise<ShopClosedActiveResponse> {
    const query = new URLSearchParams();
    if (params?.limit != null) query.set("limit", String(params.limit));
    if (params?.offset != null) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return this.request<ShopClosedActiveResponse>(`/v1/supplier/shop-closed/active${suffix}`, "GET");
  }

  async resolveSupplierShopClosed(
    request: ShopClosedResolveRequest,
    idempotencyKey: string,
  ): Promise<{ status?: string; bypass_token?: string; queued?: boolean }> {
    return this.request("/v1/supplier/shop-closed/resolve", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Quantity negotiation disabled ecosystem-wide. */
  async getSupplierNegotiationsPending(_params?: { limit?: number; offset?: number }): Promise<NegotiationPendingResponse> {
    return { data: [] };
  }

  async approveSupplierEarlyComplete(
    request: Record<string, unknown>,
    idempotencyKey?: string,
  ): Promise<void> {
    return this.request<void>("/v1/supplier/route/approve-early-complete", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  /** Quantity negotiation disabled ecosystem-wide. */
  async resolveSupplierNegotiation(
    _request: NegotiationResolveRequest,
    _idempotencyKey: string,
  ): Promise<NegotiationResolveResponse> {
    throw new Error("quantity_negotiation_disabled");
  }

  async issueSupplierPaymentBypass(
    request: PaymentBypassRequest,
    idempotencyKey: string,
  ): Promise<PaymentBypassResponse> {
    return this.request<PaymentBypassResponse>("/v1/supplier/orders/payment-bypass", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierEmpathyAdoption(): Promise<SupplierEmpathyAdoption> {
    return this.request<SupplierEmpathyAdoption>("/v1/supplier/empathy/adoption", "GET");
  }

  async postSupplierBroadcast(
    request: SupplierBroadcastRequest,
    idempotencyKey: string,
  ): Promise<SupplierBroadcastResponse> {
    return this.request<SupplierBroadcastResponse>("/v1/supplier/broadcast", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async triggerSupplierReplenishment(): Promise<SupplierReplenishmentTriggerResponse> {
    return this.request<SupplierReplenishmentTriggerResponse>("/v1/supplier/replenishment/trigger", "POST");
  }

  async getSupplierFleetOrders(): Promise<SupplierFleetOrderRow[]> {
    return this.request<SupplierFleetOrderRow[]>("/v1/supplier/fleet/orders", "GET");
  }

  async getSupplierFleetLiveMap(): Promise<SupplierFleetLiveMapResponse> {
    return this.request<SupplierFleetLiveMapResponse>("/v1/supplier/fleet/live-map", "GET");
  }

  async getSupplierEarnings(): Promise<{
    currency: string;
    today_minor: number;
    week_minor: number;
    month_minor: number;
    authority_source?: string;
    authoritative?: boolean;
    updated_at: string;
  }> {
    return this.request("/v1/supplier/earnings", "GET");
  }

  async getSupplierAIRecommendations(query: SupplierAIRecommendationsQuery = {}): Promise<SupplierAIRecommendationsResponse> {
    return this.request<SupplierAIRecommendationsResponse>(appendQuery("/v1/supplier/ai/recommendations", query as Record<string, unknown>), "GET");
  }

  async recordSupplierAIRecommendationDecision(
    request: SupplierAIRecommendationDecisionRequest,
    idempotencyKey: string,
  ): Promise<SupplierAIRecommendationDecisionResponse> {
    return this.request<SupplierAIRecommendationDecisionResponse>("/v1/supplier/ai/recommendations", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async updateSupplierPricingRule(request: SupplierPricingRuleUpdateRequest): Promise<SupplierPricingRule> {
    return this.request<SupplierPricingRule>("/v1/supplier/pricing/rules", "PATCH", { body: request });
  }

  async getRetailerSuppliers(): Promise<RetailerSupplierPreference[]> {
    return this.request<RetailerSupplierPreference[]>("/v1/retailer/suppliers", "GET");
  }

  async getRetailerPricingRule(): Promise<RetailerPricingRuleResponse> {
    return this.request<RetailerPricingRuleResponse>("/v1/retailer/pricing/rules", "GET");
  }

  async getRetailerProfile(): Promise<RetailerProfileResponse> {
    return this.request<RetailerProfileResponse>("/v1/retailer/profile", "GET");
  }

  async updateRetailerProfile(request: RetailerProfileUpdateRequest): Promise<RetailerProfileResponse> {
    return this.request<RetailerProfileResponse>("/v1/retailer/profile", "PUT", { body: request });
  }

  async getRetailerTracking(): Promise<RetailerTrackingResponse> {
    return this.request<RetailerTrackingResponse>("/v1/retailer/tracking", "GET");
  }

  async getRetailerActiveFulfillment(): Promise<RetailerActiveFulfillmentResponse> {
    return this.request<RetailerActiveFulfillmentResponse>("/v1/retailer/active-fulfillment", "GET");
  }

  async getRetailerPendingPayments(): Promise<RetailerPendingPaymentsResponse> {
    return this.request<RetailerPendingPaymentsResponse>("/v1/retailer/pending-payments", "GET");
  }

  async getRetailerAIPredictions(limit?: number): Promise<RetailerAIPredictionsResponse> {
    const path = limit && limit > 0 ? appendQuery("/v1/retailer/ai/predictions", { limit }) : "/v1/retailer/ai/predictions";
    return this.request<RetailerAIPredictionsResponse>(path, "GET");
  }

  async confirmRetailerAIOrder(
    request: ConfirmAIOrderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/retailer/orders/confirm-ai", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async rejectRetailerAIOrder(
    request: RejectAIOrderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/retailer/orders/reject-ai", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async editRetailerPreorder(
    request: EditPreorderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/edit-preorder", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async confirmRetailerPreorder(
    request: ConfirmPreorderRequest,
    idempotencyKey: string,
  ): Promise<RetailerOrderLifecycleResponse> {
    return this.request<RetailerOrderLifecycleResponse>("/v1/orders/confirm-preorder", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getWarehouseDemandForecast(query: { warehouse_id?: string; start_date?: string; days?: number } = {}): Promise<WarehouseDemandForecastResponse> {
    return this.request<WarehouseDemandForecastResponse>(appendQuery("/v1/warehouse/demand/forecast", query as Record<string, unknown>), "GET");
  }

  async getWarehouseOpsDashboard(query: { warehouse_id?: string } = {}): Promise<WarehouseOpsDashboardResponse> {
    return this.request<WarehouseOpsDashboardResponse>(appendQuery("/v1/warehouse/ops/dashboard", query as Record<string, unknown>), "GET");
  }

  async getWarehouseFleetLiveMap(query: { warehouse_id?: string } = {}): Promise<WarehouseFleetLiveMapResponse> {
    return this.request<WarehouseFleetLiveMapResponse>(appendQuery("/v1/warehouse/ops/fleet/live-map", query as Record<string, unknown>), "GET");
  }

  async getWarehouseInventory(query: { warehouse_id?: string } = {}): Promise<WarehouseInventoryResponse> {
    return this.request<WarehouseInventoryResponse>(appendQuery("/v1/warehouse/ops/inventory", query as Record<string, unknown>), "GET");
  }

  async getWarehouseOrders(query: { warehouse_id?: string; state?: string } = {}): Promise<WarehouseOrdersResponse> {
    return this.request<WarehouseOrdersResponse>(appendQuery("/v1/warehouse/ops/orders", query as Record<string, unknown>), "GET");
  }

  async previewWarehouseDispatch(query: { warehouse_id?: string } = {}): Promise<WarehouseDispatchPreview> {
    return this.request<WarehouseDispatchPreview>(appendQuery("/v1/warehouse/ops/dispatch/preview", query as Record<string, unknown>), "POST");
  }

  async executeWarehouseDispatch(
    request: WarehouseDispatchExecuteRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseDispatchExecuteResponse> {
    return this.request<WarehouseDispatchExecuteResponse>(appendQuery("/v1/warehouse/ops/dispatch/execute", query as Record<string, unknown>), "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async setupWarehouse(
    request: Record<string, unknown>,
    query: { warehouse_id?: string } = {},
  ): Promise<void> {
    return this.request<void>(appendQuery("/v1/warehouse/setup", query as Record<string, unknown>), "POST", {
      body: request,
    });
  }

  async getWarehouseSupplyRequests(query: { warehouse_id?: string } = {}): Promise<WarehouseSupplyRequestsResponse> {
    return this.request<WarehouseSupplyRequestsResponse>(appendQuery("/v1/warehouse/supply-requests", query as Record<string, unknown>), "GET");
  }

  async getWarehouseSupplyRequest(
    id: string,
    query: { warehouse_id?: string } = {},
  ): Promise<WarehouseSupplyRequest> {
    return this.request<WarehouseSupplyRequest>(appendQuery(`/v1/warehouse/supply-requests/${id}`, query as Record<string, unknown>), "GET");
  }

  async openWarehouseSupplyRequest(
    query: { warehouse_id?: string; start_date?: string; days?: number; requested_by?: string } = {},
  ): Promise<WarehouseSupplyRequest> {
    return this.request<WarehouseSupplyRequest>(appendQuery("/v1/warehouse/supply-requests", query as Record<string, unknown>), "POST");
  }

  async getWarehouseDispatchLocks(query: { warehouse_id?: string } = {}): Promise<WarehouseDispatchLocksResponse> {
    return this.request<WarehouseDispatchLocksResponse>(appendQuery("/v1/warehouse/dispatch-locks", query as Record<string, unknown>), "GET");
  }

  async acquireWarehouseDispatchLock(
    request: WarehouseDispatchLockAcquireRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseDispatchLock> {
    return this.request<WarehouseDispatchLock>(appendQuery("/v1/warehouse/dispatch-lock", query as Record<string, unknown>), "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async releaseWarehouseDispatchLock(
    query: { warehouse_id?: string; lock_id: string },
    idempotencyKey?: string,
  ): Promise<WarehouseDispatchLockReleaseResponse> {
    return this.request<WarehouseDispatchLockReleaseResponse>(appendQuery("/v1/warehouse/dispatch-lock", query as Record<string, unknown>), "DELETE", {
      idempotencyKey,
    });
  }

  async postWarehouseEmergencyTransfer(
    request: WarehouseEmergencyTransferRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseTransferMutationResponse> {
    return this.request<WarehouseTransferMutationResponse>(
      appendQuery("/v1/warehouse/transfers/emergency", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseForceReceive(
    request: WarehouseForceReceiveRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseTransferMutationResponse> {
    return this.request<WarehouseTransferMutationResponse>(
      appendQuery("/v1/warehouse/transfers/force-receive", query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseReceiveTransfer(
    transferId: string,
    query: { warehouse_id?: string } = {},
    idempotencyKey: string,
  ): Promise<WarehouseTransferMutationResponse> {
    return this.request<WarehouseTransferMutationResponse>(
      appendQuery(`/v1/warehouse/transfers/${encodeURIComponent(transferId)}/receive`, query as Record<string, unknown>),
      "POST",
      { idempotencyKey },
    );
  }

  async getWarehouseReplenishmentInsights(
    query: { warehouse_id?: string; limit?: number } = {},
  ): Promise<WarehouseReplenishmentInsightsResponse> {
    return this.request<WarehouseReplenishmentInsightsResponse>(
      appendQuery("/v1/warehouse/replenishment/insights", query as Record<string, unknown>),
      "GET",
    );
  }

  async postWarehouseReplenishmentInsightAction(
    insightId: string,
    action: "approve" | "dismiss",
    query: { warehouse_id?: string } = {},
  ): Promise<WarehouseReplenishmentInsightActionResponse> {
    return this.request<WarehouseReplenishmentInsightActionResponse>(
      appendQuery(`/v1/warehouse/replenishment/insights/${insightId}/${action}`, query as Record<string, unknown>),
      "POST",
    );
  }

  async getWarehouseDispatchSettings(
    query: { warehouse_id?: string } = {},
  ): Promise<WarehouseDispatchSettingsResponse> {
    return this.request<WarehouseDispatchSettingsResponse>(
      appendQuery("/v1/warehouse/ops/dispatch/settings", query as Record<string, unknown>),
      "GET",
    );
  }

  async patchWarehouseDispatchSettings(
    request: WarehouseDispatchSettingsPatchRequest,
    query: { warehouse_id?: string } = {},
  ): Promise<{ status: string }> {
    return this.request<{ status: string }>(
      appendQuery("/v1/warehouse/ops/dispatch/settings", query as Record<string, unknown>),
      "PATCH",
      { body: request },
    );
  }

  async getWarehouseOpsFinancials(
    query: { warehouse_id?: string; period?: string } = {},
  ): Promise<WarehouseOpsFinancialsResponse> {
    return this.request<WarehouseOpsFinancialsResponse>(
      appendQuery("/v1/warehouse/ops/financials", query as Record<string, unknown>),
      "GET",
    );
  }

  async postWarehouseOrderDelay(
    orderId: string,
    request: WarehouseOrderMutationRequest = {},
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/delay`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseOrderReject(
    orderId: string,
    request: WarehouseOrderMutationRequest,
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/reject`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async postWarehouseOrderOverflow(
    orderId: string,
    request: WarehouseOrderMutationRequest = {},
    query: { warehouse_id?: string } = {},
    idempotencyKey?: string,
  ): Promise<WarehouseOrderMutationResponse> {
    return this.request<WarehouseOrderMutationResponse>(
      appendQuery(`/v1/warehouse/ops/orders/${orderId}/overflow`, query as Record<string, unknown>),
      "POST",
      { body: request, idempotencyKey },
    );
  }

  async getPaymentLedger(query: PaymentLedgerQuery = {}): Promise<PaymentLedgerResponse> {
    return this.request<PaymentLedgerResponse>(appendQuery("/v1/payment/ledger", query as Record<string, unknown>), "GET");
  }

  async getPaymentSettlementAuthority(query: SettlementAuthorityQuery = {}): Promise<SettlementAuthorityResponse> {
    return this.request<SettlementAuthorityResponse>(appendQuery("/v1/payment/settlement/authority", query as Record<string, unknown>), "GET");
  }

  async getPaymentReconciliationMismatches(query: ReconciliationMismatchQuery = {}): Promise<ReconciliationMismatchResponse> {
    return this.request<ReconciliationMismatchResponse>(appendQuery("/v1/payment/reconciliation/mismatches", query as Record<string, unknown>), "GET");
  }

  async recordPaymentChargeback(
    request: PaymentChargebackRequest,
    idempotencyKey: string,
  ): Promise<PaymentChargebackResponse> {
    return this.request<PaymentChargebackResponse>("/v1/payment/chargeback", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async recordPaymentChargebackReversal(
    request: PaymentChargebackReversalRequest,
    idempotencyKey: string,
  ): Promise<PaymentChargebackReversalResponse> {
    return this.request<PaymentChargebackReversalResponse>("/v1/payment/chargeback/reversal", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  private async request<TResponse>(
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options: RequestOptions = {},
  ): Promise<TResponse> {
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const requiresAuth = options.requiresAuth ?? true;
    const headers = this.buildHeaders(options.headers, requiresAuth, options.idempotencyKey, options.rawBody !== undefined);

    const init: RequestInit = {
      method,
      headers,
      credentials: "include",
    };
    if (options.rawBody !== undefined) {
      init.body = options.rawBody;
    } else if (options.body !== undefined) {
      init.body = JSON.stringify(options.body);
    }

    const response = await fetchImpl(this.resolveURL(path), init);
    const payload = await parseResponsePayload(response);
    if (!response.ok) {
      const message = extractErrorMessage(response.status, payload);
      throw new ApiError(message, response.status, payload);
    }

    return payload as TResponse;
  }

  private buildHeaders(extra: HeadersInit | undefined, requiresAuth: boolean, idempotencyKey: string | undefined, hasRawBody: boolean): Headers {
    const headers = new Headers(extra);
    if (!hasRawBody) {
      headers.set("Content-Type", "application/json");
    }

    if (requiresAuth && this.config.getAuthToken) {
      const token = this.config.getAuthToken();
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }
    }
    if (idempotencyKey) {
      headers.set("Idempotency-Key", idempotencyKey);
    }

    return headers;
  }

  private resolveURL(path: string): string {
    if (/^https?:\/\//i.test(path)) {
      return path;
    }
    const base = this.config.baseUrl.endsWith("/") ? this.config.baseUrl : `${this.config.baseUrl}/`;
    const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
    return new URL(normalizedPath, base).toString();
  }
}

async function parseResponsePayload(response: Response): Promise<unknown> {
  if (response.status === 204) {
    return undefined;
  }

  const contentType = response.headers.get("content-type") || "";
  if (contentType.toLowerCase().includes("application/json")) {
    try {
      return await response.json();
    } catch {
      return undefined;
    }
  }

  try {
    const text = await response.text();
    return text.length > 0 ? text : undefined;
  } catch {
    return undefined;
  }
}

function extractErrorMessage(status: number, payload: unknown): string {
  if (payload && typeof payload === "object" && "error" in payload) {
    const candidate = (payload as { error?: unknown }).error;
    if (typeof candidate === "string" && candidate.length > 0) {
      return candidate;
    }
  }
  if (typeof payload === "string" && payload.length > 0) {
    return payload;
  }
  return `request failed (${status})`;
}

function appendQuery(path: string, query: Record<string, unknown>): string {
  const entries = Object.entries(query).filter(([, value]) => value !== undefined && value !== null && `${value}`.trim() !== "");
  if (entries.length === 0) {
    return path;
  }
  const params = new URLSearchParams();
  for (const [key, value] of entries) {
    params.set(key, String(value));
  }
  return `${path}?${params.toString()}`;
}
